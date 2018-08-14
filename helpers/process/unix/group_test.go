// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package unix

import (
	"bytes"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/opencontainers/runc/libcontainer/user"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/helpers"
	logrusHelper "gitlab.com/gitlab-org/gitlab-runner/helpers/logrus"
)

func runOnFakeLeftoversLookupWaitTime(duration time.Duration, handler func(time.Duration)) {
	oldLeftoversLookupWaitTime := leftoversLookupWaitTime
	defer func() {
		leftoversLookupWaitTime = oldLeftoversLookupWaitTime
	}()

	leftoversLookupWaitTime = duration

	handler(duration)
}

func runOnFakeKillWaitTime(duration time.Duration, handler func(time.Duration)) {
	oldKillWaitTime := killWaitTime
	defer func() {
		killWaitTime = oldKillWaitTime
	}()

	killWaitTime = duration

	handler(duration)

}

func runOnFakeUserFinder(t *testing.T, handler func(user *mockFakeUserFinder)) {
	oldUserFinder := userFinder
	defer func() {
		userFinder = oldUserFinder
	}()

	uf := new(mockFakeUserFinder)
	defer uf.AssertExpectations(t)

	userFinder = uf.Find

	handler(uf)
}

func TestGroup_PrepareWithoutUser(t *testing.T) {
	runOnFakeUserFinder(t, func(uf *mockFakeUserFinder) {
		logger := logrus.NewEntry(logrus.StandardLogger())
		cmd := exec.Command("sleep", "10")

		assert.Nil(t, cmd.SysProcAttr)

		group := New(cmd, &common.ShellScriptInfo{}, logger)
		group.Prepare()

		require.NotNil(t, cmd.SysProcAttr)
		assert.True(t, cmd.SysProcAttr.Setpgid)
		assert.Nil(t, cmd.SysProcAttr.Credential)
	})
}

func TestGroup_PrepareWithUser(t *testing.T) {
	runOnFakeUserFinder(t, func(uf *mockFakeUserFinder) {
		logger := logrus.NewEntry(logrus.StandardLogger())
		cmd := exec.Command("sleep", "10")

		assert.Nil(t, cmd.SysProcAttr)

		u := user.User{
			Name: "test-user",
			Uid:  1001,
			Gid:  1001,
		}
		uf.On("Find", u.Name).Return(u, nil).Once()

		group := New(cmd, &common.ShellScriptInfo{User: u.Name}, logger)
		group.Prepare()

		require.NotNil(t, cmd.SysProcAttr)
		assert.True(t, cmd.SysProcAttr.Setpgid)

		require.NotNil(t, cmd.SysProcAttr.Credential)
		assert.Equal(t, uint32(u.Uid), cmd.SysProcAttr.Credential.Uid)
		assert.Equal(t, uint32(u.Gid), cmd.SysProcAttr.Credential.Gid)
	})
}

func TestGroup_PrepareWithUserAndFinderError(t *testing.T) {
	runOnFakeUserFinder(t, func(uf *mockFakeUserFinder) {
		logger := logrus.NewEntry(logrus.StandardLogger())
		cmd := exec.Command("sleep", "10")

		assert.Nil(t, cmd.SysProcAttr)

		uf.On("Find", mock.Anything).Return(user.User{}, errors.New("test error")).Once()

		group := New(cmd, &common.ShellScriptInfo{User: "test-user"}, logger)
		group.Prepare()

		require.NotNil(t, cmd.SysProcAttr)
		assert.True(t, cmd.SysProcAttr.Setpgid)

		assert.Nil(t, cmd.SysProcAttr.Credential)
	})
}

func runOnFakeProcessKiller(t *testing.T, handler func(processKiller *mockFakeProcessKiller)) {
	oldProcessKiller := processKiller
	defer func() {
		processKiller = oldProcessKiller
	}()

	pk := new(mockFakeProcessKiller)
	defer pk.AssertExpectations(t)

	processKiller = pk.Kill

	handler(pk)
}

func TestGroup_KillNoProcess(t *testing.T) {
	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			runOnFakeProcessKiller(t, func(pk *mockFakeProcessKiller) {
				logger := logrus.NewEntry(logrus.StandardLogger())
				cmd := exec.Command("sleep", "10")

				group := New(cmd, &common.ShellScriptInfo{}, logger)
				group.Prepare()

				group.Kill()

				assert.NotContains(t, output.String(), "Killing process")
			})
		})
	})
}

func TestGroup_KillWithFirstSigterm(t *testing.T) {
	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			runOnFakeProcessKiller(t, func(pk *mockFakeProcessKiller) {
				logger := logrus.NewEntry(logrus.StandardLogger())
				cmd := exec.Command("go", "version")

				group := New(cmd, &common.ShellScriptInfo{}, logger)
				group.Prepare()
				cmd.Start()

				require.NotNil(t, cmd.Process)
				pid := cmd.Process.Pid

				pk.On("Kill", -pid, syscall.SIGTERM).Return(nil).Once()
				pk.On("Kill", -pid, syscall.Signal(0)).Return(errors.New("no such process")).Once()

				group.Kill()

				assert.Contains(t, output.String(), "Killing process")
				assert.NotContains(t, output.String(), "SIGTERM timed out, sending SIGKILL to process group")
				assert.Contains(t, output.String(), "Main process exited after SIGTERM")
				assert.Contains(t, output.String(), "Looking for leftovers")
				assert.Contains(t, output.String(), "No leftovers, process group terminated: no such process")
			})
		})
	})
}

func TestGroup_KillWithFirstSigtermAndLeftovers(t *testing.T) {
	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			runOnFakeProcessKiller(t, func(pk *mockFakeProcessKiller) {
				logger := logrus.NewEntry(logrus.StandardLogger())
				cmd := exec.Command("go", "version")

				group := New(cmd, &common.ShellScriptInfo{}, logger)
				group.Prepare()
				cmd.Start()

				require.NotNil(t, cmd.Process)
				pid := cmd.Process.Pid

				pk.On("Kill", -pid, syscall.SIGTERM).Return(nil).Once()
				pk.On("Kill", -pid, syscall.Signal(0)).Return(nil).Once()
				pk.On("Kill", -pid, syscall.SIGKILL).Return(nil).Once()
				pk.On("Kill", -pid, syscall.Signal(0)).Return(errors.New("no such process")).Once()

				group.Kill()

				assert.Contains(t, output.String(), "Killing process")
				assert.NotContains(t, output.String(), "SIGTERM timed out, sending SIGKILL to process group")
				assert.Contains(t, output.String(), "Main process exited after SIGTERM")
				assert.Contains(t, output.String(), "Looking for leftovers")
				assert.Contains(t, output.String(), "Found leftovers")
				assert.Contains(t, output.String(), "Sending SIGKILL to process group")
				assert.Contains(t, output.String(), "No leftovers, process group terminated: no such process")
			})
		})
	})
}

func TestGroup_KillWithSigkill(t *testing.T) {
	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			runOnFakeProcessKiller(t, func(pk *mockFakeProcessKiller) {
				runOnFakeKillWaitTime(10*time.Millisecond, func(duration time.Duration) {
					logger := logrus.NewEntry(logrus.StandardLogger())
					cmd := exec.Command("sleep", "10")

					group := New(cmd, &common.ShellScriptInfo{}, logger)
					group.Prepare()
					cmd.Start()

					require.NotNil(t, cmd.Process)
					defer cmd.Process.Kill()

					pid := cmd.Process.Pid

					pk.On("Kill", -pid, syscall.SIGTERM).Return(nil).Once()
					pk.On("Kill", -pid, syscall.SIGKILL).Return(nil).Once()
					pk.On("Kill", -pid, syscall.Signal(0)).Return(errors.New("no such process")).Once()

					group.Kill()

					assert.Contains(t, output.String(), "Killing process")
					assert.NotContains(t, output.String(), "Main process exited after SIGTERM")
					assert.Contains(t, output.String(), "SIGTERM timed out, sending SIGKILL to process group")
					assert.Contains(t, output.String(), "Looking for leftovers")
					assert.Contains(t, output.String(), "No leftovers, process group terminated: no such process")
				})
			})
		})
	})
}

func TestGroup_KillWithSigkillAndLeftovers(t *testing.T) {
	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			runOnFakeProcessKiller(t, func(pk *mockFakeProcessKiller) {
				runOnFakeKillWaitTime(10*time.Millisecond, func(duration time.Duration) {
					logger := logrus.NewEntry(logrus.StandardLogger())
					cmd := exec.Command("sleep", "10")

					group := New(cmd, &common.ShellScriptInfo{}, logger)
					group.Prepare()
					cmd.Start()

					require.NotNil(t, cmd.Process)
					defer cmd.Process.Kill()

					pid := cmd.Process.Pid

					pk.On("Kill", -pid, syscall.SIGTERM).Return(nil).Once()
					pk.On("Kill", -pid, syscall.SIGKILL).Return(nil).Once()
					pk.On("Kill", -pid, syscall.Signal(0)).Return(nil).Once()
					pk.On("Kill", -pid, syscall.SIGKILL).Return(nil).Once()
					pk.On("Kill", -pid, syscall.Signal(0)).Return(errors.New("no such process")).Once()

					group.Kill()

					assert.Contains(t, output.String(), "Killing process")
					assert.NotContains(t, output.String(), "Main process exited after SIGTERM")
					assert.Contains(t, output.String(), "SIGTERM timed out, sending SIGKILL to process group")
					assert.Contains(t, output.String(), "Looking for leftovers")
					assert.Contains(t, output.String(), "No leftovers, process group terminated: no such process")
				})
			})
		})
	})
}

func TestGroup_KillCantKillLeftovers(t *testing.T) {
	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			runOnFakeProcessKiller(t, func(pk *mockFakeProcessKiller) {
				logger := logrus.NewEntry(logrus.StandardLogger())
				cmd := exec.Command("go", "version")

				group := New(cmd, &common.ShellScriptInfo{}, logger)
				group.Prepare()
				cmd.Start()

				require.NotNil(t, cmd.Process)

				pid := cmd.Process.Pid

				pk.On("Kill", -pid, syscall.SIGTERM).Return(nil).Once()
				pk.On("Kill", -pid, syscall.Signal(0)).Return(nil).Once()
				pk.On("Kill", -pid, syscall.SIGKILL).Return(nil).Once()
				pk.On("Kill", -pid, syscall.Signal(0)).Return(nil).Once()

				assert.PanicsWithValue(t, "Process couldn't be killed!", group.Kill)
			})
		})
	})
}

var simpleScript = ": | eval $'sleep 60'"
var nonTerminatableScript = ": | eval $'trap \\'sleep 70\\' SIGTERM\nsleep 60'"

func TestIntegration_KillProcessGroupForSimpleScript(t *testing.T) {
	testKillProcessGroup(t, simpleScript)
}

func TestIntegration_KillProcessGroupForNonTerminatableScript(t *testing.T) {
	testKillProcessGroup(t, nonTerminatableScript)
}

func testKillProcessGroup(t *testing.T, script string) {
	if helpers.SkipIntegrationTests(t, "bash") {
		return
	}

	logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
		runOnFakeLeftoversLookupWaitTime(10*time.Millisecond, func(fakeLeftoversLookupTime time.Duration) {
			runOnFakeKillWaitTime(1*time.Second, func(fakeKillTime time.Duration) {
				cmd, group := createTestProcess(script)
				group.Prepare()

				err := cmd.Start()
				require.NoError(t, err)

				time.Sleep(10 * fakeLeftoversLookupTime)

				require.NotNil(t, cmd.Process)

				cmdPid := cmd.Process.Pid
				childPid := findChild(cmdPid)

				assert.NoError(t, checkProcess(cmdPid))
				assert.NoError(t, checkProcess(childPid))

				group.Kill()
				time.Sleep(10 * fakeLeftoversLookupTime)

				assert.EqualError(t, checkProcess(cmdPid), "no such process", "Process check for cmdPid should return errorFinished error")
				assert.EqualError(t, checkProcess(childPid), "no such process", "Process check for childPid should return errorFinished error")
			})
		})
	})
}

func createTestProcess(script string) (*exec.Cmd, *Group) {
	command := "bash"
	arguments := []string{"--login"}
	cmd := exec.Command(command, arguments...)

	logger := logrus.NewEntry(logrus.StandardLogger())
	group := New(cmd, &common.ShellScriptInfo{}, logger)

	cmd.Stdin = bytes.NewBufferString(script)

	return cmd, group
}

func findChild(ppid int) int {
	lines, _ := exec.Command("ps", "axo", "ppid,pid").CombinedOutput()

	for _, line := range strings.Split(string(lines), "\n") {
		row := strings.Split(strings.TrimRight(line, "\n"), " ")

		var pids []int
		for _, cell := range row {
			if cell == "" {
				continue
			}

			pid, err := strconv.Atoi(cell)
			if err != nil {
				continue
			}

			pids = append(pids, pid)
		}

		if len(pids) > 0 {
			if pids[0] == ppid {
				return pids[1]
			}
		}

		if line == "" {
			break
		}
	}

	return 0
}

func checkProcess(pid int) error {
	return syscall.Kill(pid, syscall.Signal(0))
}

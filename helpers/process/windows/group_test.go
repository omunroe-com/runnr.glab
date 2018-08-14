package windows

import (
	"bytes"
	"encoding/csv"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/helpers"
	logrusHelper "gitlab.com/gitlab-org/gitlab-runner/helpers/logrus"
)

func runOnFakeProcessKiller(t *testing.T, handler func(killer *mockFakeProcessKiller)) {
	oldProcessKiller := processKiller
	defer func() {
		processKiller = oldProcessKiller
	}()

	killer := new(mockFakeProcessKiller)
	defer killer.AssertExpectations(t)

	processKiller = killer.Kill

	handler(killer)
}

func TestGroup_KillNoProcess(t *testing.T) {
	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			runOnFakeProcessKiller(t, func(killer *mockFakeProcessKiller) {
				cmd := exec.Command("powershell.exe", "-Command", "sleep 2")
				logger := logrus.NewEntry(logrus.StandardLogger())

				group := New(cmd, logger)
				group.Prepare()
				group.Kill()

				assert.NotContains(t, output.String(), "Killing process group")
			})
		})
	})
}

func TestGroup_Kill(t *testing.T) {
	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			runOnFakeProcessKiller(t, func(killer *mockFakeProcessKiller) {
				cmd := exec.Command("powershell.exe", "-Command", "sleep 5")
				logger := logrus.NewEntry(logrus.StandardLogger())

				group := New(cmd, logger)
				group.Prepare()

				err := cmd.Start()
				require.NoError(t, err)

				require.NotNil(t, cmd.Process)
				pid := cmd.Process.Pid

				killer.On("Kill", pid).Once()

				group.Kill()

				assert.Contains(t, output.String(), "Killing process group")
			})
		})
	})
}

func TestIntegration(t *testing.T) {
	if helpers.SkipIntegrationTests(t, "powershell.exe") {
		return
	}

	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			cmd := exec.Command("powershell.exe", "-Command", "sleep 60")

			logger := logrus.NewEntry(logrus.StandardLogger())
			group := New(cmd, logger)

			group.Prepare()
			err := cmd.Start()
			require.NoError(t, err)

			time.Sleep(10 * time.Millisecond)

			require.NotNil(t, cmd.Process)
			pid := cmd.Process.Pid
			assert.True(t, processExist(t, pid))

			group.Kill()
			assert.False(t, processExist(t, pid))
		})
	})
}

func processExist(t *testing.T, pid int) bool {
	cmd := exec.Command("powershell.exe", "-Command", "tasklist /fo csv /nh")
	out, err := cmd.Output()
	require.NoError(t, err)

	r := csv.NewReader(bytes.NewBuffer(out))
	records, err := r.ReadAll()
	require.NoError(t, err)

	for _, record := range records {
		rPid, err := strconv.Atoi(record[1])
		require.NoError(t, err)

		if rPid == pid {
			t.Log("found task:", record[0])
			return true
		}
	}

	return false
}

package process

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	logrusHelper "gitlab.com/gitlab-org/gitlab-runner/helpers/logrus"
)

func getSleepCommand(duration string) *exec.Cmd {
	command := "sleep.go"
	if runtime.GOOS == "windows" {
		command = "sleep.exe"
	}

	_, filename, _, _ := runtime.Caller(0)
	sleepCommandSource := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "tests", "sleep", command))

	return exec.Command("go", "run", sleepCommandSource, duration)
}

func prepareGroup(t *testing.T) (*exec.Cmd, chan bool, Group) {
	cmd := getSleepCommand("1s")
	build := &common.Build{}
	build.ID = 10
	build.GitInfo = common.GitInfo{
		RepoURL: "https://gitlab.example.com/my-namespace/my-project.git",
	}
	info := &common.ShellScriptInfo{}
	started := make(chan bool)

	g := NewGroup(cmd, build, info, started)

	return cmd, started, g
}

func TestNewGroup(t *testing.T) {
	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			cmd, started, g := prepareGroup(t)
			assert.NotNil(t, g)

			cmd.Start()
			close(started)
			cmd.Wait()

			assert.Contains(t, output.String(), "Starting process group")
		})
	})
}

func TestNewGroup_ProcessNotStarted(t *testing.T) {
	logrusHelper.RunOnHijackedLogrusOutput(func(output *bytes.Buffer) {
		logrusHelper.RunOnHijackedLogrusLevel(logrus.DebugLevel, func() {
			_, started, g := prepareGroup(t)
			assert.NotNil(t, g)

			close(started)
			assert.NotContains(t, output.String(), "Starting process group")
		})
	})
}

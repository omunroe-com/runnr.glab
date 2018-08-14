package process

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	logrusHelper "gitlab.com/gitlab-org/gitlab-runner/helpers/logrus"
)

func prepareGroup(t *testing.T) (*exec.Cmd, chan bool, Group) {
	cmd := exec.Command("sleep", "1")
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

// +build windows

package process

import (
	"os/exec"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/process/windows"
)

func newGroup(cmd *exec.Cmd, info *common.ShellScriptInfo, logger *logrus.Entry) Group {
	return windows.New(cmd, logger)
}

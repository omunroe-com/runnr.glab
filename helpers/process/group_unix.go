// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package process

import (
	"os/exec"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/process/unix"
)

func newGroup(cmd *exec.Cmd, info *common.ShellScriptInfo, logger *logrus.Entry) Group {
	return unix.New(cmd, info, logger)
}

package process

import (
	"os/exec"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

type Group interface {
	Prepare()
	Kill()
}

func NewGroup(cmd *exec.Cmd, build *common.Build, info *common.ShellScriptInfo, startedCh chan bool) Group {
	logger := newLogger(build)

	go logStart(logger, cmd, startedCh)

	return newGroup(cmd, info, logger)
}

func newLogger(build *common.Build) *logrus.Entry {
	logger := logrus.WithFields(logrus.Fields{
		"build":   build.ID,
		"repoURL": build.RepoCleanURL(),
	})

	return logger
}

func logStart(logger *logrus.Entry, cmd *exec.Cmd, startedCh chan bool) {
	<-startedCh
	process := cmd.Process

	if process == nil {
		return
	}

	logger.WithField("PID", process.Pid).Debug("Starting process group")
}

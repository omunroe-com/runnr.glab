package process

import (
	"os/exec"
	"strconv"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

func PrepareProcessGroup(cmd *exec.Cmd, shell *common.ShellConfiguration, build *common.Build, startedCh chan struct{}) {
	SetProcessGroup(cmd)
	SetCredential(cmd, shell)

	go func() {
		<-startedCh
		process := cmd.Process
		logWithFields(process.Pid, "Starting process group", logrus.Fields{
			"build":   strconv.Itoa(build.ID),
			"repoURL": build.RepoCleanURL(),
		})
	}()
}

var logFormat = "Process [%d]: %s"

func log(pgid int, message string) {
	logrus.Debugf(logFormat, pgid, message)
}

func logWithFields(pgid int, message string, fields logrus.Fields) {
	logrus.WithFields(fields).Debugf(logFormat, pgid, message)
}

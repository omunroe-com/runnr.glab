package windows

import (
	"os/exec"
	"strconv"

	"github.com/sirupsen/logrus"
)

type Group struct {
	cmd    *exec.Cmd
	logger *logrus.Entry
}

var processKiller = func(pid int) {
	exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
}

type fakeProcessKiller interface {
	Kill(pid int)
}

func (g *Group) Prepare() {}

func (g *Group) Kill() {
	if g.cmd.Process == nil {
		return
	}

	pid := g.cmd.Process.Pid
	g.logger.WithField("PID", pid).Debug("Killing process group")

	processKiller(pid)
	g.cmd.Process.Kill()
}

func New(cmd *exec.Cmd, logger *logrus.Entry) *Group {
	return &Group{
		cmd:    cmd,
		logger: logger,
	}
}

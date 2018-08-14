// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package unix

import (
	"os/exec"
	"syscall"
	"time"

	"github.com/opencontainers/runc/libcontainer/user"
	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

var (
	killWaitTime            = 10 * time.Second
	leftoversLookupWaitTime = 10 * time.Millisecond

	userFinder    = user.LookupUser
	processKiller = syscall.Kill
)

type fakeUserFinder interface {
	Find(username string) (user.User, error)
}

type fakeProcessKiller interface {
	Kill(pid int, signal syscall.Signal) error
}

type Group struct {
	cmd    *exec.Cmd
	info   *common.ShellScriptInfo
	logger *logrus.Entry
}

func (g *Group) Prepare() {
	g.setGroup()
	g.setCredentials()
}

func (g *Group) setGroup() {
	g.prepareSysProcAttr()

	// Create process group
	g.cmd.SysProcAttr.Setpgid = true
}

func (g *Group) setCredentials() {
	if g.info.User == "" {
		return
	}

	g.prepareSysProcAttr()

	foundUser, err := userFinder(g.info.User)
	if err != nil {
		return
	}

	g.cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid: uint32(foundUser.Uid),
		Gid: uint32(foundUser.Gid),
	}
}

func (g *Group) prepareSysProcAttr() {
	if g.cmd.SysProcAttr == nil {
		g.cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
}

func (g *Group) Kill() {
	if g.cmd.Process == nil {
		return
	}

	pid := g.cmd.Process.Pid
	g.log(pid, "Killing process")

	waitCh := make(chan error)
	go func() {
		waitCh <- g.cmd.Wait()
		close(waitCh)
	}()

	g.log(pid, "Sending SIGTERM to process group")
	processKiller(-pid, syscall.SIGTERM)

	select {
	case <-waitCh:
		g.log(pid, "Main process exited after SIGTERM")
	case <-time.After(killWaitTime):
		g.log(pid, "SIGTERM timed out, sending SIGKILL to process group")
		processKiller(-pid, syscall.SIGKILL)
	}

	g.killLeftovers(pid)
}

func (g *Group) killLeftovers(pid int) {
	if !g.leftoversPresent(pid) {
		return
	}

	g.log(pid, "Sending SIGKILL to process group")
	processKiller(-pid, syscall.SIGKILL)

	if !g.leftoversPresent(pid) {
		return
	}

	panic("Process couldn't be killed!")
}

func (g *Group) leftoversPresent(pid int) bool {
	g.log(pid, "Looking for leftovers")
	time.Sleep(leftoversLookupWaitTime)

	err := processKiller(-pid, syscall.Signal(0))
	if err != nil {
		g.log(pid, "No leftovers, process group terminated: %v", err)

		return false
	}

	g.log(pid, "Found leftovers")

	return true
}

func (g *Group) log(pid int, message string, msgArgs ...interface{}) {
	g.logger.WithField("PID", pid).Debugf(message, msgArgs...)
}

func New(cmd *exec.Cmd, info *common.ShellScriptInfo, logger *logrus.Entry) *Group {
	return &Group{
		cmd:    cmd,
		info:   info,
		logger: logger,
	}
}

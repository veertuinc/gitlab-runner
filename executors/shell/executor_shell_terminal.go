// +build !windows

package shell

import (
	"errors"
	"net/http"
	"os"
	"os/exec"

	"github.com/kr/pty"
	"gitlab.com/gitlab-org/gitlab-terminal"

	terminalsession "gitlab.com/gitlab-org/gitlab-runner/session/terminal"
)

type terminalConn struct {
	shellFd *os.File
}

func (t terminalConn) Start(w http.ResponseWriter, r *http.Request, timeoutCh, disconnectCh chan error) {
	proxy := terminal.NewFileDescriptorProxy(1) // one stopper: terminal exit handler

	terminalsession.ProxyTerminal(
		timeoutCh,
		disconnectCh,
		proxy.StopCh,
		func() {
			terminal.ProxyFileDescriptor(w, r, t.shellFd, proxy)
		},
	)
}

func (t terminalConn) Close() error {
	return t.shellFd.Close()
}

func (s *executor) Connect() (terminalsession.Conn, error) {
	cmd := exec.Command(s.BuildShell.Command, s.BuildShell.Arguments...)
	if cmd == nil {
		return nil, errors.New("Failed to generate shell command")
	}

	shellFD, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	session := terminalConn{shellFd: shellFD}

	return session, nil
}

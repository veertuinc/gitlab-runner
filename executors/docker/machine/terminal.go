package machine

import (
	"errors"

	"gitlab.com/gitlab-org/gitlab-runner/session/terminal"
	terminalsession "gitlab.com/gitlab-org/gitlab-runner/session/terminal"
)

func (e *machineExecutor) Connect() (terminalsession.Conn, error) {
	if term, ok := e.executor.(terminal.InteractiveTerminal); ok {
		return term.Connect()
	}

	return nil, errors.New("executor does not have terminal")
}

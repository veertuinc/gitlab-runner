package shellstest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-runner/shells"
)

type shellWriterFactory func() shells.ShellWriter

func OnEachShell(t *testing.T, f func(t *testing.T, shell string)) {
	shells := []string{"bash", "cmd", "powershell"}

	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			if helpers.SkipIntegrationTests(t, shell) {
				t.Skip()
			}

			f(t, shell)
		})

	}
}

func OnEachShellWithWriter(t *testing.T, f func(t *testing.T, shell string, writer shells.ShellWriter)) {
	writers := map[string]shellWriterFactory{
		"bash": func() shells.ShellWriter {
			return &shells.BashWriter{}
		},
		"cmd": func() shells.ShellWriter {
			return &shells.CmdWriter{}
		},
		"powershell": func() shells.ShellWriter {
			return &shells.PsWriter{}
		},
	}

	OnEachShell(t, func(t *testing.T, shell string) {
		writer, ok := writers[shell]
		require.True(t, ok, "Missing factory for %s", shell)

		f(t, shell, writer())
	})
}

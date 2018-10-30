// +build darwin dragonfly freebsd linux netbsd openbsd

package log

import (
	"os"
	"syscall"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackDumping(t *testing.T) {
	logger, hook := test.NewNullLogger()
	logger.SetFormatter(new(logrus.TextFormatter))

	stopCh := make(chan bool)

	dumpedCh, finishedCh := watchForGoroutinesDump(logger, stopCh)
	require.NotNil(t, dumpedCh)
	require.NotNil(t, finishedCh)

	proc, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	proc.Signal(syscall.SIGUSR1)

	<-dumpedCh
	logrusOutput, err := hook.LastEntry().String()
	require.NoError(t, err)
	assert.Contains(t, logrusOutput, "=== received SIGUSR1 ===")
	assert.Contains(t, logrusOutput, "*** goroutine dump...")

	close(stopCh)
	<-finishedCh
}

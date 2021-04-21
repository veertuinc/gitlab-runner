package commands

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/log/test"
)

func TestRunCommand_doJobRequest(t *testing.T) {
	returnedJob := new(common.JobResponse)

	waitForContext := func(ctx context.Context) {
		<-ctx.Done()
	}

	tests := map[string]struct {
		requestJob             func(ctx context.Context)
		passSignal             func(c *RunCommand)
		expectedContextTimeout bool
	}{
		"requestJob returns immediately": {
			requestJob:             func(_ context.Context) {},
			passSignal:             func(_ *RunCommand) {},
			expectedContextTimeout: false,
		},
		"requestJob hangs indefinitely": {
			requestJob:             waitForContext,
			passSignal:             func(_ *RunCommand) {},
			expectedContextTimeout: true,
		},
		"requestJob interrupted by interrupt signal": {
			requestJob: waitForContext,
			passSignal: func(c *RunCommand) {
				c.runInterruptSignal <- os.Interrupt
			},
			expectedContextTimeout: false,
		},
		"runFinished signal is passed": {
			requestJob: waitForContext,
			passSignal: func(c *RunCommand) {
				close(c.runFinished)
			},
			expectedContextTimeout: false,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			runner := new(common.RunnerConfig)

			network := new(common.MockNetwork)
			defer network.AssertExpectations(t)

			network.On("RequestJob", mock.Anything, *runner, mock.Anything).
				Run(func(args mock.Arguments) {
					ctx, ok := args.Get(0).(context.Context)
					require.True(t, ok)

					tt.requestJob(ctx)
				}).
				Return(returnedJob, true).
				Once()

			c := &RunCommand{
				network:            network,
				runInterruptSignal: make(chan os.Signal),
				runFinished:        make(chan bool),
			}

			ctx, cancelFn := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancelFn()

			go tt.passSignal(c)

			job, _ := c.doJobRequest(ctx, runner, nil)

			assert.Equal(t, returnedJob, job)

			if tt.expectedContextTimeout {
				assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
				return
			}
			assert.NoError(t, ctx.Err())
		})
	}
}

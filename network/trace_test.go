package network

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

var (
	jobConfig      = common.RunnerConfig{}
	jobCredentials = &common.JobCredentials{ID: -1}
	jobOutputLimit = common.RunnerConfig{OutputLimit: 1}
)

func matchJobState(jobInfo common.UpdateJobInfo, id int, state common.JobState, failureReason common.JobFailureReason) bool {
	if jobInfo.ID != id {
		return false
	}
	if jobInfo.State != state {
		return false
	}
	if jobInfo.FailureReason != failureReason {
		return false
	}
	return true
}

func generateJobInfoMatcher(id int, state common.JobState, failureReason common.JobFailureReason) interface{} {
	return mock.MatchedBy(func(jobInfo common.UpdateJobInfo) bool {
		return matchJobState(jobInfo, id, state, failureReason)
	})
}

func TestIgnoreStatusChange(t *testing.T) {
	jobInfoMatcher := generateJobInfoMatcher(jobCredentials.ID, common.Success, common.NoneFailure)

	mockNetwork := new(common.MockNetwork)
	defer mockNetwork.AssertExpectations(t)

	// expect to receive just one status
	mockNetwork.On("UpdateJob", jobConfig, jobCredentials, jobInfoMatcher).
		Return(common.UpdateSucceeded).Once()

	b, err := newJobTrace(mockNetwork, jobConfig, jobCredentials)
	require.NoError(t, err)

	b.start()
	b.Success()
	b.Fail(errors.New("test"), "script_failure")
}

func TestJobAbort(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	keepAliveUpdateMatcher := generateJobInfoMatcher(jobCredentials.ID, common.Running, common.NoneFailure)
	updateMatcher := generateJobInfoMatcher(jobCredentials.ID, common.Success, common.NoneFailure)

	mockNetwork := new(common.MockNetwork)
	defer mockNetwork.AssertExpectations(t)

	// abort while running
	mockNetwork.On("UpdateJob", jobConfig, jobCredentials, keepAliveUpdateMatcher).
		Return(common.UpdateAbort).Once()

	// try to send status at least once more
	mockNetwork.On("UpdateJob", jobConfig, jobCredentials, updateMatcher).
		Return(common.UpdateAbort).Once()

	b, err := newJobTrace(mockNetwork, jobConfig, jobCredentials)
	require.NoError(t, err)

	b.updateInterval = 0
	b.SetCancelFunc(cancel)

	b.start()
	assert.NotNil(t, <-ctx.Done(), "should abort the job")
	b.Success()
}

func TestJobOutputLimit(t *testing.T) {
	traceMessage := "abcde"

	mockNetwork := new(common.MockNetwork)
	defer mockNetwork.AssertExpectations(t)

	b, err := newJobTrace(mockNetwork, jobOutputLimit, jobCredentials)
	require.NoError(t, err)

	// prevent any UpdateJob before `b.Success()` call
	b.updateInterval = 25 * time.Second

	updateMatcher := generateJobInfoMatcher(jobCredentials.ID, common.Success, common.NoneFailure)

	receivedTrace := bytes.NewBuffer([]byte{})
	mockNetwork.On("PatchTrace", jobOutputLimit, jobCredentials, mock.Anything, mock.Anything).
		Return(1078, common.UpdateSucceeded).
		Once().
		Run(func(args mock.Arguments) {
			// the 1078 == len(data)
			data := args.Get(2).([]byte)
			receivedTrace.Write(data)
		})

	mockNetwork.On("UpdateJob", jobOutputLimit, jobCredentials, updateMatcher).
		Return(common.UpdateSucceeded).Once()

	b.start()
	// Write 5k to the buffer
	for i := 0; i < 1024; i++ {
		fmt.Fprint(b, traceMessage)
	}
	b.Success()

	expectedLogLimitExceededMsg := "Job's log exceeded limit of"

	assert.Contains(t, receivedTrace.String(), traceMessage)
	assert.Contains(t, receivedTrace.String(), expectedLogLimitExceededMsg)
}

func TestJobMasking(t *testing.T) {
	maskedValues := []string{"masked"}
	traceMessage := "This string should be masked"
	traceMaskedMessage := "This string should be [MASKED]"

	mockNetwork := new(common.MockNetwork)
	defer mockNetwork.AssertExpectations(t)

	mockNetwork.On("PatchTrace", mock.Anything, mock.Anything, []byte(traceMaskedMessage), 0).
		Return(len(traceMaskedMessage), common.UpdateSucceeded)

	mockNetwork.On("UpdateJob", mock.Anything, mock.Anything, mock.Anything).
		Return(common.UpdateSucceeded)

	jobTrace, err := newJobTrace(mockNetwork, jobConfig, jobCredentials)
	require.NoError(t, err)

	jobTrace.SetMasked(maskedValues)
	jobTrace.start()

	_, err = jobTrace.Write([]byte(traceMessage))
	require.NoError(t, err)
	jobTrace.Success()
}

func TestJobFinishTraceUpdateRetry(t *testing.T) {
	updateMatcher := generateJobInfoMatcher(jobCredentials.ID, common.Success, common.NoneFailure)

	mockNetwork := new(common.MockNetwork)
	defer mockNetwork.AssertExpectations(t)

	// accept just 3 bytes
	mockNetwork.On("PatchTrace", jobConfig, jobCredentials, []byte("My trace send"), 0).
		Return(3, common.UpdateSucceeded).Once()

	// retry when trying to send next bytes
	mockNetwork.On("PatchTrace", jobConfig, jobCredentials, []byte("trace send"), 3).
		Return(0, common.UpdateFailed).Once()

	// accept 6 more bytes
	mockNetwork.On("PatchTrace", jobConfig, jobCredentials, []byte("trace send"), 3).
		Return(9, common.UpdateSucceeded).Once()

	// restart most of trace
	mockNetwork.On("PatchTrace", jobConfig, jobCredentials, []byte("send"), 9).
		Return(6, common.UpdateRangeMismatch).Once()

	// accept rest of trace
	mockNetwork.On("PatchTrace", jobConfig, jobCredentials, []byte("ce send"), 6).
		Return(13, common.UpdateSucceeded).Once()

	mockNetwork.On("UpdateJob", jobConfig, jobCredentials, updateMatcher).
		Return(common.UpdateSucceeded).Once()

	b, err := newJobTrace(mockNetwork, jobConfig, jobCredentials)
	require.NoError(t, err)

	b.finishRetryInterval = time.Microsecond

	b.start()
	fmt.Fprint(b, "My trace send")
	b.Success()
}

func TestJobMaxTracePatchSize(t *testing.T) {
	updateMatcher := generateJobInfoMatcher(jobCredentials.ID, common.Success, common.NoneFailure)

	mockNetwork := new(common.MockNetwork)
	defer mockNetwork.AssertExpectations(t)

	// expect just 5 bytes
	mockNetwork.On("PatchTrace", jobConfig, jobCredentials, []byte("My tr"), 0).
		Return(5, common.UpdateSucceeded).Once()

	// expect next 5 bytes
	mockNetwork.On("PatchTrace", jobConfig, jobCredentials, []byte("ace s"), 5).
		Return(10, common.UpdateSucceeded).Once()

	// expect last 3 bytes
	mockNetwork.On("PatchTrace", jobConfig, jobCredentials, []byte("end"), 10).
		Return(13, common.UpdateSucceeded).Once()

	mockNetwork.On("UpdateJob", jobConfig, jobCredentials, updateMatcher).
		Return(common.UpdateSucceeded).Once()

	b, err := newJobTrace(mockNetwork, jobConfig, jobCredentials)
	require.NoError(t, err)

	b.finishRetryInterval = time.Microsecond
	b.maxTracePatchSize = 5

	b.start()
	fmt.Fprint(b, "My trace send")
	b.Success()
}

func TestJobFinishStatusUpdateRetry(t *testing.T) {
	updateMatcher := generateJobInfoMatcher(jobCredentials.ID, common.Success, common.NoneFailure)

	mockNetwork := new(common.MockNetwork)
	defer mockNetwork.AssertExpectations(t)

	// fail job 5 times
	mockNetwork.On("UpdateJob", jobConfig, jobCredentials, updateMatcher).
		Return(common.UpdateFailed).Times(5)

	// accept job
	mockNetwork.On("UpdateJob", jobConfig, jobCredentials, updateMatcher).
		Return(common.UpdateSucceeded).Once()

	b, err := newJobTrace(mockNetwork, jobConfig, jobCredentials)
	require.NoError(t, err)

	b.finishRetryInterval = time.Microsecond

	b.start()
	b.Success()
}

func TestJobIncrementalPatchSend(t *testing.T) {
	var wg sync.WaitGroup

	finalUpdateMatcher := generateJobInfoMatcher(
		jobCredentials.ID, common.Success, common.NoneFailure)

	mockNetwork := new(common.MockNetwork)
	defer mockNetwork.AssertExpectations(t)

	// ensure that PatchTrace gets executed first
	wg.Add(1)
	mockNetwork.On("PatchTrace", jobConfig, jobCredentials, []byte("test trace"), 0).
		Return(10, common.UpdateSucceeded).Once().
		Run(func(args mock.Arguments) {
			wg.Done()
		})

	// wait for the final `UpdateJob` to be executed
	mockNetwork.On("UpdateJob", jobConfig, jobCredentials, finalUpdateMatcher).
		Return(common.UpdateSucceeded).Once()

	b, err := newJobTrace(mockNetwork, jobConfig, jobCredentials)
	require.NoError(t, err)

	b.updateInterval = time.Millisecond * 10
	b.start()
	fmt.Fprint(b, "test trace")
	wg.Wait()
	b.Success()
}

func TestJobIncrementalStatusRefresh(t *testing.T) {
	var wg sync.WaitGroup

	incrementalUpdateMatcher := generateJobInfoMatcher(
		jobCredentials.ID, common.Running, common.NoneFailure)

	finalUpdateMatcher := generateJobInfoMatcher(
		jobCredentials.ID, common.Success, common.NoneFailure)

	mockNetwork := new(common.MockNetwork)
	defer mockNetwork.AssertExpectations(t)

	// ensure that incremental UpdateJob gets executed first
	wg.Add(1)
	mockNetwork.On("UpdateJob", jobConfig, jobCredentials, incrementalUpdateMatcher).
		Return(common.UpdateSucceeded).Once().
		Run(func(args mock.Arguments) {
			wg.Done()
		})

	// wait for the final `UpdateJob` to be executed
	mockNetwork.On("UpdateJob", jobConfig, jobCredentials, finalUpdateMatcher).
		Return(common.UpdateSucceeded).Once()

	b, err := newJobTrace(mockNetwork, jobConfig, jobCredentials)
	require.NoError(t, err)

	b.updateInterval = time.Millisecond * 10

	// Test for: https://gitlab.com/gitlab-org/gitlab-ce/issues/63972
	// 1. lock, to prevent incrementalUpdate to read state
	// 2. inject final state as early as possible
	b.lock.Lock()
	b.start()
	b.state = common.Success
	b.lock.Unlock()

	wg.Wait()
	b.finish()
}

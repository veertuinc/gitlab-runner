package common

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	s := MockShell{}
	s.On("GetName").Return("script-shell")
	s.On("GenerateScript", mock.Anything, mock.Anything).Return("script", nil)
	RegisterShell(&s)
}

func TestBuildRun(t *testing.T) {
	e := MockExecutor{}
	defer e.AssertExpectations(t)

	p := MockExecutorProvider{}
	defer p.AssertExpectations(t)

	// Create executor only once
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e).Once()

	// We run everything once
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	e.On("Finish", nil).Return().Once()
	e.On("Cleanup").Return().Once()

	// Run script successfully
	e.On("Shell").Return(&ShellScriptInfo{Shell: "script-shell"})
	e.On("Run", matchBuildStage(BuildStagePrepare)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageGetSources)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageRestoreCache)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageDownloadArtifacts)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageUserScript)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageAfterScript)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageArchiveCache)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageUploadOnSuccessArtifacts)).Return(nil).Once()

	RegisterExecutor("build-run-test", &p)

	successfulBuild, err := GetSuccessfulBuild()
	assert.NoError(t, err)
	build := &Build{
		JobResponse: successfulBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-run-test",
			},
		},
	}
	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.NoError(t, err)
}

func TestRetryPrepare(t *testing.T) {
	PreparationRetryInterval = 0

	e := MockExecutor{}
	defer e.AssertExpectations(t)

	p := MockExecutorProvider{}
	defer p.AssertExpectations(t)

	// Create executor
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e).Times(3)

	// Prepare plan
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("prepare failed")).Twice()
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	e.On("Cleanup").Return().Times(3)

	// Succeed a build script
	e.On("Shell").Return(&ShellScriptInfo{Shell: "script-shell"})
	e.On("Run", mock.Anything).Return(nil)
	e.On("Finish", nil).Return().Once()

	RegisterExecutor("build-run-retry-prepare", &p)

	successfulBuild, err := GetSuccessfulBuild()
	assert.NoError(t, err)
	build := &Build{
		JobResponse: successfulBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-run-retry-prepare",
			},
		},
	}
	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.NoError(t, err)
}

func TestPrepareFailure(t *testing.T) {
	PreparationRetryInterval = 0

	e := MockExecutor{}
	defer e.AssertExpectations(t)

	p := MockExecutorProvider{}
	defer p.AssertExpectations(t)

	// Create executor
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e).Times(3)

	// Prepare plan
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("prepare failed")).Times(3)
	e.On("Cleanup").Return().Times(3)

	RegisterExecutor("build-run-prepare-failure", &p)

	successfulBuild, err := GetSuccessfulBuild()
	assert.NoError(t, err)
	build := &Build{
		JobResponse: successfulBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-run-prepare-failure",
			},
		},
	}
	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.EqualError(t, err, "prepare failed")
}

func TestPrepareFailureOnBuildError(t *testing.T) {
	e := MockExecutor{}
	defer e.AssertExpectations(t)

	p := MockExecutorProvider{}
	defer p.AssertExpectations(t)

	// Create executor
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e).Times(1)

	// Prepare plan
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).
		Return(&BuildError{}).Times(1)
	e.On("Cleanup").Return().Times(1)

	RegisterExecutor("build-run-prepare-failure-on-build-error", &p)

	successfulBuild, err := GetSuccessfulBuild()
	assert.NoError(t, err)
	build := &Build{
		JobResponse: successfulBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-run-prepare-failure-on-build-error",
			},
		},
	}
	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.IsType(t, err, &BuildError{})
}

func matchBuildStage(buildStage BuildStage) interface{} {
	return mock.MatchedBy(func(cmd ExecutorCommand) bool {
		return cmd.Stage == buildStage
	})
}

func TestRunFailureRunsAfterScriptAndArtifactsOnFailure(t *testing.T) {
	e := MockExecutor{}
	defer e.AssertExpectations(t)

	p := MockExecutorProvider{}
	defer p.AssertExpectations(t)

	// Create executor
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e).Once()

	// Prepare plan
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	e.On("Cleanup").Return().Once()

	// Fail a build script
	e.On("Shell").Return(&ShellScriptInfo{Shell: "script-shell"})
	e.On("Run", matchBuildStage(BuildStagePrepare)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageGetSources)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageRestoreCache)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageDownloadArtifacts)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageUserScript)).Return(errors.New("build fail")).Once()
	e.On("Run", matchBuildStage(BuildStageAfterScript)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageUploadOnFailureArtifacts)).Return(nil).Once()
	e.On("Finish", errors.New("build fail")).Return().Once()

	RegisterExecutor("build-run-run-failure", &p)

	failedBuild, err := GetFailedBuild()
	assert.NoError(t, err)
	build := &Build{
		JobResponse: failedBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-run-run-failure",
			},
		},
	}
	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.EqualError(t, err, "build fail")
}

func TestGetSourcesRunFailure(t *testing.T) {
	e := MockExecutor{}
	defer e.AssertExpectations(t)

	p := MockExecutorProvider{}
	defer p.AssertExpectations(t)

	// Create executor
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e).Once()

	// Prepare plan
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	e.On("Cleanup").Return()

	// Fail a build script
	e.On("Shell").Return(&ShellScriptInfo{Shell: "script-shell"})
	e.On("Run", matchBuildStage(BuildStagePrepare)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageGetSources)).Return(errors.New("build fail")).Times(3)
	e.On("Run", matchBuildStage(BuildStageUploadOnFailureArtifacts)).Return(nil).Once()
	e.On("Finish", errors.New("build fail")).Return().Once()

	RegisterExecutor("build-get-sources-run-failure", &p)

	successfulBuild, err := GetSuccessfulBuild()
	assert.NoError(t, err)
	build := &Build{
		JobResponse: successfulBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-get-sources-run-failure",
			},
		},
	}

	build.Variables = append(build.Variables, JobVariable{Key: "GET_SOURCES_ATTEMPTS", Value: "3"})
	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.EqualError(t, err, "build fail")
}

func TestArtifactDownloadRunFailure(t *testing.T) {
	e := MockExecutor{}
	defer e.AssertExpectations(t)

	p := MockExecutorProvider{}
	defer p.AssertExpectations(t)

	// Create executor
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e).Once()

	// Prepare plan
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	e.On("Cleanup").Return()

	// Fail a build script
	e.On("Shell").Return(&ShellScriptInfo{Shell: "script-shell"})
	e.On("Run", matchBuildStage(BuildStagePrepare)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageGetSources)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageRestoreCache)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageDownloadArtifacts)).Return(errors.New("build fail")).Times(3)
	e.On("Run", matchBuildStage(BuildStageUploadOnFailureArtifacts)).Return(nil).Once()
	e.On("Finish", errors.New("build fail")).Return().Once()

	RegisterExecutor("build-artifacts-run-failure", &p)

	successfulBuild, err := GetSuccessfulBuild()
	assert.NoError(t, err)
	build := &Build{
		JobResponse: successfulBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-artifacts-run-failure",
			},
		},
	}

	build.Variables = append(build.Variables, JobVariable{Key: "ARTIFACT_DOWNLOAD_ATTEMPTS", Value: "3"})
	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.EqualError(t, err, "build fail")
}

func TestArtifactUploadRunFailure(t *testing.T) {
	e := MockExecutor{}
	defer e.AssertExpectations(t)

	p := MockExecutorProvider{}
	defer p.AssertExpectations(t)

	// Create executor
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e).Once()

	// Prepare plan
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	e.On("Cleanup").Return()

	// Successful build script
	e.On("Shell").Return(&ShellScriptInfo{Shell: "script-shell"}).Times(8)
	e.On("Run", matchBuildStage(BuildStagePrepare)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageGetSources)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageRestoreCache)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageDownloadArtifacts)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageUserScript)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageAfterScript)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageArchiveCache)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageUploadOnSuccessArtifacts)).Return(errors.New("upload fail")).Once()
	e.On("Finish", errors.New("upload fail")).Return().Once()

	RegisterExecutor("build-upload-artifacts-run-failure", &p)

	successfulBuild, err := GetSuccessfulBuild()
	successfulBuild.Artifacts = make(Artifacts, 1)
	successfulBuild.Artifacts[0] = Artifact{
		Name:      "my-artifact",
		Untracked: false,
		Paths:     ArtifactPaths{"cached/*"},
		When:      ArtifactWhenAlways,
	}
	assert.NoError(t, err)
	build := &Build{
		JobResponse: successfulBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-upload-artifacts-run-failure",
			},
		},
	}

	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.EqualError(t, err, "upload fail")
}

func TestRestoreCacheRunFailure(t *testing.T) {
	e := MockExecutor{}
	defer e.AssertExpectations(t)

	p := MockExecutorProvider{}
	defer p.AssertExpectations(t)

	// Create executor
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e).Once()

	// Prepare plan
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	e.On("Cleanup").Return()

	// Fail a build script
	e.On("Shell").Return(&ShellScriptInfo{Shell: "script-shell"})
	e.On("Run", matchBuildStage(BuildStagePrepare)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageGetSources)).Return(nil).Once()
	e.On("Run", matchBuildStage(BuildStageRestoreCache)).Return(errors.New("build fail")).Times(3)
	e.On("Run", matchBuildStage(BuildStageUploadOnFailureArtifacts)).Return(nil).Once()
	e.On("Finish", errors.New("build fail")).Return().Once()

	RegisterExecutor("build-cache-run-failure", &p)

	successfulBuild, err := GetSuccessfulBuild()
	assert.NoError(t, err)
	build := &Build{
		JobResponse: successfulBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-cache-run-failure",
			},
		},
	}

	build.Variables = append(build.Variables, JobVariable{Key: "RESTORE_CACHE_ATTEMPTS", Value: "3"})
	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.EqualError(t, err, "build fail")
}

func TestRunWrongAttempts(t *testing.T) {
	e := MockExecutor{}

	p := MockExecutorProvider{}
	defer p.AssertExpectations(t)

	// Create executor
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e)

	// Prepare plan
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	e.On("Cleanup").Return()

	// Fail a build script
	e.On("Shell").Return(&ShellScriptInfo{Shell: "script-shell"})
	e.On("Run", mock.Anything).Return(nil).Once()
	e.On("Run", mock.Anything).Return(errors.New("Number of attempts out of the range [1, 10] for stage: get_sources"))
	e.On("Finish", errors.New("Number of attempts out of the range [1, 10] for stage: get_sources")).Return()

	RegisterExecutor("build-run-attempt-failure", &p)

	successfulBuild, err := GetSuccessfulBuild()
	assert.NoError(t, err)
	build := &Build{
		JobResponse: successfulBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-run-attempt-failure",
			},
		},
	}

	build.Variables = append(build.Variables, JobVariable{Key: "GET_SOURCES_ATTEMPTS", Value: "0"})
	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.EqualError(t, err, "Number of attempts out of the range [1, 10] for stage: get_sources")
}

func TestRunSuccessOnSecondAttempt(t *testing.T) {
	e := MockExecutor{}
	p := MockExecutorProvider{}

	// Create executor only once
	p.On("CanCreate").Return(true).Once()
	p.On("GetDefaultShell").Return("bash").Once()
	p.On("GetFeatures", mock.Anything).Return(nil).Twice()

	p.On("Create").Return(&e).Once()

	// We run everything once
	e.On("Prepare", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	e.On("Finish", mock.Anything).Return().Twice()
	e.On("Cleanup").Return().Twice()

	// Run script successfully
	e.On("Shell").Return(&ShellScriptInfo{Shell: "script-shell"})

	e.On("Run", mock.Anything).Return(nil)
	e.On("Run", mock.Anything).Return(errors.New("build fail")).Once()
	e.On("Run", mock.Anything).Return(nil)

	RegisterExecutor("build-run-success-second-attempt", &p)

	successfulBuild, err := GetSuccessfulBuild()
	assert.NoError(t, err)
	build := &Build{
		JobResponse: successfulBuild,
		Runner: &RunnerConfig{
			RunnerSettings: RunnerSettings{
				Executor: "build-run-success-second-attempt",
			},
		},
	}

	build.Variables = append(build.Variables, JobVariable{Key: "GET_SOURCES_ATTEMPTS", Value: "3"})
	err = build.Run(&Config{}, &Trace{Writer: os.Stdout})
	assert.NoError(t, err)
}

func TestDebugTrace(t *testing.T) {
	build := &Build{}
	assert.False(t, build.IsDebugTraceEnabled(), "IsDebugTraceEnabled should be false if CI_DEBUG_TRACE is not set")

	successfulBuild, err := GetSuccessfulBuild()
	assert.NoError(t, err)

	successfulBuild.Variables = append(successfulBuild.Variables, JobVariable{"CI_DEBUG_TRACE", "false", true, true, false})
	build = &Build{
		JobResponse: successfulBuild,
	}
	assert.False(t, build.IsDebugTraceEnabled(), "IsDebugTraceEnabled should be false if CI_DEBUG_TRACE is set to false")

	successfulBuild, err = GetSuccessfulBuild()
	assert.NoError(t, err)

	successfulBuild.Variables = append(successfulBuild.Variables, JobVariable{"CI_DEBUG_TRACE", "true", true, true, false})
	build = &Build{
		JobResponse: successfulBuild,
	}
	assert.True(t, build.IsDebugTraceEnabled(), "IsDebugTraceEnabled should be true if CI_DEBUG_TRACE is set to true")
}

func TestSharedEnvVariables(t *testing.T) {
	for _, shared := range [...]bool{true, false} {
		t.Run(fmt.Sprintf("Value:%v", shared), func(t *testing.T) {
			assert := assert.New(t)
			build := Build{
				ExecutorFeatures: FeaturesInfo{Shared: shared},
			}
			vars := build.GetAllVariables().StringList()

			assert.NotNil(vars)

			present := "CI_SHARED_ENVIRONMENT=true"
			absent := "CI_DISPOSABLE_ENVIRONMENT=true"
			if !shared {
				tmp := present
				present = absent
				absent = tmp
			}

			assert.Contains(vars, present)
			assert.NotContains(vars, absent)
			// we never expose false
			assert.NotContains(vars, "CI_SHARED_ENVIRONMENT=false")
			assert.NotContains(vars, "CI_DISPOSABLE_ENVIRONMENT=false")
		})
	}
}

func TestGetRemoteURL(t *testing.T) {
	testCases := []struct {
		runner RunnerSettings
		result string
	}{
		{
			runner: RunnerSettings{
				CloneURL: "http://test.local/",
			},
			result: "http://gitlab-ci-token:1234567@test.local/h5bp/html5-boilerplate.git",
		},
		{
			runner: RunnerSettings{
				CloneURL: "https://test.local",
			},
			result: "https://gitlab-ci-token:1234567@test.local/h5bp/html5-boilerplate.git",
		},
		{
			runner: RunnerSettings{},
			result: "http://fallback.url",
		},
	}

	for _, tc := range testCases {
		build := &Build{
			Runner: &RunnerConfig{
				RunnerSettings: tc.runner,
			},
			allVariables: JobVariables{
				JobVariable{Key: "CI_JOB_TOKEN", Value: "1234567"},
				JobVariable{Key: "CI_PROJECT_PATH", Value: "h5bp/html5-boilerplate"},
			},
			JobResponse: JobResponse{
				GitInfo: GitInfo{RepoURL: "http://fallback.url"},
			},
		}

		assert.Equal(t, tc.result, build.GetRemoteURL())
	}
}

type featureFlagOnTestCase struct {
	value          string
	expectedStatus bool
	expectedError  bool
}

func TestIsFeatureFlagOn(t *testing.T) {
	hook := test.NewGlobal()

	tests := map[string]featureFlagOnTestCase{
		"no value": {
			value:          "",
			expectedStatus: false,
			expectedError:  false,
		},
		"true": {
			value:          "true",
			expectedStatus: true,
			expectedError:  false,
		},
		"1": {
			value:          "1",
			expectedStatus: true,
			expectedError:  false,
		},
		"false": {
			value:          "false",
			expectedStatus: false,
			expectedError:  false,
		},
		"0": {
			value:          "0",
			expectedStatus: false,
			expectedError:  false,
		},
		"invalid value": {
			value:          "test",
			expectedStatus: false,
			expectedError:  true,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			build := new(Build)
			build.Variables = JobVariables{
				{Key: "FF_TEST_FEATURE", Value: testCase.value},
			}

			status := build.IsFeatureFlagOn("FF_TEST_FEATURE")
			assert.Equal(t, testCase.expectedStatus, status)

			entry := hook.LastEntry()
			if testCase.expectedError {
				require.NotNil(t, entry)

				logrusOutput, err := entry.String()
				require.NoError(t, err)

				assert.Contains(t, logrusOutput, "Error while parsing the value of feature flag")
			} else {
				assert.Nil(t, entry)
			}

			hook.Reset()
		})
	}

}

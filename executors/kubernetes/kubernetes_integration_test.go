//go:build integration && kubernetes
// +build integration,kubernetes

package kubernetes_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8s "k8s.io/client-go/kubernetes"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/common/buildtest"
	"gitlab.com/gitlab-org/gitlab-runner/executors/kubernetes"
	"gitlab.com/gitlab-org/gitlab-runner/executors/kubernetes/internal/pull"
	"gitlab.com/gitlab-org/gitlab-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/container/helperimage"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/featureflags"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/retry"
	"gitlab.com/gitlab-org/gitlab-runner/session"
	"gitlab.com/gitlab-org/gitlab-runner/shells"
	"gitlab.com/gitlab-org/gitlab-runner/shells/shellstest"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateRandomNamespace(prefix string) string {
	// If running tests inside a local cluster the following command can be used to clean-up integration namespaces:
	// kubectl get namespaces -o name| grep k8s-integration| xargs kubectl delete
	return fmt.Sprintf("k8s-integration-%s-%d", prefix, rand.Uint64())
}

type kubernetesNamespaceManagerAction int64

const (
	createNamespace kubernetesNamespaceManagerAction = iota
	deleteNamespace
)

type namespaceManager struct {
	action      kubernetesNamespaceManagerAction
	namespace   string
	client      *k8s.Clientset
	maxAttempts int
	timeout     time.Duration
}

func newNamespaceManager(client *k8s.Clientset, action kubernetesNamespaceManagerAction, namespace string) namespaceManager {
	return namespaceManager{
		namespace:   namespace,
		action:      action,
		client:      client,
		maxAttempts: 3,
		timeout:     time.Minute,
	}
}

func (n namespaceManager) Run() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), n.timeout)
	defer cancel()

	switch n.action {
	case createNamespace:
		k8sNamespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:         n.namespace,
				GenerateName: n.namespace,
			},
		}
		_, err = n.client.CoreV1().Namespaces().Create(ctx, k8sNamespace, metav1.CreateOptions{})
	case deleteNamespace:
		err = n.client.CoreV1().Namespaces().Delete(ctx, n.namespace, metav1.DeleteOptions{})
	}

	return err
}

func (n namespaceManager) ShouldRetry(tries int, err error) bool {
	return tries < n.maxAttempts && err != nil
}

type featureFlagTest func(t *testing.T, flagName string, flagValue bool)

func TestRunIntegrationTestsWithFeatureFlag(t *testing.T) {
	tests := map[string]featureFlagTest{
		"testKubernetesSuccessRun":                                testKubernetesSuccessRunFeatureFlag,
		"testKubernetesMultistepRun":                              testKubernetesMultistepRunFeatureFlag,
		"testKubernetesTimeoutRun":                                testKubernetesTimeoutRunFeatureFlag,
		"testKubernetesBuildFail":                                 testKubernetesBuildFailFeatureFlag,
		"testKubernetesBuildCancel":                               testKubernetesBuildCancelFeatureFlag,
		"testKubernetesBuildLogLimitExceeded":                     testKubernetesBuildLogLimitExceededFeatureFlag,
		"testKubernetesBuildMasking":                              testKubernetesBuildMaskingFeatureFlag,
		"testKubernetesCustomClonePath":                           testKubernetesCustomClonePathFeatureFlag,
		"testKubernetesNoRootImage":                               testKubernetesNoRootImageFeatureFlag,
		"testKubernetesMissingImage":                              testKubernetesMissingImageFeatureFlag,
		"testKubernetesMissingTag":                                testKubernetesMissingTagFeatureFlag,
		"testKubernetesFailingToPullImageTwiceFeatureFlag":        testKubernetesFailingToPullImageTwiceFeatureFlag,
		"testKubernetesFailingToPullServiceImageTwiceFeatureFlag": testKubernetesFailingToPullSvcImageTwiceFeatureFlag,
		"testKubernetesFailingToPullHelperTwiceFeatureFlag":       testKubernetesFailingToPullHelperTwiceFeatureFlag,
		"testOverwriteNamespaceNotMatch":                          testOverwriteNamespaceNotMatchFeatureFlag,
		"testOverwriteServiceAccountNotMatch":                     testOverwriteServiceAccountNotMatchFeatureFlag,
		"testInteractiveTerminal":                                 testInteractiveTerminalFeatureFlag,
		"testKubernetesReplaceEnvFeatureFlag":                     testKubernetesReplaceEnvFeatureFlag,
		"testKubernetesReplaceMissingEnvVarFeatureFlag":           testKubernetesReplaceMissingEnvVarFeatureFlag,
		"testKubernetesWithNonRootSecurityContext":                testKubernetesWithNonRootSecurityContext,
		"testBuildsDirDefaultVolumeFeatureFlag":                   testBuildsDirDefaultVolumeFeatureFlag,
		"testBuildsDirVolumeMountEmptyDirFeatureFlag":             testBuildsDirVolumeMountEmptyDirFeatureFlag,
		"testBuildsDirVolumeMountHostPathFeatureFlag":             testBuildsDirVolumeMountHostPathFeatureFlag,
		"testKubernetesBashFeatureFlag":                           testKubernetesBashFeatureFlag,
		"testKubernetesContainerHookFeatureFlag":                  testKubernetesContainerHookFeatureFlag,
		"testKubernetesGarbageCollection":                         testKubernetesGarbageCollection,
	}

	featureFlags := []string{
		featureflags.UseLegacyKubernetesExecutionStrategy,
	}

	for tn, tt := range tests {
		for _, ff := range featureFlags {
			t.Run(fmt.Sprintf("%s %s true", tn, ff), func(t *testing.T) {
				tt(t, ff, true)
			})

			t.Run(fmt.Sprintf("%s %s false", tn, ff), func(t *testing.T) {
				tt(t, ff, false)
			})
		}
	}
}

func testKubernetesSuccessRunFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteSuccessfulBuild)
	build.Image.Name = common.TestDockerGitImage
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err)
}

func TestBuildScriptSections(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	shellstest.OnEachShell(t, func(t *testing.T, shell string) {
		if shell == "cmd" || shell == "pwsh" || shell == "powershell" {
			// support for pwsh and powershell tracked in https://gitlab.com/gitlab-org/gitlab-runner/-/issues/28119
			t.Skip("CMD, pwsh, powershell not supported")
		}

		build := getTestBuild(t, common.GetRemoteSuccessfulBuild)
		build.Runner.RunnerSettings.Shell = shell

		buildtest.RunBuildWithSections(t, build)
	})
}

func TestEntrypointNotIgnored(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	testCases := map[string]struct {
		useHonorEntrypointFF bool
		expectedOutputLines  []string
	}{
		"feature flag off": {
			useHonorEntrypointFF: false,
			expectedOutputLines:  []string{"I am now root", "file not found"},
		},
		"feature flag on": {
			useHonorEntrypointFF: true,
			expectedOutputLines:  []string{"I am now nobody", "this has been executed through a custom entrypoint"},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			build := getTestBuildWithImage(t, common.TestAlpineEntrypointImage, func() (common.JobResponse, error) {
				job, err := common.GetRemoteBuildResponse(
					"if [ -f /tmp/debug.log ]; then",
					"cat /tmp/debug.log",
					"else",
					"echo 'file not found'",
					"fi",
					"echo \"I am now `whoami`\"",
				)

				job.Image = common.Image{
					Name: common.TestAlpineEntrypointImage,
				}

				return job, err
			})

			build.Variables = append(
				build.Variables,
				common.JobVariable{Key: featureflags.KubernetesHonorEntrypoint, Value: strconv.FormatBool(tc.useHonorEntrypointFF)},
			)

			out, err := buildtest.RunBuildReturningOutput(t, build)
			require.NoError(t, err)

			t.Log(out)

			for _, expectedLine := range tc.expectedOutputLines {
				assert.Contains(t, out, expectedLine)
			}
		})
	}
}

func testKubernetesMultistepRunFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	successfulBuild, err := common.GetRemoteSuccessfulMultistepBuild()
	require.NoError(t, err)

	failingScriptBuild, err := common.GetRemoteFailingMultistepBuild(common.StepNameScript)
	require.NoError(t, err)

	failingReleaseBuild, err := common.GetRemoteFailingMultistepBuild("release")
	require.NoError(t, err)

	successfulBuild.Image.Name = common.TestDockerGitImage
	failingScriptBuild.Image.Name = common.TestDockerGitImage
	failingReleaseBuild.Image.Name = common.TestDockerGitImage

	tests := map[string]struct {
		jobResponse    common.JobResponse
		expectedOutput []string
		unwantedOutput []string
		errExpected    bool
	}{
		"Successful build with release and after_script step": {
			jobResponse: successfulBuild,
			expectedOutput: []string{
				"echo Hello World",
				"echo Release",
				"echo After Script",
			},
			errExpected: false,
		},
		"Failure on script step. Release is skipped. After script runs.": {
			jobResponse: failingScriptBuild,
			expectedOutput: []string{
				"echo Hello World",
				"echo After Script",
			},
			unwantedOutput: []string{
				"echo Release",
			},
			errExpected: true,
		},
		"Failure on release step. After script runs.": {
			jobResponse: failingReleaseBuild,
			expectedOutput: []string{
				"echo Hello World",
				"echo Release",
				"echo After Script",
			},
			errExpected: true,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			build := getTestBuild(t, func() (common.JobResponse, error) {
				return tt.jobResponse, nil
			})
			buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

			var buf bytes.Buffer
			err := build.Run(&common.Config{}, &common.Trace{Writer: &buf})

			out := buf.String()
			for _, output := range tt.expectedOutput {
				assert.Contains(t, out, output)
			}

			for _, output := range tt.unwantedOutput {
				assert.NotContains(t, out, output)
			}

			if tt.errExpected {
				var buildErr *common.BuildError
				assert.ErrorAs(t, err, &buildErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func testKubernetesTimeoutRunFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteLongRunningBuild)
	build.Image.Name = common.TestDockerGitImage
	build.RunnerInfo.Timeout = 10 // seconds
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	var buildError *common.BuildError
	assert.ErrorAs(t, err, &buildError)
	assert.Equal(t, common.JobExecutionTimeout, buildError.FailureReason)
}

func testKubernetesBuildFailFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteFailedBuild)
	build.Image.Name = common.TestDockerGitImage
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err, "error")
	var buildError *common.BuildError
	assert.ErrorAs(t, err, &buildError)
	assert.Contains(t, err.Error(), "command terminated with exit code 1")
	assert.Equal(t, 1, buildError.ExitCode)
}

func testKubernetesBuildCancelFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.JobResponse{}, nil
	})
	buildtest.RunBuildWithCancel(
		t,
		build.Runner,
		func(build *common.Build) {
			buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)
		},
	)
}

func testKubernetesBuildLogLimitExceededFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.JobResponse{}, nil
	})
	buildtest.RunRemoteBuildWithJobOutputLimitExceeded(
		t,
		build.Runner,
		func(build *common.Build) {
			buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)
		},
	)
}

func testKubernetesBuildMaskingFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.JobResponse{}, nil
	})
	buildtest.RunBuildWithMasking(
		t,
		build.Runner,
		func(build *common.Build) {
			buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)
		},
	)
}

func testKubernetesCustomClonePathFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	jobResponse, err := common.GetRemoteBuildResponse(
		"ls -al $CI_BUILDS_DIR/go/src/gitlab.com/gitlab-org/repo",
	)
	require.NoError(t, err)

	tests := map[string]struct {
		clonePath   string
		expectedErr bool
	}{
		"uses custom clone path": {
			clonePath:   "$CI_BUILDS_DIR/go/src/gitlab.com/gitlab-org/repo",
			expectedErr: false,
		},
		"path has to be within CI_BUILDS_DIR": {
			clonePath:   "/unknown/go/src/gitlab.com/gitlab-org/repo",
			expectedErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			build := getTestBuild(t, func() (common.JobResponse, error) {
				return jobResponse, nil
			})
			build.Runner.Environment = []string{
				"GIT_CLONE_PATH=" + test.clonePath,
			}
			buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

			err = build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
			if test.expectedErr {
				var buildErr *common.BuildError
				assert.ErrorAs(t, err, &buildErr)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func testKubernetesNoRootImageFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteSuccessfulBuildWithDumpedVariables)
	build.Image.Name = common.TestAlpineNoRootImage
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err)
}

func testKubernetesMissingImageFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteFailedBuild)
	build.Image.Name = "some/non-existing/image"
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.ErrorIs(t, err, &common.BuildError{FailureReason: common.ScriptFailure})
	assert.Contains(t, err.Error(), "image pull failed")
}

func testKubernetesMissingTagFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteFailedBuild)
	build.Image.Name = "docker:missing-tag"
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.ErrorIs(t, err, &common.BuildError{FailureReason: common.ScriptFailure})
	assert.Contains(t, err.Error(), "image pull failed")
}

func testKubernetesFailingToPullImageTwiceFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteFailedBuild)
	build.Image.Name = "some/non-existing/image"
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := runMultiPullPolicyBuild(t, build)

	var imagePullErr *pull.ImagePullError
	require.ErrorAs(t, err, &imagePullErr)
	assert.Equal(t, build.Image.Name, imagePullErr.Image)
}

func testKubernetesFailingToPullSvcImageTwiceFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteFailedBuild)
	build.Services = common.Services{
		{
			Name: "some/non-existing/image",
		},
	}
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := runMultiPullPolicyBuild(t, build)

	var imagePullErr *pull.ImagePullError
	require.ErrorAs(t, err, &imagePullErr)
	assert.Equal(t, build.Services[0].Name, imagePullErr.Image)
}

func testKubernetesFailingToPullHelperTwiceFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteFailedBuild)
	build.Runner.Kubernetes.HelperImage = "some/non-existing/image"
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := runMultiPullPolicyBuild(t, build)

	var imagePullErr *pull.ImagePullError
	require.ErrorAs(t, err, &imagePullErr)
	assert.Equal(t, build.Runner.Kubernetes.HelperImage, imagePullErr.Image)
}

func testOverwriteNamespaceNotMatchFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.JobResponse{
			GitInfo: common.GitInfo{
				Sha: "1234567890",
			},
			Image: common.Image{
				Name: "test-image",
			},
			Variables: []common.JobVariable{
				{Key: kubernetes.NamespaceOverwriteVariableName, Value: "namespace"},
			},
		}, nil
	})
	build.Runner.Kubernetes.NamespaceOverwriteAllowed = "^not_a_match$"
	build.SystemInterrupt = make(chan os.Signal, 1)
	build.Image.Name = common.TestDockerGitImage
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")
}

func testOverwriteServiceAccountNotMatchFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.JobResponse{
			GitInfo: common.GitInfo{
				Sha: "1234567890",
			},
			Image: common.Image{
				Name: "test-image",
			},
			Variables: []common.JobVariable{
				{Key: kubernetes.ServiceAccountOverwriteVariableName, Value: "service-account"},
			},
		}, nil
	})
	build.Runner.Kubernetes.ServiceAccountOverwriteAllowed = "^not_a_match$"
	build.SystemInterrupt = make(chan os.Signal, 1)
	build.Image.Name = common.TestDockerGitImage
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")
}

func testInteractiveTerminalFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	if os.Getenv("GITLAB_CI") == "true" {
		t.Skip("Skipping inside of GitLab CI check https://gitlab.com/gitlab-org/gitlab-runner/-/issues/26421")
	}

	client := getTestKubeClusterClient(t)
	secrets, err := client.CoreV1().Secrets("default").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.GetRemoteBuildResponse("sleep 5")
	})
	build.Image.Name = "docker:git"
	build.Runner.Kubernetes.BearerToken = string(secrets.Items[0].Data["token"])
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	sess, err := session.NewSession(nil)
	build.Session = sess

	outBuffer := bytes.NewBuffer(nil)
	outCh := make(chan string)

	go func() {
		err = build.Run(
			&common.Config{
				SessionServer: common.SessionServer{
					SessionTimeout: 2,
				},
			},
			&common.Trace{Writer: outBuffer},
		)
		require.NoError(t, err)

		outCh <- outBuffer.String()
	}()

	for build.Session.Handler() == nil {
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(5 * time.Second)

	srv := httptest.NewServer(build.Session.Handler())
	defer srv.Close()

	u := url.URL{
		Scheme: "ws",
		Host:   srv.Listener.Addr().String(),
		Path:   build.Session.Endpoint + "/exec",
	}
	headers := http.Header{
		"Authorization": []string{build.Session.Token},
	}
	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), headers)
	defer func() {
		resp.Body.Close()
		if conn != nil {
			_ = conn.Close()
		}
	}()
	require.NoError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusSwitchingProtocols)

	out := <-outCh
	t.Log(out)

	assert.Contains(t, out, "Terminal is connected, will time out in 2s...")
}

func testKubernetesReplaceEnvFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")
	build := getTestBuild(t, common.GetRemoteSuccessfulBuild)
	build.Image.Name = "$IMAGE:$VERSION"
	build.JobResponse.Variables = append(
		build.JobResponse.Variables,
		common.JobVariable{Key: "IMAGE", Value: "alpine"},
		common.JobVariable{Key: "VERSION", Value: "latest"},
	)
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)
	out, err := buildtest.RunBuildReturningOutput(t, build)
	require.NoError(t, err)
	assert.Contains(t, out, "alpine:latest")
}

func testKubernetesReplaceMissingEnvVarFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")
	build := getTestBuild(t, common.GetRemoteSuccessfulBuild)
	build.Image.Name = "alpine:$NOT_EXISTING_VARIABLE"
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "image pull failed: Failed to apply default image tag \"alpine:\"")
}

func testBuildsDirDefaultVolumeFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteSuccessfulBuild)
	build.Image.Name = common.TestDockerGitImage
	build.Runner.BuildsDir = "/path/to/builds/dir"
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err)

	assert.Equal(t, "/path/to/builds/dir/gitlab-org/ci-cd/gitlab-runner-pipeline-tests/gitlab-test", build.BuildDir)
}

func testBuildsDirVolumeMountEmptyDirFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteSuccessfulBuild)
	build.Image.Name = common.TestDockerGitImage
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)
	build.Runner.BuildsDir = "/path/to/builds/dir"
	build.Runner.Kubernetes.Volumes = common.KubernetesVolumes{
		EmptyDirs: []common.KubernetesEmptyDir{
			{
				Name:      "repo",
				MountPath: "/path/to/builds/dir",
				Medium:    "Memory",
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err)

	assert.Equal(t, "/path/to/builds/dir/gitlab-org/ci-cd/gitlab-runner-pipeline-tests/gitlab-test", build.BuildDir)
}

func testBuildsDirVolumeMountHostPathFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteSuccessfulBuild)
	build.Image.Name = common.TestDockerGitImage
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)
	build.Runner.Kubernetes.Volumes = common.KubernetesVolumes{
		HostPaths: []common.KubernetesHostPath{
			{
				Name:      "repo-host",
				MountPath: "/builds",
				HostPath:  "/tmp/builds",
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err)

	allVariables := build.GetAllVariables()

	assert.Equal(t, fmt.Sprintf("/builds/%s/gitlab-org/ci-cd/gitlab-runner-pipeline-tests/gitlab-test", allVariables.Get("CI_CONCURRENT_ID")), build.BuildDir)
}

// testKubernetesGarbageCollection tests the deletion of resources via garbage collector once the owning pod is deleted
func testKubernetesGarbageCollection(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	ctxTimeout := time.Minute
	client := getTestKubeClusterClient(t)

	validateResourcesCreated := func(
		t *testing.T,
		client *k8s.Clientset,
		featureFlagValue bool,
		namespace string,
		podName string,
	) {
		credentials, err := getSecrets(client, namespace, podName)
		require.NoError(t, err)
		configMaps, err := getConfigMaps(client, namespace, podName)
		require.NoError(t, err)

		assert.NotEmpty(t, credentials)
		if !featureFlagValue {
			assert.NotEmpty(t, configMaps)
		} else {
			assert.Empty(t, configMaps)
		}
	}

	validateResourcesDeleted := func(t *testing.T, client *k8s.Clientset, namespace string, podName string) {
		credentials, err := getSecrets(client, namespace, podName)
		require.NoError(t, err)
		configMaps, err := getConfigMaps(client, namespace, podName)
		require.NoError(t, err)

		assert.Empty(t, credentials)
		assert.Empty(t, configMaps)
	}

	tests := map[string]struct {
		namespace string
		init      func(t *testing.T, build *common.Build, client *k8s.Clientset, namespace string)
		finalize  func(t *testing.T, client *k8s.Clientset, namespace string)
	}{
		"pod deletion during build step": {
			namespace: "default",
		},
		"pod deletion during prepare step": {
			namespace: "default",
		},
		"pod deletion during prepare stage in custom namespace": {
			namespace: generateRandomNamespace("gc"),
			init: func(t *testing.T, build *common.Build, client *k8s.Clientset, namespace string) {
				err := retry.New(newNamespaceManager(client, createNamespace, namespace)).Run()
				require.NoError(t, err)

				credentials, err := getSecrets(client, namespace, "")
				require.NoError(t, err)
				configMaps, err := getConfigMaps(client, namespace, "")
				require.NoError(t, err)

				assert.Empty(t, credentials)
				assert.Empty(t, configMaps)
			},
			finalize: func(t *testing.T, client *k8s.Clientset, namespace string) {
				err := retry.New(newNamespaceManager(client, deleteNamespace, namespace)).Run()
				require.NoError(t, err)
			},
		},
	}

	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			build := getTestBuild(t, func() (common.JobResponse, error) {
				jobResponse, err := common.GetRemoteBuildResponse(
					"sleep 5000",
				)
				require.NoError(t, err)

				jobResponse.Credentials = []common.Credentials{
					{
						Type:     "registry",
						URL:      "http://example.com",
						Username: "user",
						Password: "password",
					},
				}

				return jobResponse, nil
			})
			build.Runner.Kubernetes.Namespace = tc.namespace
			buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

			if tc.init != nil {
				tc.init(t, build, client, tc.namespace)
			}

			deletedPodNameCh := make(chan string)
			defer buildtest.OnUserStage(build, func() {
				ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
				defer cancel()
				pods, err := client.CoreV1().Pods(tc.namespace).List(
					ctx,
					metav1.ListOptions{
						LabelSelector: labels.Set(build.Runner.Kubernetes.PodLabels).String(),
					},
				)
				require.NoError(t, err)
				require.NotEmpty(t, pods.Items)
				pod := pods.Items[0]

				validateResourcesCreated(t, client, featureFlagValue, tc.namespace, pod.Name)

				err = client.
					CoreV1().
					Pods(tc.namespace).
					Delete(ctx, pod.Name, metav1.DeleteOptions{
						PropagationPolicy: &kubernetes.PropagationPolicy,
					})
				require.NoError(t, err)

				deletedPodNameCh <- pod.Name
			})()

			out, err := buildtest.RunBuildReturningOutput(t, build)

			podName := <-deletedPodNameCh

			if !featureFlagValue {
				assert.True(t, kubernetes.IsKubernetesPodNotFoundError(err), "expected err NotFound, but got %T", err)
				assert.Contains(
					t,
					out,
					"ERROR: Job failed (system failure):",
				)
			} else {
				assert.Errorf(t, err, "command terminated with exit code 137")
			}

			assert.Contains(
				t,
				out,
				fmt.Sprintf("pods %q not found", podName),
			)

			validateResourcesDeleted(t, client, tc.namespace, podName)

			if tc.finalize != nil {
				tc.finalize(t, client, tc.namespace)
			}
		})
	}
}

// TestLogDeletionAttach tests the outcome when the log files are all deleted
func TestLogDeletionAttach(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	t.Skip("Log deletion test temporary skipped: issue https://gitlab.com/gitlab-org/gitlab-runner/-/issues/27755")

	tests := []struct {
		stage            string
		outputAssertions func(t *testing.T, out string, pod string)
	}{
		{
			stage: "step_", // Any script the user defined
			outputAssertions: func(t *testing.T, out string, pod string) {
				assert.Contains(
					t,
					out,
					"ERROR: Job failed: command terminated with exit code 100",
				)
			},
		},
		{
			stage: string(common.BuildStagePrepare),
			outputAssertions: func(t *testing.T, out string, pod string) {
				assert.Contains(
					t,
					out,
					"ERROR: Job failed: command terminated with exit code 100",
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.stage, func(t *testing.T) {
			build := getTestBuild(t, func() (common.JobResponse, error) {
				return common.GetRemoteBuildResponse(
					"sleep 5000",
				)
			})
			buildtest.SetBuildFeatureFlag(build, featureflags.UseLegacyKubernetesExecutionStrategy, false)

			deletedPodNameCh := make(chan string)
			defer buildtest.OnUserStage(build, func() {
				client := getTestKubeClusterClient(t)
				pods, err := client.
					CoreV1().
					Pods("default").
					List(context.Background(), metav1.ListOptions{
						LabelSelector: labels.Set(build.Runner.Kubernetes.PodLabels).String(),
					})
				require.NoError(t, err)
				require.NotEmpty(t, pods.Items)
				pod := pods.Items[0]
				config, err := kubernetes.GetKubeClientConfig(new(common.KubernetesConfig))
				require.NoError(t, err)
				logsPath := fmt.Sprintf("/logs-%d-%d", build.JobInfo.ProjectID, build.JobResponse.ID)
				opts := kubernetes.ExecOptions{
					Namespace: pod.Namespace,
					PodName:   pod.Name,
					Client:    client,
					Stdin:     true,
					In:        strings.NewReader(fmt.Sprintf("rm -rf %s/*", logsPath)),
					Out:       ioutil.Discard,
					Command:   []string{"/bin/sh"},
					Config:    config,
					Executor:  &kubernetes.DefaultRemoteExecutor{},
				}
				err = opts.Run()
				require.NoError(t, err)

				deletedPodNameCh <- pod.Name
			})()

			out, err := buildtest.RunBuildReturningOutput(t, build)
			require.Error(t, err)
			assert.True(t, err != nil, "No error returned")

			tt.outputAssertions(t, out, <-deletedPodNameCh)
		})
	}
}

// This test reproduces the bug reported in https://gitlab.com/gitlab-org/gitlab-runner/issues/2583
func TestPrepareIssue2583(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	namespace := "my_namespace"
	serviceAccount := "my_account"

	runnerConfig := &common.RunnerConfig{
		RunnerSettings: common.RunnerSettings{
			Executor: "kubernetes",
			Kubernetes: &common.KubernetesConfig{
				Image:                          "an/image:latest",
				Namespace:                      namespace,
				NamespaceOverwriteAllowed:      ".*",
				ServiceAccount:                 serviceAccount,
				ServiceAccountOverwriteAllowed: ".*",
			},
		},
	}

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.JobResponse{
			Variables: []common.JobVariable{
				{Key: kubernetes.NamespaceOverwriteVariableName, Value: "namespace"},
				{Key: kubernetes.ServiceAccountOverwriteVariableName, Value: "sa"},
			},
		}, nil
	})

	e := kubernetes.NewDefaultExecutorForTest()

	// TODO: handle the context properly with https://gitlab.com/gitlab-org/gitlab-runner/-/issues/27932
	prepareOptions := common.ExecutorPrepareOptions{
		Config:  runnerConfig,
		Build:   build,
		Context: context.TODO(),
	}

	err := e.Prepare(prepareOptions)
	assert.NoError(t, err)
	assert.Equal(t, namespace, runnerConfig.Kubernetes.Namespace)
	assert.Equal(t, serviceAccount, runnerConfig.Kubernetes.ServiceAccount)
}

func TestHelperImageRegistryLogs(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	for _, tt := range []bool{true, false} {
		t.Run(fmt.Sprintf("%s is %t", featureflags.GitLabRegistryHelperImage, tt), func(t *testing.T) {
			build := getTestBuild(t, common.GetRemoteSuccessfulBuild)
			build.Runner.Kubernetes.PullPolicy = common.StringOrArray{common.PullPolicyAlways}
			buildtest.SetBuildFeatureFlag(build, featureflags.GitLabRegistryHelperImage, tt)

			trace := bytes.Buffer{}
			err := build.Run(&common.Config{}, &common.Trace{Writer: &trace})
			require.NoError(t, err)

			if !tt {
				assert.Contains(
					t,
					trace.String(),
					helperimage.DockerHubWarningMessage,
				)
				return
			}
			assert.NotContains(
				t,
				trace.String(),
				helperimage.DockerHubWarningMessage,
			)
		})
	}
}

func TestDeletedPodSystemFailureDuringExecution(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	tests := []struct {
		stage            string
		outputAssertions func(t *testing.T, out string, pod string)
	}{
		{
			stage: "step_", // Any script the user defined
			outputAssertions: func(t *testing.T, out string, pod string) {
				assert.Contains(
					t,
					out,
					fmt.Sprintf("ERROR: Job failed (system failure): pods %q not found", pod),
				)
			},
		},
		{
			stage: string(common.BuildStagePrepare),
			outputAssertions: func(t *testing.T, out string, pod string) {
				assert.Contains(
					t,
					out,
					"ERROR: Job failed (system failure):",
				)

				assert.Contains(
					t,
					out,
					fmt.Sprintf("pods %q not found", pod),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.stage, func(t *testing.T) {
			build := getTestBuild(t, common.GetRemoteLongRunningBuild)

			// It's not possible to get this kind of information on the legacy execution path.
			buildtest.SetBuildFeatureFlag(build, featureflags.UseLegacyKubernetesExecutionStrategy, false)

			deletedPodNameCh := make(chan string)
			defer buildtest.OnStage(build, tt.stage, func() {
				client := getTestKubeClusterClient(t)
				pods, err := client.CoreV1().Pods("default").List(
					context.Background(),
					metav1.ListOptions{
						LabelSelector: labels.Set(build.Runner.Kubernetes.PodLabels).String(),
					},
				)
				require.NoError(t, err)
				require.NotEmpty(t, pods.Items)
				pod := pods.Items[0]
				err = client.
					CoreV1().
					Pods("default").
					Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
				require.NoError(t, err)

				deletedPodNameCh <- pod.Name
			})()

			out, err := buildtest.RunBuildReturningOutput(t, build)
			assert.True(t, kubernetes.IsKubernetesPodNotFoundError(err), "expected err NotFound, but got %T", err)

			tt.outputAssertions(t, out, <-deletedPodNameCh)
		})
	}
}

func testKubernetesWithNonRootSecurityContext(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.GetRemoteBuildResponse("id")
	})
	build.Image.Name = common.TestAlpineNoRootImage

	runAsNonRoot := true
	runAsUser := int64(1895034)
	build.Runner.Kubernetes.PodSecurityContext = common.KubernetesPodSecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		RunAsUser:    &runAsUser,
	}

	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	out, err := buildtest.RunBuildReturningOutput(t, build)
	assert.NoError(t, err)
	assert.Contains(t, out, fmt.Sprintf("uid=%d gid=0(root)", runAsUser))
}

func testKubernetesBashFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	tests := []struct {
		script              string
		expectedContent     string
		expectedErrExitCode int
	}{
		{
			script:          "export hello=world; echo \"hello $hello\"",
			expectedContent: "hello world",
		},
		{
			script:              "return 129",
			expectedErrExitCode: 129,
		},
		{
			script:              "exit 128",
			expectedErrExitCode: 128,
		},
		{
			script:              "eco 'function error'",
			expectedErrExitCode: 127,
		},
		{
			script:              "{{}",
			expectedErrExitCode: 127,
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			build := getTestBuild(t, common.GetRemoteSuccessfulBuild)
			buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

			build.Image.Name = common.TestAlpineImage
			build.Runner.Shell = "bash"
			build.JobResponse.Steps = common.Steps{
				common.Step{
					Name:   common.StepNameScript,
					Script: []string{tc.script},
				},
			}

			out, err := buildtest.RunBuildReturningOutput(t, build)
			assert.Contains(t, out, tc.expectedContent)

			if tc.expectedErrExitCode != 0 {
				var buildError *common.BuildError
				if assert.ErrorAs(t, err, &buildError) {
					assert.Equal(t, tc.expectedErrExitCode, buildError.ExitCode)
				}
			}
		})
	}
}

func testKubernetesContainerHookFeatureFlag(t *testing.T, featureFlagName string, featureFlagValue bool) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	build := getTestBuild(t, common.GetRemoteSuccessfulBuild)
	buildtest.SetBuildFeatureFlag(build, featureFlagName, featureFlagValue)

	build.Image.Name = common.TestAlpineImage

	tests := map[string]struct {
		image           string
		shell           string
		lifecycleCfg    common.KubernetesContainerLifecyle
		steps           common.Steps
		validateOutputs func(t *testing.T, out string, err error)
	}{
		"invalid hook configuration: more than one handler type": {
			lifecycleCfg: common.KubernetesContainerLifecyle{
				PreStop: &common.KubernetesLifecycleHandler{
					Exec: &common.KubernetesLifecycleExecAction{
						Command: []string{"touch", "/builds/postStart.txt"},
					},
					HTTPGet: &common.KubernetesLifecycleHTTPGet{
						Host: "localhost",
						Port: 8080,
					},
				},
			},
			validateOutputs: func(t *testing.T, out string, err error) {
				require.Error(t, err)
				assert.Contains(t, out, "ERROR: Job failed (system failure):")
				assert.Contains(t, out, "may not specify more than 1 handler type")
			},
		},
		"postStart exec hook bash": {
			steps: common.Steps{
				common.Step{
					Name: common.StepNameScript,
					Script: []string{
						"ls -l /builds",
					},
				},
			},
			lifecycleCfg: common.KubernetesContainerLifecyle{
				PostStart: &common.KubernetesLifecycleHandler{
					Exec: &common.KubernetesLifecycleExecAction{
						Command: []string{"touch", "/builds/postStart.txt"},
					},
				},
			},
			validateOutputs: func(t *testing.T, out string, err error) {
				require.NoError(t, err)
				assert.Contains(t, out, "Job succeeded")
				assert.Contains(t, out, "postStart.txt")
			},
		},
		"postStart exec hook pwsh": {
			image: common.TestPwshImage,
			shell: shells.SNPwsh,
			steps: common.Steps{
				common.Step{
					Name: common.StepNameScript,
					Script: []string{
						"Get-ChildItem /builds",
					},
				},
			},
			lifecycleCfg: common.KubernetesContainerLifecyle{
				PostStart: &common.KubernetesLifecycleHandler{
					Exec: &common.KubernetesLifecycleExecAction{
						Command: []string{"touch", "/builds/postStart.txt"},
					},
				},
			},
			validateOutputs: func(t *testing.T, out string, err error) {
				require.NoError(t, err)
				assert.Contains(t, out, "Job succeeded")
				assert.Contains(t, out, "postStart.txt")
			},
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			build.Runner.RunnerSettings.Kubernetes.ContainerLifecycle = tt.lifecycleCfg

			if tt.image != "" {
				build.Image.Name = tt.image
			}

			if tt.shell != "" {
				build.Runner.Shell = tt.shell
			}

			if tt.steps != nil {
				build.JobResponse.Steps = tt.steps
			}

			out, err := buildtest.RunBuildReturningOutput(t, build)
			tt.validateOutputs(t, out, err)
		})
	}
}

func getTestBuildWithImage(t *testing.T, image string, getJobResponse func() (common.JobResponse, error)) *common.Build {
	jobResponse, err := getJobResponse()
	assert.NoError(t, err)

	podUUID, err := helpers.GenerateRandomUUID(8)
	require.NoError(t, err)

	return &common.Build{
		JobResponse: jobResponse,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "kubernetes",
				Kubernetes: &common.KubernetesConfig{
					Image:      image,
					PullPolicy: common.StringOrArray{common.PullPolicyIfNotPresent},
					PodLabels: map[string]string{
						"test.k8s.gitlab.com/name": podUUID,
					},
				},
			},
		},
	}

}

func getTestBuild(t *testing.T, getJobResponse func() (common.JobResponse, error)) *common.Build {
	return getTestBuildWithImage(t, common.TestAlpineImage, getJobResponse)
}

func getTestKubeClusterClient(t *testing.T) *k8s.Clientset {
	config, err := kubernetes.GetKubeClientConfig(new(common.KubernetesConfig))
	require.NoError(t, err)
	client, err := k8s.NewForConfig(config)
	require.NoError(t, err)

	return client
}

// getSecrets retrieves all the secrets found in the given namespace
// with at least one ownerReference name matching the name given
// If ownerName is an empty string, all the secrets resources found are returned
func getSecrets(client *k8s.Clientset, namespace, ownerName string) ([]v1.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	credList, err := client.CoreV1().Secrets(namespace).List(
		ctx,
		metav1.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	credentials := []v1.Secret{}
	for _, cred := range credList.Items {
		if len(cred.OwnerReferences) == 1 && cred.OwnerReferences[0].Name == ownerName {
			credentials = append(credentials, cred)
		}
	}

	return credentials, nil
}

// getConfigMaps retrieves all the configMaps found in the given namespace
// with at least one ownerReference name matching the name given
// If ownerName is an empty string, all the configMaps resources found are returned
func getConfigMaps(client *k8s.Clientset, namespace, ownerName string) ([]v1.ConfigMap, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	configMapList, err := client.CoreV1().ConfigMaps(namespace).List(
		ctx,
		metav1.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	configMaps := []v1.ConfigMap{}
	for _, cfg := range configMapList.Items {
		if len(cfg.OwnerReferences) == 1 && cfg.OwnerReferences[0].Name == ownerName {
			configMaps = append(configMaps, cfg)
		}
	}

	return configMaps, nil
}

func runMultiPullPolicyBuild(t *testing.T, build *common.Build) error {
	build.Runner.Kubernetes.PullPolicy = common.StringOrArray{
		common.PullPolicyAlways,
		common.PullPolicyIfNotPresent,
	}

	outBuffer := bytes.NewBuffer(nil)

	err := build.Run(&common.Config{}, &common.Trace{Writer: outBuffer})
	require.Error(t, err)
	assert.ErrorIs(t, err, &common.BuildError{FailureReason: common.ScriptFailure})

	assert.Regexp(
		t,
		`(?s).*WARNING: Failed to pull image with policy "Always": image pull failed:.*`+
			`Attempt #2: Trying "IfNotPresent" pull policy for "some\/non-existing\/image" image.*`+
			`WARNING: Failed to pull image with policy "IfNotPresent":.*`+
			`image pull failed:.*`,
		outBuffer.String(),
	)

	return err
}

func TestKubernetesAllowedImages(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	type testDef struct {
		AllowedImages []string
		Image         string
		VerifyFn      func(*testing.T, error)
	}
	tests := map[string]testDef{
		// allowed image case
		"allowed image case": {
			AllowedImages: []string{"alpine"},
			Image:         "alpine",
			VerifyFn: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		// disallowed image case
		"disallowed image case": {
			AllowedImages: []string{"alpine"},
			Image:         "ubuntu",
			VerifyFn: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, common.ErrDisallowedImage)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			successfulBuild, err := common.GetRemoteSuccessfulBuild()
			assert.NoError(t, err)
			build := &common.Build{
				JobResponse: successfulBuild,
				Runner: &common.RunnerConfig{
					RunnerSettings: common.RunnerSettings{
						Executor: "kubernetes",
						Kubernetes: &common.KubernetesConfig{
							AllowedImages: test.AllowedImages,
						},
					},
				},
			}
			build.Image.Name = test.Image

			err = build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})

			test.VerifyFn(t, err)
		})
	}
}

func TestKubernetesAllowedServices(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	type testDef struct {
		AllowedServices []string
		Services        common.Services
		VerifyFn        func(*testing.T, error)
	}
	tests := map[string]testDef{
		"allowed service case": {
			AllowedServices: []string{"alpine", "debian"},
			Services: common.Services{
				common.Image{Name: "alpine"},
			},
			VerifyFn: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		"disallowed service case": {
			AllowedServices: []string{"alpine", "debian"},
			Services: common.Services{
				common.Image{Name: "alpine"},
				common.Image{Name: "ubuntu"},
			},
			VerifyFn: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, common.ErrDisallowedImage)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			successfulBuild, err := common.GetRemoteSuccessfulBuild()
			assert.NoError(t, err)
			build := &common.Build{
				JobResponse: successfulBuild,
				Runner: &common.RunnerConfig{
					RunnerSettings: common.RunnerSettings{
						Executor: "kubernetes",
						Kubernetes: &common.KubernetesConfig{
							AllowedServices: test.AllowedServices,
						},
					},
				},
			}
			build.Image.Name = "alpine"
			build.Services = test.Services

			err = build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})

			test.VerifyFn(t, err)
		})
	}
}

func TestCleanupProjectGitClone(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	buildtest.RunBuildWithCleanupGitClone(t, getTestBuild(t, common.GetRemoteSuccessfulBuild))
}

func TestCleanupProjectGitFetch(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	untrackedFilename := "untracked"

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.GetRemoteBuildResponse(
			buildtest.GetNewUntrackedFileIntoSubmodulesCommands(untrackedFilename, "", "")...,
		)
	})

	buildtest.RunBuildWithCleanupGitFetch(t, build, untrackedFilename)
}

func TestCleanupProjectGitSubmoduleNormal(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	untrackedFile := "untracked"
	untrackedSubmoduleFile := "untracked_submodule"

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.GetRemoteBuildResponse(
			buildtest.GetNewUntrackedFileIntoSubmodulesCommands(untrackedFile, untrackedSubmoduleFile, "")...,
		)
	})

	buildtest.RunBuildWithCleanupNormalSubmoduleStrategy(t, build, untrackedFile, untrackedSubmoduleFile)
}

func TestCleanupProjectGitSubmoduleRecursive(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	untrackedFile := "untracked"
	untrackedSubmoduleFile := "untracked_submodule"
	untrackedSubSubmoduleFile := "untracked_submodule_submodule"

	build := getTestBuild(t, func() (common.JobResponse, error) {
		return common.GetRemoteBuildResponse(
			buildtest.GetNewUntrackedFileIntoSubmodulesCommands(
				untrackedFile,
				untrackedSubmoduleFile,
				untrackedSubSubmoduleFile)...,
		)
	})

	buildtest.RunBuildWithCleanupRecursiveSubmoduleStrategy(
		t,
		build,
		untrackedFile,
		untrackedSubmoduleFile,
		untrackedSubSubmoduleFile,
	)
}

func TestKubernetesPwshFeatureFlag(t *testing.T) {
	helpers.SkipIntegrationTests(t, "kubectl", "cluster-info")

	tests := []struct {
		script              string
		expectedRegexp      string
		expectedErrExitCode int
	}{
		{
			script:         "Write-Output $PSVersionTable",
			expectedRegexp: "PSEdition +Core",
		},
		{
			script:         "return 129",
			expectedRegexp: "Job succeeded",
		},
		{
			script:              "Write-Error 'should fail'",
			expectedErrExitCode: 1,
		},
		{
			script:              "Exit 128",
			expectedErrExitCode: 128,
		},
		{
			script:              "$host.SetShouldExit(130)",
			expectedErrExitCode: 130,
		},
		{
			script:              "eco 'function error'",
			expectedErrExitCode: 1,
		},
		{
			script:              "syntax error {{}",
			expectedErrExitCode: 1,
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			build := getTestBuild(t, common.GetRemoteSuccessfulBuild)

			build.Image.Name = common.TestPwshImage
			build.Runner.Shell = shells.SNPwsh
			build.JobResponse.Steps = common.Steps{
				common.Step{
					Name:   common.StepNameScript,
					Script: []string{tc.script},
				},
			}

			out, err := buildtest.RunBuildReturningOutput(t, build)
			assert.Regexp(t, regexp.MustCompile(tc.expectedRegexp), out)

			if tc.expectedErrExitCode != 0 {
				var buildError *common.BuildError
				if assert.ErrorAs(t, err, &buildError) {
					assert.Equal(t, tc.expectedErrExitCode, buildError.ExitCode)
				}
			}
		})
	}
}

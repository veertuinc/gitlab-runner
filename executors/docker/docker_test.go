package docker

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/executors"
	"gitlab.com/gitlab-org/gitlab-runner/executors/docker/internal/networks"
	"gitlab.com/gitlab-org/gitlab-runner/executors/docker/internal/pull"
	"gitlab.com/gitlab-org/gitlab-runner/executors/docker/internal/user"
	"gitlab.com/gitlab-org/gitlab-runner/executors/docker/internal/volumes"
	"gitlab.com/gitlab-org/gitlab-runner/executors/docker/internal/volumes/parser"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/container/helperimage"
	service_test "gitlab.com/gitlab-org/gitlab-runner/helpers/container/services/test"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/docker"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/docker/auth"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/featureflags"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/test"
	"gitlab.com/gitlab-org/gitlab-runner/shells"
)

func TestMain(m *testing.M) {
	PrebuiltImagesPaths = []string{"../../out/helper-images/"}

	flag.Parse()
	os.Exit(m.Run())
}

func TestParseDeviceStringOne(t *testing.T) {
	e := new(executor)

	device, err := e.parseDeviceString("/dev/kvm")

	assert.NoError(t, err)
	assert.Equal(t, "/dev/kvm", device.PathOnHost)
	assert.Equal(t, "/dev/kvm", device.PathInContainer)
	assert.Equal(t, "rwm", device.CgroupPermissions)
}

func TestParseDeviceStringTwo(t *testing.T) {
	e := new(executor)

	device, err := e.parseDeviceString("/dev/kvm:/devices/kvm")

	assert.NoError(t, err)
	assert.Equal(t, "/dev/kvm", device.PathOnHost)
	assert.Equal(t, "/devices/kvm", device.PathInContainer)
	assert.Equal(t, "rwm", device.CgroupPermissions)
}

func TestParseDeviceStringThree(t *testing.T) {
	e := new(executor)

	device, err := e.parseDeviceString("/dev/kvm:/devices/kvm:r")

	assert.NoError(t, err)
	assert.Equal(t, "/dev/kvm", device.PathOnHost)
	assert.Equal(t, "/devices/kvm", device.PathInContainer)
	assert.Equal(t, "r", device.CgroupPermissions)
}

func TestParseDeviceStringFour(t *testing.T) {
	e := new(executor)

	_, err := e.parseDeviceString("/dev/kvm:/devices/kvm:r:oops")

	assert.Error(t, err)
}

func TestBindDeviceRequests(t *testing.T) {
	tests := []struct {
		gpus                  string
		expectedDeviceRequest []container.DeviceRequest
		expectedErr           bool
	}{
		{
			gpus: "all",
			expectedDeviceRequest: []container.DeviceRequest{
				{
					Driver:       "",
					Count:        -1,
					DeviceIDs:    nil,
					Capabilities: [][]string{{"gpu"}},
					Options:      map[string]string{},
				},
			},
		},
		{
			gpus:                  "",
			expectedDeviceRequest: nil,
		},
		{
			gpus:                  "somestring=thatshouldtriggeranerror",
			expectedDeviceRequest: nil,
			expectedErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.gpus, func(t *testing.T) {
			e := executor{
				AbstractExecutor: executors.AbstractExecutor{
					Config: common.RunnerConfig{
						RunnerSettings: common.RunnerSettings{
							Docker: &common.DockerConfig{
								Gpus: tt.gpus,
							},
						},
					},
				},
			}

			err := e.bindDeviceRequests()
			if tt.expectedErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedDeviceRequest, e.deviceRequests)
		})
	}
}

type testAllowedImageDescription struct {
	allowed       bool
	image         string
	allowedImages []string
}

var testAllowedImages = []testAllowedImageDescription{
	{true, "ruby", []string{"*"}},
	{true, "ruby:2.6", []string{"*"}},
	{true, "ruby:latest", []string{"*"}},
	{true, "library/ruby", []string{"*/*"}},
	{true, "library/ruby:2.6", []string{"*/*"}},
	{true, "library/ruby:2.6", []string{"*/*:*"}},
	{true, "my.registry.tld/library/ruby", []string{"my.registry.tld/*/*"}},
	{true, "my.registry.tld/library/ruby:2.6", []string{"my.registry.tld/*/*:*"}},
	{true, "my.registry.tld/group/subgroup/ruby", []string{"my.registry.tld/*/*/*"}},
	{true, "my.registry.tld/group/subgroup/ruby:2.6", []string{"my.registry.tld/*/*/*:*"}},
	{true, "ruby", []string{"**/*"}},
	{true, "ruby:2.6", []string{"**/*"}},
	{true, "ruby:latest", []string{"**/*"}},
	{true, "library/ruby", []string{"**/*"}},
	{true, "library/ruby:2.6", []string{"**/*"}},
	{true, "library/ruby:2.6", []string{"**/*:*"}},
	{true, "my.registry.tld/library/ruby", []string{"my.registry.tld/**/*"}},
	{true, "my.registry.tld/library/ruby:2.6", []string{"my.registry.tld/**/*:*"}},
	{true, "my.registry.tld/group/subgroup/ruby", []string{"my.registry.tld/**/*"}},
	{true, "my.registry.tld/group/subgroup/ruby:2.6", []string{"my.registry.tld/**/*:*"}},
	{false, "library/ruby", []string{"*"}},
	{false, "library/ruby:2.6", []string{"*"}},
	{false, "my.registry.tld/ruby", []string{"*"}},
	{false, "my.registry.tld/ruby:2.6", []string{"*"}},
	{false, "my.registry.tld/library/ruby", []string{"*"}},
	{false, "my.registry.tld/library/ruby:2.6", []string{"*"}},
	{false, "my.registry.tld/group/subgroup/ruby", []string{"*"}},
	{false, "my.registry.tld/group/subgroup/ruby:2.6", []string{"*"}},
	{false, "library/ruby", []string{"*/*:*"}},
	{false, "my.registry.tld/group/subgroup/ruby", []string{"my.registry.tld/*/*"}},
	{false, "my.registry.tld/group/subgroup/ruby:2.6", []string{"my.registry.tld/*/*:*"}},
	{false, "library/ruby", []string{"**/*:*"}},
}

func TestVerifyAllowedImage(t *testing.T) {
	e := new(executor)

	for _, test := range testAllowedImages {
		err := e.verifyAllowedImage(test.image, "", test.allowedImages, []string{})

		if err != nil && test.allowed {
			t.Errorf("%q must be allowed by %q", test.image, test.allowedImages)
		} else if err == nil && !test.allowed {
			t.Errorf("%q must not be allowed by %q", test.image, test.allowedImages)
		}
	}
}

func testServiceFromNamedImage(t *testing.T, description, imageName, serviceName string) {
	c := new(docker.MockClient)
	defer c.AssertExpectations(t)

	p := new(pull.MockManager)
	defer p.AssertExpectations(t)

	servicePart := fmt.Sprintf("-%s-0", strings.ReplaceAll(serviceName, "/", "__"))
	containerNameRegex, err := regexp.Compile("runner-abcdef12-project-0-concurrent-0-[^-]+" + servicePart)
	require.NoError(t, err)

	containerNameMatcher := mock.MatchedBy(containerNameRegex.MatchString)
	networkID := "network-id"

	e := &executor{
		client: c,
		info: types.Info{
			OSType:       helperimage.OSTypeLinux,
			Architecture: "amd64",
		},
		volumeParser: parser.NewLinuxParser(),
		pullManager:  p,
	}

	e.Config = common.RunnerConfig{}
	e.Config.Docker = &common.DockerConfig{}
	e.Build = &common.Build{
		ProjectRunnerID: 0,
		Runner:          &common.RunnerConfig{},
	}
	e.Build.JobInfo.ProjectID = 0
	e.Build.Runner.Token = "abcdef1234567890"
	e.Context = context.Background()

	e.helperImageInfo, err = helperimage.Get(common.REVISION, helperimage.Config{
		OSType:          e.info.OSType,
		Architecture:    e.info.Architecture,
		OperatingSystem: e.info.OperatingSystem,
	})
	require.NoError(t, err)

	err = e.createLabeler()
	require.NoError(t, err)

	e.BuildShell = &common.ShellConfiguration{
		Environment: []string{},
	}

	realServiceContainerName := e.getProjectUniqRandomizedName() + servicePart

	p.On("GetDockerImage", imageName).
		Return(&types.ImageInspect{ID: "helper-image"}, nil).
		Once()

	c.On(
		"ContainerRemove",
		e.Context,
		containerNameMatcher,
		types.ContainerRemoveOptions{RemoveVolumes: true, Force: true},
	).
		Return(nil).
		Once()

	networkContainersMap := map[string]types.EndpointResource{
		"1": {Name: realServiceContainerName},
	}

	c.On("NetworkList", e.Context, types.NetworkListOptions{}).
		Return([]types.NetworkResource{{ID: networkID, Name: "network-name", Containers: networkContainersMap}}, nil).
		Once()

	c.On("NetworkDisconnect", e.Context, networkID, containerNameMatcher, true).
		Return(nil).
		Once()

	c.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{ID: realServiceContainerName}, nil).
		Once()

	c.On("ContainerStart", e.Context, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	err = e.createVolumesManager()
	require.NoError(t, err)

	linksMap := make(map[string]*types.Container)
	err = e.createFromServiceDefinition(0, common.Image{Name: description}, linksMap)
	assert.NoError(t, err)
}

func TestServiceFromNamedImage(t *testing.T) {
	for _, test := range service_test.Services {
		t.Run(test.Description, func(t *testing.T) {
			testServiceFromNamedImage(t, test.Description, test.Image, test.Service)
		})
	}
}

func executorWithMockClient(c *docker.MockClient) *executor {
	e := &executor{client: c}
	e.Context = context.Background()
	e.Build = new(common.Build)
	return e
}

func TestHelperImageWithVariable(t *testing.T) {
	c := new(docker.MockClient)
	defer c.AssertExpectations(t)

	p := new(pull.MockManager)
	defer p.AssertExpectations(t)

	runnerImageTag := "gitlab/gitlab-runner:" + common.REVISION

	p.On("GetDockerImage", runnerImageTag).
		Return(&types.ImageInspect{ID: "helper-image"}, nil).
		Once()

	e := executorWithMockClient(c)
	e.pullManager = p

	e.Config = common.RunnerConfig{}
	e.Config.Docker = &common.DockerConfig{
		HelperImage: "gitlab/gitlab-runner:${CI_RUNNER_REVISION}",
	}

	img, err := e.getPrebuiltImage()
	assert.NoError(t, err)
	require.NotNil(t, img)
	assert.Equal(t, "helper-image", img.ID)
}

func TestPrepareBuildsDir(t *testing.T) {
	tests := map[string]struct {
		parser                  parser.Parser
		rootDir                 string
		volumes                 []string
		expectedSharedBuildsDir bool
		expectedError           string
	}{
		"rootDir mounted as host based volume": {
			parser:                  parser.NewLinuxParser(),
			rootDir:                 "/build",
			volumes:                 []string{"/build:/build"},
			expectedSharedBuildsDir: true,
		},
		"rootDir mounted as container based volume": {
			parser:                  parser.NewLinuxParser(),
			rootDir:                 "/build",
			volumes:                 []string{"/build"},
			expectedSharedBuildsDir: false,
		},
		"rootDir not mounted as volume": {
			parser:                  parser.NewLinuxParser(),
			rootDir:                 "/build",
			volumes:                 []string{"/folder:/folder"},
			expectedSharedBuildsDir: false,
		},
		"rootDir's parent mounted as volume": {
			parser:                  parser.NewLinuxParser(),
			rootDir:                 "/build/other/directory",
			volumes:                 []string{"/build/:/build"},
			expectedSharedBuildsDir: true,
		},
		"rootDir is not an absolute path": {
			parser:        parser.NewLinuxParser(),
			rootDir:       "builds",
			expectedError: "build directory needs to be an absolute path",
		},
		"rootDir is /": {
			parser:        parser.NewLinuxParser(),
			rootDir:       "/",
			expectedError: "build directory needs to be a non-root path",
		},
		"error on volume parsing": {
			parser:        parser.NewLinuxParser(),
			rootDir:       "/build",
			volumes:       []string{""},
			expectedError: "invalid volume specification",
		},
		"error on volume parser creation": {
			expectedError: `missing volume parser`,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			c := common.RunnerConfig{
				RunnerSettings: common.RunnerSettings{
					BuildsDir: test.rootDir,
					Docker: &common.DockerConfig{
						Volumes: test.volumes,
					},
				},
			}

			options := common.ExecutorPrepareOptions{
				Config: &c,
			}

			e := &executor{
				AbstractExecutor: executors.AbstractExecutor{
					Config: c,
				},
				volumeParser: test.parser,
			}

			err := e.prepareBuildsDir(options)
			if test.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, test.expectedSharedBuildsDir, e.SharedBuildsDir)
		})
	}
}

type volumesTestCase struct {
	volumes                  []string
	buildsDir                string
	gitStrategy              string
	adjustConfiguration      func(e *executor)
	volumesManagerAssertions func(*volumes.MockManager)
	clientAssertions         func(*docker.MockClient)
	createVolumeManager      bool
	expectedError            error
}

var volumesTestsDefaultBuildsDir = "/default-builds-dir"
var volumesTestsDefaultCacheDir = "/default-cache-dir"

func getExecutorForVolumesTests(t *testing.T, test volumesTestCase) (*executor, func()) {
	e := &executor{}

	clientMock := new(docker.MockClient)
	clientMock.On("Close").Return(nil).Once()

	volumesManagerMock := new(volumes.MockManager)
	if !errors.Is(test.expectedError, errVolumesManagerUndefined) {
		volumesManagerMock.On("RemoveTemporary", mock.Anything).Return(nil).Once()
	}

	oldCreateVolumesManager := createVolumesManager
	closureFn := func() {
		e.Cleanup()

		createVolumesManager = oldCreateVolumesManager

		volumesManagerMock.AssertExpectations(t)
		clientMock.AssertExpectations(t)
	}

	createVolumesManager = func(_ *executor) (volumes.Manager, error) {
		return volumesManagerMock, nil
	}

	if test.volumesManagerAssertions != nil {
		test.volumesManagerAssertions(volumesManagerMock)
	}

	if test.clientAssertions != nil {
		test.clientAssertions(clientMock)
	}

	c := common.RunnerConfig{
		RunnerCredentials: common.RunnerCredentials{
			Token: "abcdef1234567890",
		},
		RunnerSettings: common.RunnerSettings{
			BuildsDir: test.buildsDir,
			Docker: &common.DockerConfig{
				Volumes: test.volumes,
			},
		},
	}

	logger, _ := logrustest.NewNullLogger()
	e.AbstractExecutor = executors.AbstractExecutor{
		BuildLogger: common.NewBuildLogger(&common.Trace{Writer: ioutil.Discard}, logger.WithField("test", t.Name())),
		Build: &common.Build{
			ProjectRunnerID: 0,
			Runner:          &c,
			JobResponse: common.JobResponse{
				JobInfo: common.JobInfo{
					ProjectID: 0,
				},
				GitInfo: common.GitInfo{
					RepoURL: "https://gitlab.example.com/group/project.git",
				},
			},
		},
		Config: c,
		ExecutorOptions: executors.ExecutorOptions{
			DefaultBuildsDir: volumesTestsDefaultBuildsDir,
			DefaultCacheDir:  volumesTestsDefaultCacheDir,
		},
	}
	e.client = clientMock
	e.info = types.Info{
		OSType: helperimage.OSTypeLinux,
	}

	e.Build.Variables = append(e.Build.Variables, common.JobVariable{
		Key:   "GIT_STRATEGY",
		Value: test.gitStrategy,
	})

	if test.adjustConfiguration != nil {
		test.adjustConfiguration(e)
	}

	err := e.Build.StartBuild(
		e.RootDir(),
		e.CacheDir(),
		e.CustomBuildEnabled(),
		e.SharedBuildsDir,
	)
	require.NoError(t, err)

	if test.createVolumeManager {
		err = e.createVolumesManager()
		require.NoError(t, err)
	}

	return e, closureFn
}

func TestCreateVolumes(t *testing.T) {
	tests := map[string]volumesTestCase{
		"volumes manager not created": {
			expectedError: errVolumesManagerUndefined,
		},
		"no volumes defined, empty buildsDir, clone strategy, no errors": {
			gitStrategy:         "clone",
			createVolumeManager: true,
		},
		"no volumes defined, defined buildsDir, clone strategy, no errors": {
			buildsDir:           "/builds",
			gitStrategy:         "clone",
			createVolumeManager: true,
		},
		"no volumes defined, defined buildsDir, fetch strategy, no errors": {
			buildsDir:           "/builds",
			gitStrategy:         "fetch",
			createVolumeManager: true,
		},
		"volumes defined, empty buildsDir, clone strategy, no errors on user volume": {
			volumes:     []string{"/volume"},
			gitStrategy: "clone",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/volume").
					Return(nil).
					Once()
			},
			createVolumeManager: true,
		},
		"volumes defined, empty buildsDir, clone strategy, cache containers disabled error on user volume": {
			volumes:     []string{"/volume"},
			gitStrategy: "clone",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/volume").
					Return(volumes.ErrCacheVolumesDisabled).
					Once()
			},
			createVolumeManager: true,
		},
		"volumes defined, empty buildsDir, clone strategy, cache containers disabled wrapped error on user volume": {
			volumes:     []string{"/volume"},
			gitStrategy: "clone",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/volume").
					Return(fmt.Errorf("wrap: %w", volumes.ErrCacheVolumesDisabled)).
					Once()
			},
			createVolumeManager: true,
		},
		"volumes defined, empty buildsDir, clone strategy, duplicated error on user volume": {
			volumes:     []string{"/volume"},
			gitStrategy: "clone",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/volume").
					Return(volumes.NewErrVolumeAlreadyDefined("/volume")).
					Once()
			},
			createVolumeManager: true,
			expectedError:       volumes.NewErrVolumeAlreadyDefined("/volume"),
		},
		"volumes defined, empty buildsDir, clone strategy, other error on user volume": {
			volumes:     []string{"/volume"},
			gitStrategy: "clone",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/volume").
					Return(errors.New("test-error")).
					Once()
			},
			createVolumeManager: true,
			expectedError:       errors.New("test-error"),
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			e, closureFn := getExecutorForVolumesTests(t, test)
			defer closureFn()

			err := e.createVolumes()
			assert.Equal(t, test.expectedError, err)
		})
	}
}

func TestCreateBuildVolume(t *testing.T) {
	tests := map[string]volumesTestCase{
		"volumes manager not created": {
			expectedError: errVolumesManagerUndefined,
		},
		"git strategy clone, empty buildsDir, no error": {
			gitStrategy: "clone",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("CreateTemporary", mock.Anything, volumesTestsDefaultBuildsDir).
					Return(nil).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy clone, empty buildsDir, duplicated error": {
			gitStrategy: "clone",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("CreateTemporary", mock.Anything, volumesTestsDefaultBuildsDir).
					Return(volumes.NewErrVolumeAlreadyDefined(volumesTestsDefaultBuildsDir)).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy clone, empty buildsDir, other error": {
			gitStrategy: "clone",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("CreateTemporary", mock.Anything, volumesTestsDefaultBuildsDir).
					Return(errors.New("test-error")).
					Once()
			},
			createVolumeManager: true,
			expectedError:       errors.New("test-error"),
		},
		"git strategy clone, non-empty buildsDir, no error": {
			gitStrategy: "clone",
			buildsDir:   "/builds",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("CreateTemporary", mock.Anything, "/builds").
					Return(nil).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy clone, non-empty buildsDir, duplicated error": {
			gitStrategy: "clone",
			buildsDir:   "/builds",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("CreateTemporary", mock.Anything, "/builds").
					Return(volumes.NewErrVolumeAlreadyDefined("/builds")).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy clone, non-empty buildsDir, other error": {
			gitStrategy: "clone",
			buildsDir:   "/builds",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("CreateTemporary", mock.Anything, "/builds").
					Return(errors.New("test-error")).
					Once()
			},
			createVolumeManager: true,
			expectedError:       errors.New("test-error"),
		},
		"git strategy fetch, empty buildsDir, no error": {
			gitStrategy: "fetch",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, volumesTestsDefaultBuildsDir).
					Return(nil).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy fetch, empty buildsDir, duplicated error": {
			gitStrategy: "fetch",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, volumesTestsDefaultBuildsDir).
					Return(volumes.NewErrVolumeAlreadyDefined(volumesTestsDefaultBuildsDir)).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy fetch, empty buildsDir, other error": {
			gitStrategy: "fetch",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, volumesTestsDefaultBuildsDir).
					Return(errors.New("test-error")).
					Once()
			},
			createVolumeManager: true,
			expectedError:       errors.New("test-error"),
		},
		"git strategy fetch, non-empty buildsDir, no error": {
			gitStrategy: "fetch",
			buildsDir:   "/builds",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/builds").
					Return(nil).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy fetch, non-empty buildsDir, duplicated error": {
			gitStrategy: "fetch",
			buildsDir:   "/builds",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/builds").
					Return(volumes.NewErrVolumeAlreadyDefined("/builds")).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy fetch, non-empty buildsDir, wrapped duplicated error": {
			gitStrategy: "fetch",
			buildsDir:   "/builds",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/builds").
					Return(fmt.Errorf("wrap: %w", volumes.NewErrVolumeAlreadyDefined("/builds"))).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy fetch, non-empty buildsDir, other error": {
			gitStrategy: "fetch",
			buildsDir:   "/builds",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/builds").
					Return(errors.New("test-error")).
					Once()
			},
			createVolumeManager: true,
			expectedError:       errors.New("test-error"),
		},
		"git strategy fetch, non-empty buildsDir, cache volumes disabled": {
			gitStrategy: "fetch",
			buildsDir:   "/builds",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/builds").
					Return(volumes.ErrCacheVolumesDisabled).
					Once()
				vm.On("CreateTemporary", mock.Anything, "/builds").
					Return(nil).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy fetch, non-empty buildsDir, cache volumes disabled wrapped error": {
			gitStrategy: "fetch",
			buildsDir:   "/builds",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/builds").
					Return(fmt.Errorf("wrap: %w", volumes.ErrCacheVolumesDisabled)).
					Once()
				vm.On("CreateTemporary", mock.Anything, "/builds").
					Return(nil).
					Once()
			},
			createVolumeManager: true,
		},
		"git strategy fetch, non-empty buildsDir, cache volumes disabled, duplicated error": {
			gitStrategy: "fetch",
			buildsDir:   "/builds",
			volumesManagerAssertions: func(vm *volumes.MockManager) {
				vm.On("Create", mock.Anything, "/builds").
					Return(volumes.ErrCacheVolumesDisabled).
					Once()
				vm.On("CreateTemporary", mock.Anything, "/builds").
					Return(volumes.NewErrVolumeAlreadyDefined("/builds")).
					Once()
			},
			createVolumeManager: true,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			e, closureFn := getExecutorForVolumesTests(t, test)
			defer closureFn()

			err := e.createBuildVolume()
			assert.Equal(t, test.expectedError, err)
		})
	}
}

func TestCreateDependencies(t *testing.T) {
	const containerID = "container-ID"
	containerNameRegex, err := regexp.Compile("runner-abcdef12-project-0-concurrent-0-[^-]+-alpine-0")
	require.NoError(t, err)

	containerNameMatcher := mock.MatchedBy(containerNameRegex.MatchString)
	testError := errors.New("test-error")

	testCase := volumesTestCase{
		buildsDir: "/builds",
		volumes:   []string{"/volume"},
		adjustConfiguration: func(e *executor) {
			e.Build.Services = append(e.Build.Services, common.Image{
				Name: "alpine:latest",
			})

			e.BuildShell = &common.ShellConfiguration{
				Environment: []string{},
			}
		},
		volumesManagerAssertions: func(vm *volumes.MockManager) {
			binds := make([]string, 0)

			vm.On("CreateTemporary", mock.Anything, "/builds").
				Return(nil).
				Run(func(args mock.Arguments) {
					binds = append(binds, args.Get(1).(string))
				}).
				Once()
			vm.On("Create", mock.Anything, "/volume").
				Return(nil).
				Run(func(args mock.Arguments) {
					binds = append(binds, args.Get(1).(string))
				}).
				Once()
			vm.On("Binds").
				Return(func() []string {
					return binds
				}).
				Once()
		},
		clientAssertions: func(c *docker.MockClient) {
			hostConfigMatcher := mock.MatchedBy(func(conf *container.HostConfig) bool {
				return assert.Equal(t, []string{"/volume", "/builds"}, conf.Binds)
			})

			c.On("ImageInspectWithRaw", mock.Anything, "alpine:latest").
				Return(types.ImageInspect{}, nil, nil).
				Once()
			c.On("NetworkList", mock.Anything, mock.Anything).
				Return(nil, nil).
				Times(2)
			c.On("ContainerRemove", mock.Anything, containerNameMatcher, mock.Anything).
				Return(nil).
				Once()
			c.On("ContainerRemove", mock.Anything, containerID, mock.Anything).
				Return(nil).
				Once()
			c.On(
				"ContainerCreate",
				mock.Anything,
				mock.Anything,
				hostConfigMatcher,
				mock.Anything,
				containerNameMatcher,
			).
				Return(container.ContainerCreateCreatedBody{ID: containerID}, nil).
				Once()
			c.On("ContainerStart", mock.Anything, containerID, mock.Anything).
				Return(testError).
				Once()
		},
	}

	e, closureFn := getExecutorForVolumesTests(t, testCase)
	defer closureFn()

	err = e.createDependencies()
	assert.Equal(t, testError, err)
}

type containerConfigExpectations func(*testing.T, *container.Config, *container.HostConfig)

type dockerConfigurationTestFakeDockerClient struct {
	docker.MockClient

	cce containerConfigExpectations
	t   *testing.T
}

func (c *dockerConfigurationTestFakeDockerClient) ContainerCreate(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	containerName string,
) (container.ContainerCreateCreatedBody, error) {
	c.cce(c.t, config, hostConfig)
	return container.ContainerCreateCreatedBody{ID: "abc"}, nil
}

func prepareTestDockerConfiguration(
	t *testing.T,
	dockerConfig *common.DockerConfig,
	cce containerConfigExpectations,
) (*dockerConfigurationTestFakeDockerClient, *executor) {
	c := &dockerConfigurationTestFakeDockerClient{
		cce: cce,
		t:   t,
	}

	e := new(executor)
	e.client = c
	e.volumeParser = parser.NewLinuxParser()
	e.info = types.Info{
		OSType:       helperimage.OSTypeLinux,
		Architecture: "amd64",
	}
	e.Config.Docker = dockerConfig
	e.Build = &common.Build{
		Runner: &common.RunnerConfig{},
	}
	e.Build.Token = "abcd123456"
	e.BuildShell = &common.ShellConfiguration{
		Environment: []string{},
	}
	var err error
	e.helperImageInfo, err = helperimage.Get(common.REVISION, helperimage.Config{
		OSType:          e.info.OSType,
		Architecture:    e.info.Architecture,
		OperatingSystem: e.info.OperatingSystem,
	})
	require.NoError(t, err)

	err = e.createLabeler()
	require.NoError(t, err)

	c.On("ImageInspectWithRaw", mock.Anything, "alpine").
		Return(types.ImageInspect{ID: "123"}, []byte{}, nil).Twice()
	c.On("ImagePullBlocking", mock.Anything, "alpine:latest", mock.Anything).
		Return(nil).Once()
	c.On("NetworkList", mock.Anything, mock.Anything).
		Return([]types.NetworkResource{}, nil).Once()
	c.On("ContainerRemove", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	return c, e
}

func testDockerConfigurationWithJobContainer(
	t *testing.T,
	dockerConfig *common.DockerConfig,
	cce containerConfigExpectations,
) {
	c, e := prepareTestDockerConfiguration(t, dockerConfig, cce)
	defer c.AssertExpectations(t)

	c.On("ContainerInspect", mock.Anything, "abc").
		Return(types.ContainerJSON{}, nil).Once()

	err := e.createVolumesManager()
	require.NoError(t, err)

	err = e.createPullManager()
	require.NoError(t, err)

	_, err = e.createContainer("build", common.Image{Name: "alpine"}, []string{"/bin/sh"}, []string{})
	assert.NoError(t, err, "Should create container without errors")
}

func testDockerConfigurationWithServiceContainer(
	t *testing.T,
	dockerConfig *common.DockerConfig,
	cce containerConfigExpectations,
) {
	c, e := prepareTestDockerConfiguration(t, dockerConfig, cce)
	defer c.AssertExpectations(t)

	c.On("ContainerStart", mock.Anything, "abc", mock.Anything).
		Return(nil).Once()

	err := e.createVolumesManager()
	require.NoError(t, err)

	err = e.createPullManager()
	require.NoError(t, err)

	_, err = e.createService(
		0,
		"build",
		"latest",
		"alpine",
		common.Image{Command: []string{"/bin/sh"}},
		nil,
	)
	assert.NoError(t, err, "Should create service container without errors")
}

func TestDockerMemorySetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		Memory: "42m",
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		assert.Equal(t, int64(44040192), hostConfig.Memory)
	}

	testDockerConfigurationWithJobContainer(t, dockerConfig, cce)
}

func TestDockerMemorySwapSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		MemorySwap: "2g",
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		assert.Equal(t, int64(2147483648), hostConfig.MemorySwap)
	}

	testDockerConfigurationWithJobContainer(t, dockerConfig, cce)
}

func TestDockerMemoryReservationSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		MemoryReservation: "64m",
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		assert.Equal(t, int64(67108864), hostConfig.MemoryReservation)
	}

	testDockerConfigurationWithJobContainer(t, dockerConfig, cce)
}

func TestDockerCPUSSetting(t *testing.T) {
	examples := []struct {
		cpus     string
		nanocpus int64
	}{
		{"0.5", 500000000},
		{"0.25", 250000000},
		{"1/3", 333333333},
		{"1/8", 125000000},
		{"0.0001", 100000},
	}

	for _, example := range examples {
		t.Run(example.cpus, func(t *testing.T) {
			dockerConfig := &common.DockerConfig{
				CPUS: example.cpus,
			}

			cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
				assert.Equal(t, example.nanocpus, hostConfig.NanoCPUs)
			}

			testDockerConfigurationWithJobContainer(t, dockerConfig, cce)
		})
	}
}

func TestDockerCPUSetCPUsSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		CPUSetCPUs: "1-3,5",
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		assert.Equal(t, "1-3,5", hostConfig.CpusetCpus)
	}

	testDockerConfigurationWithJobContainer(t, dockerConfig, cce)
}

func TestDockerServicesTmpfsSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		ServicesTmpfs: map[string]string{
			"/tmpfs": "rw,noexec",
		},
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		require.NotEmpty(t, hostConfig.Tmpfs)
	}

	testDockerConfigurationWithServiceContainer(t, dockerConfig, cce)
}

func TestDockerTmpfsSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		Tmpfs: map[string]string{
			"/tmpfs": "rw,noexec",
		},
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		require.NotEmpty(t, hostConfig.Tmpfs)
	}

	testDockerConfigurationWithJobContainer(t, dockerConfig, cce)
}

func TestDockerServicesDNSSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		DNS: []string{"2001:db8::1", "192.0.2.1"},
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		require.Equal(t, dockerConfig.DNS, hostConfig.DNS)
	}

	testDockerConfigurationWithServiceContainer(t, dockerConfig, cce)
}

func TestDockerServicesDNSSearchSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		DNSSearch: []string{"mydomain.example"},
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		require.Equal(t, dockerConfig.DNSSearch, hostConfig.DNSSearch)
	}

	testDockerConfigurationWithServiceContainer(t, dockerConfig, cce)
}

func TestDockerServicesExtraHostsSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		ExtraHosts: []string{"foo.example:2001:db8::1", "bar.example:192.0.2.1"},
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		require.Equal(t, dockerConfig.ExtraHosts, hostConfig.ExtraHosts)
	}

	testDockerConfigurationWithServiceContainer(t, dockerConfig, cce)
}

func TestDockerServiceUserNSSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{}
	dockerConfigWithHostUsernsMode := &common.DockerConfig{
		UsernsMode: "host",
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		assert.Equal(t, container.UsernsMode(""), hostConfig.UsernsMode)
	}
	cceWithHostUsernsMode := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		assert.Equal(t, container.UsernsMode("host"), hostConfig.UsernsMode)
	}

	testDockerConfigurationWithServiceContainer(t, dockerConfig, cce)
	testDockerConfigurationWithServiceContainer(t, dockerConfigWithHostUsernsMode, cceWithHostUsernsMode)
}

func TestDockerUserNSSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{}
	dockerConfigWithHostUsernsMode := &common.DockerConfig{
		UsernsMode: "host",
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		assert.Equal(t, container.UsernsMode(""), hostConfig.UsernsMode)
	}
	cceWithHostUsernsMode := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		assert.Equal(t, container.UsernsMode("host"), hostConfig.UsernsMode)
	}

	testDockerConfigurationWithJobContainer(t, dockerConfig, cce)
	testDockerConfigurationWithJobContainer(t, dockerConfigWithHostUsernsMode, cceWithHostUsernsMode)
}

func TestDockerRuntimeSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		Runtime: "runc",
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		assert.Equal(t, "runc", hostConfig.Runtime)
	}

	testDockerConfigurationWithJobContainer(t, dockerConfig, cce)
}

func TestDockerSysctlsSetting(t *testing.T) {
	dockerConfig := &common.DockerConfig{
		SysCtls: map[string]string{
			"net.ipv4.ip_forward": "1",
		},
	}

	cce := func(t *testing.T, config *container.Config, hostConfig *container.HostConfig) {
		assert.Equal(t, "1", hostConfig.Sysctls["net.ipv4.ip_forward"])
	}

	testDockerConfigurationWithJobContainer(t, dockerConfig, cce)
}

type networksTestCase struct {
	clientAssertions          func(*docker.MockClient)
	networksManagerAssertions func(*networks.MockManager)
	createNetworkManager      bool
	networkPerBuild           string
	expectedBuildError        error
	expectedCleanError        error
}

func TestDockerCreateNetwork(t *testing.T) {
	testErr := errors.New("test-err")

	tests := map[string]networksTestCase{
		"networks manager not created": {
			networkPerBuild:    "false",
			expectedBuildError: errNetworksManagerUndefined,
			expectedCleanError: errNetworksManagerUndefined,
		},
		"network not created": {
			createNetworkManager: true,
			networkPerBuild:      "false",
			networksManagerAssertions: func(nm *networks.MockManager) {
				nm.On("Create", mock.Anything, mock.Anything).
					Return(container.NetworkMode("test"), nil).
					Once()
				nm.On("Inspect", mock.Anything).
					Return(types.NetworkResource{}, nil).
					Once()
				nm.On("Cleanup", mock.Anything).
					Return(nil).
					Once()
			},
		},
		"network created": {
			createNetworkManager: true,
			networkPerBuild:      "true",
			networksManagerAssertions: func(nm *networks.MockManager) {
				nm.On("Create", mock.Anything, mock.Anything).
					Return(container.NetworkMode("test"), nil).
					Once()
				nm.On("Inspect", mock.Anything).
					Return(types.NetworkResource{}, nil).
					Once()
				nm.On("Cleanup", mock.Anything).
					Return(nil).
					Once()
			},
		},
		"network creation failed": {
			createNetworkManager: true,
			networkPerBuild:      "true",
			networksManagerAssertions: func(nm *networks.MockManager) {
				nm.On("Create", mock.Anything, mock.Anything).
					Return(container.NetworkMode("fail"), testErr).
					Once()
			},
			expectedBuildError: testErr,
		},
		"network inspect failed": {
			createNetworkManager: true,
			networkPerBuild:      "true",
			networksManagerAssertions: func(nm *networks.MockManager) {
				nm.On("Create", mock.Anything, mock.Anything).
					Return(container.NetworkMode("test"), nil).
					Once()
				nm.On("Inspect", mock.Anything).
					Return(types.NetworkResource{}, testErr).
					Once()
			},
			expectedCleanError: nil,
		},
		"removing container failed": {
			createNetworkManager: true,
			networkPerBuild:      "true",
			clientAssertions: func(c *docker.MockClient) {
				c.On("NetworkList", mock.Anything, mock.Anything).
					Return([]types.NetworkResource{}, nil).
					Once()
				c.On("ContainerRemove", mock.Anything, mock.Anything, mock.Anything).
					Return(testErr).
					Once()
			},
			networksManagerAssertions: func(nm *networks.MockManager) {
				nm.On("Create", mock.Anything, mock.Anything).
					Return(container.NetworkMode("test"), nil).
					Once()
				nm.On("Inspect", mock.Anything).
					Return(
						types.NetworkResource{
							Containers: map[string]types.EndpointResource{
								"abc": {},
							},
						},
						nil,
					).
					Once()
				nm.On("Cleanup", mock.Anything).
					Return(nil).
					Once()
			},
			expectedCleanError: nil,
		},
		"network cleanup failed": {
			createNetworkManager: true,
			networkPerBuild:      "true",
			networksManagerAssertions: func(nm *networks.MockManager) {
				nm.On("Create", mock.Anything, mock.Anything).
					Return(container.NetworkMode("test"), nil).
					Once()
				nm.On("Inspect", mock.Anything).
					Return(types.NetworkResource{}, nil).
					Once()
				nm.On("Cleanup", mock.Anything).
					Return(testErr).
					Once()
			},
			expectedCleanError: testErr,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			e, closureFn := getExecutorForNetworksTests(t, test)
			defer closureFn()

			err := e.createBuildNetwork()
			assert.ErrorIs(t, err, test.expectedBuildError)

			err = e.cleanupNetwork(context.Background())
			assert.ErrorIs(t, err, test.expectedCleanError)
		})
	}
}

func getExecutorForNetworksTests(t *testing.T, test networksTestCase) (*executor, func()) {
	t.Helper()

	clientMock := new(docker.MockClient)
	networksManagerMock := new(networks.MockManager)

	oldCreateNetworksManager := createNetworksManager
	closureFn := func() {
		createNetworksManager = oldCreateNetworksManager

		networksManagerMock.AssertExpectations(t)
		clientMock.AssertExpectations(t)
	}

	createNetworksManager = func(_ *executor) (networks.Manager, error) {
		return networksManagerMock, nil
	}

	if test.networksManagerAssertions != nil {
		test.networksManagerAssertions(networksManagerMock)
	}

	if test.clientAssertions != nil {
		test.clientAssertions(clientMock)
	}

	c := common.RunnerConfig{
		RunnerCredentials: common.RunnerCredentials{
			Token: "abcdef1234567890",
		},
	}
	c.Docker = &common.DockerConfig{
		NetworkMode: "",
	}
	e := &executor{
		AbstractExecutor: executors.AbstractExecutor{
			Build: &common.Build{
				ProjectRunnerID: 0,
				Runner:          &c,
				JobResponse: common.JobResponse{
					JobInfo: common.JobInfo{
						ProjectID: 0,
					},
					GitInfo: common.GitInfo{
						RepoURL: "https://gitlab.example.com/group/project.git",
					},
				},
			},
			Config: c,
			ExecutorOptions: executors.ExecutorOptions{
				DefaultBuildsDir: volumesTestsDefaultBuildsDir,
				DefaultCacheDir:  volumesTestsDefaultCacheDir,
			},
		},
		client: clientMock,
		info: types.Info{
			OSType: helperimage.OSTypeLinux,
		},
	}

	e.Context = context.Background()
	e.Build.Variables = append(e.Build.Variables, common.JobVariable{
		Key:   featureflags.NetworkPerBuild,
		Value: test.networkPerBuild,
	})

	if test.createNetworkManager {
		err := e.createNetworksManager()
		require.NoError(t, err)
	}

	return e, closureFn
}

func TestCheckOSType(t *testing.T) {
	cases := map[string]struct {
		executorMetadata map[string]string
		dockerInfoOSType string
		expectedErr      string
	}{
		"executor and docker info mismatch": {
			executorMetadata: map[string]string{
				metadataOSType: osTypeWindows,
			},
			dockerInfoOSType: osTypeLinux,
			expectedErr:      "executor requires OSType=windows, but Docker Engine supports only OSType=linux",
		},
		"executor and docker info match": {
			executorMetadata: map[string]string{
				metadataOSType: osTypeLinux,
			},
			dockerInfoOSType: osTypeLinux,
			expectedErr:      "",
		},
		"executor OSType not defined": {
			executorMetadata: nil,
			dockerInfoOSType: osTypeLinux,
			expectedErr:      " does not have any OSType specified",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := executor{
				info: types.Info{
					OSType: c.dockerInfoOSType,
				},
				AbstractExecutor: executors.AbstractExecutor{
					ExecutorOptions: executors.ExecutorOptions{
						Metadata: c.executorMetadata,
					},
				},
			}

			err := e.validateOSType()
			if c.expectedErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, c.expectedErr)
		})
	}
}

func TestGetServiceDefinitions(t *testing.T) {
	e := new(executor)
	e.Build = &common.Build{
		Runner: &common.RunnerConfig{},
	}
	e.Config = common.RunnerConfig{}
	e.Config.Docker = &common.DockerConfig{}

	tests := map[string]struct {
		services         []common.Service
		buildServices    []common.Image
		allowedServices  []string
		expectedServices common.Services
		expectedErr      string
	}{
		"all services with proper name and alias": {
			services: []common.Service{
				{
					Name:       "name",
					Alias:      "alias",
					Command:    []string{"executable", "param1", "param2"},
					Entrypoint: []string{"executable", "param3", "param4"},
				},
				{
					Name:    "name2",
					Alias:   "alias2",
					Command: []string{"executable", "param1", "param2"},
				},
				{
					Name:       "name3",
					Alias:      "alias3",
					Entrypoint: []string{"executable", "param3", "param4"},
				},
			},
			expectedServices: common.Services{
				{
					Name:       "name",
					Alias:      "alias",
					Command:    []string{"executable", "param1", "param2"},
					Entrypoint: []string{"executable", "param3", "param4"},
				},
				{
					Name:    "name2",
					Alias:   "alias2",
					Command: []string{"executable", "param1", "param2"},
				},
				{
					Name:       "name3",
					Alias:      "alias3",
					Entrypoint: []string{"executable", "param3", "param4"},
				},
			},
		},
		"build service not in internal images but empty allowed services": {
			services: []common.Service{
				{
					Name:  "name",
					Alias: "alias",
				},
			},
			buildServices: []common.Image{
				{
					Name: "name_not_in_internal",
				},
			},
			expectedServices: common.Services{
				{
					Name:  "name",
					Alias: "alias",
				},
				{
					Name: "name_not_in_internal",
				},
			},
		},
		"build service not in internal images": {
			services: []common.Service{
				{
					Name: "name",
				},
			},
			buildServices: []common.Image{
				{
					Name: "name_not_in_internal",
				},
			},
			allowedServices: []string{"name"},
			expectedErr:     "invalid image",
		},
		"build service not in allowed services but in internal images": {
			services: []common.Service{
				{
					Name: "name",
				},
			},
			buildServices: []common.Image{
				{
					Name: "name",
				},
			},
			allowedServices: []string{"allowed_name"},
			expectedServices: common.Services{
				{
					Name: "name",
				},
				{
					Name: "name",
				},
			},
		},
		"empty service name": {
			services: []common.Service{
				{
					Name: "",
				},
			},
			buildServices: []common.Image{},
			expectedServices: common.Services{
				{
					Name: "",
				},
			},
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			e.Config.Docker.Services = tt.services
			e.Config.Docker.AllowedServices = tt.allowedServices
			e.Build.Services = tt.buildServices

			svcs, err := e.getServicesDefinitions()
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedServices, svcs)
		})
	}
}

func TestAddServiceHealthCheck(t *testing.T) {
	tests := map[string]struct {
		networkMode            string
		dockerClientAssertions func(*docker.MockClient)
		expectedEnvironment    []string
		expectedErr            error
	}{
		"network mode not defined": {
			expectedEnvironment: []string{},
		},
		"get ports via environment": {
			networkMode: "test",
			dockerClientAssertions: func(c *docker.MockClient) {
				c.On("ContainerInspect", mock.Anything, mock.Anything).
					Return(types.ContainerJSON{
						Config: &container.Config{
							ExposedPorts: nat.PortSet{
								"1000/tcp": {},
							},
						},
					}, nil).
					Once()
			},
			expectedEnvironment: []string{
				"WAIT_FOR_SERVICE_TCP_ADDR=000000000000",
				"WAIT_FOR_SERVICE_TCP_PORT=1000",
			},
		},
		"get port from many": {
			networkMode: "test",
			dockerClientAssertions: func(c *docker.MockClient) {
				c.On("ContainerInspect", mock.Anything, mock.Anything).
					Return(types.ContainerJSON{
						Config: &container.Config{
							ExposedPorts: nat.PortSet{
								"1000/tcp": {},
								"500/udp":  {},
								"600/tcp":  {},
								"1500/tcp": {},
							},
						},
					}, nil).
					Once()
			},
			expectedEnvironment: []string{
				"WAIT_FOR_SERVICE_TCP_ADDR=000000000000",
				"WAIT_FOR_SERVICE_TCP_PORT=600",
			},
		},
		"no ports defined": {
			networkMode: "test",
			dockerClientAssertions: func(c *docker.MockClient) {
				c.On("ContainerInspect", mock.Anything, mock.Anything).
					Return(types.ContainerJSON{
						Config: &container.Config{
							ExposedPorts: nat.PortSet{},
						},
					}, nil).
					Once()
			},
			expectedErr: fmt.Errorf("service %q has no exposed ports", "default"),
		},
		"container inspect error": {
			networkMode: "test",
			dockerClientAssertions: func(c *docker.MockClient) {
				c.On("ContainerInspect", mock.Anything, mock.Anything).
					Return(types.ContainerJSON{}, fmt.Errorf("%v", "test error")).
					Once()
			},
			expectedErr: fmt.Errorf("get container exposed ports: %v", "test error"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client := new(docker.MockClient)

			if test.dockerClientAssertions != nil {
				test.dockerClientAssertions(client)
			}
			defer client.AssertExpectations(t)

			executor := &executor{
				networkMode: container.NetworkMode(test.networkMode),
				client:      client,
			}

			service := &types.Container{
				ID:    "0000000000000000000000000000000000000000000000000000000000000000",
				Names: []string{"default"},
			}

			environment, err := executor.addServiceHealthCheckEnvironment(service)

			assert.Equal(t, test.expectedEnvironment, environment)

			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestHelperImageRegistry(t *testing.T) {
	dockerOS := helperimage.OSTypeLinux
	if runtime.GOOS == helperimage.OSTypeWindows {
		dockerOS = runtime.GOOS
	}

	tests := map[string]struct {
		build *common.Build
		// We only validate the name because we only care if the right image is
		// used. We don't want to end up having this test as a "spellcheck" to
		// make sure tags and commands are generated correctly since that is
		// done at a unit level already and we would be duplicating internal
		// logic and leaking abstractions.
		expectedHelperImageName string
	}{
		"Docker Hub helper image": {
			build: &common.Build{
				JobResponse: common.JobResponse{
					Image: common.Image{
						Name: "test",
					},
				},
				Runner: &common.RunnerConfig{
					RunnerSettings: common.RunnerSettings{
						Docker: &common.DockerConfig{},
					},
				},
			},
			expectedHelperImageName: helperimage.DockerHubName,
		},
		"GitLab Registry helper image": {
			build: &common.Build{
				JobResponse: common.JobResponse{
					Image: common.Image{
						Name: "test",
					},
					Variables: common.JobVariables{
						common.JobVariable{
							Key:      featureflags.GitLabRegistryHelperImage,
							Value:    "true",
							Public:   false,
							Internal: false,
							File:     false,
							Masked:   false,
							Raw:      false,
						},
					},
				},
				Runner: &common.RunnerConfig{
					RunnerSettings: common.RunnerSettings{
						Docker: &common.DockerConfig{},
					},
				},
			},
			expectedHelperImageName: helperimage.GitLabRegistryName,
		},
		"helper image overridden still use default helper image in prepare": {
			build: &common.Build{
				JobResponse: common.JobResponse{
					Image: common.Image{
						Name: "test",
					},
				},
				Runner: &common.RunnerConfig{
					RunnerSettings: common.RunnerSettings{
						Docker: &common.DockerConfig{
							HelperImage: "private.registry.com/helper",
						},
					},
				},
			},
			// We expect the default image to still be chosen since the check of
			// the override happens at a later stage.
			expectedHelperImageName: helperimage.DockerHubName,
		},
		"helper image overridden still use registry.gitlab.com helper image in prepare": {
			build: &common.Build{
				JobResponse: common.JobResponse{
					Variables: common.JobVariables{
						common.JobVariable{
							Key:   featureflags.GitLabRegistryHelperImage,
							Value: "true",
						},
					},
					Image: common.Image{
						Name: "test",
					},
				},
				Runner: &common.RunnerConfig{
					RunnerSettings: common.RunnerSettings{
						Docker: &common.DockerConfig{
							HelperImage: "private.registry.com/helper",
						},
					},
				},
			},
			// We expect the registry.gitlab.com image to still be chosen since
			// the check of the override happens at a later stage.
			expectedHelperImageName: helperimage.GitLabRegistryName,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			e := &executor{
				AbstractExecutor: executors.AbstractExecutor{
					ExecutorOptions: executors.ExecutorOptions{
						Metadata: map[string]string{
							metadataOSType: dockerOS,
						},
					},
				},
				volumeParser: parser.NewLinuxParser(),
			}

			prepareOptions := common.ExecutorPrepareOptions{
				Config: &common.RunnerConfig{
					RunnerSettings: common.RunnerSettings{
						BuildsDir: "/tmp",
						CacheDir:  "/tmp",
						Shell:     "bash",
						Docker:    tt.build.Runner.Docker,
					},
				},
				Build:   tt.build,
				Context: context.Background(),
			}

			err := e.Prepare(prepareOptions)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedHelperImageName, e.helperImageInfo.Name)
		})
	}
}

func TestLocalHelperImage(t *testing.T) {
	// Docker Windows doesn't support docker import, only docker load which we
	// do not yet support
	// https://gitlab.com/gitlab-org/gitlab-runner/-/issues/26678
	test.SkipIfGitLabCIOn(t, test.OSWindows)

	dockerHubHelperImage := fmt.Sprintf("%s:%s", helperimage.DockerHubName, "localimageimport")
	gitlabRegistryHelperImage := fmt.Sprintf("%s:%s", helperimage.GitLabRegistryName, "localimageimport")
	defaultHelperImageInfo := helperimage.Info{
		Architecture:            "x86_64",
		Name:                    helperimage.DockerHubName,
		Tag:                     "localimageimport",
		IsSupportingLocalImport: true,
	}

	cleanupFn := createFakePrebuiltImages(t, defaultHelperImageInfo.Architecture)
	defer cleanupFn()

	tests := map[string]struct {
		jobVariables     common.JobVariables
		helperImageInfo  helperimage.Info
		shell            string
		clientAssertions func(*docker.MockClient)
		expectedImage    *types.ImageInspect
	}{
		"doesn't support local import": {
			helperImageInfo: helperimage.Info{
				Architecture:            "x86_64",
				Name:                    "nosupport",
				Tag:                     "localimageimport",
				IsSupportingLocalImport: false,
			},
			clientAssertions: func(c *docker.MockClient) {},
			expectedImage:    nil,
		},
		"Docker import using Docker Hub name": {
			helperImageInfo: defaultHelperImageInfo,
			clientAssertions: func(c *docker.MockClient) {
				c.On(
					"ImageImportBlocking",
					mock.Anything,
					mock.Anything,
					helperimage.DockerHubName,
					mock.Anything,
				).Return(nil)

				imageInspect := types.ImageInspect{
					RepoTags: []string{
						dockerHubHelperImage,
					},
				}

				c.On(
					"ImageInspectWithRaw",
					mock.Anything,
					dockerHubHelperImage,
				).Return(imageInspect, []byte{}, nil)
			},
			expectedImage: &types.ImageInspect{
				RepoTags: []string{
					dockerHubHelperImage,
				},
			},
		},
		"Docker import using registry.gitlab.com name": {
			helperImageInfo: helperimage.Info{
				Architecture:            "x86_64",
				Name:                    helperimage.GitLabRegistryName,
				Tag:                     "localimageimport",
				IsSupportingLocalImport: true,
			},
			clientAssertions: func(c *docker.MockClient) {
				c.On(
					"ImageImportBlocking",
					mock.Anything,
					mock.Anything,
					helperimage.GitLabRegistryName,
					mock.Anything,
				).Return(nil)

				imageInspect := types.ImageInspect{
					RepoTags: []string{
						gitlabRegistryHelperImage,
					},
				}

				c.On(
					"ImageInspectWithRaw",
					mock.Anything,
					gitlabRegistryHelperImage,
				).Return(imageInspect, []byte{}, nil)
			},
			expectedImage: &types.ImageInspect{
				RepoTags: []string{
					gitlabRegistryHelperImage,
				},
			},
		},
		"entrypoint added": {
			helperImageInfo: defaultHelperImageInfo,
			clientAssertions: func(c *docker.MockClient) {
				c.On(
					"ImageImportBlocking",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					types.ImageImportOptions{
						Tag: "localimageimport",
						Changes: []string{
							`ENTRYPOINT ["/usr/bin/dumb-init", "/entrypoint"]`,
						},
					},
				).Return(nil)

				c.On(
					"ImageInspectWithRaw",
					mock.Anything,
					mock.Anything,
				).Return(types.ImageInspect{}, []byte{}, nil)
			},
			expectedImage: &types.ImageInspect{},
		},
		"entrypoint not added when feature flag turned off": {
			jobVariables: common.JobVariables{
				common.JobVariable{
					Key:   featureflags.ResetHelperImageEntrypoint,
					Value: "false",
				},
			},
			helperImageInfo: defaultHelperImageInfo,
			clientAssertions: func(c *docker.MockClient) {
				c.On(
					"ImageImportBlocking",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					types.ImageImportOptions{
						Tag:     "localimageimport",
						Changes: nil,
					},
				).Return(nil)

				c.On(
					"ImageInspectWithRaw",
					mock.Anything,
					mock.Anything,
				).Return(types.ImageInspect{}, []byte{}, nil)
			},
			expectedImage: &types.ImageInspect{},
		},
		"nil is returned if error on import": {
			helperImageInfo: defaultHelperImageInfo,
			clientAssertions: func(c *docker.MockClient) {
				c.On(
					"ImageImportBlocking",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(errors.New("error"))
			},
			expectedImage: nil,
		},
		"nil is returned if error on inspect": {
			helperImageInfo: defaultHelperImageInfo,
			clientAssertions: func(c *docker.MockClient) {
				c.On(
					"ImageImportBlocking",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil)

				c.On(
					"ImageInspectWithRaw",
					mock.Anything,
					mock.Anything,
				).Return(types.ImageInspect{}, []byte{}, errors.New("error"))
			},
			expectedImage: nil,
		},
		"Powershell image is used when shell is pwsh": {
			helperImageInfo: defaultHelperImageInfo,
			shell:           shells.SNPwsh,
			clientAssertions: func(c *docker.MockClient) {
				c.On(
					"ImageImportBlocking",
					mock.Anything,
					mock.MatchedBy(func(source types.ImageImportSource) bool {
						return assert.IsType(t, new(os.File), source.Source) &&
							assert.Equal(
								t,
								"prebuilt-x86_64-pwsh.tar.xz",
								path.Base((source.Source.(*os.File)).Name()),
							)
					}),
					helperimage.DockerHubName,
					mock.Anything,
				).Return(nil)

				imageInspect := types.ImageInspect{
					RepoTags: []string{
						dockerHubHelperImage,
					},
				}

				c.On(
					"ImageInspectWithRaw",
					mock.Anything,
					dockerHubHelperImage,
				).Return(imageInspect, []byte{}, nil)
			},
			expectedImage: &types.ImageInspect{
				RepoTags: []string{
					dockerHubHelperImage,
				},
			},
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			c := new(docker.MockClient)
			defer c.AssertExpectations(t)

			e := &executor{
				AbstractExecutor: executors.AbstractExecutor{
					Build: &common.Build{
						JobResponse: common.JobResponse{
							Variables: tt.jobVariables,
						},
						Runner: &common.RunnerConfig{},
					},

					Config: common.RunnerConfig{
						RunnerSettings: common.RunnerSettings{
							Shell: tt.shell,
						},
					},
				},
				client:          c,
				volumeParser:    parser.NewLinuxParser(),
				helperImageInfo: tt.helperImageInfo,
			}

			tt.clientAssertions(c)

			image := e.getLocalHelperImage()
			assert.Equal(t, tt.expectedImage, image)
		})
	}
}

func createFakePrebuiltImages(t *testing.T, architecture string) func() {
	// Create fake image files so that tests do not need helper images built
	tempImgDir, err := ioutil.TempDir(os.TempDir(), "runner-tests")
	require.NoError(t, err)

	prevPrebuiltImagesPaths := PrebuiltImagesPaths
	PrebuiltImagesPaths = []string{tempImgDir}
	for _, fakeImgName := range []string{
		fmt.Sprintf("prebuilt-%s.tar.xz", architecture),
		fmt.Sprintf("prebuilt-%s-pwsh.tar.xz", architecture),
	} {
		fakeLocalImage, err := os.Create(path.Join(tempImgDir, fakeImgName))
		require.NoError(t, err)
		fakeLocalImage.Close()
	}

	return func() {
		os.RemoveAll(tempImgDir)
		PrebuiltImagesPaths = prevPrebuiltImagesPaths
	}
}

func TestGetUIDandGID(t *testing.T) {
	ctx := context.Background()
	testContainerID := "test-ID"
	testImageSHA := "test-SHA"
	testUID := 456
	testGID := 789

	tests := map[string]struct {
		mockInspect   func(t *testing.T, i *user.MockInspect)
		expectedError error
	}{
		"UID check returns error": {
			mockInspect: func(t *testing.T, i *user.MockInspect) {
				i.On("UID", ctx, testContainerID).Return(0, assert.AnError).Once()
			},
			expectedError: assert.AnError,
		},
		"UID check succeeds, GID check returns error": {
			mockInspect: func(t *testing.T, i *user.MockInspect) {
				i.On("UID", ctx, testContainerID).Return(testUID, nil).Once()
				i.On("GID", ctx, testContainerID).Return(0, assert.AnError).Once()
			},
			expectedError: assert.AnError,
		},
		"both checks succeed": {
			mockInspect: func(t *testing.T, i *user.MockInspect) {
				i.On("UID", ctx, testContainerID).Return(testUID, nil).Once()
				i.On("GID", ctx, testContainerID).Return(testGID, nil).Once()
			},
			expectedError: nil,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			inspectMock := new(user.MockInspect)
			defer inspectMock.AssertExpectations(t)

			tt.mockInspect(t, inspectMock)

			log, _ := logrustest.NewNullLogger()
			uid, gid, err := getUIDandGID(ctx, log, inspectMock, testContainerID, testImageSHA)

			if tt.expectedError != nil {
				assert.Equal(t, 0, uid)
				assert.Equal(t, 0, gid)
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, testUID, uid)
			assert.Equal(t, testGID, gid)
		})
	}
}

func init() {
	auth.HomeDirectory = ""
}

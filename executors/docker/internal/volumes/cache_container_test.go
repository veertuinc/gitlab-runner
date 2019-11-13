package volumes

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewCacheContainerManager(t *testing.T) {
	logger := newDebugLoggerMock()

	m := NewCacheContainerManager(context.Background(), logger, nil, nil)
	assert.IsType(t, &cacheContainerManager{}, m)
}

func getCacheContainerManager() (*cacheContainerManager, *mockContainerClient) {
	cClient := new(mockContainerClient)

	m := &cacheContainerManager{
		logger:             newDebugLoggerMock(),
		containerClient:    cClient,
		failedContainerIDs: make([]string, 0),
		helperImage:        &types.ImageInspect{ID: "helper-image"},
	}

	return m, cClient
}

func TestCacheContainerManager_FindExistingCacheContainer(t *testing.T) {
	containerName := "container-name"
	containerPath := "container-path"

	testCases := map[string]struct {
		inspectResult       types.ContainerJSON
		inspectError        error
		expectedContainerID string
		expectedRemoveID    string
	}{
		"error on container inspection": {
			inspectError:        errors.New("test error"),
			expectedContainerID: "",
		},
		"container with valid cache exists": {
			inspectResult: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					ID: "existingWithValidCacheID",
				},
				Config: &container.Config{
					Volumes: map[string]struct{}{
						containerPath: {},
					},
				},
			},
			inspectError:        nil,
			expectedContainerID: "existingWithValidCacheID",
		},
		"container without valid cache exists": {
			inspectResult: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					ID: "existingWithInvalidCacheID",
				},
				Config: &container.Config{
					Volumes: map[string]struct{}{
						"different-path": {},
					},
				},
			},
			inspectError:        nil,
			expectedContainerID: "",
			expectedRemoveID:    "existingWithInvalidCacheID",
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			m, cClient := getCacheContainerManager()
			defer cClient.AssertExpectations(t)

			cClient.On("ContainerInspect", mock.Anything, containerName).
				Return(testCase.inspectResult, testCase.inspectError).
				Once()

			if testCase.expectedRemoveID != "" {
				cClient.On("RemoveContainer", mock.Anything, testCase.expectedRemoveID).
					Return(nil).
					Once()
			}

			containerID := m.FindOrCleanExisting(containerName, containerPath)
			assert.Equal(t, testCase.expectedContainerID, containerID)
		})
	}
}

func TestCacheContainerManager_CreateCacheContainer(t *testing.T) {
	containerName := "container-name"
	containerPath := "container-path"
	expectedCacheCmd := []string{"gitlab-runner-helper", "cache-init", containerPath}

	testCases := map[string]struct {
		expectedContainerID       string
		createResult              container.ContainerCreateCreatedBody
		createError               error
		containerID               string
		startError                error
		waitForContainerError     error
		expectedFailedContainerID string
		expectedError             error
	}{
		"error on container create": {
			createError:   errors.New("test error"),
			expectedError: errors.New("test error"),
		},
		"error on container create with returnedID": {
			createResult: container.ContainerCreateCreatedBody{
				ID: "containerID",
			},
			createError:               errors.New("test error"),
			expectedFailedContainerID: "containerID",
			expectedError:             errors.New("test error"),
		},
		"error on container start": {
			createResult: container.ContainerCreateCreatedBody{
				ID: "containerID",
			},
			containerID:               "containerID",
			startError:                errors.New("test error"),
			expectedFailedContainerID: "containerID",
			expectedError:             errors.New("test error"),
		},
		"error on wait for container": {
			createResult: container.ContainerCreateCreatedBody{
				ID: "containerID",
			},
			containerID:               "containerID",
			waitForContainerError:     errors.New("test error"),
			expectedFailedContainerID: "containerID",
			expectedError:             errors.New("test error"),
		},
		"success": {
			createResult: container.ContainerCreateCreatedBody{
				ID: "containerID",
			},
			containerID:         "containerID",
			expectedContainerID: "containerID",
			expectedError:       nil,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			m, cClient := getCacheContainerManager()

			defer cClient.AssertExpectations(t)

			configMatcher := mock.MatchedBy(func(config *container.Config) bool {
				if config.Image != "helper-image" {
					return false
				}

				if len(config.Cmd) != len(expectedCacheCmd) {
					return false
				}

				return config.Cmd[0] == expectedCacheCmd[0]
			})

			cClient.On("LabelContainer", configMatcher, "cache", fmt.Sprintf("cache.dir=%s", containerPath)).
				Once()

			cClient.On("ContainerCreate", mock.Anything, configMatcher, mock.Anything, mock.Anything, containerName).
				Return(testCase.createResult, testCase.createError).
				Once()

			if testCase.createError == nil {
				cClient.On("ContainerStart", mock.Anything, testCase.containerID, mock.Anything).
					Return(testCase.startError).
					Once()

				if testCase.startError == nil {
					cClient.On("WaitForContainer", testCase.containerID).
						Return(testCase.waitForContainerError).
						Once()
				}
			}

			require.Empty(t, m.failedContainerIDs, "Initial list of failed containers should be empty")

			containerID, err := m.Create(containerName, containerPath)
			assert.Equal(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedContainerID, containerID)

			if testCase.expectedFailedContainerID != "" {
				assert.Len(t, m.failedContainerIDs, 1)
				assert.Contains(
					t, m.failedContainerIDs, testCase.expectedFailedContainerID,
					"List of failed container should be updated with %s", testCase.expectedContainerID,
				)
			} else {
				assert.Empty(t, m.failedContainerIDs, "List of failed containers should not be updated")
			}
		})
	}
}

func TestCacheContainerManager_Cleanup(t *testing.T) {
	ctx := context.Background()

	containerClientMock := new(mockContainerClient)
	defer containerClientMock.AssertExpectations(t)

	loggerMock := new(mockDebugLogger)
	defer loggerMock.AssertExpectations(t)

	containerClientMock.On("RemoveContainer", ctx, "failed-container-1").
		Return(nil).
		Once()
	containerClientMock.On("RemoveContainer", ctx, "container-1-with-remove-error").
		Return(errors.New("test-error")).
		Once()
	containerClientMock.On("RemoveContainer", ctx, "container-1").
		Return(nil).
		Once()

	loggerMock.On("Debugln", "Error while removing the container: test-error").
		Once()

	m := &cacheContainerManager{
		containerClient:    containerClientMock,
		logger:             loggerMock,
		failedContainerIDs: []string{"failed-container-1", "container-1-with-remove-error"},
	}

	done := m.Cleanup(ctx, []string{"container-1"})

	<-done
}

package docker_helpers

import (
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"golang.org/x/net/context"
)

type Client interface {
	ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)

	ImagePullBlocking(ctx context.Context, ref string, options types.ImagePullOptions) error
	ImageImportBlocking(ctx context.Context, source types.ImageImportSource, ref string, options types.ImageImportOptions) error

	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error
	ContainerWait(ctx context.Context, containerID string) (int64, error)
	ContainerKill(ctx context.Context, containerID, signal string) error
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerAttach(ctx context.Context, container string, options types.ContainerAttachOptions) (types.HijackedResponse, error)
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
	ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error)
	ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error)
	ContainerExecAttach(ctx context.Context, execID string, config types.ExecConfig) (types.HijackedResponse, error)

	NetworkDisconnect(ctx context.Context, networkID, containerID string, force bool) error
	NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)

	Info(ctx context.Context) (types.Info, error)

	Close() error
}

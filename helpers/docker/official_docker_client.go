package docker_helpers

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/docker/docker/api/types"
	container "github.com/docker/docker/api/types/container"
	network "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// IsErrNotFound checks whether a returned error is due to an image or container
// not being found. Proxies the docker implementation.
func IsErrNotFound(err error) bool {
	return client.IsErrNotFound(err)
}

// type officialDockerClient wraps a "github.com/docker/docker/client".Client,
// giving it the methods it needs to satisfy the docker_helpers.Client interface
type officialDockerClient struct {
	client *client.Client

	// Close() means "close idle connections held by engine-api's transport"
	Transport *http.Transport
}

func newOfficialDockerClient(c DockerCredentials, apiVersion string) (*officialDockerClient, error) {
	transport, err := newHTTPTransport(c)
	if err != nil {
		logrus.Errorln("Error creating TLS Docker client:", err)
		return nil, err
	}
	httpClient := &http.Client{Transport: transport}

	dockerClient, err := client.NewClient(c.Host, apiVersion, httpClient, nil)
	if err != nil {
		transport.CloseIdleConnections()
		logrus.Errorln("Error creating Docker client:", err)
		return nil, err
	}

	return &officialDockerClient{
		client:    dockerClient,
		Transport: transport,
	}, nil
}

func wrapError(method string, err error, started time.Time) error {
	if err == nil {
		return nil
	}

	seconds := int(time.Since(started).Seconds())

	if _, file, line, ok := runtime.Caller(2); ok {
		return fmt.Errorf("%s (%s:%d:%ds)", err.Error(), filepath.Base(file), line, seconds)
	}

	return fmt.Errorf("%s (%s:%ds)", err.Error(), method, seconds)
}

func (c *officialDockerClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	started := time.Now()
	image, data, err := c.client.ImageInspectWithRaw(ctx, imageID)
	return image, data, wrapError("ImageInspectWithRaw", err, started)
}

func (c *officialDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	started := time.Now()
	container, err := c.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, containerName)
	return container, wrapError("ContainerCreate", err, started)
}

func (c *officialDockerClient) ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
	started := time.Now()
	err := c.client.ContainerStart(ctx, containerID, options)
	return wrapError("ContainerCreate", err, started)
}

func (c *officialDockerClient) ContainerWait(ctx context.Context, containerID string) (int64, error) {
	started := time.Now()
	result, err := c.client.ContainerWait(ctx, containerID)
	return result, wrapError("ContainerWait", err, started)
}

func (c *officialDockerClient) ContainerKill(ctx context.Context, containerID string, signal string) error {
	started := time.Now()
	err := c.client.ContainerKill(ctx, containerID, signal)
	return wrapError("ContainerWait", err, started)
}

func (c *officialDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	started := time.Now()
	data, err := c.client.ContainerInspect(ctx, containerID)
	return data, wrapError("ContainerInspect", err, started)
}

func (c *officialDockerClient) ContainerAttach(ctx context.Context, container string, options types.ContainerAttachOptions) (types.HijackedResponse, error) {
	started := time.Now()
	response, err := c.client.ContainerAttach(ctx, container, options)
	return response, wrapError("ContainerAttach", err, started)
}

func (c *officialDockerClient) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	started := time.Now()
	err := c.client.ContainerRemove(ctx, containerID, options)
	return wrapError("ContainerRemove", err, started)
}

func (c *officialDockerClient) ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	started := time.Now()
	rc, err := c.client.ContainerLogs(ctx, container, options)
	return rc, wrapError("ContainerLogs", err, started)
}

func (c *officialDockerClient) ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
	started := time.Now()
	resp, err := c.client.ContainerExecCreate(ctx, container, config)
	return resp, wrapError("ContainerExecCreate", err, started)
}

func (c *officialDockerClient) ContainerExecAttach(ctx context.Context, execID string, config types.ExecConfig) (types.HijackedResponse, error) {
	started := time.Now()
	resp, err := c.client.ContainerExecAttach(ctx, execID, config)
	return resp, wrapError("ContainerExecAttach", err, started)
}

func (c *officialDockerClient) NetworkDisconnect(ctx context.Context, networkID string, containerID string, force bool) error {
	started := time.Now()
	err := c.client.NetworkDisconnect(ctx, networkID, containerID, force)
	return wrapError("NetworkDisconnect", err, started)
}

func (c *officialDockerClient) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	started := time.Now()
	networks, err := c.client.NetworkList(ctx, options)
	return networks, wrapError("NetworkList", err, started)
}

func (c *officialDockerClient) Info(ctx context.Context) (types.Info, error) {
	started := time.Now()
	info, err := c.client.Info(ctx)
	return info, wrapError("Info", err, started)
}

func (c *officialDockerClient) ImageImportBlocking(ctx context.Context, source types.ImageImportSource, ref string, options types.ImageImportOptions) error {
	started := time.Now()
	readCloser, err := c.client.ImageImport(ctx, source, ref, options)
	if err != nil {
		return wrapError("ImageImport", err, started)
	}
	defer readCloser.Close()

	// TODO: respect the context here
	if _, err := io.Copy(ioutil.Discard, readCloser); err != nil {
		return wrapError("io.Copy: Failed to import image", err, started)
	}

	return nil
}

func (c *officialDockerClient) ImagePullBlocking(ctx context.Context, ref string, options types.ImagePullOptions) error {
	started := time.Now()
	readCloser, err := c.client.ImagePull(ctx, ref, options)
	if err != nil {
		return wrapError("ImagePull", err, started)
	}
	defer readCloser.Close()

	// TODO: respect the context here
	if _, err := io.Copy(ioutil.Discard, readCloser); err != nil {
		return wrapError("io.Copy: Failed to pull image", err, started)
	}

	return nil
}

func (c *officialDockerClient) Close() error {
	c.Transport.CloseIdleConnections()
	return nil
}

// New attempts to create a new Docker client of the specified version.
//
// If no host is given in the DockerCredentials, it will attempt to look up
// details from the environment. If that fails, it will use the default
// connection details for your platform.
func New(c DockerCredentials, apiVersion string) (Client, error) {
	if c.Host == "" {
		c = credentialsFromEnv()
	}

	// Use the default if nothing is specified by caller *or* environment
	if c.Host == "" {
		c.Host = client.DefaultDockerHost
	}

	return newOfficialDockerClient(c, apiVersion)
}

func newHTTPTransport(c DockerCredentials) (*http.Transport, error) {
	proto, addr, _, err := client.ParseHost(c.Host)
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{}

	if err := configureTransport(tr, proto, addr); err != nil {
		return nil, err
	}

	// FIXME: is a TLS connection with InsecureSkipVerify == true ever wanted?
	if c.TLSVerify {
		options := tlsconfig.Options{}

		if c.CertPath != "" {
			options.CAFile = filepath.Join(c.CertPath, "ca.pem")
			options.CertFile = filepath.Join(c.CertPath, "cert.pem")
			options.KeyFile = filepath.Join(c.CertPath, "key.pem")
		}

		tlsConfig, err := tlsconfig.Client(options)
		if err != nil {
			tr.CloseIdleConnections()
			return nil, err
		}

		tr.TLSClientConfig = tlsConfig
	}

	return tr, nil
}

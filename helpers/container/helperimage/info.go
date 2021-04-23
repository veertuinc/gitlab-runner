package helperimage

import (
	"fmt"

	"gitlab.com/gitlab-org/gitlab-runner/helpers/docker/errors"
	"gitlab.com/gitlab-org/gitlab-runner/shells"
)

const (
	OSTypeLinux   = "linux"
	OSTypeWindows = "windows"

	//nolint:lll
	// DockerHubWarningMessage is the message that is printed to the user when
	// it's using the helper image hosted in Docker Hub. It is up to the caller
	// to print this message.
	DockerHubWarningMessage = "Pulling GitLab Runner helper image from Docker Hub. " +
		"Helper image is migrating to registry.gitlab.com, " +
		"for more information see " +
		"https://docs.gitlab.com/runner/configuration/advanced-configuration.html#migrate-helper-image-to-registrygitlabcom"

	// DockerHubName is the name of the helper image hosted in Docker Hub.
	DockerHubName = "gitlab/gitlab-runner-helper"
	// GitLabRegistryName is the name of the helper image hosted in registry.gitlab.com.
	GitLabRegistryName = "registry.gitlab.com/gitlab-org/gitlab-runner/gitlab-runner-helper"

	headRevision        = "HEAD"
	latestImageRevision = "latest"
)

type Info struct {
	Architecture            string
	Name                    string
	Tag                     string
	IsSupportingLocalImport bool
	Cmd                     []string
}

func (i Info) String() string {
	return fmt.Sprintf("%s:%s", i.Name, i.Tag)
}

// Config specifies details about the consumer of this package that need to be
// taken in consideration when building Container.
type Config struct {
	OSType          string
	Architecture    string
	OperatingSystem string
	Shell           string
	GitLabRegistry  bool
}

type creator interface {
	Create(revision string, cfg Config) (Info, error)
}

var supportedOsTypesFactories = map[string]creator{
	OSTypeWindows: new(windowsInfo),
	OSTypeLinux:   new(linuxInfo),
}

func Get(revision string, cfg Config) (Info, error) {
	factory, ok := supportedOsTypesFactories[cfg.OSType]
	if !ok {
		return Info{}, errors.NewErrOSNotSupported(cfg.OSType)
	}

	return factory.Create(imageRevision(revision), cfg)
}

func imageRevision(revision string) string {
	if revision != headRevision {
		return revision
	}

	return latestImageRevision
}

func imageName(gitlabRegistry bool) string {
	if gitlabRegistry {
		return GitLabRegistryName
	}

	return DockerHubName
}

func getPowerShellCmd(shell string) []string {
	if shell == "" {
		// TODO: Replace with shells.SNPwsh in 14.0 in https://gitlab.com/gitlab-org/gitlab-runner/-/issues/26419
		shell = shells.SNPowershell
	}

	return shells.PowershellDockerCmd(shell)
}

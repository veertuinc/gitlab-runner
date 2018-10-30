package common

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/helpers"
)

type ShellConfiguration struct {
	Environment   []string
	DockerCommand []string
	Command       string
	Arguments     []string
	PassFile      bool
	Extension     string
}

type ShellType int

const (
	NormalShell ShellType = iota
	LoginShell
)

func (s *ShellConfiguration) GetCommandWithArguments() []string {
	parts := []string{s.Command}
	for _, arg := range s.Arguments {
		parts = append(parts, arg)
	}
	return parts
}

func (s *ShellConfiguration) String() string {
	return helpers.ToYAML(s)
}

type ShellScriptInfo struct {
	Shell           string
	Build           *Build
	Type            ShellType
	User            string
	RunnerCommand   string
	PreCloneScript  string
	PreBuildScript  string
	PostBuildScript string
}

type Shell interface {
	GetName() string
	GetFeatures(features *FeaturesInfo)
	IsDefault() bool

	GetConfiguration(info ShellScriptInfo) (*ShellConfiguration, error)
	GenerateScript(buildStage BuildStage, info ShellScriptInfo) (string, error)
}

var shells map[string]Shell

func RegisterShell(shell Shell) {
	log.Debugln("Registering", shell.GetName(), "shell...")

	if shells == nil {
		shells = make(map[string]Shell)
	}
	if shells[shell.GetName()] != nil {
		panic("Shell already exist: " + shell.GetName())
	}
	shells[shell.GetName()] = shell
}

func GetShell(shell string) Shell {
	if shells == nil {
		return nil
	}

	return shells[shell]
}

func GetShells() []string {
	names := []string{}
	if shells != nil {
		for name := range shells {
			names = append(names, name)
		}
	}
	return names
}

func GetShellConfiguration(info ShellScriptInfo) (*ShellConfiguration, error) {
	shell := GetShell(info.Shell)
	if shell == nil {
		return nil, fmt.Errorf("shell %s not found", info.Shell)
	}

	return shell.GetConfiguration(info)
}

func GenerateShellScript(buildStage BuildStage, info ShellScriptInfo) (string, error) {
	shell := GetShell(info.Shell)
	if shell == nil {
		return "", fmt.Errorf("shell %s not found", info.Shell)
	}

	return shell.GenerateScript(buildStage, info)
}

func GetDefaultShell() string {
	if shells == nil {
		panic("no shells defined")
	}

	for _, shell := range shells {
		if shell.IsDefault() {
			return shell.GetName()
		}
	}
	panic("no default shell defined")
}

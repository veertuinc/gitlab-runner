package helperimage

import (
	"fmt"
	"runtime"

	"gitlab.com/gitlab-org/gitlab-runner/shells"
)

const (
	platformAmd64   = "amd64"
	platformArm6vl  = "armv6l"
	platformArmv7l  = "armv7l"
	platformAarch64 = "aarch64"
	archX8664       = "x86_64"
	archArm         = "arm"
	archArm64       = "arm64"
)

var bashCmd = []string{"gitlab-runner-build"}

type linuxInfo struct{}

func (l *linuxInfo) Create(revision string, cfg Config) (Info, error) {
	arch := l.architecture(cfg.Architecture)

	shell := cfg.Shell
	if shell == "" {
		shell = "bash"
	}

	cmd := bashCmd
	tag := fmt.Sprintf("%s-%s", arch, revision)
	if shell == shells.SNPwsh {
		cmd = getPowerShellCmd(shell)
		tag = fmt.Sprintf("%s-%s", tag, shell)
	}

	return Info{
		Architecture:            arch,
		Name:                    imageName(cfg.GitLabRegistry),
		Tag:                     tag,
		IsSupportingLocalImport: true,
		Cmd:                     cmd,
	}, nil
}

func (l *linuxInfo) architecture(arch string) string {
	switch arch {
	case platformArm6vl, platformArmv7l:
		return archArm
	case platformAarch64:
		return archArm64
	case platformAmd64:
		return archX8664
	}

	if arch != "" {
		return arch
	}

	switch runtime.GOARCH {
	case platformAmd64:
		return archX8664
	default:
		return runtime.GOARCH
	}
}

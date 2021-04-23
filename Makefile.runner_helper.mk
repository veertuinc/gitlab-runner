# -------------------------------------------------------------------------------
# The following make file does two things:
#   1. Create binaries for the gitlab-runner-helper app which can be found in
#   `./apps/gitlab-runner-helper` for all the platforms we want to support.
#   2. Create Linux containers and extract their file system to be used later to
#   build/publish.
#
# If you want to add a new arch or OS you would need to add a new
# file path to the $BINARIES variables and a new GO_ARCH_{{arch}}-{{OS}}
# variable. Note that Linux is implied by default.
# ---------------------------------------------------------------------------

TAR_XZ_ARGS ?= -f -0

# Tar files that we want to generate from the Docker file system, this is
# generally used for linux based Dockerfiles.
BASE_TAR_PATH := out/helper-images/prebuilt
TAR_XZ += ${BASE_TAR_PATH}-x86_64.tar.xz
TAR_XZ += ${BASE_TAR_PATH}-x86_64-pwsh.tar.xz
TAR_XZ += ${BASE_TAR_PATH}-arm.tar.xz
TAR_XZ += ${BASE_TAR_PATH}-arm64.tar.xz
TAR_XZ += ${BASE_TAR_PATH}-s390x.tar.xz

# Binaries that we support for the helper image. We are using the following
# pattern match:
# out/binaries/gitlab-runner-helper/gitlab-runner-helper.{{arch}}-{{os}}, these should
# match up with GO_ARCH_* variables names. Note that Linux is implied by
# default.
BASE_BINARY_PATH := out/binaries/gitlab-runner-helper/gitlab-runner-helper
BINARIES := ${BASE_BINARY_PATH}.x86_64-windows
BINARIES += ${BASE_BINARY_PATH}.x86_64
BINARIES += ${BASE_BINARY_PATH}.arm
BINARIES += ${BASE_BINARY_PATH}.arm64
BINARIES += ${BASE_BINARY_PATH}.s390x

# Define variables with the architecture for each matching binary. We are using
# the following pattern match GO_ARCH_{{arch}}-{{os}}, these should match up
# with BINARIES variables. The value of the variable is the dist name from `go tool dist list`
GO_ARCH_x86_64 = linux/amd64
GO_ARCH_arm = linux/arm
GO_ARCH_arm64 = linux/arm64
GO_ARCH_s390x = linux/s390x
GO_ARCH_x86_64-windows = windows/amd64

# Go files that are used to create the helper binary.
HELPER_GO_FILES ?= $(shell find common network vendor -name '*.go')

# Build the Runner Helper binaries for the host platform.
.PHONY: helper-bin-host
helper-bin-host: ${BASE_BINARY_PATH}.$(shell uname -m)

# Build the Runner Helper binaries for all supported platforms.
.PHONY: helper-bin
helper-bin: $(BINARIES)

${BASE_BINARY_PATH}.%: $(HELPER_GO_FILES) $(GOX)
	$(GOX) -osarch=$(GO_ARCH_$*) -ldflags "$(GO_LDFLAGS)" -output=$@ $(PKG)/apps/gitlab-runner-helper

# Build the Runner Helper tar files for host platform.
.PHONY: _helper-dockerarchive-host
_helper-dockerarchive-host: ${BASE_TAR_PATH}-$(shell uname -m)$(IMAGE_VARIANT_SUFFIX).tar.xz
	@ # NOTE: The ENTRYPOINT metadata is not preserved on export, so we need to reapply this metadata on import.
	@ # See https://gitlab.com/gitlab-org/gitlab-runner/-/merge_requests/2058#note_388341301
	docker import ${BASE_TAR_PATH}-$(shell uname -m)$(IMAGE_VARIANT_SUFFIX).tar.xz \
		--change "ENTRYPOINT [\"/usr/bin/dumb-init\", \"/entrypoint\"]" \
		gitlab/gitlab-runner-helper:$(shell uname -m)-$(REVISION)$(IMAGE_VARIANT_SUFFIX)

.PHONY: helper-dockerarchive-host
helper-dockerarchive-host:
	@$(MAKE) _helper-dockerarchive-host IMAGE_VARIANT_SUFFIX=''
	@$(MAKE) _helper-dockerarchive-host IMAGE_VARIANT_SUFFIX='-pwsh'

# Build the Runner Helper tar files for all supported platforms.
.PHONY: helper-dockerarchive
helper-dockerarchive: $(TAR_XZ)

${BASE_TAR_PATH}-%-pwsh.tar.xz: ${BASE_TAR_PATH}-%-pwsh.tar
	xz $(TAR_XZ_ARGS) $<

${BASE_TAR_PATH}-%.tar.xz: ${BASE_TAR_PATH}-%.tar
	xz $(TAR_XZ_ARGS) $<

# See https://github.com/PowerShell/powershell/releases for values of PWSH_VERSION/PWSH_IMAGE_DATE
${BASE_TAR_PATH}-%-pwsh.tar: export PWSH_VERSION ?= 7.1.1
${BASE_TAR_PATH}-%-pwsh.tar: export PWSH_ALPINE_IMAGE_VERSION ?= 3.12
${BASE_TAR_PATH}-%-pwsh.tar: export PWSH_IMAGE_DATE ?= 20210114
${BASE_TAR_PATH}-%-pwsh.tar: export IMAGE_SHELL := pwsh
${BASE_TAR_PATH}-%-pwsh.tar: ${BASE_BINARY_PATH}.%
	@mkdir -p $$(dirname $@_)
	@./ci/build_helper_docker $* $@

${BASE_TAR_PATH}-%.tar: export ALPINE_IMAGE_VERSION ?= 3.12.0
${BASE_TAR_PATH}-%.tar: ${BASE_BINARY_PATH}.%
	@mkdir -p $$(dirname $@_)
	@./ci/build_helper_docker $* $@

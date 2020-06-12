# Anka Gitlab Runner

[Offical Usage Guide](http://ankadocs.veertu.com/docs/anka-build-cloud/ci-plugins/gitlab/)

For a list of compatible versions between GitLab and GitLab Runner, consult
the [compatibility section](https://docs.gitlab.com/runner/#compatibility-with-gitlab-versions).

> This is a stripped down and modified version of [the official gitlab-runner](https://github.com/gitlabhq/gitlab-runner/tree/12-10-stable) (version 12.10-stable).

# Development Setup and Details

```
brew install xz
go get gitlab.com/gitlab-org/gitlab-runner
make deps
export PATH="$PATH:$HOME/go/bin" # To load in gox

# Below steps require docker

# Generate helpers and run tests
make test

# Build a single binary for testing
make build_simple

# Build all binaries for linux and darwin
make build_all
```

Test your changes manually with `anka-gitlab-runner --debug --log-level debug`

> When adding new options/flags, add them to `testRegisterCommandRun`

Changes we made from the offical gitlab-runner repo:

  - `executors/anka`
  - `common/version.go`
      - `var NAME` -> anka-gitlab-runner
      - prometheus `Name` -> anka_gitlab...
  - `main.go`: 
      - Commented out executor imports we don't need
      - Added Veertu as author and changed Usage
  - `commands/exec.go`: 
      - Commented out executor imports we don't need
      - Commented out `Add self-volume to docker` code
  - `network/trace.go` + `common/trace.go` + `common/network.go`: 
      - Added `IsJobSuccessful` function
  - `common/config.go`: 
      - Added `AnkaConfig` struct
      - `getDefaultConfigFile`: `config.toml` -> `anka-config.toml` (allows multiple gitlab-runners on same host)
      - Commented out Runners we don't support from `RunnerSettings`
      - Commented out for loop that checkes runner.Machine from `LoadConfig`
      - Commented out any structs and funcs we don't use
  - `common/config_test.go`: 
      - Added `TestConfigParse` tests for anka since the default didn't work after pulling out docker
  - `Makefile`: 
      - Modified `NAME` ENV in  to be `anka-gitlab-runner`
      - Fixed `PKG = ` so it doesn't try to use anka-gitlab-runner as the repo name
      - Removed virtualbox and parallels from `development_setup`
  - `commands/register.go`: 
      - Commented out ask* functions we don't use
      - We duplicated `askSSHLogin`, renamed it to `askAnkaSSHLogin`, then added it to the `exectorFns` so it prompts
      - Commented out exectors we don't use from `exectorFns`
      - Added better prompt messages (s.Name was called description and made unregister confusing)
      - Commented out `transformDockerServices` and anything else Docker related
      - Changed `Invalid executor specified` message to include the executor name
      - Added askAnka
  - `commands/register_test.go`:
      - Replaced everything with anka examples
      - Various modifications to get tests passing
      - Added all available anka options to `testRegisterCommandRun`
  - `commands/multi_test.go`
      - `multi-runner-build-limit` -> anka
  - `commands/service_test.go`
      - gitlab-runner -> anka-gitlab-runner
  - `commands/build_logger_test.go`
      - Added missing `IsJobSuccessful`
  - `commands/service.go`:
      - Updated `defaultServiceName` + `defaultDescription` with anka name
      - Added `service` definition to the import of `gitlab.com/gitlab-org/gitlab-runner/helpers/service`.
      - Added `service "github.com/ayufan/golang-kardianos-service"` to `commands/service.go`
      - Printing a message to `RunServiceControl` when a command is successful
  - `commands/unregister.go`: 
      - Added a failure if you don't specify the runner to unregister
      - Disabled all-runners
  - `commands/user_mode_warning.go`: 
      - gitlab-runner -> anka-gitlab-runner
  - `common/build.go`
      - Added Retries logic to support the new --preparation-retries option
  - `common/build_test.go`
      - Fixed `TestRetryPrepare` and `TestPrepareFailure` for PrepareRetries being 0 by default + failure fix
  - `common/const.go`
      - PreparationRetries = 2
      - const PreparationRetries -> var so tests can change it
  - `VERSION`
      - Added {gitlab runner version}/{anka executor version}
  - `ci/version`
      - Modified echo so the version doesn't contain useless stuff
  - `create-docker.bash`
      - Script for building, tagging, and pushing to veertu/ dockerhub
  - `helpers/gitlab_ci_yaml_parser/`
      - Updated parser.go (and test.go) to handle `anka_template` the same way it does for `image`

> `executor/ssh.go` must stay as an available executor for tests to pass.
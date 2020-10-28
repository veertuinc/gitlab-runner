# Anka GitLab Runner

### [Official Anka GitLab Runner Usage Guide](https://ankadocs.veertu.com/docs/ci-plugins-and-integrations/gitlab/)

For a list of compatible versions between GitLab and GitLab Runner, see the [compatibility section](https://docs.gitlab.com/runner/#compatibility-with-gitlab-versions).

> This is a stripped down and modified version of [the official gitlab-runner](https://github.com/gitlabhq/gitlab-runner/tree/13-2-stable) (version 13.2-stable).

## Anka GitLab Runner Registration Example

[Official GitLab Runner Documentation](https://docs.gitlab.com/runner/)

```bash
./anka-gitlab-runner-darwin-amd64 register --non-interactive \
--url "http://anka.gitlab:8093" \
--registration-token 48EZAzxiF92TsqAVmkph \
--ssh-host host.docker.internal \
--ssh-user anka \
--ssh-password admin \
--name "localhost shared runner" \
--anka-controller-address "https://anka.controller:8090/" \
--anka-template-uuid d09f2a1a-e621-463d-8dfd-8ce9ba9f4160 \
--anka-tag base:port-forward-22:brew-git:gitlab \
--executor anka \
--anka-root-ca-path /Users/hostUser/anka-ca-crt.pem \
--anka-cert-path /Users/hostUser/anka-gitlab-crt.pem \
--anka-key-path /Users/hostUser/anka-gitlab-key.pem \
--clone-url "http://anka.gitlab:8093" \
--tag-list "localhost-shared,localhost,iOS"
```

> Examples of the different log formats can be found [HERE](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#log_format-examples-truncated)

## Example `gitlab-ci.yml`

```yaml
test:
  tags:
    - localhost-shared
  stage: test
  variables:
    # Only use these variables to override the defaults you set when you register the runner.
    ANKA_TEMPLATE_UUID: "c0847bc9-5d2d-4dbc-ba6a-240f7ff08032"
    ANKA_TAG_NAME: "base"
  script:
    - hostname
    - echo "Echo from inside of the VM!"
```

## Development Setup and Details

```bash
brew install xz
go get gitlab.com/gitlab-org/gitlab-runner
make deps
export PATH="$PATH:$HOME/go/bin" # To load in gox
make development_setup

# Run all tests
make simple-test

# Build a single binary for testing
make runner-bin-host

# Build all binaries for linux and darwin (make sure docker daemon experimental = true)
make runner-and-helper-bin-host
```

Test your changes manually with `anka-gitlab-runner --debug --log-level debug run`:

> Using https://github.com/veertuinc/getting-started scripts to setup Gitlab locally

> The gitlab-rails will take a few minutes to do the query. Be patient.

```bash
export GITLAB_DOCKER_CONTAINER_NAME="anka.gitlab"
export GITLAB_PORT="8093"
export GITLAB_ROOT_PASSWORD="rootpassword"
export GITLAB_EXAMPLE_PROJECT_NAME="gitlab-examples"
export GITLAB_ACCESS_TOKEN=$(curl -s --request POST --data "grant_type=password&username=root&password=$GITLAB_ROOT_PASSWORD" http://$GITLAB_DOCKER_CONTAINER_NAME:$GITLAB_PORT/oauth/token | jq -r '.access_token')
export GITLAB_EXAMPLE_PROJECT_ID=$(curl -s --request GET -H "Authorization: Bearer $GITLAB_ACCESS_TOKEN" "http://$GITLAB_DOCKER_CONTAINER_NAME:$GITLAB_PORT/api/v4/projects" | jq -r ".[] | select(.name==\"$GITLAB_EXAMPLE_PROJECT_NAME\") | .id")
export SHARED_REGISTRATION_TOKEN="$(docker exec -i $GITLAB_DOCKER_CONTAINER_NAME bash -c "gitlab-rails runner -e production \"puts Gitlab::CurrentSettings.current_application_settings.runners_registration_token\"")"
export PROJECT_REGISTRATION_TOKEN=$(docker exec -i $GITLAB_DOCKER_CONTAINER_NAME bash -c "gitlab-rails runner -e production \"puts Project.find_by_id($GITLAB_EXAMPLE_PROJECT_ID).runners_token\"")
```

```bash
./out/binaries/anka-gitlab-runner stop; ./out/binaries/anka-gitlab-runner unregister -n "localhost shared runner"; ./out/binaries/anka-gitlab-runner unregister -n "localhost specific runner"; rm -f ./out/binaries/anka-gitlab-runner; make runner-bin-host && \
./out/binaries/anka-gitlab-runner register --non-interactive \
--url "http://$GITLAB_DOCKER_CONTAINER_NAME:$GITLAB_PORT/" \
--registration-token $SHARED_REGISTRATION_TOKEN \
--ssh-user anka \
--ssh-password admin \
--name "localhost shared runner" \
--anka-controller-address "http://anka.controller:8090/" \
--anka-template-uuid c0847bc9-5d2d-4dbc-ba6a-240f7ff08032 \
--anka-tag base:port-forward-22:brew-git:gitlab \
--executor anka \
--anka-controller-http-headers "{ \"HOST\": \"testing123.com\", \"Content-Typee\": \"test\" }" \
--clone-url "http://$GITLAB_DOCKER_CONTAINER_NAME:$GITLAB_PORT" \
--tag-list "localhost-shared,localhost,iOS" && \
./out/binaries/anka-gitlab-runner register --non-interactive \
--url "http://$GITLAB_DOCKER_CONTAINER_NAME:$GITLAB_PORT" \
--registration-token $PROJECT_REGISTRATION_TOKEN \
--ssh-user anka \
--ssh-password admin \
--name "localhost specific runner" \
--anka-controller-address "http://anka.controller:8090/" \
--anka-template-uuid c0847bc9-5d2d-4dbc-ba6a-240f7ff08032 \
--anka-tag base:port-forward-22:brew-git:gitlab \
--executor anka \
--anka-controller-http-headers "{ \"HOST\": \"testing123.com\", \"Content-Typee\": \"test\" }" \
--clone-url "http://$GITLAB_DOCKER_CONTAINER_NAME:$GITLAB_PORT" \
--tag-list "localhost-specific,localhost,iOS" && \
./out/binaries/anka-gitlab-runner stop && ./out/binaries/anka-gitlab-runner --debug --log-level debug run
```

> When adding new options/flags, add them to `testRegisterCommandRun`

### Change Log

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
      - Fixed `TestRetryPrepare` and `TestPrepareFailure` for PrepareRetries being 0 by default (consts.go updated const to var)
  - `common/const.go`
      - PreparationRetries = 2
      - const PreparationRetries -> var so tests can change it
      - shortened TraceForceSendInterval for when users cancel jobs in the UI (30 seconds is too long)
  - `VERSION`
      - Added {gitlab runner version}/{anka executor version}
  - `ci/version`
      - Modified echo so the version doesn't contain useless stuff
  - `create-docker.bash`
      - Script for building, tagging, and pushing to veertu/ dockerhub

> **`executor/ssh.go` must stay as an available executor**
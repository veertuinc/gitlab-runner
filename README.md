# Anka GitLab Runner (Deprecated)

**This Anka Gitlab Runner has been deprecated and replaced with the [Anka Custom Executor]({{< relref "plugins-and-integrations/controller-+-registry/gitlab-custom-executor.md" >}}) you can load into the official Gitlab Runner.**

### [Official Anka GitLab Runner Usage Guide](https://docs.veertu.com/anka/plugins-and-integrations/controller-+-registry/gitlab-runner/)

For a list of compatible versions between GitLab and GitLab Runner, see the [compatibility section](https://docs.gitlab.com/runner/#compatibility-with-gitlab-versions).

> This is a lightly modified version of [the official gitlab-runner](https://github.com/gitlabhq/gitlab-runner). See the VERSION file for information on which version of the core gitlab-runner we're using.

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
--anka-tag v1 \
--executor anka \
--anka-controller-instance-name "triggered by shared runner" \ # Defaults to "Anka Gitlab Runner Name: localhost shared runner" in Controller Instances Name column
--anka-controller-external-id "http://test123.com" \           # Defaults to full job URL. Example: http://anka.gitlab:8093/root/gitlab-examples/-/jobs/28
--anka-root-ca-path /Users/hostUser/anka-ca-crt.pem \
--anka-cert-path /Users/hostUser/anka-gitlab-crt.pem \
--anka-key-path /Users/hostUser/anka-gitlab-key.pem \
--clone-url "http://anka.gitlab:8093" \
--tag-list "localhost-shared,localhost,iOS"
```

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
    ANKA_NODE_GROUP: "larger-vm-pool"
    # ANKA_CONTROLLER_INSTANCE_NAME: "" # Defaults to "Anka Gitlab Runner Name: localhost shared runner" in Controller Instances Name column
    # ANKA_CONTROLLER_EXTERNAL_ID: "" # Defaults to full job URL. Example: http://anka.gitlab:8093/root/gitlab-examples/-/jobs/28
  script:
    - hostname
    - echo "Echo from inside of the VM!"
```

### Variable interpolation in your YAML

Gitlab level variables set under the Project, Group, Instance, etc, can be used in your `gitlab-ci.yml`. These are set using $ before the variable's name:

```yaml
build:
  stage: build
  variables:
    ANKA_TEMPLATE_UUID: "$DEFAULT_ANKA_TEMPLATE_UUID"
```

## Development Setup and Details

```bash
brew install xz
go get gitlab.com/gitlab-org/gitlab-runner
make deps
export PATH="$PATH:$HOME/go/bin" # To load in gox

# Build a single binary for testing
make runner-bin-host

# Build all binaries for linux and darwin (make sure docker daemon experimental = true)
make runner-and-helper-bin-host
```

Test your changes manually with `./out/binaries/anka-gitlab-runner-darwin-amd64 --debug --log-level debug run`:

> Try our https://github.com/veertuinc/getting-started scripts to run Gitlab locally inside of a docker container

```bash
export GITLAB_HOSTNAME="anka.gitlab";
export GITLAB_DOCKER_CONTAINER_NAME="anka.gitlab";
export GITLAB_PORT="8093";
export GITLAB_ROOT_PASSWORD="kLF2Cx2XmaWdBwcKAmWRD/Ew9eifMCnyPUFnUPlk6Lw=";
export GITLAB_EXAMPLE_PROJECT_NAME="gitlab-examples";
export GITLAB_ACCESS_TOKEN="token-string-here123";
export GITLAB_EXAMPLE_PROJECT_ID=$(curl -s --request GET -H "PRIVATE-TOKEN: $GITLAB_ACCESS_TOKEN" "http://$GITLAB_HOSTNAME:$GITLAB_PORT/api/v4/projects" | jq -r ".[] | select(.name==\"$GITLAB_EXAMPLE_PROJECT_NAME\") | .id");
export SHARED_REGISTRATION_TOKEN="$(docker exec -i $GITLAB_DOCKER_CONTAINER_NAME bash -c "gitlab-rails runner -e production \"puts Gitlab::CurrentSettings.current_application_settings.runners_registration_token\"")";
export PROJECT_REGISTRATION_TOKEN=$(docker exec -i $GITLAB_DOCKER_CONTAINER_NAME bash -c "gitlab-rails runner -e production \"puts Project.find_by_id($GITLAB_EXAMPLE_PROJECT_ID).runners_token\"");
```

> The gitlab-rails will take a few minutes to do the query. Be patient.

```bash
./out/binaries/anka-gitlab-runner-darwin-amd64 stop; ./out/binaries/anka-gitlab-runner-darwin-amd64 unregister -n "localhost shared runner"; ./out/binaries/anka-gitlab-runner-darwin-amd64 unregister -n "localhost specific runner"; rm -f ./out/binaries/anka-gitlab-runner-darwin-amd64; make runner-bin-host && \
./out/binaries/anka-gitlab-runner-darwin-amd64 register --non-interactive \
--url "http://$GITLAB_HOSTNAME:$GITLAB_PORT/" \
--registration-token $SHARED_REGISTRATION_TOKEN \
--ssh-user anka \
--ssh-password admin \
--name "localhost shared runner" \
--anka-controller-address "http://anka.controller:8090/" \
--anka-template-uuid 5d1b40b9-7e68-4807-a290-c59c66e926b4 \
--anka-tag v1 \
--executor anka \
--anka-controller-instance-name "triggered by shared runner" \
--anka-controller-external-id "http://test123share.com" \
--anka-controller-http-headers "{ \"HOST\": \"testing123.com\", \"Content-Type\": \"test\" }" \
--clone-url "http://$GITLAB_HOSTNAME:$GITLAB_PORT" \
--tag-list "localhost-shared,localhost,iOS" && \
./out/binaries/anka-gitlab-runner-darwin-amd64 register --non-interactive \
--url "http://$GITLAB_HOSTNAME:$GITLAB_PORT" \
--registration-token $PROJECT_REGISTRATION_TOKEN \
--ssh-user anka \
--ssh-password admin \
--name "localhost specific runner" \
--anka-controller-address "http://anka.controller:8090/" \
--anka-template-uuid 5d1b40b9-7e68-4807-a290-c59c66e926b4 \
--anka-tag v1 \
--executor anka \
--anka-controller-instance-name "triggered by project runner" \
--anka-controller-external-id "http://test123proj.com" \
--anka-controller-http-headers "{ \"HOST\": \"testing123.com\", \"Content-Type\": \"test\" }" \
--clone-url "http://$GITLAB_HOSTNAME:$GITLAB_PORT" \
--tag-list "localhost-specific,localhost,iOS" && \
./out/binaries/anka-gitlab-runner-darwin-amd64 stop; ./out/binaries/anka-gitlab-runner-darwin-amd64 --debug --log-level debug run -c $HOME/.gitlab-runner/anka-config.toml
```

> When adding new options/flags, add them to `testRegisterCommandRun`

### Change Log

Changes we made from the offical gitlab-runner repo:

  - `executors/anka`
  - `common/version.go`
      - `var NAME` -> anka-gitlab-runner
      - prometheus `Name` -> anka_gitlab...
  - `main.go`: 
      - Added anka executor import
      - Added Veertu as author and changed Usage
  - `commands/exec.go`: 
      - Added anka executor import
  - `network/trace.go` + `common/trace.go` + `common/network.go`: 
      - Added `IsJobSuccessful` function
  - `common/config.go`: 
      - Added `AnkaConfig` struct
      - Added `Anka` and `PreparationRetries` to RunnerSettings struct
  - `commands/config.go`:
      - `getDefaultConfigFile`: `config.toml` -> `anka-config.toml` (allows multiple gitlab-runners on same host)
  - `Makefile`: 
      - Modified `NAME` ENV in  to be `anka-gitlab-runner`
      - Fixed `PKG = ` so it doesn't try to use anka-gitlab-runner as the repo name
      - Removed platforms and archs we don't build for
  - `commands/register.go`:
      - Added several imports
      - We duplicated `askSSHLogin`, renamed it to `askAnkaSSHLogin`, then added it to the `exectorFns` so it prompts
      - Added askAnka
      - Updated the description for s.Name, Token, TagList to remove any confusion as to what they're for (they're for gitlab, not anka executor)
      - Changed `Invalid executor specified` message/Paniclns to include the executor name
  - `commands/service_test.go`
      - gitlab-runner -> anka-gitlab-runner
  - `commands/service.go`:
      - Updated `defaultServiceName` + `defaultDescription` with anka name
      - `runServiceInstall` Fatal message update: anka-gitlab-runner
      - Printing a message to `RunServiceControl` when a command is successful
  - `commands/unregister.go`: 
      - Added a failure if you don't specify the runner to unregister
      - Disabled all-runners
  - `commands/user_mode_warning.go`: 
      - gitlab-runner -> anka-gitlab-runner
  - `common/build.go`
      - Added Retries logic to support the new --preparation-retries option
  - `common/const.go`
      - PreparationRetries = 0
      - const PreparationRetries -> var
      - shortened TraceForceSendInterval for when users cancel jobs in the UI to 10s
  - `VERSION`
      - Added {gitlab runner version}/{anka executor version}
  - `ci/version`
      - Modified echo to just show version
  - `build-and*` script for building, tagging, and pushing to veertu/ dockerhub

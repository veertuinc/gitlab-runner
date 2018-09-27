# GitLab Runner Fork

This is **NOT** the repository of the official GitLab Runner.

# Anka cloud support

You can use this fork in order to integrate your Anka cloud with gitlab.  

For more information about Anka go to <a href="https://veertu.com" target="_blank">veertu.com</a>.  

# How To Use This Package

## Step 1 - Prepare the base image

Start or create an Anka VM. 
Install git.
Install your dependencies.
Suspend the VM.
Push it to Registry.

## Step 2 - Install the alternative runner

Download a binary from the <a href="https://github.com/veertuinc/gitlab-runner/releases/">releases page</a> 

Copy the file to /usr/local/bin/gitlab-runner.

Give the file run permissions


You can also follow the additional instructions in the <a href="https://docs.gitlab.com/runner/install/linux-manually.html">gitlab installation instructions</a>


## Install the runner service 

```
gitlab-runner install
```

## Configure the runner

Follow the instructions on <a href="https://docs.gitlab.com/runner/register/index.html">gitlab instructions page</a>.
In step 6 "Enter the Runner executor:"

enter "anka"


After the runner is configured a config.toml file should be in ~/.gitlab-runner/config.toml or in /etc/gitlab-runner/config.toml


under [runners.ssh] , add user and password for ssh

under [runners.anka] , add controller_address and image_id

controller address is the Anka cloud controller address (with port)

image_id is the id of the VM you prepared in step 1


your configuration should look similar to this


```

concurrent = 1
check_interval = 0

[[runners]]
  name = "RUNNER_NAME"
  url = "http://gitlab-server.net"
  token = "********"
  executor = "anka"
  [runners.ssh]
    user = "anka"
    password = "admin"
  [runners.cache]
  [runners.anka]
    controller_address = "http://CONTROLLER_HOST:CONTROLLER_PORT"
    image_id = "IMAGE_ID"

```

## Run the runner

```
gitlab-runner start
```

## License

This code is distributed under the MIT license, see the [LICENSE](LICENSE) file.

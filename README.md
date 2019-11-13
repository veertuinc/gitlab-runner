# GitLab Runner Fork

This is **NOT** the repository of the official GitLab Runner.

# Anka cloud support
You can use this fork in order to integrate your Anka cloud with gitlab.  

For more information about Anka go to <a href="https://veertu.com" target="_blank">veertu.com</a>.  
For a list of compatible versions between GitLab and GitLab Runner, consult
the [compatibility section](https://docs.gitlab.com/runner/#compatibility-with-gitlab-versions).

# How To Use This Package

### Prepare a base image

Start or create an Anka VM. 
Install git.
Install your dependencies.
Suspend/Stop the VM.

Configure port forwarding for ssh:

```
anka modify $VM_NAME add port-forwarding --guest-port 22 --host-port 0 --protocol tcp ssh1
```

Push it to Registry.



Once you have a base image, there are 2 options.

## Option 1 - Run with Docker

```
docker run -t asafg6/gitlab-anka-runner --executor anka \
--url http://YOUR_GITLAB_HOST \
--registration-token ** \
--ssh-user VM_USER \
--ssh-password VM_PASSWORD \
--anka-controller-address http://ANKA_CLOUD_ADDRESS \
--anka-image-id IMAGE_ID \
--name my-anka-runner
```

You can also append whatever arguments you would pass to gitlab-runner register.
For example ---locked, --run-untagged, --tag-list, etc....

More Anka optional parameters:

--anka-tag value Use a specific tag  
--anka-node-id value Run on a specific node  
--anka-priority value Override the task's default priority  
--anka-group-id value Run on a specific node group  
--anka-keep-alive-on-error Keep the VM alive in case of error for debugging purposes  


## Option 2 - Install And Run Manually

### Install the alternative runner

Download a binary from the <a href="https://github.com/veertuinc/gitlab-runner/releases/">releases page</a> 

Copy the file to /usr/local/bin/gitlab-runner.

Give the file run permission


You can also follow the additional instructions in the <a href="https://docs.gitlab.com/runner/install/linux-manually.html">gitlab installation instructions</a>


### Install the runner service 

```
gitlab-runner install
```

### Configure the runner

Follow the instructions on <a href="https://docs.gitlab.com/runner/register/index.html">gitlab instructions page</a>.
In step 6 "Enter the Runner executor:  

enter "anka"

The register process will ask you for more configuration parameters.


For Anka Cloud Address, enter your controller url.  
For Anka Image id, enter the id of the VM you prepared in step 1  
If you want enter the registry tag you want Anka to use.  
For SSH User, enter the ssh user for your anka VM  
For SSH Password, enter the ssh password for your anka VM  

After the runner is configured a config.toml file should be in ~/.gitlab-runner/config.toml or in /etc/gitlab-runner/config.toml


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

- 2014 - 2015   : [Kamil Trzciński](mailto:ayufan@ayufan.eu)
- 2015 - now    : GitLab Inc. team and contributors

## License

This code is distributed under the MIT license, see the [LICENSE](LICENSE) file.

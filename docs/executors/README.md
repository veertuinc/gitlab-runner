# Executors

GitLab Runner implements a number of executors that can be used to run your
builds in different scenarios. If you are not sure what to select, read the
[I am not sure](#i-am-not-sure) section.
Visit the [compatibility chart](#compatibility-chart) to find
out what features each executor does and does not support.

To jump into the specific documentation for each executor, visit:

- [SSH](ssh.md)
- [Shell](shell.md)
- [Parallels](parallels.md)
- [VirtualBox](virtualbox.md)
- [Docker](docker.md)
- [Docker Machine (auto-scaling)](docker_machine.md)
- [Kubernetes](kubernetes.md)

## Selecting the executor

The executors support different platforms and methodologies for building a
project. The table below shows the key facts for each executor which will help
you decide which executor to use.

| Executor                                          | SSH  | Shell   | VirtualBox | Parallels | Docker | Kubernetes |
|:--------------------------------------------------|:----:|:-------:|:----------:|:---------:|:------:|:----------:|
| Clean build environment for every build           | ✗    | ✗       | ✓          | ✓         | ✓      | ✓          |
| Migrate runner machine                            | ✗    | ✗       | partial    | partial   | ✓      | ✓          |
| Zero-configuration support for concurrent builds  | ✗    | ✗ (1)   | ✓          | ✓         | ✓      | ✓          |
| Complicated build environments                    | ✗    | ✗ (2)   | ✓ (3)      | ✓ (3)     | ✓      | ✓          |
| Debugging build problems                          | easy | easy    | hard       | hard      | medium | medium     |

1. It's possible, but in most cases it is problematic if the build uses services
   installed on the build machine
2. It requires to install all dependencies by hand
3. For example using [Vagrant](https://www.vagrantup.com/docs/virtualbox/ "Vagrant documentation for VirtualBox")

### I am not sure

#### Shell Executor

**Shell** is the simplest executor to configure. All required dependencies for
your builds need to be installed manually on the same machine that the Runner is
installed on.

#### Virtual Machine Executor (VirtualBox / Parallels)

This type of executor allows you to use an already created virtual machine, which
is cloned and used to run your build. We offer two full system virtualization
options: **VirtualBox** and **Parallels**. They can prove useful if you want to run
your builds on different operating systems, since it allows the creation of virtual
machines on Windows, Linux, OSX or FreeBSD, then GitLab Runner connects to the
virtual machine and runs the build on it. Its usage can also be useful for reducing
infrastructure costs.

#### Docker Executor

A great option is to use **Docker** as it allows a clean build environment,
with easy dependency management (all dependencies for building the project can
be put in the Docker image). The Docker executor allows you to easily create
a build environment with dependent [services](https://docs.gitlab.com/ee/ci/services/README.html),
like MySQL.

#### Docker Machine

The **Docker Machine** is a special version of the **Docker** executor
with support for auto-scaling. It works like the normal **Docker** executor
but with build hosts created on demand by _Docker Machine_.

#### Kubernetes Executor

The **Kubernetes** executor allows you to use an existing Kubernetes cluster
for your builds. The executor will call the Kubernetes cluster API
and create a new Pod (with a build container and services containers) for
each GitLab CI job.

#### SSH Executor

The **SSH** executor is added for completeness, but it's the least supported
among all executors. It makes GitLab Runner connect to an external server
and runs the builds there. We have some success stories from organizations using
this executor, but usually we recommend using one of the other types.

## Compatibility chart

Supported features by different executors:

| Executor                                     | SSH  | Shell   | VirtualBox | Parallels | Docker | Kubernetes |
|:---------------------------------------------|:----:|:-------:|:----------:|:---------:|:------:|:----------:|
| Secure Variables                             | ✓    | ✓       | ✓          | ✓         | ✓      | ✓          |
| GitLab Runner Exec command                   | ✗    | ✓       | ✗          | ✗         | ✓      | ✓          |
| gitlab-ci.yml: image                         | ✗    | ✗       | ✗          | ✗         | ✓      | ✓          |
| gitlab-ci.yml: services                      | ✗    | ✗       | ✗          | ✗         | ✓      | ✓          |
| gitlab-ci.yml: cache                         | ✓    | ✓       | ✓          | ✓         | ✓      | ✓          |
| gitlab-ci.yml: artifacts                     | ✓    | ✓       | ✓          | ✓         | ✓      | ✓          |
| Absolute paths: caching, artifacts           | ✗    | ✗       | ✗          | ✗         | ✗      | ✓          |
| Passing artifacts between stages             | ✓    | ✓       | ✓          | ✓         | ✓      | ✓          |
| Use GitLab Container Registry private images | n/a  | n/a     | n/a        | n/a       | ✓      | ✓          |
| Interactive Web terminal                     | ✗    | ✓ (bash)| ✗          | ✗         | ✓      | ✓          |

Supported systems by different shells:

| Shells  | Bash        | Windows Batch | PowerShell |
|:-------:|:-----------:|:-------------:|:----------:|
| Windows | ✓           | ✓ (default)   | ✓          |
| Linux   | ✓ (default) | ✗             | ✗          |
| OSX     | ✓ (default) | ✗             | ✗          |
| FreeBSD | ✓ (default) | ✗             | ✗          |

Supported systems for interactive web terminals by different shells:

| Shells  | Bash        | Windows Batch | PowerShell |
|:-------:|:-----------:|:-------------:|:----------:|
| Windows | ✗           | ✗             | ✗          |
| Linux   | ✓           | ✗             | ✗          |
| OSX     | ✓           | ✗             | ✗          |
| FreeBSD | ✓           | ✗             | ✗          |

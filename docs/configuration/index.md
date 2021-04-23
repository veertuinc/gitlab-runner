---
stage: Verify
group: Runner
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/engineering/ux/technical-writing/#assignments
comments: false
---

# Configuring GitLab Runner

Learn how to configure GitLab Runner.

- [Advanced configuration options](advanced-configuration.md): Use
  the [`config.toml`](https://github.com/toml-lang/toml) configuration file
  to edit runner settings.
- [Use self-signed certificates](tls-self-signed.md): Configure certificates
  that verify TLS peers when connecting to the GitLab server.
- [Autoscale with Docker Machine](autoscale.md): Execute jobs on machines
  created automatically by Docker Machine.
- [Autoscale GitLab Runner on AWS EC2](runner_autoscale_aws/index.md): Execute jobs on auto-scaled AWS EC2 instances.
- [Autoscale GitLab CI on AWS Fargate](runner_autoscale_aws_fargate/index.md):
  Use the AWS Fargate driver with the GitLab custom executor to run jobs in AWS ECS.
- [Graphical Processing Units](gpus.md): Use GPUs to execute jobs.
- [The init system](init.md): GitLab Runner installs
  its init service files based on your operating system.
- [Supported shells](../shells/index.md): Execute builds on different systems by
  using shell script generators.
- [Security considerations](../security/index.md): Be aware of potential
  security implications when running your jobs with GitLab Runner.
- [Runner monitoring](../monitoring/README.md): Monitor the behavior of your
  runners.
- [Clean up Docker cache automatically](../executors/docker.md#clearing-docker-cache):
  If you are running low on disk space, use a cron job to clean old containers and volumes.
- [Configure GitLab Runner to run behind a proxy](proxy.md): Set
  up a Linux proxy and configure GitLab Runner. Useful for the
  Docker executor.
- [Best practices](../best_practice/index.md).
- [Handling rate limited requests](proxy.md#handling-rate-limited-requests).
- [Configure GitLab Runner Operator on OpenShift](configuring_runner_openshift.md).

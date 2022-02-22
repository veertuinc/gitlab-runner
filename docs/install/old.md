---
stage: Verify
group: Runner
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/engineering/ux/technical-writing/#assignments
---

# Old GitLab Runner URLs **(FREE)**

NOTE:
Take a look at the [compatibility section](../index.md#gitlab-runner-versions) to check GitLab Runner compatibility
with your version of GitLab.

In GitLab Runner 10, the name of the executable was renamed from
`gitlab-ci-multi-runner` to `gitlab-runner`. With that change, GitLab Runner
[has a new home](https://gitlab.com/gitlab-org/gitlab-runner) and the package
repository [was renamed as well](https://packages.gitlab.com/runner/gitlab-runner).

## Using the Linux repository

For versions **prior to 10.0**, the repository URLs are:

```shell
# For Debian/Ubuntu
curl -L "https://packages.gitlab.com/install/repositories/runner/gitlab-ci-multi-runner/script.deb.sh" | sudo bash

# For RHEL/CentOS
curl -L "https://packages.gitlab.com/install/repositories/runner/gitlab-ci-multi-runner/script.rpm.sh" | sudo bash
```

## Downloading the binaries manually

For manual installations, the old GitLab Runner binaries can be found under
<https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/index.html>.

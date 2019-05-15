# GitLab Runner

This is the repository of the official GitLab Runner written in Go.
It runs tests and sends the results to GitLab.
[GitLab CI](https://about.gitlab.com/gitlab-ci) is the open-source
continuous integration service included with GitLab that coordinates the testing.
The old name of this project was GitLab CI Multi Runner but please use "GitLab Runner" (without CI) from now on.

![Build Status](https://gitlab.com/gitlab-org/gitlab-runner/badges/master/build.svg)

## Runner and GitLab CE/EE compatibility

For a list of compatible versions between GitLab and GitLab Runner, consult
the [compatibility chart](https://docs.gitlab.com/runner/#compatibility-chart).

## Release process

The description of release process of GitLab Runner project can be found in the [release documentation](docs/release_process/README.md).

## Contributing

Contributions are welcome, see [`CONTRIBUTING.md`](CONTRIBUTING.md) for more details.

### Closing issues and merge requests

GitLab is growing very fast and we have a limited resources to deal with reported issues
and merge requests opened by the community volunteers. We appreciate all the contributions
coming from our community. But to help all of us with issues and merge requests management
we need to create some closing policy.

If an issue or merge request has a ~"waiting for feedback" label and the response from the
reporter has not been received for 14 days, we can close it using the following response
template:

```
We haven't received an update for more than 14 days so we will assume that the
problem is fixed or is no longer valid. If you still experience the same problem
try upgrading to the latest version. If the issue persists, reopen this issue
or merge request with the relevant information.
```

### Contributing to documentation

If your contribution contains only documentation changes, you can speed up the CI process
by following some branch naming conventions, as described in https://docs.gitlab.com/ce/development/documentation/index.html#branch-naming

## Documentation

The documentation source files can be found under the [docs/](docs/) directory. You can
read the documentation online at https://docs.gitlab.com/runner/.

## Requirements

[Read about the requirements of GitLab Runner.](https://docs.gitlab.com/runner/#requirements)

## Features

[Read about the features of GitLab Runner.](https://docs.gitlab.com/runner/#features)

## Executors compatibility chart

[Read about what options each executor can offer.](https://docs.gitlab.com/runner/executors/#compatibility-chart)

## Install GitLab Runner

Visit the [installation documentation](https://docs.gitlab.com/runner/install/).

## Use GitLab Runner

See [https://docs.gitlab.com/runner/#using-gitlab-runner](https://docs.gitlab.com/runner/#using-gitlab-runner).

## Select executor

See [https://docs.gitlab.com/runner/executors/#selecting-the-executor](https://docs.gitlab.com/runner/executors/#selecting-the-executor).

## Troubleshooting

Read the [FAQ](https://docs.gitlab.com/runner/faq/).

## Advanced Configuration

See [https://docs.gitlab.com/runner/#advanced-configuration](https://docs.gitlab.com/runner/#advanced-configuration).

## Building and development

See [https://docs.gitlab.com/runner/development/](https://docs.gitlab.com/runner/development/).

## Changelog

Visit the [Changelog](CHANGELOG.md) to view recent changes.

## The future

* Please see the [GitLab Direction page](https://about.gitlab.com/direction/).
* Feel free submit issues with feature proposals on the issue tracker.

## Author

- 2014 - 2015   : [Kamil Trzciński](mailto:ayufan@ayufan.eu)
- 2015 - now    : GitLab Inc. team and contributors

## License

This code is distributed under the MIT license, see the [LICENSE](LICENSE) file.

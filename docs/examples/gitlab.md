# How to configure GitLab Runner for GitLab CE integration tests

We will register the Runner using a confined Docker executor.

The registration token can be found at `https://gitlab.com/project_namespace/project_name/runners`.
You can export it as a variable and run the commands below as-is. Start by
creating a template configuration file in order to pass complex configuration:

```shell
$ cat > /tmp/test-config.template.toml << EOF
[[runners]]
[runners.docker]
[[runners.docker.services]]
name = "mysql:latest"
[[runners.docker.services]]
name = "redis:latest"
EOF
```

Finally, register the runner, passing the newly created template configuration file:

```shell
gitlab-runner register \
  --non-interactive \
  --url "https://gitlab.com" \
  --registration-token "$REGISTRATION_TOKEN" \
  --template-config /tmp/test-config.template.toml \
  --description "gitlab-ce-ruby-2.6" \
  --executor "docker" \
  --docker-image ruby:2.6
```

You now have a GitLab CE integration testing instance with bundle caching.
Push some commits to test it.

For [advanced configuration](../configuration/advanced-configuration.md), look into
`/etc/gitlab-runner/config.toml` and tune it.

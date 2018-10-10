# Run this after running build.sh
FROM ubuntu:16.04
MAINTAINER Veertu Inc. "support@veertu.com"

COPY gitlab-runner-linux-386 /usr/local/bin/gitlab-runner
RUN gitlab-runner install --user root
# RUN gitlab-runner start
COPY register_and_run.sh /tmp/register_and_run.sh
RUN chmod +x /tmp/register_and_run.sh

ENTRYPOINT ["/bin/bash", "/tmp/register_and_run.sh"]
CMD ['/usr/local/bin/gitlab-runner', 'run']
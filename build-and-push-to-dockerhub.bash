#!/bin/bash
set -eo pipefail
ROOT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
VERSION=$(cat $ROOT_DIR/VERSION | cut -d/ -f2)
cd $ROOT_DIR
# Build binaries
echo "Building binaries..."
make runner-bin
# Create dockerfile
for arch in amd64 386; do
  cd $ROOT_DIR
  echo "Building $arch docker tags..."
  FROM="$arch/ubuntu:20.04"
  [[ $arch == 386 ]] && FROM="i$arch/ubuntu:20.04"
cat > out/binaries/register_and_run.sh <<BLOCK
#!/bin/bash
set -exo pipefail
export RUNNER_OPTIONS="\${RUNNER_OPTIONS:-}"
export RUN_OPTIONS="\${RUN_OPTIONS:-}"
export ARR=("\$@")
unregister() {
  for i in "\${!ARR[@]}"; do
    [[ "\${ARR[\$i]}" == "--name" ]] && NAME_INDEX=\$i
  done
  RUNNER_NAME="\${ARR[\${NAME_INDEX}+1]}"
  echo "UNREGISTERING \$RUNNER_NAME"
  /usr/local/bin/anka-gitlab-runner unregister -n "\$RUNNER_NAME"
}
trap unregister EXIT
update-ca-certificates
/usr/local/bin/anka-gitlab-runner \${RUNNER_OPTIONS} register --non-interactive "\$@"
/usr/local/bin/anka-gitlab-runner \${RUNNER_OPTIONS} run \${RUN_OPTIONS}
BLOCK
cat > out/binaries/Dockerfile <<BLOCK
  FROM $FROM
  MAINTAINER Veertu Inc. "support@veertu.com"
  COPY anka-gitlab-runner-linux-$arch /usr/local/bin/anka-gitlab-runner
  RUN anka-gitlab-runner install --user root
  # RUN gitlab-runner start
  COPY register_and_run.sh /tmp/register_and_run.sh
  RUN chmod +x /tmp/register_and_run.sh
  RUN apt-get update
  RUN apt-get install -y ca-certificates
  ENTRYPOINT ["/bin/bash", "/tmp/register_and_run.sh"]
BLOCK
  # Build dockerfile
  cd out/binaries/
  [[ $arch == 386 ]] && arch="i$arch"
  [[ $2 == '--and-latest' ]] && LATEST="-t veertu/anka-gitlab-runner-$arch:latest"
  docker build $LATEST -t veertu/anka-gitlab-runner-$arch:v$VERSION .
  if [[ $1 == "--push" ]]; then
    # Push to dockerhub
    [[ $2 == '--and-latest' ]] && docker push veertu/anka-gitlab-runner-$arch:latest
    docker push veertu/anka-gitlab-runner-$arch:v$VERSION
  fi
done
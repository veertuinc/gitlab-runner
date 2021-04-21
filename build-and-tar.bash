#!/bin/bash
set -eo pipefail
ROOT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
VERSION=$(cat $ROOT_DIR/VERSION | cut -d/ -f2)
cd $ROOT_DIR
# Build binaries
echo "Building binaries..."
make runner-bin
for arch in linux-amd64 linux-386 darwin-amd64; do
  mkdir -p $ROOT_DIR/out/archived_binaries
  cp $ROOT_DIR/out/binaries/anka-gitlab-runner-$arch $ROOT_DIR/out/archived_binaries/
  pushd $ROOT_DIR/out/archived_binaries
  echo "Creating tar.gz for $arch binary..."
  rm -f anka-gitlab-runner-v$VERSION-$arch.tar.gz
  tar -czvf anka-gitlab-runner-v$VERSION-$arch.tar.gz anka-gitlab-runner-$arch
  rm -f anka-gitlab-runner-$arch
  popd
done

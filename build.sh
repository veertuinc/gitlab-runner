
set -e 

OSS=("darwin" "linux")
ARCHS=("386" "amd64")

for OS in ${OSS[@]}; do
    for ARCH in ${ARCHS[@]}; do
        echo "Building for $OS $ARCH"
        env GOOS=$OS GOARCH=$ARCH go build -o gitlab-runner-$OS-$ARCH .
        echo "Built gitlab-runner-$OS-$ARCH"
    done
done
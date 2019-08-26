#!/usr/bin/env bash
set -e
cd `dirname $0`
cd ..

# This script builds executables for multiple platforms and architectures 
# it is used by the CI system to output releases. When testing locally it shouldn't be required
# unless you wish to share a build with someone on a different platform
platforms=("linux/amd64" "windows/amd64" "windows/386" "darwin/amd64" "linux/386" "linux/arm")

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    CGO_ENABLED=0 
    output_name='./bin/terraform-provider-kubectl-'$GOOS'-'$GOARCH
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi  
    echo "Building for $GOOS $GOARCH..."

    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=$CGO_ENABLED go build -a -installsuffix cgo -o $output_name

    if [ $? -ne 0 ]; then
        echo 'An error has occurred! Aborting the script execution...'
        exit 1
    fi
done

echo "Completed builds, for output see ./bin"

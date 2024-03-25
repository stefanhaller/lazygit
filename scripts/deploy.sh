#!/bin/sh

set -ex

server=/Volumes/teams/development/tools/lazygit

test -d $server || {
    echo File server not mounted
    exit 1
}

VERSION=0.53.0

GOOS=darwin GOARCH=amd64 go build -o lazygit_amd64 -ldflags="-X 'main.version=$VERSION'"
GOOS=darwin GOARCH=arm64 go build -o lazygit_arm64 -ldflags="-X 'main.version=$VERSION'"
lipo -create -output lazygit lazygit_amd64 lazygit_arm64
rm lazygit_amd64 lazygit_arm64

GOOS=windows GOARCH=amd64 go build -ldflags="-X 'main.version=$VERSION'"

mv lazygit $server/Mac
mv lazygit.exe $server/Windows

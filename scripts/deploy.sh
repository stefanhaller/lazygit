#!/bin/sh

set -ex

server=/Volumes/teams/development/tools/lazygit

test -d $server || {
    echo File server not mounted
    exit 1
}

GOOS=darwin GOARCH=amd64 go build -o lazygit_amd64
GOOS=darwin GOARCH=arm64 go build -o lazygit_arm64
lipo -create -output lazygit lazygit_amd64 lazygit_arm64
rm lazygit_amd64 lazygit_arm64

GOOS=windows GOARCH=amd64 go build

mv lazygit $server/Mac
mv lazygit.exe $server/Windows

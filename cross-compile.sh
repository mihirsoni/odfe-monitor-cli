#!/bin/bash

VERSION="1.2"

rm -rf dist; mkdir -p dist
for GOOS in darwin linux windows; do
  for GOARCH in 386 amd64; do
    echo "Building $GOOS-$GOARCH"
    if [[ $GOOS == "windows" ]]; then
      env GOOS=$GOOS GOARCH=$GOARCH go build -o dist/odfe-monitor-cli-${VERSION}-${GOOS}-${GOARCH}.exe
    elif [[ $GOOS == "darwin" ]]; then
      env GOOS=$GOOS GOARCH=$GOARCH go build -o dist/odfe-monitor-cli-${VERSION}-mac-${GOARCH}
    else
      env GOOS=$GOOS GOARCH=$GOARCH go build -o dist/odfe-monitor-cli-${VERSION}-${GOOS}-${GOARCH}
    fi
  done
done

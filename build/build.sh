#!/bin/bash

echo " > Running build.sh"
set -e
CGO_ENABLED=0 go build -a -v -i -installsuffix cgo -ldflags '-s -w' -o output/search-collector ./
make docker/build
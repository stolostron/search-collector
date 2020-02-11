#!/bin/bash
# build
echo "building"
CGO_ENABLED=0 go build -a -v -i -installsuffix cgo -ldflags '-s -w' -o output/search-collector ./

#build linix??
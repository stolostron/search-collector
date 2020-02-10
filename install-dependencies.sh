#!/bin/bash

# install dependencies
echo "installing dependencies for collector ..."
go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
go get -u github.com/golang/dep/cmd/dep
dep ensure -v
#!/bin/bash
GOLANGCI_LINT_VERSION="v1.47.1"
GOLANGCI_LINT_CACHE=/tmp/golangci-cache
GOPATH=$(go env GOPATH)
export GOFLAGS=""
DOWNLOAD_URL="https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
curl -sSfL "${DOWNLOAD_URL}" | sh -s -- -b "${GOPATH}/bin" v1.47.1
GOLANGCI_LINT_CACHE=${GOLANGCI_LINT_CACHE} CGO_ENABLED=0 GOGC=25 golangci-lint run
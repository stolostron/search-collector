# Copyright (c) 2020 Red Hat, Inc.
# Copyright Contributors to the Open Cluster Management project

include build/Configfile
BINDIR ?= output

USE_VENDORIZED_BUILD_HARNESS ?=

ifndef USE_VENDORIZED_BUILD_HARNESS
-include $(shell curl -s -H 'Authorization: token ${GITHUB_TOKEN}' -H 'Accept: application/vnd.github.v4.raw' -L https://api.github.com/repos/stolostron/build-harness-extensions/contents/templates/Makefile.build-harness-bootstrap -o .build-harness-bootstrap; echo .build-harness-bootstrap)
else
-include vbh/.build-harness-vendorized
endif

default::
	@echo "Build Harness Bootstrapped"

.PHONY: deps
deps:
	go mod tidy

.PHONY: build
build:
	CGO_ENABLED=0 GOGC=25 go build -o $(BINDIR)/search-collector ./

.PHONY: build-linux
build-linux:
	make build GOOS=linux

.PHONY: lint
lint:
	GOPATH=$(go env GOPATH)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${GOPATH}/bin" v1.47.1
	CGO_ENABLED=0 GOGC=25 golangci-lint run --timeout=3m
	
run:
	GOGC=25 go run main.go --v=2

.PHONY: test
test:
	DEPLOYED_IN_HUB=true go test ./... -v -coverprofile cover.out

.PHONY: coverage
coverage:
	go tool cover -html=cover.out -o=cover.html


.PHONY: clean
clean::
	go clean
	rm -f cover*
	rm -rf ./$(BINDIR)
	rm -rf ./.vendor-new
	rm -rf ./vendor

# Build the docker image
docker-build: 
	docker build -f Dockerfile . -t search-collector
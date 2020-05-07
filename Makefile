# Copyright (c) 2020 Red Hat, Inc.

include build/Configfile
BINDIR ?= output

USE_VENDORIZED_BUILD_HARNESS ?=

ifndef USE_VENDORIZED_BUILD_HARNESS
-include $(shell curl -s -H 'Authorization: token ${GITHUB_TOKEN}' -H 'Accept: application/vnd.github.v4.raw' -L https://api.github.com/repos/open-cluster-management/build-harness-extensions/contents/templates/Makefile.build-harness-bootstrap -o .build-harness-bootstrap; echo .build-harness-bootstrap)
else
-include vbh/.build-harness-vendorized
endif

default::
	@echo "Build Harness Bootstrapped"

.PHONY: deps
deps:
    # TEMPORARILY disable linter because we are unable to install it with the current version of go 1.11.13
	# We have been unsuccessful tracing what suddenly cause this problem. Skipping linter to unblock us from
	# delivering code fixes while we find the solution.
	#
	# go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	go get -u github.com/golang/dep/cmd/dep
	dep ensure -v
	# go install ./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: search-collector
search-collector:
	CGO_ENABLED=0 go build -a -v -i -installsuffix cgo -ldflags '-s -w' -o $(BINDIR)/search-collector ./

.PHONY: build
build: search-collector

.PHONY: build-linux
build-linux:
	make search-collector GOOS=linux

.PHONY: lint
lint:
	@echo "!!! LINTER temporarily disabled because of issues getting golangci-lint !!!"
	# golangci-lint run --timeout=2m

.PHONY: test
test:
	DEPLOYED_IN_HUB=true go test ./... -v -coverprofile cover.out

.PHONY: coverage
coverage:
	go tool cover -html=cover.out -o=cover.html

.PHONY: copyright-check
copyright-check:
	./build/rh-copyright-check.sh
	./build/copyright-check.sh

.PHONY: clean
clean::
	go clean
	rm -f cover*
	rm -rf ./$(BINDIR)
	rm -rf ./.vendor-new
	rm -rf ./vendor
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
	GO111MODULE=on go mod tidy

.PHONY: search-collector
search-collector:
	GO111MODULE=on CGO_ENABLED=0 GOGC=50 go build -a -v -i -installsuffix cgo -ldflags '-s -w' -o $(BINDIR)/search-collector ./

.PHONY: build
build: search-collector

.PHONY: build-linux
build-linux:
	make search-collector GOOS=linux

.PHONY: lint
lint:
	GO111MODULE=on go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.27.0
	# Flag GOGC=75 needed to avoid out of memory issue.
	GO111MODULE=on GOGC=75 golangci-lint run --timeout=2m

run:
	GO111MODULE=on GOGC=50 go run main.go

.PHONY: test
test:
	GO111MODULE=on DEPLOYED_IN_HUB=true go test ./... -v -coverprofile cover.out

.PHONY: coverage
coverage:
	GO111MODULE=on go tool cover -html=cover.out -o=cover.html

.PHONY: copyright-check
copyright-check:
	./build/copyright-check.sh

.PHONY: clean
clean::
	go clean
	rm -f cover*
	rm -rf ./$(BINDIR)
	rm -rf ./.vendor-new
	rm -rf ./vendor
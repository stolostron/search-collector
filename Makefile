# Copyright (c) 2020 Red Hat, Inc.
# Copyright Contributors to the Open Cluster Management project

include build/Configfile
BINDIR ?= output


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
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${GOPATH}/bin" v1.52.2
	CGO_ENABLED=0 GOGC=25 golangci-lint run --timeout=3m
	gosec ./...
	
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
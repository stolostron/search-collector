include build/Configfile
BINDIR ?= output

-include $(shell curl -H 'Authorization: token ${GITHUB_TOKEN}' -H 'Accept: application/vnd.github.v4.raw' -L https://api.github.com/repos/open-cluster-management/build-harness-extensions/contents/templates/Makefile.build-harness-bootstrap -o .build-harness-bootstrap; echo .build-harness-bootstrap)

default::
	@echo "Build Harness Bootstrapped"

.PHONY: deps
deps:
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	go get -u github.com/golang/dep/cmd/dep
	dep ensure -v

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
	golangci-lint run

.PHONY: test
test:
	DEPLOYED_IN_HUB=true go test ./... -v -coverprofile cover.out

.PHONY: coverage
coverage:
	go tool cover -html=cover.out -o=cover.html

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
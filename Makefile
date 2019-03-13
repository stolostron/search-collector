BINDIR        ?= output

.PHONY: deps default build lint test coverage clean

default: search-collector

deps:
	go get -t -v ./...
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

search-collector:
	go build -v -i -ldflags '-s -w' -o $(BINDIR)/search-collector ./

build: search-collector

lint:
	golangci-lint run

test:
	go test ./... -v -coverprofile cover.out

coverage:
	go tool cover -html=cover.out -o=cover.html

clean:
	go clean
	rm -f cover*
	rm -rf ./$(BINDIR)

include Makefile.docker

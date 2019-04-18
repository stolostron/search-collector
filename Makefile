BINDIR        ?= output

.PHONY: deps default build lint test coverage clean search-collector

default: search-collector

deps:
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	go get -u github.com/golang/dep/cmd/dep
	dep ensure -v

search-collector:
	CGO_ENABLED=0 go build -a -v -i -installsuffix cgo -ldflags '-s -w' -o $(BINDIR)/search-collector ./

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


# To build image on Mac and Linux
local-docker-search-collector:
	CGO_ENABLED=0 GOOS=linux go build -a -v -i -installsuffix cgo -ldflags '-s -w' -o $(BINDIR)/search-collector ./

.PHONY: local
local: check-env app-version local-docker-search-collector
	docker build -t $(IMAGE_REPO)/$(IMAGE_NAME_ARCH):$(IMAGE_VERSION) \
		--build-arg "VCS_REF=$(VCS_REF)" \
		--build-arg "VCS_URL=$(GIT_REMOTE_URL)" \
		--build-arg "IMAGE_NAME=$(IMAGE_NAME)" \
		--build-arg "IMAGE_DISPLAY_NAME=$(IMAGE_DISPLAY_NAME)" \
		--build-arg "IMAGE_MAINTAINER=$(IMAGE_MAINTAINER)" \
		--build-arg "IMAGE_VENDOR=$(IMAGE_VENDOR)" \
		--build-arg "IMAGE_VERSION=$(IMAGE_VERSION)" \
		--build-arg "IMAGE_RELEASE=$(IMAGE_RELEASE)" \
		--build-arg "IMAGE_SUMMARY=$(IMAGE_SUMMARY)" \
		--build-arg "IMAGE_OPENSHIFT_TAGS=$(IMAGE_OPENSHIFT_TAGS)" \
		--build-arg "IMAGE_NAME_ARCH=$(IMAGE_NAME_ARCH)" \
		--build-arg "IMAGE_DESCRIPTION=$(IMAGE_DESCRIPTION)" $(DOCKER_FLAG) .
	docker tag $(IMAGE_REPO)/$(IMAGE_NAME_ARCH):$(IMAGE_VERSION) $(IMAGE_REPO)/$(IMAGE_NAME_ARCH):$(RELEASE_TAG)

.PHONY: copyright-check
copyright-check:
	./build-tools/copyright-check.sh

include Makefile.docker

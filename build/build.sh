#!/bin/bash

echo " > Running build.sh"
set -e

export DOCKER_IMAGE_AND_TAG=${1}

echo "!!! TODO: lint is temporarily disabled. Must re-enable on ./build/build.sh !!!"
# make lint
make build
make docker/build
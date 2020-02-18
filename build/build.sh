#!/bin/bash

echo " > Running build.sh"
set -e

export DOCKER_IMAGE_AND_TAG=${1}

echo "!!! TODO: Need to enable lint!!!"
# make lint
make build
make docker/build
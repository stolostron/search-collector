#!/bin/bash

# Copyright Contributors to the Open Cluster Management project

echo " > Running build.sh"
set -e

export DOCKER_IMAGE_AND_TAG=${1}

make build
make docker/build
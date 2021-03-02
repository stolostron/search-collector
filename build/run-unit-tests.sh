#!/bin/bash

# Copyright Contributors to the Open Cluster Management project

echo " > Running run-unit-tests.sh"
set -e

make deps
make lint
make test
make coverage

exit 0
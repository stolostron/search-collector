#!/bin/bash
# Copyright (c) 2021 Red Hat, Inc.
# Copyright Contributors to the Open Cluster Management project

echo " > Running run-unit-tests.sh"
set -e

make deps
make lint
make test
make coverage

exit 0
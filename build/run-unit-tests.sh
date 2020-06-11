#!/bin/bash

echo " > Running run-unit-tests.sh"
set -e

make copyright-check
make deps
make lint
make test
make coverage

exit 0
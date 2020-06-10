#!/bin/bash

echo " > Running run-unit-tests.sh"
set -e

git config --global url."git@github.com:".insteadOf "https://github.com/"
make deps
make test
make coverage

exit 0
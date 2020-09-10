#!/bin/bash

echo " > Running run-e2e-test.sh"
set -e
sh tests/e2e/runTests.sh $1

exit 0
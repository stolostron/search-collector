#!/bin/bash

echo " > Running run-e2e-test.sh"
echo "   args: $1"

sh tests/e2e/runTests.sh $1

exit 0
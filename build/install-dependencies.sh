#!/bin/bash

echo " > Running install-dependencies.sh"
set -e

git config --global url."git@github.com:".insteadOf "https://github.com/"
make deps

exit 0
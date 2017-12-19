#!/bin/bash

set -e

function cleanup {
  rm -f usagereport.exe
  rm -f usagereport-plugin
}
trap cleanup EXIT

SCRIPT_DIR=`dirname $0`
cd ${SCRIPT_DIR}/..

echo "Go formatting..."
go fmt ./...

echo "Go vetting..."
go vet ./...

echo "Recursive ginkgo... ${*:+(with parameter(s) }$*${*:+)}"
ginkgo -r --race --randomizeAllSpecs --failOnPending -cover $*

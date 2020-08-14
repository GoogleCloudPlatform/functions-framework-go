#!/bin/bash

set -x

TESTDATA_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
VENDOR_DIR="${TESTDATA_DIR?}/vendor/github.com/GoogleCloudPlatform/functions-framework-go"
FF_DIR="$(dirname "${TESTDATA_DIR}")"

function goget {
  go get github.com/GoogleCloudPlatform/functions-framework-go@${GITHUB_SHA?}
  [ $? -eq 0 ] || return 1
}

function vendor {
  # Remove the dependency
  go get github.com/GoogleCloudPlatform/functions-framework-go@none
  go mod vendor
  rm -rf ${VENDOR_DIR?}
  mkdir -p ${VENDOR_DIR?}
  cp "${FF_DIR?}/go.mod" ${VENDOR_DIR?}
  cp -r "${FF_DIR?}/funcframework" ${VENDOR_DIR?}
  rm ${TESTDATA_DIR?}/go.mod
}

pushd ${TESTDATA_DIR?}
goget || vendor
popd

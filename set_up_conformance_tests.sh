#!/bin/bash

set -e

pushd testdata
go mod edit -require github.com/GoogleCloudPlatform/functions-framework-go@${GITHUB_SHA?}
popd

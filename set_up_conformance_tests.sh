#!/bin/bash

set -e

pushd testdata
go get github.com/GoogleCloudPlatform/functions-framework-go@${GITHUB_SHA?}
popd

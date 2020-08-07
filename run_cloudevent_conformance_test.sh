#!/bin/bash

set -e

go install github.com/GoogleCloudPlatform/functions-framework-conformance/client

go run github.com/GoogleCloudPlatform/functions-framework-conformance/client \
  -cmd "go run conformance-tests/cloudevent/main.go" \
  -type "cloudevent" \
  -validate-mapping=false

# Clean up.
rm serverlog_stderr.txt
rm serverlog_stdout.txt
rm function_output.json

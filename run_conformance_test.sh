#!/bin/bash

set -e

go install github.com/GoogleCloudPlatform/functions-framework-conformance/client

# # Validate HTTP
# go run github.com/GoogleCloudPlatform/functions-framework-conformance/client \
#   -cmd "go run conformance-tests/http/main.go" \
#   -type "http" \

# Validate legacy events
go run github.com/GoogleCloudPlatform/functions-framework-conformance/client \
  -cmd "go run conformance-tests/event/main.go" \
  -type "legacyevent" \
  # TODO: enable mapping once we support CloudEvents.
  # -validateMapping false

# Clean up.
rm serverlog_stderr.txt
rm serverlog_stdout.txt
rm function_output.json

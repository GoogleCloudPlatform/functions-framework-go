# Conformance Test Functions

## Layout
`function/` contains the functions for conformance tests. This function is GCF-deployable.

`function/prerun.sh` is a helper script used for the buildpack integration tests GitHub Workflow.

`cmd/` contains the `main.go` required to run conformance test functions as servers.

## Testing Locally

`./run-conformance-tests.sh`

Builds and runs the functions using Go tooling and running the Go binaries.

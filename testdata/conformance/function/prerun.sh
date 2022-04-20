# prerun.sh sets up the test function to use the functions framework commit
# specified by generating a `go.mod`. This makes the function `pack` buildable
# with GCF buildpacks.
#
# This should only be used for testing with buildpacks since the `go.mod` will
# cause problems with import paths when trying to run conformance test functions
# locally using the `main.go` files in the `cmd` directory.`
#
# `pack` build example command:
# pack build myfn --builder us.gcr.io/fn-img/buildpacks/go116/builder:go116_20220320_1_16_13_RC00 --env GOOGLE_RUNTIME=go116 --env GOOGLE_FUNCTION_TARGET=declarativeHTTP
FRAMEWORK_VERSION=$1

# exit when any command fails
set -e

cd $(dirname $0)

if [ -z "${FRAMEWORK_VERSION}" ]
    then
        echo "Functions Framework version required as first parameter"
        exit 1
fi

echo "module example.com/function

go 1.13

require (
        cloud.google.com/go/functions v1.0.0
        github.com/GoogleCloudPlatform/functions-framework-go $FRAMEWORK_VERSION
        github.com/cloudevents/sdk-go/v2 v2.6.1
)" >> go.mod

cat go.mod

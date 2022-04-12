# prerun.sh sets up the test function to use the functions framework commit
# specified by generating a `go.mod`. This makes the function `pack` buildable
# with GCF buildpacks.
#
# This should only be used for testing with buildpacks since the `go.mod` will
# cause problems with import paths when trying to run conformance test functions
# locally using the `main.go` files in the `cmd` directory.`
FRAMEWORK_VERSION=$1

# exit when any command fails
set -e

cd $(dirname $0)

if [ -z "${FRAMEWORK_VERSION}" ]
    then
        echo "Functions Framework version required as first parameter"
        exit 1
fi

go mod init example.com/function
go mod tidy
go mod edit -require=github.com/GoogleCloudPlatform/functions-framework-go@$FRAMEWORK_VERSION
cat go.mod
# Functions Framework for Go  [![Build Status](https://travis-ci.com/GoogleCloudPlatform/functions-framework-go.svg?branch=master)](https://travis-ci.com/GoogleCloudPlatform/functions-framework-go) [![GoDoc](https://godoc.org/github.com/GoogleCloudPlatform/functions-framework-go?status.svg)](http://godoc.org/github.com/GoogleCloudPlatform/functions-framework-go) [![Go version](https://img.shields.io/badge/go-v1.11+-blue)](https://golang.org/dl/#stable)

An open source FaaS (Function as a Service) framework for writing portable
Go functions, brought to you by the Google Cloud Functions team.

The Functions Framework lets you write lightweight functions that run in many
different environments, including:

*   [Google Cloud Functions](https://cloud.google.com/functions/)
*   Your local development machine
*   [Knative](https://github.com/knative/)-based environments
*   [Google App Engine](https://cloud.google.com/appengine/docs/go/)
*   [Google Cloud Run](https://cloud.google.com/run/docs/quickstarts/build-and-deploy)

The framework allows you to go from:

```golang
func HelloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, World!")
}
```

To:

```sh
curl http://my-url
# Output: Hello, World!
```

All without needing to worry about writing an HTTP server or request
handling logic.

## Features

*   Spin up a local development server for quick testing with little extra code
*   Invoke a function in response to a request
*   Automatically unmarshal events conforming to the
    [CloudEvents](https://cloudevents.io/) spec
*   Portable between serverless platforms

## Quickstart: Hello, World on your local machine

1. Install Go 1.11+, [Docker](https://store.docker.com/search?type=edition&offering=community), and the [`pack` tool](https://buildpacks.io/docs/install-pack/).

1. Create a Go module:
	```sh
	go mod init example.com/hello
	```

	> Note: You can use a different module name rather than `example.com/hello`.

1. Create a `function.go` file with the following contents:
	```golang
	package hello

	import (
		"net/http"
		"fmt"
	)

	// HelloWorld writes "Hello, World!" to the HTTP response.
	func HelloWorld(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, World!\n")
	}
	```

	> Note that you can use any file name or package name (convention is to make
	package name same as directory name).

1. Build a container from your function using the Functions [buildpacks](https://github.com/GoogleCloudPlatform/buildpacks):
  ```sh
  pack build \
    --builder gcr.io/buildpacks/builder:v1 \
    --env GOOGLE_FUNCTION_SIGNATURE_TYPE=http \
    --env GOOGLE_FUNCTION_TARGET=HelloWorld \
    my-first-function
  ```

1. Start the built container:
	```sh
	docker run --rm -p 8080:8080 my-first-function
	# Output: Serving function...
	```

2. Send requests to this function using `curl` from another terminal window:
	```sh
	curl localhost:8080
	# Output: Hello, World!
	```

## Run your function on serverless platforms

### Google Cloud Functions

Deploy from your local machine using the `gcloud` command-line tool.
[Check out the Cloud Functions quickstart](https://cloud.google.com/functions/docs/quickstart).

### Container environments based on Knative

The Functions Framework is designed to be compatible with Knative environments.
Just build and deploy your container to a Knative environment. Note that your app needs to listen
`PORT` environment variable per [Knative runtime contract](https://github.com/knative/serving/blob/master/docs/runtime-contract.md#inbound-network-connectivity).

## Functions Framework Features

The Go Functions Framework conforms to the [Functions Framework Contract](https://github.com/GoogleCloudPlatform/functions-framework), As such, it
supports HTTP functions, background event functions, and CloudEvent functions
(as of v1.1.0). The primary build mechanism is the [GCP buildpacks stack](https://github.com/GoogleCloudPlatform/buildpacks), which takes a function of
one of the accepted types, converts it to a full HTTP serving app, and creates a
launchable container to run the server.

### HTTP Functions

The Framework provides support for handling native Go HTTP-style functions:

```golang
func HTTPFunction(w http.ResponseWriter, r *http.Request) error {
	// Do something with r, and write response to w.
}
```

The functions are registered with the handler via `funcframework.RegisterHTTPFunctionContext` and should behave according to idiomatic Go HTTP expectations.

### Background Event Functions

[Background events](https://cloud.google.com/functions/docs/writing/background)
are also supported. This type of function takes two parameters: a Go context and
a user-defined data struct.

```golang
func BackgroundEventFunction(ctx context.Context, data userDefinedEventStruct) error {
	// Do something with ctx and data.
}
```

This type of event requires you to define a struct with the
appropriate data fields (e.g. those for a PubSub message or GCS event) and pass
that struct as the data parameter. See the [samples](https://cloud.google.com/functions/docs/writing/background) for details.

The context parameter is a Go `context.Context`, and contains additional event
metadata under a functions-specific key. This data is accesible via the `cloud.google.com/go/functions/metadata` package:

```golang
m := metadata.FromContext(ctx)
```

### CloudEvent Functions

The Functions Framework provides support for unmarshalling an incoming
[CloudEvent](https://cloudevents.io/) payload into a `cloudevents.Event` object.
These will be passed as arguments to your function when it receives a request.

```golang
func CloudEventFunction(ctx context.Context, e cloudevents.Event) error {
	// Do something with event.Context and event.Data (via event.DataAs(foo)).
}
```

To learn more about CloudEvents, see the [Go SDK for CloudEvents](https://github.com/cloudevents/sdk-go).

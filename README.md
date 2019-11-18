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

# Features

*   Spin up a local development server for quick testing with little extra code
*   Invoke a function in response to a request
*   Automatically unmarshal events conforming to the
    [CloudEvents](https://cloudevents.io/) spec
*   Portable between serverless platforms

# Quickstart: Hello, World on your local machine

1. Make sure you have Go 1.11+ installed with:
	```
	go version
	```
	The output should be Go 1.11 or higher.

1. Create the necessary directories.
	```sh
	mkdir -p hello/cmd
	cd hello
	```

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

1. Now go to the `cmd` subdirectory.
	```sh
	cd cmd
	```

1. Create a `main.go` file with the following contents:
	```golang
	package main

	import (
		"log"
		"os"

		"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
		"example.com/hello"
	)

	func main() {
		funcframework.RegisterHTTPFunction("/", hello.HelloWorld)
		// Use PORT environment variable, or default to 8080.
		port := "8080"
		if envPort := os.Getenv("PORT"); envPort != "" {
			port = envPort
		}

		if err := funcframework.Start(port); err != nil {
			log.Fatalf("funcframework.Start: %v\n", err)
		}
	}
	```

1. Start the local development server:
	```sh
	go build
	./cmd
	Serving function...
	```

2. Send requests to this function using `curl` from another terminal window:
	```sh
	curl localhost:8080
	# Output: Hello, World!
	```

# Run your function on serverless platforms

## Google Cloud Functions

You cannot deploy main packages to Google Cloud Functions. You need to go back to the parent directory
in which your function code is.

```sh
cd ..
```

and you can deploy it from your local machine using the `gcloud` command-line tool.
[Check out the Cloud Functions quickstart](https://cloud.google.com/functions/docs/quickstart).

## Container environments based on Knative

The Functions Framework is designed to be compatible with Knative environments.
Just build and deploy your container to a Knative environment. Note that your app needs to listen
`PORT` environment variable per [Knative runtime contract](https://github.com/knative/serving/blob/master/docs/runtime-contract.md#inbound-network-connectivity).

# Configure the Functions Framework

If you're deploying to Google Cloud Functions, you don't need to worry about writing a
`package main`. But if you want to run your function locally (e.g., for local development),
you may want to configure the port, the function to be executed, and the function signature type
(which specifies event unmarshalling logic). You can do this by modifying the `main.go`
file described above:

To select a port, set the `$PORT` environment variable when running.

```sh
PORT=8000 ./cmd
```

To select a function, pass your function to `funcframework.RegisterHTTPFunction` in the second variable.

```golang
funcframework.RegisterHTTPFunction("/", myFunction);
```

If your function handles events, use `funcframework.RegisterEventFunction` instead of `funcframework.RegisterHTTPFunction`.

```golang
funcframework.RegisterEventFunction("/", eventFunction);

func eventFunction(ctx context.Context, e myEventType){
	// function logic
}
```

> Note that the first parameter to a function that handles events has to be `context.Context`
and the type of second parameter needs to be a type of an unmarshallable event.

# Enable Cloud Events

The Functions Framework can unmarshal to custom structs, and provides support for 
unmarshalling an incoming [CloudEvents](http://cloudevents.io) payload to a
`cloudevents.Event` object. These will be passed as arguments to your function when it receives a request.
Note that your function must use the event-style function signature.

```golang
func CloudEventsFunction(ctx context.Context, e cloudevents.Event) {
	// Do something with event.Context and event.Data (via event.DataAs(foo)).
}
```

To learn more about CloudEvents, see the [Go SDK for CloudEvents](https://github.com/cloudevents/sdk-go).

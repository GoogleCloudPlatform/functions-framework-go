# Functions Framework for Go

An open source FaaS (Function as a Service) framework for writing portable
Go functions -- brought to you by the Google Cloud Functions team.

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

All without needing to worry about writing an HTTP server or complicated request
handling logic.

# Features

*   Spin up a local development server for quick testing with little extra code
*   Invoke a function in response to a request
*   Automatically unmarshal events conforming to the
    [CloudEvents](https://cloudevents.io/) spec
*   Portable between serverless platforms

# Quickstart: Hello, World on your local machine

Create the necessary directories.
```sh
mkdir -p hello/cmd
cd hello
```

Create a Go module:

```sh
go mod init example.com/hello
```

> Note: You can use a different module name rather than `example.com/hello`.

Create a `function.go` file with the following contents:

```golang
package hello

import (
	"net/http"
	"fmt"
)

// HelloWorld writes "Hello, World!" to the HTTP response.
func HelloWorld(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello, World!")
}
```

> Note that you can use any file name or package name(convention is to make
package name same as directory name).

Now go to the `cmd` subdirectory.
```sh
cd cmd
```

Create a `main.go` file with the following contents:

```golang
package main

import (
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/functions/framework"
	"example.com/hello"
)

func main() {
	if err := framework.RegisterHTTPFunction("/", hello.HelloWorld); err != nil {
		log.Fatalf("framework.RegisterHTTPFunction: %v\n", err)
	}
	port := ":8080"
	if err := framework.Start(port); err != nil {
		log.Fatalf("framework.Start: %v\n", err)
	}
}
```

Start the built-in local development server:

```sh
go build
./cmd
Function serving...
URL: http://localhost:8080/
```

Send requests to this function using `curl` from another terminal window:

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
you may want to configure the port, function to be executed and function signature type
(which specifies event unmarshalling logic). You can do this by modifying the `main.go`
file described above.

To select a port change the port variable that is passed to `framework.Start` like this:

```golang
framework.Start(myPort)
```

To select a function pass your function to `framework.RegisterHTTPFunction` in the second variable:

```golang
framework.RegisterHTTPFunction("/", myFunction);
```

If your function handles events, use `framework.RegisterEventFunction` instead of `framework.RegisterHTTPFunction`:

```golang
framework.RegisterEventFunction("/", eventFunction);

func eventFunction(ctx context.Context, e myEventType){
	// function logic
}
```

> Note that the first parameter to a function that handles events has to be `context.Context`
and the type of second parameter needs to be a type of an unmarshallable event.

# Enable CloudEvents handling for use with the event function signature

The Functions Framework can unmarshal to custom structs, for example, unmarshalling an incoming
[CloudEvents](http://cloudevents.io) payloads to a `cloudevents.Event` object.
These will be passed as arguments to your function when it receives a request.
Note that your function must use the event-style function signature:

```golang
func CloudEventsFunction(ctx context.Context, e cloudevents.Event) {
    // do something with event.Context and event.Data (via event.DataAs(foo))
}
```

To learn more about CloudEvents see [Go SDK for CloudEvents](https://github.com/cloudevents/sdk-go)

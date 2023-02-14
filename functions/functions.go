// Package functions provides a way to declaratively register functions
// that can be used to handle incoming requests.
package functions

import (
	"context"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/internal/registry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// HTTP registers an HTTP function that becomes the function handler served
// at "/" when environment variable `FUNCTION_TARGET=name`
func HTTP(name string, fn func(http.ResponseWriter, *http.Request)) {
	if err := registry.Default().RegisterHTTP(fn, registry.WithName(name)); err != nil {
		log.Fatalf("failure to register function: %s", err)
	}
}

// CloudEvent registers a CloudEvent function that becomes the function handler
// served at "/" when environment variable `FUNCTION_TARGET=name`
func CloudEvent(name string, fn func(context.Context, cloudevents.Event) error) {
	if err := registry.Default().RegisterCloudEvent(fn, registry.WithName(name)); err != nil {
		log.Fatalf("failure to register function: %s", err)
	}
}

// Typed registers a Typed function that becomes the function handler
// served at "/" when environment variable `FUNCTION_TARGET=name`
// This function takes a strong type T as an input and can return a strong type T,
// built in types, nil and/or error as an output
func Typed(name string, fn interface{}) {
	if err := registry.Default().RegisterTyped(fn, registry.WithName(name)); err != nil {
		log.Fatalf("failure to register function: %s", err)
	}
}

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
	if err := registry.Default().RegisterHTTP(name, fn); err != nil {
		log.Fatalf("failure to register function: %s", err)
	}
}

// CloudEvent registers a CloudEvent function that becomes the function handler
// served at "/" when environment variable `FUNCTION_TARGET=name`
func CloudEvent(name string, fn func(context.Context, cloudevents.Event) error) {
	if err := registry.Default().RegisterCloudEvent(name, fn); err != nil {
		log.Fatalf("failure to register function: %s", err)
	}
}

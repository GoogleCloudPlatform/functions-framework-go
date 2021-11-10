package functions

import (
	"context"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/internal/registry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Declaratively registers a HTTP function.
func HTTP(name string, fn func(http.ResponseWriter, *http.Request)) {
	if err := registry.Default().RegisterHTTP(name, fn); err != nil {
		log.Fatalf("failure to register function: %s", err)
	}
}

// Declaratively registers a CloudEvent function.
func CloudEvent(name string, fn func(context.Context, cloudevents.Event) error) {
	if err := registry.Default().RegisterCloudEvent(name, fn); err != nil {
		log.Fatalf("failure to register function: %s", err)
	}
}

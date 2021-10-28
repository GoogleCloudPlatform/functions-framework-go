package funcframework

import (
	"context"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// A declaratively registered function
type RegisteredFunction struct {
	Name         string
	CloudEventFn func(context.Context, cloudevents.Event) error
	HTTPFn       func(http.ResponseWriter, *http.Request)
}

var (
	function_registry = map[string]RegisteredFunction{}
)

// Registers a HTTP function with a given name
func HTTPFunction(name string, fn func(http.ResponseWriter, *http.Request)) error {
	function_registry[name] = RegisteredFunction{
		Name:         name,
		CloudEventFn: nil,
		HTTPFn:       fn,
	}
	RegisterHTTPFunctionContext(context.Background(), name, fn)
	return nil
}

// Registers a CloudEvent function with a given name
func CloudEventFunction(name string, fn func(context.Context, cloudevents.Event) error) error {
	function_registry[name] = RegisteredFunction{
		Name:         name,
		CloudEventFn: fn,
		HTTPFn:       nil,
	}
	RegisterCloudEventFunctionContext(context.Background(), name, fn)
	return nil
}

// Gets a registered function by name
func GetRegisteredFunction(name string) (RegisteredFunction, bool) {
	fn, ok := function_registry[name]
	return fn, ok
}

package registry

import (
	"context"
	"fmt"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// A declaratively registered function
type RegisteredFunction struct {
	Name         string                                         // The name of the function
	CloudEventFn func(context.Context, cloudevents.Event) error // Optional: The user's CloudEvent function
	HTTPFn       func(http.ResponseWriter, *http.Request)       // Optional: The user's HTTP function
}

var (
	function_registry = map[string]RegisteredFunction{}
)

// Registers a HTTP function with a given name
func HTTPFunction(name string, fn func(http.ResponseWriter, *http.Request)) error {
	if _, ok := function_registry[name]; ok {
		return fmt.Errorf("function name already registered: %s", name)
	}
	function_registry[name] = RegisteredFunction{
		Name:         name,
		CloudEventFn: nil,
		HTTPFn:       fn,
	}
	return nil
}

// Registers a CloudEvent function with a given name
func CloudEventFunction(name string, fn func(context.Context, cloudevents.Event) error) error {
	if _, ok := function_registry[name]; ok {
		return fmt.Errorf("function name already registered: %s", name)
	}

	function_registry[name] = RegisteredFunction{
		Name:         name,
		CloudEventFn: fn,
		HTTPFn:       nil,
	}
	return nil
}

// Gets a registered function by name
func GetRegisteredFunction(name string) (RegisteredFunction, bool) {
	fn, ok := function_registry[name]
	return fn, ok
}

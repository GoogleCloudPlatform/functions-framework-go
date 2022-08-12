package registry

import (
	"context"
	"fmt"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// RegisteredFunction represents a function that has been
// registered with the registry.
type RegisteredFunction struct {
	Name         string                                         // The name of the function
	CloudEventFn func(context.Context, cloudevents.Event) error // Optional: The user's CloudEvent function
	HTTPFn       func(http.ResponseWriter, *http.Request)       // Optional: The user's HTTP function
}

// Registry is a registry of functions.
type Registry struct {
	functions map[string]RegisteredFunction
}

var defaultInstance = New()

// Default returns the default, singleton registry instance.
func Default() *Registry {
	return defaultInstance
}

func New() *Registry {
	return &Registry{
		functions: map[string]RegisteredFunction{},
	}
}

// RegisterHTTP a HTTP function with a given name
func (r *Registry) RegisterHTTP(name string, fn func(http.ResponseWriter, *http.Request)) error {
	if _, ok := r.functions[name]; ok {
		return fmt.Errorf("function name already registered: %s", name)
	}
	r.functions[name] = RegisteredFunction{
		Name:         name,
		CloudEventFn: nil,
		HTTPFn:       fn,
	}
	return nil
}

// RegistryCloudEvent a CloudEvent function with a given name
func (r *Registry) RegisterCloudEvent(name string, fn func(context.Context, cloudevents.Event) error) error {
	if _, ok := r.functions[name]; ok {
		return fmt.Errorf("function name already registered: %s", name)
	}
	r.functions[name] = RegisteredFunction{
		Name:         name,
		CloudEventFn: fn,
		HTTPFn:       nil,
	}
	return nil
}

// GetRegisteredFunction a registered function by name
func (r *Registry) GetRegisteredFunction(name string) (RegisteredFunction, bool) {
	fn, ok := r.functions[name]
	return fn, ok
}

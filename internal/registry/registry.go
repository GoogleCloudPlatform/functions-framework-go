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
	Path         string                                         // The serving path of the function
	CloudEventFn func(context.Context, cloudevents.Event) error // Optional: The user's CloudEvent function
	HTTPFn       func(http.ResponseWriter, *http.Request)       // Optional: The user's HTTP function
	EventFn      interface{}                                    // Optional: The user's Event function
}

// Option is an option used when registering a function.
type Option func(*RegisteredFunction)

func WithPath(path string) Option {
	return func(fn *RegisteredFunction) {
		fn.Path = path
	}
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
func (r *Registry) RegisterHTTP(name string, fn func(http.ResponseWriter, *http.Request), options ...Option) error {
	if _, ok := r.functions[name]; ok {
		return fmt.Errorf("function name already registered: %q", name)
	}
	function := RegisteredFunction{
		Name:         name,
		Path:         "/" + name,
		CloudEventFn: nil,
		HTTPFn:       fn,
		EventFn:      nil,
	}
	for _, o := range options {
		o(&function)
	}
	r.functions[name] = function
	return nil
}

// RegistryCloudEvent a CloudEvent function with a given name
func (r *Registry) RegisterCloudEvent(name string, fn func(context.Context, cloudevents.Event) error, options ...Option) error {
	if _, ok := r.functions[name]; ok {
		return fmt.Errorf("function name already registered: %q", name)
	}
	function := RegisteredFunction{
		Name:         name,
		Path:         "/" + name,
		CloudEventFn: fn,
		HTTPFn:       nil,
		EventFn:      nil,
	}
	for _, o := range options {
		o(&function)
	}
	r.functions[name] = function
	return nil
}

// RegistryCloudEvent a Event function with a given name
func (r *Registry) RegisterEvent(name string, fn interface{}, options ...Option) error {
	if _, ok := r.functions[name]; ok {
		return fmt.Errorf("function name already registered: %q", name)
	}
	function := RegisteredFunction{
		Name:         name,
		Path:         "/" + name,
		CloudEventFn: nil,
		HTTPFn:       nil,
		EventFn:      fn,
	}
	for _, o := range options {
		o(&function)
	}
	r.functions[name] = function
	return nil
}

// GetRegisteredFunction a registered function by name
func (r *Registry) GetRegisteredFunction(name string) (RegisteredFunction, bool) {
	fn, ok := r.functions[name]
	return fn, ok
}

// GetAllFunctions returns all the registered functions.
func (r *Registry) GetAllFunctions() map[string]RegisteredFunction {
	return r.functions
}

// DeleteRegisteredFunction deletes a registered function.
func (r *Registry) DeleteRegisteredFunction(name string) {
	delete(r.functions, name)
}

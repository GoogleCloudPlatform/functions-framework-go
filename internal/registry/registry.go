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

func WithName(name string) Option {
	return func(fn *RegisteredFunction) {
		fn.Name = name
	}
}

// Registry is a registry of functions.
type Registry struct {
	functions             map[string]*RegisteredFunction
	functionsWithoutNames []*RegisteredFunction // The functions that are not registered declaratively.
}

var defaultInstance = New()

// Default returns the default, singleton registry instance.
func Default() *Registry {
	return defaultInstance
}

func New() *Registry {
	return &Registry{
		functions: map[string]*RegisteredFunction{},
	}
}

func Reset() {
	defaultInstance = New()
}

// RegisterHTTP registes a HTTP function.
func (r *Registry) RegisterHTTP(fn func(http.ResponseWriter, *http.Request), options ...Option) error {
	function := RegisteredFunction{
		CloudEventFn: nil,
		HTTPFn:       fn,
		EventFn:      nil,
	}
	for _, o := range options {
		o(&function)
	}
	if function.Name == "" {
		// The function is not registered declaratively.
		r.functionsWithoutNames = append(r.functionsWithoutNames, &function)
		return nil
	}
	if _, ok := r.functions[function.Name]; ok {
		return fmt.Errorf("function name already registered: %q", function.Name)
	}
	function.Path = "/" + function.Name
	r.functions[function.Name] = &function
	return nil
}

// RegistryCloudEvent registers a CloudEvent function.
func (r *Registry) RegisterCloudEvent(fn func(context.Context, cloudevents.Event) error, options ...Option) error {
	function := RegisteredFunction{
		CloudEventFn: fn,
		HTTPFn:       nil,
		EventFn:      nil,
	}
	for _, o := range options {
		o(&function)
	}
	if function.Name == "" {
		// The function is not registered declaratively.
		r.functionsWithoutNames = append(r.functionsWithoutNames, &function)
		return nil
	}
	if _, ok := r.functions[function.Name]; ok {
		return fmt.Errorf("function name already registered: %q", function.Name)
	}
	function.Path = "/" + function.Name
	r.functions[function.Name] = &function
	return nil
}

// RegistryCloudEvent registers a Event function.
func (r *Registry) RegisterEvent(fn interface{}, options ...Option) error {
	function := RegisteredFunction{
		CloudEventFn: nil,
		HTTPFn:       nil,
		EventFn:      fn,
	}
	for _, o := range options {
		o(&function)
	}
	if function.Name == "" {
		// The function is not registered declaratively.
		r.functionsWithoutNames = append(r.functionsWithoutNames, &function)
		return nil
	}
	if _, ok := r.functions[function.Name]; ok {
		return fmt.Errorf("function name already registered: %q", function.Name)
	}
	function.Path = "/" + function.Name
	r.functions[function.Name] = &function
	return nil
}

// GetRegisteredFunction a registered function by name
func (r *Registry) GetRegisteredFunction(name string) (*RegisteredFunction, bool) {
	fn, ok := r.functions[name]
	return fn, ok
}

// GetAllFunctions returns all the registered functions.
func (r *Registry) GetAllFunctions() []*RegisteredFunction {
	all := r.functionsWithoutNames
	for _, fn := range r.functions {
		all = append(all, fn)
	}
	return all
}

// GetLastFunctionWithoutName returns the last function that's not registered declaratively.
func (r *Registry) GetLastFunctionWithoutName() *RegisteredFunction {
	count := len(r.functionsWithoutNames)
	if count == 0 {
		return nil
	}
	return r.functionsWithoutNames[count-1]
}

// DeleteRegisteredFunction deletes a registered function.
func (r *Registry) DeleteRegisteredFunction(name string) {
	delete(r.functions, name)
}

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
	TypedFn      interface{}                                    // Optional: The user's typed function
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

func (r *Registry) Reset() {
	r.functions = map[string]*RegisteredFunction{}
	r.functionsWithoutNames = []*RegisteredFunction{}
}

// RegisterHTTP registes a HTTP function.
func (r *Registry) RegisterHTTP(fn func(http.ResponseWriter, *http.Request), options ...Option) error {
	return r.register(&RegisteredFunction{HTTPFn: fn}, options...)
}

// RegisterCloudEvent registers a CloudEvent function.
func (r *Registry) RegisterCloudEvent(fn func(context.Context, cloudevents.Event) error, options ...Option) error {
	return r.register(&RegisteredFunction{CloudEventFn: fn}, options...)
}

// RegisterEvent registers an Event function.
func (r *Registry) RegisterEvent(fn interface{}, options ...Option) error {
	return r.register(&RegisteredFunction{EventFn: fn}, options...)
}

// RegisterTyped registers a strongly typed function.
func (r *Registry) RegisterTyped(fn interface{}, options ...Option) error {
	return r.register(&RegisteredFunction{TypedFn: fn}, options...)
}

func (r *Registry) register(function *RegisteredFunction, options ...Option) error {
	for _, o := range options {
		o(function)
	}
	if function.Name == "" && function.Path == "" {
		return fmt.Errorf("either the function path or the function name should be specified")
	}
	if function.Name == "" {
		// The function is not registered declaratively.
		r.functionsWithoutNames = append(r.functionsWithoutNames, function)
		return nil
	}
	if _, ok := r.functions[function.Name]; ok {
		return fmt.Errorf("function name already registered: %q", function.Name)
	}
	if function.Path == "" {
		function.Path = "/" + function.Name
	}
	r.functions[function.Name] = function
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
// As the function is registered without a name, it can not be found by setting FUNCTION_TARGET
// when deploying. In this case, the last function that's not registered declaratively
// will be served.
func (r *Registry) GetLastFunctionWithoutName() *RegisteredFunction {
	count := len(r.functionsWithoutNames)
	if count == 0 {
		return nil
	}
	return r.functionsWithoutNames[count-1]
}

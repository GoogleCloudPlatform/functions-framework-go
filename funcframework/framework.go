// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package funcframework is a Functions Framework implementation for Go. It allows you to register
// HTTP and event functions, then start an HTTP server serving those functions.
package funcframework

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/GoogleCloudPlatform/functions-framework-go/internal/registry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

const (
	functionStatusHeader     = "X-Google-Status"
	crashStatus              = "crash"
	errorStatus              = "error"
	panicMessageTmpl         = "A panic occurred during %s. Please see logs for more details."
	fnErrorMessageStderrTmpl = "Function error: %v"
)

var (
	server            = http.DefaultServeMux
	handlerRegistered = false
)

// recoverPanic recovers from a panic in a consistent manner. panicSrc should
// describe what was happening when the panic was encountered, for example
// "user function execution". w is an http.ResponseWriter to write a generic
// response body to that does not expose the details of the panic; w can be
// nil to skip this.
func recoverPanic(w http.ResponseWriter, panicSrc string) {
	if r := recover(); r != nil {
		genericMsg := fmt.Sprintf(panicMessageTmpl, panicSrc)
		fmt.Fprintf(os.Stderr, "%s\npanic message: %v\nstack trace: %s", genericMsg, r, debug.Stack())
		if w != nil {
			writeHTTPErrorResponse(w, http.StatusInternalServerError, crashStatus, genericMsg)
		}
	}
}

// RegisterHTTPFunction registers fn as an HTTP function.
// Maintained for backward compatibility. Please use RegisterHTTPFunctionContext instead.
func RegisterHTTPFunction(path string, fn interface{}) {
	defer recoverPanic(nil, "function registration")

	fnHTTP, ok := fn.(func(http.ResponseWriter, *http.Request))
	if !ok {
		panic("expected function to have signature func(http.ResponseWriter, *http.Request)")
	}

	ctx := context.Background()
	if err := RegisterHTTPFunctionContext(ctx, path, fnHTTP); err != nil {
		panic(fmt.Sprintf("unexpected error in RegisterEventFunctionContext: %v", err))
	}
}

// RegisterEventFunction registers fn as an event function.
// Maintained for backward compatibility. Please use RegisterEventFunctionContext instead.
func RegisterEventFunction(path string, fn interface{}) {
	ctx := context.Background()
	defer recoverPanic(nil, "function registration")
	if err := RegisterEventFunctionContext(ctx, path, fn); err != nil {
		panic(fmt.Sprintf("unexpected error in RegisterEventFunctionContext: %v", err))
	}
}

// RegisterHTTPFunctionContext registers fn as an HTTP function.
func RegisterHTTPFunctionContext(ctx context.Context, path string, fn func(http.ResponseWriter, *http.Request)) error {
	handler, err := wrapHTTPFunction(fn)
	if err == nil {
		server.Handle(path, handler)
		handlerRegistered = true
	}
	return err
}

// RegisterEventFunctionContext registers fn as an event function. The function must have two arguments, a
// context.Context and a struct type depending on the event, and return an error. If fn has the
// wrong signature, RegisterEventFunction returns an error.
func RegisterEventFunctionContext(ctx context.Context, path string, fn interface{}) error {
	handler, err := wrapEventFunction(fn)
	if err == nil {
		server.Handle(path, handler)
		handlerRegistered = true
	}
	return err
}

// RegisterCloudEventFunctionContext registers fn as an cloudevent function.
func RegisterCloudEventFunctionContext(ctx context.Context, path string, fn func(context.Context, cloudevents.Event) error) error {
	handler, err := wrapCloudEventFunction(ctx, fn)
	if err == nil {
		server.Handle(path, handler)
		handlerRegistered = true
	}
	return err
}

// Start serves an HTTP server with registered function(s).
func Start(port string) error {
	if err := initServer(); err != nil {
		return err
	}
	return http.ListenAndServe(":"+port, server)
}

func initServer() error {
	target := os.Getenv("FUNCTION_TARGET")
	switch {
	// If FUNCTION_TARGET is set, only serve this target function.
	case len(target) > 0:
		// Start a new server because the same route "/" can only be
		// set once on a server.
		server = http.NewServeMux()
		handlerRegistered = false
		h, err := wrapFunction(target)
		if err != nil {
			return err
		}
		server.Handle("/", h)
	// If any handlers are registered, only serve these registered handlers.
	case handlerRegistered:
		// No-op because the handlers are already registered with the server.
	// Serve all the registered functions.
	default:
		server = http.NewServeMux()
		handlerRegistered = false
		fns := registry.Default().GetAllRegisteredFunction()
		for funcName := range fns {
			h, err := wrapFunction(funcName)
			if err != nil {
				fmt.Printf("Failed to serve function %s: %v", funcName, err)
				continue
			}
			server.Handle("/"+funcName, h)
		}
	}
	return nil
}

func wrapFunction(name string) (http.Handler, error) {
	// Check if we have a function resource set, and if so, log progress.
	if os.Getenv("K_SERVICE") == "" {
		fmt.Printf("Serving function: %s\n", name)
	}
	// Check if there's a registered function, and use if possible.
	fn, ok := registry.Default().GetRegisteredFunction(name)
	if !ok {
		return nil, fmt.Errorf("no matching function found with name: %q", name)
	}
	ctx := context.Background()
	if fn.HTTPFn != nil {
		handler, err := wrapHTTPFunction(fn.HTTPFn)
		if err != nil {
			return nil, fmt.Errorf("unexpected error in wrapHTTPFunction: %v", err)
		}
		return handler, nil
	} else if fn.CloudEventFn != nil {
		handler, err := wrapCloudEventFunction(ctx, fn.CloudEventFn)
		if err != nil {
			return nil, fmt.Errorf("unexpected error in wrapCloudEventFunction: %v", err)
		}
		return handler, nil
	}
	return nil, fmt.Errorf("missing function entry in %v", fn)
}

func wrapHTTPFunction(fn func(http.ResponseWriter, *http.Request)) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(b/111823046): Remove following once Cloud Functions does not need flushing the logs anymore.
		if os.Getenv("K_SERVICE") != "" {
			// Force flush of logs after every function trigger when running on GCF.
			defer fmt.Println()
			defer fmt.Fprintln(os.Stderr)
		}
		defer recoverPanic(w, "user function execution")
		fn(w, r)
	}), nil
}

func wrapEventFunction(fn interface{}) (http.Handler, error) {
	err := validateEventFunction(fn)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("K_SERVICE") != "" {
			// Force flush of logs after every function trigger when running on GCF.
			defer fmt.Println()
			defer fmt.Fprintln(os.Stderr)
		}

		if shouldConvertCloudEventToBackgroundRequest(r) {
			if err := convertCloudEventToBackgroundRequest(r); err != nil {
				writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, fmt.Sprintf("error converting CloudEvent to Background Event: %v", err))
			}
		}

		handleEventFunction(w, r, fn)
	}), nil
}

func wrapCloudEventFunction(ctx context.Context, fn func(context.Context, cloudevents.Event) error) (http.Handler, error) {
	p, err := cloudevents.NewHTTP()
	if err != nil {
		return nil, fmt.Errorf("failed to create protocol: %v", err)
	}

	// Always log errors returned by the function to stderr
	logErrFn := func(ctx context.Context, ce cloudevents.Event) error {
		err := fn(ctx, ce)
		if err != nil {
			fmt.Fprintf(os.Stderr, fmtFunctionError(err))
		}
		return err
	}

	h, err := cloudevents.NewHTTPReceiveHandler(ctx, p, logErrFn)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %v", err)
	}

	return convertBackgroundToCloudEvent(h), nil
}

func handleEventFunction(w http.ResponseWriter, r *http.Request, fn interface{}) {
	body, err := readHTTPRequestBody(r)
	if err != nil {
		writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, fmt.Sprintf("%v", err))
		return
	}

	// Background events have data and an associated metadata, so parse those and run if present.
	if metadata, data, err := getBackgroundEvent(body, r.URL.Path); err != nil {
		writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, fmt.Sprintf("Error: %s, parsing background event: %s", err.Error(), string(body)))
		return
	} else if data != nil && metadata != nil {
		runBackgroundEvent(w, r, metadata, data, fn)
		return
	}

	// Otherwise, we assume the body is a JSON blob containing the user-specified data structure.
	runUserFunction(w, r, body, fn)
}

func readHTTPRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("request body not found")
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read request body %s: %v", r.Body, err)
	}

	return body, nil
}

func runUserFunction(w http.ResponseWriter, r *http.Request, data []byte, fn interface{}) {
	runUserFunctionWithContext(r.Context(), w, r, data, fn)
}

func runUserFunctionWithContext(ctx context.Context, w http.ResponseWriter, r *http.Request, data []byte, fn interface{}) {
	argVal := reflect.New(reflect.TypeOf(fn).In(1))
	if err := json.Unmarshal(data, argVal.Interface()); err != nil {
		writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, fmt.Sprintf("Error: %s, while converting event data: %s", err.Error(), string(data)))
		return
	}

	defer recoverPanic(w, "user function execution")
	userFunErr := reflect.ValueOf(fn).Call([]reflect.Value{
		reflect.ValueOf(ctx),
		argVal.Elem(),
	})
	if userFunErr[0].Interface() != nil {
		writeHTTPErrorResponse(w, http.StatusInternalServerError, errorStatus, fmtFunctionError(userFunErr[0].Interface()))
		return
	}
}

func fmtFunctionError(err interface{}) string {
	formatted := fmt.Sprintf(fnErrorMessageStderrTmpl, err)
	if !strings.HasSuffix(formatted, "\n") {
		formatted += "\n"
	}

	return formatted
}

func writeHTTPErrorResponse(w http.ResponseWriter, statusCode int, status, msg string) {
	// Ensure logs end with a newline otherwise they are grouped incorrectly in SD.
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Fprint(os.Stderr, msg)

	// Flush stdout and stderr when running on GCF. This must be done before writing
	// the HTTP response in order for all logs to appear in Stackdriver.
	if os.Getenv("K_SERVICE") != "" {
		fmt.Println()
		fmt.Fprintln(os.Stderr)
	}

	w.Header().Set(functionStatusHeader, status)
	w.WriteHeader(statusCode)
	fmt.Fprint(w, msg)
}

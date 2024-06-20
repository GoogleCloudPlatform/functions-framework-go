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
	"strconv"
	"strings"
	"time"

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

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// recoverPanic recovers from a panic in a consistent manner. panicSrc should
// describe what was happening when the panic was encountered, for example
// "user function execution". w is an http.ResponseWriter to write a generic
// response body to that does not expose the details of the panic; w can be
// nil to skip this. If panic needs to be recovered by different caller
// set shouldPanic to true.
func recoverPanic(w http.ResponseWriter, panicSrc string, shouldPanic bool) {
	if r := recover(); r != nil {
		genericMsg := fmt.Sprintf(panicMessageTmpl, panicSrc)
		fmt.Fprintf(os.Stderr, "%s\npanic message: %v\nstack trace: %v\n%s", genericMsg, r, r, debug.Stack())
		if w != nil {
			writeHTTPErrorResponse(w, http.StatusInternalServerError, crashStatus, genericMsg)
		}
		if shouldPanic {
			panic(r)
		}
	}
}

// RegisterHTTPFunction registers fn as an HTTP function.
// Maintained for backward compatibility. Please use RegisterHTTPFunctionContext instead.
func RegisterHTTPFunction(path string, fn interface{}) {
	defer recoverPanic(nil, "function registration", false)

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
	defer recoverPanic(nil, "function registration", false)
	if err := RegisterEventFunctionContext(ctx, path, fn); err != nil {
		panic(fmt.Sprintf("unexpected error in RegisterEventFunctionContext: %v", err))
	}
}

// RegisterHTTPFunctionContext registers fn as an HTTP function.
func RegisterHTTPFunctionContext(ctx context.Context, path string, fn func(http.ResponseWriter, *http.Request)) error {
	return registry.Default().RegisterHTTP(fn, registry.WithPath(path))
}

// RegisterEventFunctionContext registers fn as an event function. The function must have two arguments, a
// context.Context and a struct type depending on the event, and return an error. If fn has the
// wrong signature, RegisterEventFunction returns an error.
func RegisterEventFunctionContext(ctx context.Context, path string, fn interface{}) error {
	return registry.Default().RegisterEvent(fn, registry.WithPath(path))
}

// RegisterCloudEventFunctionContext registers fn as an cloudevent function.
func RegisterCloudEventFunctionContext(ctx context.Context, path string, fn func(context.Context, cloudevents.Event) error) error {
	return registry.Default().RegisterCloudEvent(fn, registry.WithPath(path))
}

// Start serves an HTTP server with registered function(s).
func Start(port string) error {
	return StartHostPort("", port)
}

// StartHostPort serves an HTTP server with registered function(s) on the given host and port.
func StartHostPort(hostname, port string) error {
	server, err := initServer()
	if err != nil {
		return err
	}
	return http.ListenAndServe(fmt.Sprintf("%s:%s", hostname, port), server)
}

func initServer() (*http.ServeMux, error) {
	server := http.NewServeMux()

	// If FUNCTION_TARGET is set, only serve this target function at path "/".
	// If not set, serve all functions at the registered paths.
	if target := os.Getenv("FUNCTION_TARGET"); len(target) > 0 {
		var targetFn *registry.RegisteredFunction

		fn, ok := registry.Default().GetRegisteredFunction(target)
		if ok {
			targetFn = fn
		} else if lastFnWithoutName := registry.Default().GetLastFunctionWithoutName(); lastFnWithoutName != nil {
			// If no function was found with the target name, assume the last function that's not registered declaratively
			// should be served at '/'.
			targetFn = lastFnWithoutName
		} else {
			return nil, fmt.Errorf("no matching function found with name: %q", target)
		}

		h, err := wrapFunction(targetFn)
		if err != nil {
			return nil, fmt.Errorf("failed to serve function %q: %v", target, err)
		}
		server.Handle("/", h)
		return server, nil
	}

	fns := registry.Default().GetAllFunctions()
	for _, fn := range fns {
		h, err := wrapFunction(fn)
		if err != nil {
			return nil, fmt.Errorf("failed to serve function at path %q: %v", fn.Path, err)
		}
		server.Handle(fn.Path, h)
	}
	return server, nil
}

func wrapFunction(fn *registry.RegisteredFunction) (http.Handler, error) {
	// Check if we have a function resource set, and if so, log progress.
	if os.Getenv("FUNCTION_TARGET") == "" {
		fmt.Printf("Serving function: %q\n", fn.Name)
	}

	if fn.HTTPFn != nil {
		handler, err := wrapHTTPFunction(fn.HTTPFn)
		if err != nil {
			return nil, fmt.Errorf("unexpected error in wrapHTTPFunction: %v", err)
		}
		return handler, nil
	} else if fn.CloudEventFn != nil {
		handler, err := wrapCloudEventFunction(context.Background(), fn.CloudEventFn)
		if err != nil {
			return nil, fmt.Errorf("unexpected error in wrapCloudEventFunction: %v", err)
		}
		return handler, nil
	} else if fn.EventFn != nil {
		handler, err := wrapEventFunction(fn.EventFn)
		if err != nil {
			return nil, fmt.Errorf("unexpected error in wrapEventFunction: %v", err)
		}
		return handler, nil
	} else if fn.TypedFn != nil {
		handler, err := wrapTypedFunction(fn.TypedFn)
		if err != nil {
			return nil, fmt.Errorf("unexpected error in wrapTypedFunction: %v", err)
		}
		return handler, nil
	}
	return nil, fmt.Errorf("missing function entry in %v", fn)
}

func wrapHTTPFunction(fn func(http.ResponseWriter, *http.Request)) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("K_SERVICE") != "" {
			// Force flush of logs after every function trigger when running on GCF.
			defer fmt.Println()
			defer fmt.Fprintln(os.Stderr)
		}
		r, cancel := setupRequestContext(r)
		if cancel != nil {
			defer cancel()
		}
		defer recoverPanic(w, "user function execution", false)
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
		r, cancel := setupRequestContext(r)
		if cancel != nil {
			defer cancel()
		}
		if shouldConvertCloudEventToBackgroundRequest(r) {
			if err := convertCloudEventToBackgroundRequest(r); err != nil {
				writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, fmt.Sprintf("error converting CloudEvent to Background Event: %v", err))
			}
		}

		handleEventFunction(w, r, fn)
	}), nil
}

func wrapTypedFunction(fn interface{}) (http.Handler, error) {
	inputType, err := validateTypedFunction(fn)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := readHTTPRequestBody(r)
		if err != nil {
			writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, fmt.Sprintf("%v", err))
			return
		}
		argVal := inputType

		if err := json.Unmarshal(body, argVal.Interface()); err != nil {
			writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, fmt.Sprintf("Error while converting input data. %s", err.Error()))
			return
		}

		defer recoverPanic(w, "user function execution", false)
		funcReturn := reflect.ValueOf(fn).Call([]reflect.Value{
			argVal.Elem(),
		})

		handleTypedReturn(w, funcReturn)
	}), nil
}

func handleTypedReturn(w http.ResponseWriter, funcReturn []reflect.Value) {
	if len(funcReturn) == 0 {
		return
	}
	errorVal := funcReturn[len(funcReturn)-1].Interface() // last return must be of type error
	if errorVal != nil && reflect.TypeOf(errorVal).AssignableTo(errorType) {
		writeHTTPErrorResponse(w, http.StatusInternalServerError, errorStatus, fmtFunctionError(errorVal))
		return
	}

	firstVal := funcReturn[0].Interface()
	if !reflect.TypeOf(firstVal).AssignableTo(errorType) {
		returnVal, _ := json.Marshal(firstVal)
		fmt.Fprintf(w, string(returnVal))
	}
}

func validateTypedFunction(fn interface{}) (*reflect.Value, error) {
	ft := reflect.TypeOf(fn)
	if ft.NumIn() != 1 {
		return nil, fmt.Errorf("expected function to have one parameters, found %d", ft.NumIn())
	}
	if ft.NumOut() > 2 {
		return nil, fmt.Errorf("expected function to have maximum two return values")
	}
	if ft.NumOut() > 0 && !ft.Out(ft.NumOut()-1).AssignableTo(errorType) {
		return nil, fmt.Errorf("expected last return type to be of error")
	}
	var inputType = reflect.New(ft.In(0))
	return &inputType, nil
}

func wrapCloudEventFunction(ctx context.Context, fn func(context.Context, cloudevents.Event) error) (http.Handler, error) {
	p, err := cloudevents.NewHTTP()
	if err != nil {
		return nil, fmt.Errorf("failed to create protocol: %v", err)
	}

	// Always log errors returned by the function to stderr
	logErrFn := func(ctx context.Context, ce cloudevents.Event) error {
		defer recoverPanic(nil, "user function execution", true)
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

	defer recoverPanic(w, "user function execution", false)
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
	// the HTTP response in order for all logs to appear in GCF.
	if os.Getenv("K_SERVICE") != "" {
		fmt.Println()
		fmt.Fprintln(os.Stderr)
	}

	w.Header().Set(functionStatusHeader, status)
	w.WriteHeader(statusCode)
	fmt.Fprint(w, msg)
}

func setupRequestContext(r *http.Request) (*http.Request, func()) {
	r, cancel := setContextTimeoutIfRequested(r)
	r = addLoggingIDsToRequest(r)
	return r, cancel
}

// setContextTimeoutIfRequested replaces the request's context with a cancellation if requested
func setContextTimeoutIfRequested(r *http.Request) (*http.Request, func()) {
	timeoutStr := os.Getenv("CLOUD_RUN_TIMEOUT_SECONDS")
	if timeoutStr == "" {
		return r, nil
	}
	timeoutSecs, err := strconv.Atoi(timeoutStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse CLOUD_RUN_TIMEOUT_SECONDS as an integer value in seconds: %v\n", err)
		return r, nil
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSecs)*time.Second)
	return r.WithContext(ctx), cancel
}

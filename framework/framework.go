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

// Package framework is a Functions Framework implementation for Go. It allows you to register
// HTTP and event functions, then start an HTTP server serving those functions.
package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
)

const (
	functionStatusHeader = "X-Google-Status"
	crashStatus          = "crash"
	errorStatus          = "error"
)

// RegisterHTTPFunction registers fn as an HTTP function.
func RegisterHTTPFunction(path string, fn interface{}) {
	fnHTTP, ok := fn.(func(http.ResponseWriter, *http.Request))
	if !ok {
		panic("expected function to have signature func(http.ResponseWriter, *http.Request)")
	}
	registerHTTPFunction(path, fnHTTP, http.DefaultServeMux)
}

// RegisterEventFunction registers fn as an event function. The function must have two arguments, a
// context.Context and a struct type depending on the event, and return an error. If fn has the
// wrong signature, RegisterEventFunction panics.
func RegisterEventFunction(path string, fn interface{}) {
	registerEventFunction(path, fn, http.DefaultServeMux)
}

// Start serves an HTTP server with registered function(s).
func Start(port string) error {
	// Check if we have a function resource set, and if so, log progress.
	if os.Getenv("K_SERVICE") == "" {
		fmt.Println("Serving function...")
	}
	return http.ListenAndServe(":"+port, nil)
}

func registerHTTPFunction(path string, fn func(http.ResponseWriter, *http.Request), h *http.ServeMux) {
	h.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// TODO(b/111823046): Remove following once Cloud Functions does not need flushing the logs anymore.
		// Force flush of logs after every function trigger.
		defer fmt.Println()
		defer fmt.Fprintln(os.Stderr)
		defer func() {
			if r := recover(); r != nil {
				writeHTTPErrorResponse(w, http.StatusInternalServerError, crashStatus, fmt.Sprintf("Function panic: %v\n\n%s", r, debug.Stack()))
			}
		}()
		fn(w, r)
	})
}

func registerEventFunction(path string, fn interface{}, h *http.ServeMux) {
	validateEventFunction(fn)
	h.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// TODO(b/111823046): Remove following once Cloud Functions does not need flushing the logs anymore.
		// Force flush of logs after every function trigger.
		defer fmt.Println()
		defer fmt.Fprintln(os.Stderr)
		defer func() {
			if r := recover(); r != nil {
				writeHTTPErrorResponse(w, http.StatusInternalServerError, crashStatus, fmt.Sprintf("Function panic: %v\n\n%s", r, debug.Stack()))
			}
		}()
		handleEventFunction(w, r, fn)
	})
}

func validateEventFunction(fn interface{}) {
	ft := reflect.TypeOf(fn)
	if ft.NumIn() != 2 {
		panic(fmt.Sprintf("expected function to have two parameters, found %d", ft.NumIn()))
	}
	var err error
	errorType := reflect.TypeOf(&err).Elem()
	if ft.NumOut() != 1 || !ft.Out(0).AssignableTo(errorType) {
		panic("expected function to return only an error")
	}
	var ctx context.Context
	ctxType := reflect.TypeOf(&ctx).Elem()
	if !ctxType.AssignableTo(ft.In(0)) {
		panic("expected first parameter to be context.Context")
	}
}

func isBinaryCloudEvent(r *http.Request) bool {
	ceReqHeaders := []string{"Ce-Type", "Ce-Specversion", "Ce-Source", "Ce-Id"}
	for _, h := range ceReqHeaders {
		if _, ok := r.Header[http.CanonicalHeaderKey(h)]; ok {
			return false
		}
	}
	return true
}

func handleEventFunction(w http.ResponseWriter, r *http.Request, fn interface{}) {
	body := readHTTPRequestBody(w, r)
	if body == nil {
		// No body, error has already been written.
		return
	}

	if isBinaryCloudEvent(r) {
		runUserFunction(w, r, body, fn)
		return
	}

	// Otherwise, we have to parse the request to extract the context and the body for the data.
	event := make(map[string]interface{})
	event["data"] = string(body)
	for k, v := range r.Header {
		k = strings.ToLower(k)
		if !strings.HasPrefix(k, "ce-") {
			continue
		}
		k = strings.TrimPrefix(k, "ce-")
		if len(v) != 1 {
			writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, fmt.Sprintf("Too many header values: %s", k))
			return
		}
		var mapVal map[string]interface{}
		if err := json.Unmarshal([]byte(v[0]), &mapVal); err != nil {
			// If there's an error, represent the field as the string from the header. Errors will be caught by the event constructor if present.
			event[k] = v[0]
		} else {
			// Otherwise, represent the unmarshalled map value.
			event[k] = mapVal
		}
	}

	// We don't want any escaping to happen here.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(event)
	if err != nil {
		writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, fmt.Sprintf("Unable to construct event: %s", err.Error()))
		return
	}

	runUserFunction(w, r, buf.Bytes(), fn)
}

func readHTTPRequestBody(w http.ResponseWriter, r *http.Request) []byte {
	if r.Body == nil {
		writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, "Request body not found")
		return nil
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeHTTPErrorResponse(w, http.StatusUnsupportedMediaType, crashStatus, fmt.Sprintf("Could not read request body: %s", err.Error()))
		return nil
	}

	return body
}

func runUserFunction(w http.ResponseWriter, r *http.Request, data []byte, fn interface{}) {
	argVal := reflect.New(reflect.TypeOf(fn).In(1))
	if err := json.Unmarshal(data, argVal.Interface()); err != nil {
		writeHTTPErrorResponse(w, http.StatusUnsupportedMediaType, crashStatus, fmt.Sprintf("Error while converting event data: %s", err.Error()))
		return
	}

	userFunErr := reflect.ValueOf(fn).Call([]reflect.Value{
		reflect.ValueOf(r.Context()),
		argVal.Elem(),
	})
	if userFunErr[0].Interface() != nil {
		writeHTTPErrorResponse(w, http.StatusInternalServerError, errorStatus, fmt.Sprintf("Function error: %v", userFunErr[0]))
		return
	}
}

func writeHTTPErrorResponse(w http.ResponseWriter, statusCode int, status, msg string) {
	w.Header().Set(functionStatusHeader, status)
	w.WriteHeader(statusCode)
	fmt.Fprintf(os.Stderr, msg)
	fmt.Fprintf(w, msg)
}

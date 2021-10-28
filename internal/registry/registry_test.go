// Copyright 2021 Google LLC
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
package registry

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

func TestRegisterHTTP(t *testing.T) {
	HTTPFunction("httpfn", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	})

	fn, ok := GetRegisteredFunction("httpfn")
	if !ok {
		t.Error("Expected function to be registered")
	}
	if fn.Name != "httpfn" {
		t.Errorf("Expected function name to be 'httpfn', got %s", fn.Name)
	}
}

func TestRegisterCE(t *testing.T) {
	CloudEventFunction("cefn", func(context.Context, cloudevents.Event) error {
		return nil
	})

	fn, ok := GetRegisteredFunction("cefn")
	if !ok {
		t.Error("Expected function to be registered")
	}
	if fn.Name != "cefn" {
		t.Errorf("Expected function name to be 'cefn', got %s", fn.Name)
	}
}

func TestRegisterMultipleFunctions(t *testing.T) {
	HTTPFunction("multifn1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	})
	HTTPFunction("multifn2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World 2!")
	})
	CloudEventFunction("multifn3", func(context.Context, cloudevents.Event) error {
		return nil
	})
}

func TestRegisterMultipleFunctionsError(t *testing.T) {
	// Expect no error
	err := HTTPFunction("samename", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	})
	if err != nil {
		t.Error("Expected no error registering function")
	}

	// Expect err error
	err = HTTPFunction("samename", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World 2!")
	})
	if err == nil {
		t.Error("Expected error registering function with same name")
	}
}

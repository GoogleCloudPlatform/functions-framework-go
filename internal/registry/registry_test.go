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
	registry := New()
	registry.RegisterHTTP("httpfn", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	})

	fn, ok := registry.GetRegisteredFunction("httpfn")
	if !ok {
		t.Fatalf("Expected function to be registered")
	}
	if fn.Name != "httpfn" {
		t.Errorf("Expected function name to be 'httpfn', got %s", fn.Name)
	}
}

func TestRegisterCE(t *testing.T) {
	registry := New()
	registry.RegisterCloudEvent("cefn", func(context.Context, cloudevents.Event) error {
		return nil
	})

	fn, ok := registry.GetRegisteredFunction("cefn")
	if !ok {
		t.Fatalf("Expected function to be registered")
	}
	if fn.Name != "cefn" {
		t.Errorf("Expected function name to be 'cefn', got %s", fn.Name)
	}
}

func TestRegisterMultipleFunctions(t *testing.T) {
	registry := New()
	if err := registry.RegisterHTTP("multifn1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	}); err != nil {
		t.Error("Expected \"multifn1\" function to be registered")
	}
	if err := registry.RegisterHTTP("multifn2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World 2!")
	}); err != nil {
		t.Error("Expected \"multifn2\" function to be registered")
	}
	if err := registry.RegisterCloudEvent("multifn3", func(context.Context, cloudevents.Event) error {
		return nil
	}); err != nil {
		t.Error("Expected \"multifn3\" function to be registered")
	}
}

func TestRegisterMultipleFunctionsError(t *testing.T) {
	registry := New()
	if err := registry.RegisterHTTP("samename", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	}); err != nil {
		t.Error("Expected no error registering function")
	}

	if err := registry.RegisterHTTP("samename", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World 2!")
	}); err == nil {
		t.Error("Expected error registering function with same name")
	}
}

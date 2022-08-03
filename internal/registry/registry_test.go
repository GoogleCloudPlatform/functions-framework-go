// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	testCases := []struct {
		name     string
		option   Option
		wantName string
		wantPath string
	}{
		{
			name:     "hello",
			wantName: "hello",
			wantPath: "/hello",
		},
		{
			name:     "withPath",
			option:   WithPath("/world"),
			wantName: "withPath",
			wantPath: "/world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := New()

			httpfn := func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "Hello World!") }
			if tc.option != nil {
				registry.RegisterHTTP(tc.name, httpfn, tc.option)
			} else {
				registry.RegisterHTTP(tc.name, httpfn)
			}

			fn, ok := registry.GetRegisteredFunction(tc.name)
			if !ok {
				t.Fatalf("Expected function to be registered")
			}
			if fn.Name != tc.wantName {
				t.Errorf("Expected function name to be %s, got %s", tc.wantName, fn.Name)
			}
			if fn.Path != tc.wantPath {
				t.Errorf("Expected function path to be %s, got %s", tc.wantPath, fn.Path)
			}
		})
	}
}

func TestRegisterCloudEvent(t *testing.T) {
	testCases := []struct {
		name     string
		option   Option
		wantName string
		wantPath string
	}{
		{
			name:     "hello",
			wantName: "hello",
			wantPath: "/hello",
		},
		{
			name:     "withPath",
			option:   WithPath("/world"),
			wantName: "withPath",
			wantPath: "/world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := New()

			cefn := func(context.Context, cloudevents.Event) error { return nil}
			if tc.option != nil {
				registry.RegisterCloudEvent(tc.name, cefn, tc.option)
			} else {
				registry.RegisterCloudEvent(tc.name, cefn)
			}

			fn, ok := registry.GetRegisteredFunction(tc.name)
			if !ok {
				t.Fatalf("Expected function to be registered")
			}
			if fn.Name != tc.wantName {
				t.Errorf("Expected function name to be %s, got %s", tc.wantName, fn.Name)
			}
			if fn.Path != tc.wantPath {
				t.Errorf("Expected function path to be %s, got %s", tc.wantPath, fn.Path)
			}
		})
	}
}

func TestRegisterEvent(t *testing.T) {
	testCases := []struct {
		name     string
		option   Option
		wantName string
		wantPath string
	}{
		{
			name:     "hello",
			wantName: "hello",
			wantPath: "/hello",
		},
		{
			name:     "withPath",
			option:   WithPath("/world"),
			wantName: "withPath",
			wantPath: "/world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := New()

			eventfn := func() {}
			if tc.option != nil {
				registry.RegisterEvent(tc.name, eventfn, tc.option)
			} else {
				registry.RegisterEvent(tc.name, eventfn)
			}

			fn, ok := registry.GetRegisteredFunction(tc.name)
			if !ok {
				t.Fatalf("Expected function to be registered")
			}
			if fn.Name != tc.wantName {
				t.Errorf("Expected function name to be %s, got %s", tc.wantName, fn.Name)
			}
			if fn.Path != tc.wantPath {
				t.Errorf("Expected function path to be %s, got %s", tc.wantPath, fn.Path)
			}
		})
	}
}

func TestRegisterMultipleFunctions(t *testing.T) {
	registry := New()
	if err := registry.RegisterHTTP("multifn1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	}); err != nil {
		t.Error("Expected \"multifn1\" function to be registered")
	}
	if err := registry.RegisterEvent("multifn2", func() {}); err != nil {
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

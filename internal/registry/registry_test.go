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
			option:   WithName("hello"),
			wantName: "hello",
			wantPath: "/hello",
		},
		{
			name:     "withPath",
			option:   WithPath("/world"),
			wantPath: "/world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := New()

			httpfn := func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "Hello World!") }
			registry.RegisterHTTP(httpfn, tc.option)

			if tc.wantName != "" {
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
			} else {
				fn := registry.GetLastFunctionWithoutName()
				if fn == nil {
					t.Fatalf("Expected function to be registered")
				}
				if fn.Path != tc.wantPath {
					t.Errorf("Expected function path to be %s, got %s", tc.wantPath, fn.Path)
				}
			}
		})
	}
}

func TestRegisterCloudEvent(t *testing.T) {
	testCases := []struct {
		name       string
		option     Option
		wantName   string
		wantPath   string
		wantLegacy bool
	}{
		{
			name:     "hello",
			option:   WithName("hello"),
			wantName: "hello",
			wantPath: "/hello",
		},
		{
			name:     "withPath",
			option:   WithPath("/world"),
			wantPath: "/world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := New()

			cefn := func(context.Context, cloudevents.Event) error { return nil }
			registry.RegisterCloudEvent(cefn, tc.option)

			if tc.wantName != "" {
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
			} else {
				fn := registry.GetLastFunctionWithoutName()
				if fn == nil {
					t.Fatalf("Expected function to be registered")
				}
				if fn.Path != tc.wantPath {
					t.Errorf("Expected function path to be %s, got %s", tc.wantPath, fn.Path)
				}
			}
		})
	}
}

func TestRegisterEvent(t *testing.T) {
	testCases := []struct {
		name       string
		option     Option
		wantName   string
		wantPath   string
		wantLegacy bool
	}{
		{
			name:     "hello",
			option:   WithName("hello"),
			wantName: "hello",
			wantPath: "/hello",
		},
		{
			name:     "withPath",
			option:   WithPath("/world"),
			wantPath: "/world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := New()

			eventfn := func() {}
			registry.RegisterEvent(eventfn, tc.option)

			if tc.wantName != "" {
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
			} else {
				fn := registry.GetLastFunctionWithoutName()
				if fn == nil {
					t.Fatalf("Expected function to be registered")
				}
				if fn.Path != tc.wantPath {
					t.Errorf("Expected function path to be %s, got %s", tc.wantPath, fn.Path)
				}
			}
		})
	}
}

func TestRegisterMultipleFunctions(t *testing.T) {
	registry := New()
	if err := registry.RegisterHTTP(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	}, WithName("multifn1")); err != nil {
		t.Error("Expected \"multifn1\" function to be registered")
	}
	if err := registry.RegisterEvent(func() {}, WithName("multifn2")); err != nil {
		t.Error("Expected \"multifn2\" function to be registered")
	}
	if err := registry.RegisterCloudEvent(func(context.Context, cloudevents.Event) error {
		return nil
	}, WithName("multifn3")); err != nil {
		t.Error("Expected \"multifn3\" function to be registered")
	}
}

func TestRegisterMultipleFunctionsError(t *testing.T) {
	registry := New()
	if err := registry.RegisterHTTP(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	}, WithName("samename")); err != nil {
		t.Error("Expected no error registering function")
	}

	if err := registry.RegisterHTTP(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World 2!")
	}, WithName("samename")); err == nil {
		t.Error("Expected error registering function with same name")
	}
}

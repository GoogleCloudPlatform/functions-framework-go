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

package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go"
)

func TestHTTPFunction(t *testing.T) {
	h := http.NewServeMux()
	registerHTTPFunction("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	}, h)

	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll: %v", err)
	}

	if got, want := string(body), "Hello World!"; got != want {
		t.Fatalf("TestHTTPFunction: got %v; want %v", got, want)
	}
}

type customStruct struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestEventFunction(t *testing.T) {
	cloudeventsJSON := []byte(`{
		"specversion" : "0.3",
		"type" : "com.github.pull.create",
		"source" : "https://github.com/cloudevents/spec/pull",
		"subject" : "123",
		"id" : "A234-1234-1234",
		"time" : "2018-04-05T17:31:00Z",
		"comexampleextension1" : "value",
		"comexampleextension2" : {"othervalue":5},
		"datacontenttype" : "text/xml",
		"data" : "<much wow=\"xml\"/>"
	}`)
	var testCE cloudevents.Event
	err := json.Unmarshal(cloudeventsJSON, &testCE)
	if err != nil {
		t.Fatalf("TestEventFunction: unable to create Event from JSON: %v", err)
	}

	var tests = []struct {
		name      string
		data      []byte
		fn        interface{}
		status    int
		header    string
		ceHeaders map[string]string
	}{
		{
			name: "valid function",
			data: []byte(`{"id": 12345,"name": "custom"}`),
			fn: func(c context.Context, s customStruct) error {
				if s.ID != 12345 {
					return fmt.Errorf("expected id=12345, got %d", s.ID)
				}
				if s.Name != "custom" {
					return fmt.Errorf("TestEventFunction(valid function): got name=%s, want name=\"custom\"", s.Name)
				}
				return nil
			},
			status: http.StatusOK,
			header: "",
		},
		{
			name: "incorrect type",
			data: []byte(`{"id": 12345,"name": 123}`),
			fn: func(c context.Context, s customStruct) error {
				return nil
			},
			status: http.StatusUnsupportedMediaType,
			header: "crash",
		},
		{
			name: "erroring function",
			data: []byte(`{"id": 12345,"name": "custom"}`),
			fn: func(c context.Context, s customStruct) error {
				return fmt.Errorf("TestEventFunction(erroring function): this error should fire")
			},
			status: http.StatusInternalServerError,
			header: "error",
		},
		{
			name: "cloudevent with context from headers",
			data: []byte("<much wow=\"xml\"/>"),
			fn: func(c context.Context, e cloudevents.Event) error {
				if e.String() != testCE.String() {
					return fmt.Errorf("TestEventFunction(cloudevent with context from header): got: %v, want: %v", e, testCE)
				}
				return nil
			},
			status: http.StatusOK,
			header: "",
			ceHeaders: map[string]string{
				"ce-specversion":          "0.3",
				"ce-type":                 "com.github.pull.create",
				"ce-source":               "https://github.com/cloudevents/spec/pull",
				"ce-subject":              "123",
				"ce-id":                   "A234-1234-1234",
				"ce-time":                 "2018-04-05T17:31:00Z",
				"ce-comexampleextension1": "value",
				"ce-comexampleextension2": `{"othervalue": 5}`,
				"ce-datacontenttype":      "text/xml",
			},
		},
		{
			name: "binary cloudevent request",
			data: cloudeventsJSON,
			fn: func(c context.Context, e cloudevents.Event) error {
				if e.String() != testCE.String() {
					return fmt.Errorf("TestEventFunction(binary cloudevent request): got: %v, want: %v", e, testCE)
				}
				return nil
			},
			status: http.StatusOK,
			header: "",
		},
	}

	for _, tc := range tests {
		h := http.NewServeMux()
		registerEventFunction("/", tc.fn, h)

		srv := httptest.NewServer(h)
		defer srv.Close()

		req, err := http.NewRequest("POST", srv.URL, bytes.NewBuffer(tc.data))
		req.Header.Set("Content-Type", "application/json")
		for k, v := range tc.ceHeaders {
			req.Header.Set(k, v)
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("client.Do(%s): %v", tc.name, err)
			continue
		}

		if resp.StatusCode != tc.status {
			t.Errorf("TestEventFunction(%s): response status = %v, want %v", tc.name, resp.StatusCode, tc.status)
			continue
		}
		if resp.Header.Get(functionStatusHeader) != tc.header {
			t.Errorf("TestEventFunction(%s): response header = %s, want %s", tc.name, resp.Header.Get(functionStatusHeader), tc.header)
			continue
		}
	}
}

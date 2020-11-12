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

package funcframework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

func TestHTTPFunction(t *testing.T) {
	h := http.NewServeMux()
	if err := registerHTTPFunction("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	}, h); err != nil {
		t.Fatalf("registerHTTPFunction(): %v", err)
	}

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

type eventData struct {
	Data string `json:"data"`
}

func TestEventFunction(t *testing.T) {
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
			name: "pubsub event",
			data: []byte(`{
				"context": {
					"eventId": "1234567",
					"timestamp": "2019-11-04T23:01:10.112Z",
					"eventType": "google.pubsub.topic.publish",
					"resource": {
						"service": "pubsub.googleapis.com",
						"name": "mytopic",
						"type": "type.googleapis.com/google.pubsub.v1.PubsubMessage"
					}
				},
				"data": {
					"@type": "type.googleapis.com/google.pubsub.v1.PubsubMessage",
					"attributes": null,
					"data": "test data"
					}
				}`),
			fn: func(c context.Context, e eventData) error {
				if e.Data != "test data" {
					return fmt.Errorf("TestEventFunction(pubsub event): got: %v, want: 'test data'", e.Data)
				}
				return nil
			},
			status: http.StatusOK,
			header: "",
		},
		{
			name: "pubsub legacy event",
			data: []byte(`{
				"eventId": "1234567",
				"timestamp": "2019-11-04T23:01:10.112Z",
				"eventType": "google.pubsub.topic.publish",
				"resource": {
					"service": "pubsub.googleapis.com",
					"name": "mytopic",
					"type": "type.googleapis.com/google.pubsub.v1.PubsubMessage"
				},
				"data": {
					"@type": "type.googleapis.com/google.pubsub.v1.PubsubMessage",
					"attributes": null,
					"data": "test data"
					}
				}`),
			fn: func(c context.Context, e eventData) error {
				if e.Data != "test data" {
					return fmt.Errorf("TestEventFunction(pubsub legacy event): got: %v, want: 'test data'", e.Data)
				}
				return nil
			},
			status: http.StatusOK,
			header: "",
		},
	}

	for _, tc := range tests {
		h := http.NewServeMux()
		if err := registerEventFunction("/", tc.fn, h); err != nil {
			t.Fatalf("registerEventFunction(): %v", err)
		}

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

func TestCloudEventFunction(t *testing.T) {
	cloudeventsJSON := []byte(`{
		"specversion" : "1.0",
		"type" : "com.github.pull.create",
		"source" : "https://github.com/cloudevents/spec/pull",
		"subject" : "123",
		"id" : "A234-1234-1234",
		"time" : "2018-04-05T17:31:00Z",
		"comexampleextension1" : "value",
		"datacontenttype" : "application/xml",
		"data" : "<much wow=\"xml\"/>"
	}`)
	var testCE cloudevents.Event
	err := json.Unmarshal(cloudeventsJSON, &testCE)
	if err != nil {
		t.Fatalf("TestCloudEventFunction: unable to create Event from JSON: %v", err)
	}

	replyType := fmt.Sprint("com.google.test.", rand.Uint64())
	replySource := fmt.Sprint("https://github.com/GoogleCloudPlatform/functions-framework-go/funcframework/", rand.Uint64())

	var tests = []struct {
		name        string
		data        []byte
		fn          interface{}
		ceHeaders   map[string]string
		status      int
		wantHeaders map[string]string
		wantBody    string
	}{
		{
			name: "binary cloudevent",
			data: []byte("<much wow=\"xml\"/>"),
			fn: func(ctx context.Context, e cloudevents.Event) error {
				if e.String() != testCE.String() {
					return fmt.Errorf("TestCloudEventFunction(binary cloudevent): got: %v, want: %v", e, testCE)
				}
				return nil
			},
			status: http.StatusOK,
			wantHeaders: map[string]string{
				functionStatusHeader: "",
			},
			ceHeaders: map[string]string{
				"ce-specversion":          "1.0",
				"ce-type":                 "com.github.pull.create",
				"ce-source":               "https://github.com/cloudevents/spec/pull",
				"ce-subject":              "123",
				"ce-id":                   "A234-1234-1234",
				"ce-time":                 "2018-04-05T17:31:00Z",
				"ce-comexampleextension1": "value",
				"Content-Type":            "application/xml",
			},
		},
		{
			name: "structured cloudevent",
			data: cloudeventsJSON,
			fn: func(ctx context.Context, e cloudevents.Event) error {
				if e.String() != testCE.String() {
					return fmt.Errorf("TestCloudEventFunction(structured cloudevent): got: %v, want: %v", e, testCE)
				}
				return nil
			},
			status: http.StatusOK,
			wantHeaders: map[string]string{
				functionStatusHeader: "",
			},
			ceHeaders: map[string]string{
				"Content-Type": "application/cloudevents+json",
			},
		},
		{
			name: "cloudevent reply",
			data: cloudeventsJSON,
			fn: func(ctx context.Context, e cloudevents.Event) (*cloudevents.Event, error) {
				// The other tests check content of the request, so just reply.
				ce := cloudevents.NewEvent(cloudevents.VersionV1)
				ce.SetType(replyType)
				ce.SetSource(replySource)

				data := struct {
					A int `json:"a"`
					B int `json:"b"`
				}{
					A: 3,
					B: 4,
				}
				if err := ce.SetData("application/json", data); err != nil {
					return nil, err
				}
				return &ce, nil
			},
			ceHeaders: map[string]string{
				"Content-Type": "application/cloudevents+json",
			},
			status: http.StatusOK,
			wantHeaders: map[string]string{
				functionStatusHeader: "",
				"Content-Type":       "application/json",
				"CE-Type":            replyType,
				"CE-Source":          replySource,
			},
			wantBody: `{"a":3,"b":4}`,
		},
	}

	for _, tc := range tests {
		ctx := context.Background()
		h := http.NewServeMux()
		if err := registerCloudEventFunction(ctx, "/", tc.fn, h); err != nil {
			t.Fatalf("registerCloudEventFunction(): %v", err)
		}

		srv := httptest.NewServer(h)
		defer srv.Close()

		req, err := http.NewRequest("POST", srv.URL, bytes.NewBuffer(tc.data))
		for k, v := range tc.ceHeaders {
			req.Header.Add(k, v)
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("client.Do(%s): %v", tc.name, err)
			continue
		}

		gotBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("unable to read got response body: %v", err)
		} else if string(gotBody) != tc.wantBody {
			t.Errorf("client.Do(%s) = %s, wanted %s", tc.name, string(gotBody), tc.wantBody)
		}

		if resp.StatusCode != tc.status {
			t.Errorf("TestCloudEventFunction(%s): response status = %v, want %v, %q.", tc.name, resp.StatusCode, tc.status, string(gotBody))
		}
		for k, v := range tc.wantHeaders {
			if resp.Header.Get(k) != v {
				t.Errorf("TestCloudEventFunction(%s): response header[%q] = %q, want %q", tc.name, k, resp.Header.Get(k), v)
			}
		}
	}
}

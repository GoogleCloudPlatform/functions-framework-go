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
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/GoogleCloudPlatform/functions-framework-go/internal/registry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-cmp/cmp"
)

func TestRegisterHTTPFunctionContext(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(w http.ResponseWriter, r *http.Request)
		wantStatus int // defaults to http.StatusOK
		wantResp   string
	}{
		{
			name: "helloworld",
			fn: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "Hello World!")
			},
			wantResp: "Hello World!",
		},
		{
			name: "panic in function",
			fn: func(w http.ResponseWriter, r *http.Request) {
				panic("intentional panic for test")
			},
			wantStatus: http.StatusInternalServerError,
			wantResp:   fmt.Sprintf(panicMessageTmpl, "user function execution"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer resetGlobalVars()

			if err := RegisterHTTPFunctionContext(context.Background(), "/", tc.fn); err != nil {
				t.Fatalf("RegisterHTTPFunctionContext(): %v", err)
			}

			srv := httptest.NewServer(server)
			defer srv.Close()

			resp, err := http.Get(srv.URL)
			if err != nil {
				t.Fatalf("http.Get: %v", err)
			}

			if tc.wantStatus == 0 {
				tc.wantStatus = http.StatusOK
			}
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("unexpected status code: got %d, want: %d", resp.StatusCode, tc.wantStatus)
			}

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("ioutil.ReadAll: %v", err)
			}

			if got := strings.TrimSpace(string(body)); got != tc.wantResp {
				t.Errorf("TestHTTPFunction: got %q; want: %q", got, tc.wantResp)
			}
		})
	}
}

type customStruct struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type eventData struct {
	Data string `json:"data"`
}

func TestRegisterEventFunctionContext(t *testing.T) {
	var tests = []struct {
		name       string
		body       []byte
		fn         interface{}
		status     int
		header     string
		ceHeaders  map[string]string
		wantResp   string
		wantStderr string
	}{
		{
			name: "valid function",
			body: []byte(`{"id": 12345,"name": "custom"}`),
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
			body: []byte(`{"id": 12345,"name": 123}`),
			fn: func(c context.Context, s customStruct) error {
				return nil
			},
			status: http.StatusBadRequest,
			header: "crash",
		},
		{
			name: "erroring function",
			body: []byte(`{"id": 12345,"name": "custom"}`),
			fn: func(c context.Context, s customStruct) error {
				return fmt.Errorf("TestEventFunction(erroring function): this error should fire")
			},
			status:     http.StatusInternalServerError,
			header:     "error",
			wantResp:   fmt.Sprintf(fnErrorMessageStderrTmpl, "TestEventFunction(erroring function): this error should fire"),
			wantStderr: "TestEventFunction(erroring function): this error should fire",
		},
		{
			name: "panicking function",
			body: []byte(`{"id": 12345,"name": "custom"}`),
			fn: func(c context.Context, s customStruct) error {
				panic("intential panic for test")
			},
			status:   http.StatusInternalServerError,
			header:   "crash",
			wantResp: fmt.Sprintf(panicMessageTmpl, "user function execution"),
		},
		{
			name: "pubsub event",
			body: []byte(`{
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
			body: []byte(`{
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
		{
			name: "cloudevent",
			body: []byte(`{
				"data": {
				  "bucket": "some-bucket",
				  "contentType": "text/plain",
				  "crc32c": "rTVTeQ==",
				  "etag": "CNHZkbuF/ugCEAE=",
				  "generation": "1587627537231057",
				  "id": "some-bucket/folder/Test.cs/1587627537231057",
				  "kind": "storage#object",
				  "md5Hash": "kF8MuJ5+CTJxvyhHS1xzRg==",
				  "mediaLink": "https://www.googleapis.com/download/storage/v1/b/some-bucket/o/folder%2FTest.cs?generation=1587627537231057\u0026alt=media",
				  "metageneration": "1",
				  "name": "folder/Test.cs",
				  "selfLink": "https://www.googleapis.com/storage/v1/b/some-bucket/o/folder/Test.cs",
				  "size": "352",
				  "storageClass": "MULTI_REGIONAL",
				  "timeCreated": "2020-04-23T07:38:57.230Z",
				  "timeStorageClassUpdated": "2020-04-23T07:38:57.230Z",
				  "updated": "2020-04-23T07:38:57.230Z"
				}
			  }`),
			fn: func(c context.Context, gotData map[string]interface{}) error {
				want := `{
					"data": {
					  "bucket": "some-bucket",
					  "contentType": "text/plain",
					  "crc32c": "rTVTeQ==",
					  "etag": "CNHZkbuF/ugCEAE=",
					  "generation": "1587627537231057",
					  "id": "some-bucket/folder/Test.cs/1587627537231057",
					  "kind": "storage#object",
					  "md5Hash": "kF8MuJ5+CTJxvyhHS1xzRg==",
					  "mediaLink": "https://www.googleapis.com/download/storage/v1/b/some-bucket/o/folder%2FTest.cs?generation=1587627537231057\u0026alt=media",
					  "metageneration": "1",
					  "name": "folder/Test.cs",
					  "selfLink": "https://www.googleapis.com/storage/v1/b/some-bucket/o/folder/Test.cs",
					  "size": "352",
					  "storageClass": "MULTI_REGIONAL",
					  "timeCreated": "2020-04-23T07:38:57.230Z",
					  "timeStorageClassUpdated": "2020-04-23T07:38:57.230Z",
					  "updated": "2020-04-23T07:38:57.230Z"
					}
				  }`

				var wantData map[string]interface{}
				if err := json.Unmarshal([]byte(want), &wantData); err != nil {
					return fmt.Errorf("unable to unmarshal test data: %s, error: %v", want, err)
				}

				if diff := cmp.Diff(wantData, gotData); diff != "" {
					return fmt.Errorf("TestEventFunction() mismatch (-want +got):\n%s", diff)
				}
				return nil
			},
			status: http.StatusOK,
			header: "",
			ceHeaders: map[string]string{
				"ce-specversion":     "1.0",
				"ce-type":            "google.cloud.storage.object.v1.finalized",
				"ce-source":          "//storage.googleapis.com/projects/_/buckets/some-bucket",
				"ce-subject":         "objects/folder/Test.cs",
				"ce-id":              "aaaaaa-1111-bbbb-2222-cccccccccccc",
				"ce-time":            "2020-09-29T11:32:00.000Z",
				"ce-datacontenttype": "application/json",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer resetGlobalVars()

			if err := RegisterEventFunctionContext(context.Background(), "/", tc.fn); err != nil {
				t.Fatalf("RegisterEventFunctionContext(): %v", err)
			}

			// Capture stderr for the duration of the test case. This includes
			// the stderr of the HTTP test server.
			origStderrPipe := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			defer func() { os.Stderr = origStderrPipe }()

			srv := httptest.NewServer(server)
			defer srv.Close()

			req, err := http.NewRequest("POST", srv.URL, bytes.NewBuffer(tc.body))
			if err != nil {
				t.Fatalf("error creating HTTP request for test: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			for k, v := range tc.ceHeaders {
				req.Header.Set(k, v)
			}
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do(%s): %v", tc.name, err)
			}

			if err := w.Close(); err != nil {
				t.Fatalf("failed to close stderr write pipe: %v", err)
			}

			stderr, err := ioutil.ReadAll(r)
			if err != nil {
				t.Errorf("failed to read stderr read pipe: %v", err)
			}

			if err := r.Close(); err != nil {
				t.Fatalf("failed to close stderr read pipe: %v", err)
			}

			if tc.wantStderr != "" && !strings.Contains(string(stderr), tc.wantStderr) {
				t.Errorf("stderr mismatch, got: %q, must contain: %q", string(stderr), tc.wantStderr)
			}

			if tc.wantResp != "" {
				gotBody, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("unable to read got request body: %v", err)
				}
				if strings.TrimSpace(string(gotBody)) != tc.wantResp {
					t.Errorf("TestEventFunction(%s): response body = %q, want %q on error status code %d.", tc.name, string(gotBody), tc.wantResp, tc.status)
				}
			}

			if resp.StatusCode != tc.status {
				t.Errorf("TestEventFunction(%s): response status = %v, want %v", tc.name, resp.StatusCode, tc.status)
			}

			if resp.Header.Get(functionStatusHeader) != tc.header {
				t.Errorf("TestEventFunction(%s): response header = %s, want %s", tc.name, resp.Header.Get(functionStatusHeader), tc.header)
			}
		})
	}
}

func TestRegisterCloudEventFunctionContext(t *testing.T) {
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

	var tests = []struct {
		name       string
		body       []byte
		fn         func(context.Context, cloudevents.Event) error
		status     int
		header     string
		ceHeaders  map[string]string
		wantResp   string
		wantStderr string
	}{
		{
			name: "binary cloudevent",
			body: []byte("<much wow=\"xml\"/>"),
			fn: func(ctx context.Context, e cloudevents.Event) error {
				if e.String() != testCE.String() {
					return fmt.Errorf("TestCloudEventFunction(binary cloudevent): got: %v, want: %v", e, testCE)
				}
				return nil
			},
			status: http.StatusOK,
			header: "",
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
			body: cloudeventsJSON,
			fn: func(ctx context.Context, e cloudevents.Event) error {
				if e.String() != testCE.String() {
					return fmt.Errorf("TestCloudEventFunction(structured cloudevent): got: %v, want: %v", e, testCE)
				}
				return nil
			},
			status: http.StatusOK,
			header: "",
			ceHeaders: map[string]string{
				"Content-Type": "application/cloudevents+json",
			},
		},
		{
			name: "background event",
			body: []byte(`{
				"context": {
				   "eventId": "aaaaaa-1111-bbbb-2222-cccccccccccc",
				   "timestamp": "2020-09-29T11:32:00.000Z",
				   "eventType": "google.storage.object.finalize",
				   "resource": {
					  "service": "storage.googleapis.com",
					  "name": "projects/_/buckets/some-bucket/objects/folder/Test.cs",
					  "type": "storage#object"
				   }
				},
				"data": {
				   "bucket": "some-bucket",
				   "contentType": "text/plain",
				   "crc32c": "rTVTeQ==",
				   "etag": "CNHZkbuF/ugCEAE=",
				   "generation": "1587627537231057",
				   "id": "some-bucket/folder/Test.cs/1587627537231057",
				   "kind": "storage#object",
				   "md5Hash": "kF8MuJ5+CTJxvyhHS1xzRg==",
				   "mediaLink": "https://www.googleapis.com/download/storage/v1/b/some-bucket/o/folder%2FTest.cs?generation=1587627537231057\u0026alt=media",
				   "metageneration": "1",
				   "name": "folder/Test.cs",
				   "selfLink": "https://www.googleapis.com/storage/v1/b/some-bucket/o/folder/Test.cs",
				   "size": "352",
				   "storageClass": "MULTI_REGIONAL",
				   "timeCreated": "2020-04-23T07:38:57.230Z",
				   "timeStorageClassUpdated": "2020-04-23T07:38:57.230Z",
				   "updated": "2020-04-23T07:38:57.230Z"
				}
			  }`),
			fn: func(ctx context.Context, e cloudevents.Event) error {
				want := `{
					"specversion": "1.0",
					"type": "google.cloud.storage.object.v1.finalized",
					"source": "//storage.googleapis.com/projects/_/buckets/some-bucket",
					"subject": "objects/folder/Test.cs",
					"id": "aaaaaa-1111-bbbb-2222-cccccccccccc",
					"time": "2020-09-29T11:32:00.000Z",
					"datacontenttype": "application/json",
					"data": {
					  "bucket": "some-bucket",
					  "contentType": "text/plain",
					  "crc32c": "rTVTeQ==",
					  "etag": "CNHZkbuF/ugCEAE=",
					  "generation": "1587627537231057",
					  "id": "some-bucket/folder/Test.cs/1587627537231057",
					  "kind": "storage#object",
					  "md5Hash": "kF8MuJ5+CTJxvyhHS1xzRg==",
					  "mediaLink": "https://www.googleapis.com/download/storage/v1/b/some-bucket/o/folder%2FTest.cs?generation=1587627537231057\u0026alt=media",
					  "metageneration": "1",
					  "name": "folder/Test.cs",
					  "selfLink": "https://www.googleapis.com/storage/v1/b/some-bucket/o/folder/Test.cs",
					  "size": "352",
					  "storageClass": "MULTI_REGIONAL",
					  "timeCreated": "2020-04-23T07:38:57.230Z",
					  "timeStorageClassUpdated": "2020-04-23T07:38:57.230Z",
					  "updated": "2020-04-23T07:38:57.230Z"
					}
				  }`
				wantCE := cloudevents.NewEvent()
				err := json.Unmarshal([]byte(want), &wantCE)
				if err != nil {
					return fmt.Errorf("error unmarshaling JSON to CloudEvent: %v", err)
				}

				if e.String() != wantCE.String() {
					return fmt.Errorf("TestCloudEventFunction got: %s, want: %s", e.String(), wantCE.String())
				}
				return nil
			},
			status: http.StatusOK,
		},
		{
			name: "panic returns 500",
			body: cloudeventsJSON,
			fn: func(ctx context.Context, e cloudevents.Event) error {
				panic("intentional panic for test")
			},
			status: http.StatusInternalServerError,
			ceHeaders: map[string]string{
				"Content-Type": "application/cloudevents+json",
			},
		},
		{
			name: "error returns 500",
			body: cloudeventsJSON,
			fn: func(ctx context.Context, e cloudevents.Event) error {
				return fmt.Errorf("error for test")
			},
			status: http.StatusInternalServerError,
			ceHeaders: map[string]string{
				"Content-Type": "application/cloudevents+json",
			},
			wantStderr: "error for test",
			wantResp:   "", // CloudEvent functions do not put the error message in the response body
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer resetGlobalVars()

			if err := RegisterCloudEventFunctionContext(context.Background(), "/", tc.fn); err != nil {
				t.Fatalf("RegisterCloudEventFunctionContext(): %v", err)
			}

			// Capture stderr for the duration of the test case. This includes
			// the stderr of the HTTP test server.
			origStderrPipe := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			defer func() { os.Stderr = origStderrPipe }()

			srv := httptest.NewServer(server)
			defer srv.Close()

			req, err := http.NewRequest("POST", srv.URL, bytes.NewBuffer(tc.body))
			if err != nil {
				t.Fatalf("error creating HTTP request for test: %v", err)
			}
			for k, v := range tc.ceHeaders {
				req.Header.Add(k, v)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do(%s): %v", tc.name, err)
			}

			if err := w.Close(); err != nil {
				t.Fatalf("failed to close stderr write pipe: %v", err)
			}

			stderr, err := ioutil.ReadAll(r)
			if err != nil {
				t.Errorf("failed to read stderr read pipe: %v", err)
			}

			if err := r.Close(); err != nil {
				t.Fatalf("failed to close stderr read pipe: %v", err)
			}

			if !strings.Contains(string(stderr), tc.wantStderr) {
				t.Errorf("stderr mismatch, got: %q, must contain: %q", string(stderr), tc.wantStderr)
			}

			gotBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unable to read got request body: %v", err)
			}
			if string(gotBody) != tc.wantResp {
				t.Errorf("TestCloudEventFunction(%s): response body = %q, want %q on error status code %d.", tc.name, gotBody, tc.wantResp, tc.status)
			}

			if resp.StatusCode != tc.status {
				t.Errorf("TestCloudEventFunction(%s): response status = %v, want %v, %q.", tc.name, resp.StatusCode, tc.status, string(gotBody))
			}
			if resp.Header.Get(functionStatusHeader) != tc.header {
				t.Errorf("TestCloudEventFunction(%s): response header = %q, want %q", tc.name, resp.Header.Get(functionStatusHeader), tc.header)
			}
		})
	}
}

func TestDeclarativeFunctionHTTP(t *testing.T) {
	defer resetGlobalVars()

	funcName := "httpfunc"
	funcResp := "Hello World!"
	os.Setenv("FUNCTION_TARGET", funcName)
	defer os.Unsetenv("FUNCTION_TARGET")

	// register functions
	functions.HTTP(funcName, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, funcResp)
	})
	if _, ok := registry.Default().GetRegisteredFunction(funcName); !ok {
		t.Fatalf("could not get registered function: %s", funcName)
	}

	if err := initServer(); err != nil {
		t.Fatalf("initServer(): %v", err)
	}
	srv := httptest.NewServer(server)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("could not make HTTP GET request to function: %s", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if got := strings.TrimSpace(string(body)); got != funcResp {
		t.Errorf("unexpected http response: got %q; want: %q", got, funcResp)
	}
}

func TestDeclarativeFunctionCloudEvent(t *testing.T) {
	defer resetGlobalVars()

	funcName := "cloudeventfunc"
	os.Setenv("FUNCTION_TARGET", funcName)
	defer os.Unsetenv("FUNCTION_TARGET")

	// register functions
	functions.CloudEvent(funcName, dummyCloudEvent)
	if _, ok := registry.Default().GetRegisteredFunction(funcName); !ok {
		t.Fatalf("could not get registered function: %s", funcName)
	}

	if err := initServer(); err != nil {
		t.Fatalf("initServer(): %v", err)
	}
	srv := httptest.NewServer(server)
	defer srv.Close()

	if _, err := http.Get(srv.URL); err != nil {
		t.Fatalf("could not make HTTP GET request to function: %s", err)
	}
}

func TestFunctionsNotRegisteredError(t *testing.T) {
	defer resetGlobalVars()

	funcName := "HelloWorld"
	os.Setenv("FUNCTION_TARGET", funcName)
	defer os.Unsetenv("FUNCTION_TARGET")

	wantErr := fmt.Sprintf("no matching function found with name: %q", funcName)

	if err := Start("0"); err.Error() != wantErr {
		t.Fatalf("Expected error: %s and received error: %s", wantErr, err.Error())
	}
}

func dummyCloudEvent(ctx context.Context, e cloudevents.Event) error {
	return nil
}

func TestServeMultipleFunctions(t *testing.T) {
	defer resetGlobalVars()

	fns := []struct {
		name     string
		fn       func(w http.ResponseWriter, r *http.Request)
		wantResp string
	}{
		{
			name: "fn1",
			fn: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "Hello Foo!")
			},
			wantResp: "Hello Foo!",
		},
		{
			name: "fn2",
			fn: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "Hello Bar!")
			},
			wantResp: "Hello Bar!",
		},
	}

	// Register functions.
	for _, f := range fns {
		functions.HTTP(f.name, f.fn)
		if _, ok := registry.Default().GetRegisteredFunction(f.name); !ok {
			t.Fatalf("could not get registered function: %s", f.name)
		}
	}

	if err := initServer(); err != nil {
		t.Fatalf("initServer(): %v", err)
	}
	srv := httptest.NewServer(server)
	defer srv.Close()

	for _, f := range fns {
		resp, err := http.Get(srv.URL + "/" + f.name)
		if err != nil {
			t.Fatalf("could not make HTTP GET request to function: %s", err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("io.ReadAll: %v", err)
		}
		if got := strings.TrimSpace(string(body)); got != f.wantResp {
			t.Errorf("unexpected http response: got %q; want: %q", got, f.wantResp)
		}
	}
}

func resetGlobalVars() {
	server = http.NewServeMux()
	handlerRegistered = false
}

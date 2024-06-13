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
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/GoogleCloudPlatform/functions-framework-go/internal/registry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/go-cmp/cmp"
)

func TestRegisterHTTPFunctionContext(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		fn         func(w http.ResponseWriter, r *http.Request)
		target     string
		wantStatus int // defaults to http.StatusOK
		wantResp   string
	}{
		{
			name: "helloworld",
			path: "/TestRegisterHTTPFunctionContext_helloworld",
			fn: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "Hello World!")
			},
			wantResp: "Hello World!",
		},
		{
			name: "FUNCTION_TARGET defined",
			path: "/TestRegisterHTTPFunctionContext_target",
			fn: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "Hello World!")
			},
			target:   "helloworld",
			wantResp: "Hello World!",
		},
		{
			name: "panic in function",
			path: "/TestRegisterHTTPFunctionContext_panic",
			fn: func(w http.ResponseWriter, r *http.Request) {
				panic("intentional panic for test")
			},
			wantStatus: http.StatusInternalServerError,
			wantResp:   fmt.Sprintf(panicMessageTmpl, "user function execution"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer cleanup()
			if len(tc.target) > 0 {
				os.Setenv("FUNCTION_TARGET", tc.target)
			}

			if err := RegisterHTTPFunctionContext(context.Background(), tc.path, tc.fn); err != nil {
				t.Fatalf("RegisterHTTPFunctionContext(): %v", err)
			}

			server, err := initServer()
			if err != nil {
				t.Fatalf("initServer(): %v", err)
			}
			srv := httptest.NewServer(server)
			defer srv.Close()

			resp, err := http.Get(srv.URL + tc.path)
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

type testStruct struct {
	Age  int
	Name string
}

type eventData struct {
	Data string `json:"data"`
}

func TestRegisterTypedFunction(t *testing.T) {
	var tests = []struct {
		name       string
		path       string
		body       []byte
		fn         interface{}
		target     string
		status     int
		header     string
		ceHeaders  map[string]string
		wantResp   string
		wantStderr string
	}{
		{
			name: "TestTypedFunction_typed",
			body: []byte(`{"id": 12345,"name": "custom"}`),
			fn: func(s customStruct) (customStruct, error) {
				return s, nil
			},
			status:   http.StatusOK,
			header:   "",
			wantResp: `{"id":12345,"name":"custom"}`,
		},
		{
			name: "TestTypedFunction_no_return",
			body: []byte(`{"id": 12345,"name": "custom"}`),
			fn: func(s customStruct) {

			},
			status:   http.StatusOK,
			header:   "",
			wantResp: "",
		},
		{
			name: "TestTypedFunction_untagged_struct",
			body: []byte(`{"Age": 30,"Name": "john"}`),
			fn: func(s testStruct) (testStruct, error) {
				return s, nil
			},
			status:   http.StatusOK,
			header:   "",
			wantResp: `{"Age":30,"Name":"john"}`,
		},
		{
			name: "TestTypedFunction_two_returns",
			body: []byte(`{"id": 12345,"name": "custom"}`),
			fn: func(s customStruct) (customStruct, error) {
				return s, nil
			},
			status:   http.StatusOK,
			header:   "",
			wantResp: `{"id":12345,"name":"custom"}`,
		},
		{
			name: "TestTypedFunction_return_int",
			body: []byte(`{"id": 12345,"name": "custom"}`),
			fn: func(s customStruct) (int, error) {
				return s.ID, nil
			},
			status:   http.StatusOK,
			header:   "",
			wantResp: "12345",
		},
		{
			name: "TestTypedFunction_different_types",
			body: []byte(`{"id": 12345,"name": "custom"}`),
			fn: func(s customStruct) (testStruct, error) {
				var t = testStruct{99, "John"}
				return t, nil
			},
			status:   http.StatusOK,
			header:   "",
			wantResp: `{"Age":99,"Name":"John"}`,
		},
		{
			name: "TestTypedFunction_return_error",
			body: []byte(`{"id": 12345,"name": "custom"}`),
			fn: func(s customStruct) error {
				return fmt.Errorf("Some error message")
			},
			status:     http.StatusInternalServerError,
			header:     "error",
			wantResp:   fmt.Sprintf(fnErrorMessageStderrTmpl, "Some error message"),
			wantStderr: "Some error message",
		},
		{
			name: "TestTypedFunction_data_error",
			body: []byte(`{"id": 12345,"name": 5}`),
			fn: func(s customStruct) (customStruct, error) {
				return s, nil
			},
			status:     http.StatusBadRequest,
			header:     "crash",
			wantStderr: "while converting input data",
		},
		{
			name: "TestTypedFunction_func_error",
			body: []byte(`{"id": 0,"name": "john"}`),
			fn: func(s customStruct) (customStruct, error) {
				s.ID = 10 / s.ID
				return s, nil
			},
			status:     http.StatusInternalServerError,
			header:     "crash",
			wantStderr: "A panic occurred during user function execution",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer cleanup()
			if len(tc.target) > 0 {
				os.Setenv("FUNCTION_TARGET", tc.target)
			}
			functions.Typed(tc.name, tc.fn)
			if _, ok := registry.Default().GetRegisteredFunction(tc.name); !ok {
				t.Fatalf("could not get registered function: %s", tc.name)
			}

			origStderrPipe := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			defer func() { os.Stderr = origStderrPipe }()

			server, err := initServer()
			if err != nil {
				t.Fatalf("initServer(): %v", err)
			}
			srv := httptest.NewServer(server)
			defer srv.Close()

			req, err := http.NewRequest("POST", srv.URL+"/"+tc.name, bytes.NewBuffer(tc.body))
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

			if tc.wantStderr != "" && !strings.Contains(string(stderr), tc.wantStderr) {
				t.Errorf("stderr mismatch, got: %q, must contain: %q", string(stderr), tc.wantStderr)
			}

			gotBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unable to read got request body: %v", err)
			}

			if tc.wantResp != "" && strings.TrimSpace(string(gotBody)) != tc.wantResp {
				t.Errorf("TestTypedFunction(%s): response body = %q, want %q on error status code %d.", tc.name, gotBody, tc.wantResp, tc.status)
			}

			if resp.StatusCode != tc.status {
				t.Errorf("TestTypedFunction(%s): response status = %v, want %v, %q.", tc.name, resp.StatusCode, tc.status, string(gotBody))
			}
			if resp.Header.Get(functionStatusHeader) != tc.header {
				t.Errorf("TestTypedFunction(%s): response header = %q, want %q", tc.name, resp.Header.Get(functionStatusHeader), tc.header)
			}
		})
	}
}

func TestRegisterEventFunctionContext(t *testing.T) {
	var tests = []struct {
		name       string
		path       string
		body       []byte
		fn         interface{}
		target     string
		status     int
		header     string
		ceHeaders  map[string]string
		wantResp   string
		wantStderr string
	}{
		{
			name: "valid function",
			path: "/TestRegisterEventFunctionContext_valid",
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
			name: "FUNCTION_TARGET defined",
			path: "/TestRegisterEventFunctionContext_target",
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
			target: "target",
			status: http.StatusOK,
			header: "",
		},
		{
			name: "incorrect type",
			path: "/TestRegisterEventFunctionContext_incorrect",
			body: []byte(`{"id": 12345,"name": 123}`),
			fn: func(c context.Context, s customStruct) error {
				return nil
			},
			status: http.StatusBadRequest,
			header: "crash",
		},
		{
			name: "erroring function",
			path: "/TestRegisterEventFunctionContext_erroring",
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
			path: "/TestRegisterEventFunctionContext_panicking",
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
			path: "/TestRegisterEventFunctionContext_pubsub1",
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
			path: "/TestRegisterEventFunctionContext_pubsub2",
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
			path: "/TestRegisterEventFunctionContext_cloudevent",
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
			defer cleanup()
			if len(tc.target) > 0 {
				os.Setenv("FUNCTION_TARGET", tc.target)
			}

			if err := RegisterEventFunctionContext(context.Background(), tc.path, tc.fn); err != nil {
				t.Fatalf("RegisterEventFunctionContext(): %v", err)
			}

			// Capture stderr for the duration of the test case. This includes
			// the stderr of the HTTP test server.
			origStderrPipe := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			defer func() { os.Stderr = origStderrPipe }()

			server, err := initServer()
			if err != nil {
				t.Fatalf("initServer(): %v", err)
			}
			srv := httptest.NewServer(server)
			defer srv.Close()

			req, err := http.NewRequest("POST", srv.URL+tc.path, bytes.NewBuffer(tc.body))
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
		path       string
		body       []byte
		fn         func(context.Context, cloudevents.Event) error
		target     string
		status     int
		header     string
		ceHeaders  map[string]string
		wantResp   string
		wantStderr string
	}{
		{
			name: "binary cloudevent",
			path: "/TestRegisterCloudEventFunctionContext_binary",
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
			path: "/TestRegisterCloudEventFunctionContext_structured",
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
			name: "FUNCTION_TARGET defined",
			path: "/TestRegisterCloudEventFunctionContext_target",
			body: cloudeventsJSON,
			fn: func(ctx context.Context, e cloudevents.Event) error {
				if e.String() != testCE.String() {
					return fmt.Errorf("TestCloudEventFunction(structured cloudevent): got: %v, want: %v", e, testCE)
				}
				return nil
			},
			target: "target",
			status: http.StatusOK,
			header: "",
			ceHeaders: map[string]string{
				"Content-Type": "application/cloudevents+json",
			},
		},
		{
			name: "background event",
			path: "/TestRegisterCloudEventFunctionContext_background",
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
			path: "/TestRegisterCloudEventFunctionContext_panic",
			body: cloudeventsJSON,
			fn: func(ctx context.Context, e cloudevents.Event) error {
				panic("intentional panic for test")
			},
			status: http.StatusInternalServerError,
			ceHeaders: map[string]string{
				"Content-Type": "application/cloudevents+json",
			},
			wantStderr: fmt.Sprintf(panicMessageTmpl, "user function execution"),
		},
		{
			name: "error returns 500",
			path: "/TestRegisterCloudEventFunctionContext_error",
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
			defer cleanup()
			if len(tc.target) > 0 {
				os.Setenv("FUNCTION_TARGET", tc.target)
			}

			if err := RegisterCloudEventFunctionContext(context.Background(), tc.path, tc.fn); err != nil {
				t.Fatalf("RegisterCloudEventFunctionContext(): %v", err)
			}

			// Capture stderr for the duration of the test case. This includes
			// the stderr of the HTTP test server.
			origStderrPipe := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			defer func() { os.Stderr = origStderrPipe }()

			server, err := initServer()
			if err != nil {
				t.Fatalf("initServer(): %v", err)
			}
			srv := httptest.NewServer(server)
			defer srv.Close()

			req, err := http.NewRequest("POST", srv.URL+tc.path, bytes.NewBuffer(tc.body))
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
	defer cleanup()
	funcName := "httpfunc"
	funcResp := "Hello World!"
	os.Setenv("FUNCTION_TARGET", funcName)

	// Verify RegisterHTTPFunctionContext and functions.HTTP don't conflict.
	if err := RegisterHTTPFunctionContext(context.Background(), "/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	}); err != nil {
		t.Fatalf("RegisterHTTPFunctionContext(): %v", err)
	}
	// Register functions.
	functions.HTTP(funcName, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, funcResp)
	})
	if _, ok := registry.Default().GetRegisteredFunction(funcName); !ok {
		t.Fatalf("could not get registered function: %q", funcName)
	}

	server, err := initServer()
	if err != nil {
		t.Fatalf("initServer(): %v", err)
	}
	srv := httptest.NewServer(server)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("could not make HTTP GET request to function: %q", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll: %v", err)
	}
	if got := strings.TrimSpace(string(body)); got != funcResp {
		t.Errorf("unexpected http response: got %q; want: %q", got, funcResp)
	}
}

func TestDeclarativeFunctionCloudEvent(t *testing.T) {
	defer cleanup()
	funcName := "cloudeventfunc"
	os.Setenv("FUNCTION_TARGET", funcName)

	// Verify RegisterCloudEventFunctionContext and functions.CloudEvent don't conflict.
	if err := RegisterCloudEventFunctionContext(context.Background(), "/", dummyCloudEvent); err != nil {
		t.Fatalf("registerHTTPFunction(): %v", err)
	}
	// register functions
	functions.CloudEvent(funcName, dummyCloudEvent)
	if _, ok := registry.Default().GetRegisteredFunction(funcName); !ok {
		t.Fatalf("could not get registered function: %s", funcName)
	}

	server, err := initServer()
	if err != nil {
		t.Fatalf("initServer(): %v", err)
	}
	srv := httptest.NewServer(server)
	defer srv.Close()

	if _, err := http.Get(srv.URL); err != nil {
		t.Fatalf("could not make HTTP GET request to function: %s", err)
	}
}

func TestFunctionsNotRegisteredError(t *testing.T) {
	defer cleanup()
	funcName := "HelloWorld"
	os.Setenv("FUNCTION_TARGET", funcName)

	wantErr := fmt.Sprintf("no matching function found with name: %q", funcName)
	if err := Start("0"); err.Error() != wantErr {
		t.Fatalf("Expected error: %s and received error: %s", wantErr, err.Error())
	}
}

func dummyCloudEvent(ctx context.Context, e cloudevents.Event) error {
	return nil
}

func TestServeMultipleFunctions(t *testing.T) {
	defer cleanup()
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

	server, err := initServer()
	if err != nil {
		t.Fatalf("initServer(): %v", err)
	}
	srv := httptest.NewServer(server)
	defer srv.Close()

	for _, f := range fns {
		resp, err := http.Get(srv.URL + "/" + f.name)
		if err != nil {
			t.Fatalf("could not make HTTP GET request to function: %s", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("ioutil.ReadAll: %v", err)
		}
		if got := strings.TrimSpace(string(body)); got != f.wantResp {
			t.Errorf("unexpected http response: got %q; want: %q", got, f.wantResp)
		}
	}
}

func TestHTTPRequestTimeout(t *testing.T) {
	timeoutEnvVar := "CLOUD_RUN_TIMEOUT_SECONDS"
	prev := os.Getenv(timeoutEnvVar)
	defer os.Setenv(timeoutEnvVar, prev)

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

	tcs := []struct {
		name              string
		wantDeadline      bool
		waitForExpiration bool
		timeout           string
	}{
		{
			name:              "deadline not requested",
			wantDeadline:      false,
			waitForExpiration: false,
			timeout:           "",
		},
		{
			name:              "NaN deadline",
			wantDeadline:      false,
			waitForExpiration: false,
			timeout:           "aaa",
		},
		{
			name:              "very long deadline",
			wantDeadline:      true,
			waitForExpiration: false,
			timeout:           "3600",
		},
		{
			name:              "short deadline should terminate",
			wantDeadline:      true,
			waitForExpiration: true,
			timeout:           "1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			defer cleanup()
			os.Setenv(timeoutEnvVar, tc.timeout)

			var httpReqCtx context.Context
			functions.HTTP("http", func(w http.ResponseWriter, r *http.Request) {
				if tc.waitForExpiration {
					<-r.Context().Done()
				}
				httpReqCtx = r.Context()
			})
			var ceReqCtx context.Context
			functions.CloudEvent("cloudevent", func(ctx context.Context, event event.Event) error {
				if tc.waitForExpiration {
					<-ctx.Done()
				}
				ceReqCtx = ctx
				return nil
			})
			server, err := initServer()
			if err != nil {
				t.Fatalf("initServer(): %v", err)
			}
			srv := httptest.NewServer(server)
			defer srv.Close()

			t.Run("http", func(t *testing.T) {
				_, err = http.Get(srv.URL + "/http")
				if err != nil {
					t.Fatalf("expected success")
				}
				if httpReqCtx == nil {
					t.Fatalf("expected non-nil request context")
				}
				deadline, ok := httpReqCtx.Deadline()
				if ok != tc.wantDeadline {
					t.Errorf("expected deadline %v but got %v", tc.wantDeadline, ok)
				}
				if expired := deadline.Before(time.Now()); ok && expired != tc.waitForExpiration {
					t.Errorf("expected expired %v but got %v", tc.waitForExpiration, expired)
				}
			})

			t.Run("cloudevent", func(t *testing.T) {
				req, err := http.NewRequest("POST", srv.URL+"/cloudevent", bytes.NewBuffer(cloudeventsJSON))
				if err != nil {
					t.Fatalf("failed to create request")
				}
				req.Header.Add("Content-Type", "application/cloudevents+json")
				client := &http.Client{}
				_, err = client.Do(req)
				if err != nil {
					t.Fatalf("request failed")
				}
				if ceReqCtx == nil {
					t.Fatalf("expected non-nil request context")
				}
				deadline, ok := ceReqCtx.Deadline()
				if ok != tc.wantDeadline {
					t.Errorf("expected deadline %v but got %v", tc.wantDeadline, ok)
				}
				if expired := deadline.Before(time.Now()); ok && expired != tc.waitForExpiration {
					t.Errorf("expected expired %v but got %v", tc.waitForExpiration, expired)
				}
			})
		})
	}
}

func cleanup() {
	os.Unsetenv("FUNCTION_TARGET")
	registry.Default().Reset()
}

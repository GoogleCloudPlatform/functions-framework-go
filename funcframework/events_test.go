package funcframework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"cloud.google.com/go/functions/metadata"
	"github.com/google/go-cmp/cmp"
)

func TestValidateEventFunction(t *testing.T) {
	tcs := []struct {
		name  string
		valid bool
		fn    interface{}
	}{
		{
			name:  "valid signature",
			valid: true,
			fn: func(context.Context, interface{}) error {
				return nil
			},
		},
		{
			name:  "missing error return",
			valid: false,
			fn:    func(context.Context, interface{}) {},
		},
		{
			name:  "missing parameter",
			valid: false,
			fn: func(context.Context) error {
				return nil
			},
		},
		{
			name:  "incorrect context parameter",
			valid: false,
			fn: func(time.Time, interface{}) error {
				return nil
			},
		},
		{
			name:  "additional parameter",
			valid: false,
			fn: func(context.Context, interface{}, interface{}) error {
				return nil
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEventFunction(tc.fn)
			if tc.valid && err != nil {
				t.Errorf("expected signature to be valid, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Errorf("expected signature to be invalid, but validation passed")
			}
		})
	}
}

func TestGetBackgroundEvent(t *testing.T) {
	timestamp, err := time.Parse(time.RFC3339, "2020-05-18T12:13:19.209Z")
	if err != nil {
		t.Fatalf("unable to parse time: %v", err)
	}
	tcs := []struct {
		name     string
		hasErr   bool
		input    []byte
		metadata *metadata.Metadata
		data     interface{}
	}{
		{
			name:   "invalid json",
			hasErr: true,
			input:  []byte(`{bad json`),
		},
		{
			name:   "not a background event but no error",
			input:  []byte(`{"random": "x"}`),
		},
		{
			name:   "data and context event",
			input: []byte(`{
   "context": {
      "eventId":"1144231683168617",
      "timestamp":"2020-05-18T12:13:19.209Z",
      "eventType":"google.pubsub.topic.publish",
      "resource":{
        "service":"pubsub.googleapis.com",
        "name":"projects/sample-project/topics/gcf-test",
        "type":"type.googleapis.com/google.pubsub.v1.PubsubMessage"
      }
   },
   "data": {
      "data": "dGVzdCBtZXNzYWdlIDM="
   }
}`),
			metadata: &metadata.Metadata{
				EventID:   "1144231683168617",
				Timestamp: timestamp,
				EventType: "google.pubsub.topic.publish",
				Resource: &metadata.Resource{
					Service: "pubsub.googleapis.com",
					Name:    "projects/sample-project/topics/gcf-test",
					Type:    "type.googleapis.com/google.pubsub.v1.PubsubMessage",
				},
			},
			data: map[string]interface{}{
				"data": "dGVzdCBtZXNzYWdlIDM=",
			},
		},
		{
			name:   "data and embedded context event",
			input: []byte(`{
  "eventId": "1215011316659232",
  "timestamp": "2020-05-18T12:13:19.209Z",
  "eventType": "providers/cloud.pubsub/eventTypes/topic.publish",
  "resource": "projects/sample-project/topics/gcf-test",
  "data": {
    "data": "VGhpcyBpcyBhIHNhbXBsZSBtZXNzYWdl"
  }
}`),
			metadata: &metadata.Metadata{
				EventID:   "1215011316659232",
				Timestamp: timestamp,
				EventType: "providers/cloud.pubsub/eventTypes/topic.publish",
				Resource: &metadata.Resource{
					RawPath: "projects/sample-project/topics/gcf-test",
				},
			},
			data: map[string]interface{}{
				"data": "VGhpcyBpcyBhIHNhbXBsZSBtZXNzYWdl",
			},
		},
		{
			name:   "data and invalid embedded context event no error",
			input: []byte(`{
  "data": {
    "data": "VGhpcyBpcyBhIHNhbXBsZSBtZXNzYWdl"
  }
}`),
		},
		{
			name:   "legacy pubsub event",
			input: []byte(`{
				"subscription": "projects/FOO/subscriptions/BAR_SUB",
				"message": {
					"data": "eyJmb28iOiJiYXIifQ==",
					"messageId": "1",
					"attributes": {
						"test": "123"
					}
				}
			}`),
			metadata: &metadata.Metadata{
				EventID:   "1",
				EventType: "google.pubsub.topic.publish",
				Resource: &metadata.Resource{
					Name: "projects/sample-project/topics/gcf-test",
					Type:    "type.googleapis.com/google.pubusb.v1.PubsubMessage",
					Service: "pubsub.googleapis.com",
				},
			},
			data: map[string]interface{}{
				"@type":      "type.googleapis.com/google.pubusb.v1.PubsubMessage",
				"data":       []byte(`{"foo":"bar"}`),
				"attributes": map[string]string{
					"test": "123",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			md, d, err := getBackgroundEvent(tc.input, "projects/sample-project/topics/gcf-test")
			if tc.hasErr && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tc.hasErr && err != nil {
				t.Errorf("expected no error, got error: %v", err)
			}

			if diff := cmp.Diff(tc.metadata, md); diff != "" {
				t.Errorf("MakeGatewayInfo() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.data, d); diff != "" {
				t.Errorf("getBackgroundEvent() data mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateCloudEvent(t *testing.T) {

	bgWithoutCtxBody := bytes.NewBuffer([]byte(`{
  "eventId": "1215011316659232",
  "timestamp": "2020-05-18T12:13:19.209Z",
  "eventType": "providers/cloud.pubsub/eventTypes/topic.publish",
  "resource": "projects/sample-project/topics/gcf-test",
  "data": {
    "data": "10"
  }
}`))
	bgWithoutCtxReq, err := http.NewRequest(http.MethodPost, "example.com", bgWithoutCtxBody)
	if err != nil {
		t.Fatalf("unable to create test request data: %v", err)
	}

	bgWithCtxBody := bytes.NewBuffer([]byte(`{
	"context": {
  	"eventId": "1215011316659232",
  	"timestamp": "2020-05-18T12:13:19.209Z",
		"eventType":"google.pubsub.topic.publish",
		"resource":{
			"service":"pubsub.googleapis.com",
			"name":"projects/sample-project/topics/gcf-test",
			"type":"type.googleapis.com/google.pubsub.v1.PubsubMessage"
		}
	},
  "data": {
    "data": "10"
  }
}`))
	bgWithCtxReq, err := http.NewRequest(http.MethodPost, "example.com", bgWithCtxBody)
	if err != nil {
		t.Fatalf("unable to create test request data: %v", err)
	}

	ce := map[string]interface{}{
		"specversion":     "1.0",
		"id":              "1215011316659232",
		"source":          "//pubsub.googleapis.com/projects/sample-project/topics/gcf-test",
		"time":            "2020-05-18T12:13:19Z",
		"type":            "google.cloud.pubsub.topic.v1.messagePublished",
		"datacontenttype": "application/json",
		"data": map[string]interface{}{
			"message": map[string]interface{}{
				"data": "10",
			},
		},
	}

	tcs := []struct {
		name         string
		responseCode int
		hasErr       bool
		req          *http.Request
		output       map[string]interface{}
	}{
		{
			name:         "background event without context attribute",
			responseCode: http.StatusOK,
			hasErr:       false,
			req:          bgWithoutCtxReq,
			output:       ce,
		},
		{
			name:         "background event with context attribute",
			responseCode: http.StatusOK,
			hasErr:       false,
			req:          bgWithCtxReq,
			output:       ce,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := createCloudEventRequest(tc.req)
			if tc.hasErr && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tc.hasErr && err != nil {
				t.Errorf("expected no error, got error: %v", err)
			}

			if tc.responseCode != rc {
				t.Errorf("incorrect response code, got %d, want %d", rc, tc.responseCode)
			}

			gotBody, err := ioutil.ReadAll(tc.req.Body)
			if err != nil {
				t.Fatalf("unable to read got request body: %v", err)
			}

			got := make(map[string]interface{})
			if err := json.Unmarshal(gotBody, &got); err != nil {
				t.Fatalf("unable to unmarshal got request body: %v", err)
			}

			if !cmp.Equal(got, tc.output) {
				t.Errorf("incorrect request output, got %v, want %v", got, tc.output)
			}

			if got := tc.req.Header.Get(contentTypeHeader); got != jsonContentType {
				t.Errorf("incorrect request content type header, got %s, want %s", got, jsonContentType)
			}

			if got := tc.req.Header.Get(contentLengthHeader); got != fmt.Sprint(len(gotBody)) {
				t.Errorf("incorrect request content length header, got %s, want %s", got, fmt.Sprint(len(gotBody)))
			}
		})
	}
}

func TestSplitResource(t *testing.T) {
	tcs := []struct {
		name         string
		service      string
		resource     string
		wantResource string
		wantSubject  string
	}{
		{
			// Firebase Auth resources are not split.
			name:         firebaseAuthCEService,
			service:      firebaseAuthCEService,
			resource:     "projects/my-project-id",
			wantResource: "projects/my-project-id",
		},
		{
			name:         firebaseCEService,
			service:      firebaseCEService,
			resource:     "projects/my-project-id/events/my-event",
			wantResource: "projects/my-project-id",
			wantSubject:  "events/my-event",
		},
		{
			name:         firebaseDBCEService,
			service:      firebaseDBCEService,
			resource:     "projects/_/instances/my-instance/refs/abc/xyz",
			wantResource: "projects/_/instances/my-instance",
			wantSubject:  "refs/abc/xyz",
		},
		{
			name:         firestoreCEService,
			service:      firestoreCEService,
			resource:     "projects/my-project-id/databases/(default)/documents/abc/xyz",
			wantResource: "projects/my-project-id/databases/(default)",
			wantSubject:  "documents/abc/xyz",
		},
		{
			// Pub/Sub resources are not split.
			// TODO(mtraver) Should we split on /topics/?
			name:         pubSubCEService,
			service:      pubSubCEService,
			resource:     "projects/my-project-id/topics/my-topic",
			wantResource: "projects/my-project-id/topics/my-topic",
		},
		{
			name:         storageCEService,
			service:      storageCEService,
			resource:     "projects/_/buckets/my-bucket/objects/abc/xyz",
			wantResource: "projects/_/buckets/my-bucket",
			wantSubject:  "objects/abc/xyz",
		},
		{
			name:         "nonexistent_service",
			service:      "not.a.valid.service",
			resource:     "projects/my-project-id/stuff/thing/abc/xyz",
			wantResource: "projects/my-project-id/stuff/thing/abc/xyz",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			gotResource, gotSubject, err := splitResource(tc.service, tc.resource)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tc.wantResource != gotResource {
				t.Errorf("incorrect resource, got %s, want %s", gotResource, tc.wantResource)
			}

			if tc.wantSubject != gotSubject {
				t.Errorf("incorrect subject, got %s, want %s", gotSubject, tc.wantSubject)
			}
		})
	}
}

func TestSplitResourceFailures(t *testing.T) {
	tcs := []struct {
		name     string
		service  string
		resource string
	}{
		{
			name:     "no_match",
			service:  storageCEService,
			resource: "projects/my-project-id/stuff/thing/abc/xyz",
		},
		{
			name:    "truncated_resource",
			service: storageCEService,
			// This resource should include an object path, e.g. "objects/abc/xyz",
			// and we match against the whole string so this will not match.
			resource: "projects/_/buckets/my-bucket/",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			gotResource, gotSubject, err := splitResource(tc.service, tc.resource)
			if err == nil {
				t.Errorf("expected error but got nil, resource %q, subject %q", gotResource, gotSubject)
			}
		})
	}
}

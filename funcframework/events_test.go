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
			hasErr: false,
			input:  []byte(`{"random": "x"}`),
		},
		{
			name:   "data and context event",
			hasErr: false,
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
			hasErr: false,
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
			hasErr: false,
			input: []byte(`{
  "data": {
    "data": "VGhpcyBpcyBhIHNhbXBsZSBtZXNzYWdl"
  }
}`),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			md, d, err := getBackgroundEvent(tc.input)
			if tc.hasErr && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tc.hasErr && err != nil {
				t.Errorf("expected no error, got error: %v", err)
			}

			if !cmp.Equal(md, tc.metadata) {
				t.Errorf("incorrect metadata, got %+v, want %+v", md, tc.metadata)
			}

			if !cmp.Equal(d, tc.data) {
				t.Errorf("incorrect data, got %+v, want %+v", d, tc.data)
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
			"data": "10",
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

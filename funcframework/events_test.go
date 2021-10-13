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
	"github.com/google/go-cmp/cmp/cmpopts"
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
		name         string
		hasErr       bool
		body         []byte
		url          string
		wantMetadata *metadata.Metadata
		wantData     interface{}
	}{
		{
			name:   "invalid json",
			hasErr: true,
			body:   []byte(`{bad json`),
		},
		{
			name: "not a background event but no error",
			body: []byte(`{"random": "x"}`),
		},
		{
			name: "data and context event",
			body: []byte(`{
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
			url: "projects/sample-project/topics/gcf-test",
			wantMetadata: &metadata.Metadata{
				EventID:   "1144231683168617",
				Timestamp: timestamp,
				EventType: "google.pubsub.topic.publish",
				Resource: &metadata.Resource{
					Service: "pubsub.googleapis.com",
					Name:    "projects/sample-project/topics/gcf-test",
					Type:    "type.googleapis.com/google.pubsub.v1.PubsubMessage",
				},
			},
			wantData: map[string]interface{}{
				"data": "dGVzdCBtZXNzYWdlIDM=",
			},
		},
		{
			name: "data and embedded context event",
			body: []byte(`{
  "eventId": "1215011316659232",
  "timestamp": "2020-05-18T12:13:19.209Z",
  "eventType": "providers/cloud.pubsub/eventTypes/topic.publish",
  "resource": "projects/sample-project/topics/gcf-test",
  "data": {
    "data": "VGhpcyBpcyBhIHNhbXBsZSBtZXNzYWdl"
  }
}`),
			wantMetadata: &metadata.Metadata{
				EventID:   "1215011316659232",
				Timestamp: timestamp,
				EventType: "providers/cloud.pubsub/eventTypes/topic.publish",
				Resource: &metadata.Resource{
					RawPath: "projects/sample-project/topics/gcf-test",
				},
			},
			wantData: map[string]interface{}{
				"data": "VGhpcyBpcyBhIHNhbXBsZSBtZXNzYWdl",
			},
		},
		{
			name: "data and invalid embedded context event no error",
			body: []byte(`{
  "data": {
    "data": "VGhpcyBpcyBhIHNhbXBsZSBtZXNzYWdl"
  }
}`),
		},
		{
			name: "data and embedded context event missing url",
			body: []byte(`{
  "eventId": "1215011316659232",
  "timestamp": "2020-05-18T12:13:19.209Z",
  "eventType": "providers/cloud.pubsub/eventTypes/topic.publish",
  "resource": "projects/sample-project/topics/gcf-test",
  "data": {
    "data": "VGhpcyBpcyBhIHNhbXBsZSBtZXNzYWdl"
  }
}`),
			// missing url has no effect on standard background events
			url: "",
			wantMetadata: &metadata.Metadata{
				EventID:   "1215011316659232",
				Timestamp: timestamp,
				EventType: "providers/cloud.pubsub/eventTypes/topic.publish",
				Resource: &metadata.Resource{
					RawPath: "projects/sample-project/topics/gcf-test",
				},
			},
			wantData: map[string]interface{}{
				"data": "VGhpcyBpcyBhIHNhbXBsZSBtZXNzYWdl",
			},
		},
		{
			name: "data and invalid embedded context event no error",
			body: []byte(`{
  "data": {
    "data": "VGhpcyBpcyBhIHNhbXBsZSBtZXNzYWdl"
  }
}`),
		},
		{
			name: "raw pubsub event",
			body: []byte(`{
				"subscription": "projects/FOO/subscriptions/BAR_SUB",
				"message": {
					"data": "eyJmb28iOiJiYXIifQ==",
					"messageId": "1",
					"attributes": {
						"test": "123"
					}
				}
			}`),
			url: "projects/sample-project/topics/gcf-test",
			wantMetadata: &metadata.Metadata{
				EventID:   "1",
				EventType: "google.pubsub.topic.publish",
				Resource: &metadata.Resource{
					Name:    "projects/sample-project/topics/gcf-test",
					Type:    "type.googleapis.com/google.pubusb.v1.PubsubMessage",
					Service: "pubsub.googleapis.com",
				},
			},
			wantData: map[string]interface{}{
				"@type": "type.googleapis.com/google.pubusb.v1.PubsubMessage",
				"data":  []byte(`{"foo":"bar"}`),
				"attributes": map[string]string{
					"test": "123",
				},
			},
		},
		{
			name: "raw pubsub event missing url",
			body: []byte(`{
				"subscription": "projects/FOO/subscriptions/BAR_SUB",
				"message": {
					"data": "eyJmb28iOiJiYXIifQ==",
					"messageId": "1",
					"attributes": {
						"test": "123"
					}
				}
			}`),
			wantMetadata: &metadata.Metadata{
				EventID:   "1",
				EventType: "google.pubsub.topic.publish",
				Resource: &metadata.Resource{
					Type:    "type.googleapis.com/google.pubusb.v1.PubsubMessage",
					Service: "pubsub.googleapis.com",
				},
			},
			wantData: map[string]interface{}{
				"@type": "type.googleapis.com/google.pubusb.v1.PubsubMessage",
				"data":  []byte(`{"foo":"bar"}`),
				"attributes": map[string]string{
					"test": "123",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			md, d, err := getBackgroundEvent(tc.body, tc.url)
			if tc.hasErr && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tc.hasErr && err != nil {
				t.Errorf("expected no error, got error: %v", err)
			}

			// If timestamp is not being tested in this test case, skip comparing the field
			// since some timestamps are auto-populated with time.Now()
			diffOpts := []cmp.Option{}
			if tc.wantMetadata != nil && tc.wantMetadata.Timestamp.IsZero() {
				diffOpts = append(diffOpts, cmpopts.IgnoreFields(metadata.Metadata{}, "Timestamp"))
			}

			if diff := cmp.Diff(tc.wantMetadata, md, diffOpts...); diff != "" {
				t.Errorf("getBackgroundEvent() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantData, d); diff != "" {
				t.Errorf("getBackgroundEvent() data mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConvertBackgroundToCloudEventRequest(t *testing.T) {
	pubsubCE := `{
		"specversion":     "1.0",
		"id":              "1215011316659232",
		"source":          "//pubsub.googleapis.com/projects/sample-project/topics/gcf-test",
		"time":            "2020-05-18T12:13:19.209Z",
		"type":            "google.cloud.pubsub.topic.v1.messagePublished",
		"datacontenttype": "application/json",
		"data": {
			"message": {
				"data": "10",
				"messageId": "1215011316659232",
				"publishTime": "2020-05-18T12:13:19.209Z"
			}
		}
	}`

	tcs := []struct {
		name    string
		reqBody string
		wantCE  string
	}{
		{
			name: "pubsub event without context attribute",
			reqBody: `{
				"eventId": "1215011316659232",
				"timestamp": "2020-05-18T12:13:19.209Z",
				"eventType": "providers/cloud.pubsub/eventTypes/topic.publish",
				"resource": "projects/sample-project/topics/gcf-test",
				"data": {
				  "data": "10"
				}
			}`,
			wantCE: pubsubCE,
		},
		{
			name: "background event with context attribute",
			reqBody: `{
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
			}`,
			wantCE: pubsubCE,
		},
		{
			name: "firebase auth event upcast",
			reqBody: `{
				"data": {
				  "email": "test@nowhere.com",
				  "metadata": {
					"createdAt": "2020-05-26T10:42:27Z",
					"lastSignedInAt": "2020-10-24T11:00:00Z"
				  },
				  "providerData": [
					{
					  "email": "test@nowhere.com",
					  "providerId": "password",
					  "uid": "test@nowhere.com"
					}
				  ],
				  "uid": "UUpby3s4spZre6kHsgVSPetzQ8l2"
				},
				"eventId": "aaaaaa-1111-bbbb-2222-cccccccccccc",
				"eventType": "providers/firebase.auth/eventTypes/user.create",
				"notSupported": {
				},
				"resource": "projects/my-project-id",
				"timestamp": "2020-09-29T11:32:00.000Z"
			  }`,
			wantCE: `{
				"specversion": "1.0",
				"type": "google.firebase.auth.user.v1.created",
				"source": "//firebaseauth.googleapis.com/projects/my-project-id",
				"subject": "users/UUpby3s4spZre6kHsgVSPetzQ8l2",
				"id": "aaaaaa-1111-bbbb-2222-cccccccccccc",
				"time": "2020-09-29T11:32:00.000Z",
				"datacontenttype": "application/json",
				"data": {
				  "email": "test@nowhere.com",
				  "metadata": {
					"createTime": "2020-05-26T10:42:27Z",
					"lastSignInTime": "2020-10-24T11:00:00Z"
				  },
				  "providerData": [
					{
					  "email": "test@nowhere.com",
					  "providerId": "password",
					  "uid": "test@nowhere.com"
					}
				  ],
				  "uid": "UUpby3s4spZre6kHsgVSPetzQ8l2"
				}
			  }`,
		},
		{
			name: "firebase db event upcast firebaseio.com domain",
			reqBody: `{
				"eventType": "providers/google.firebase.database/eventTypes/ref.write",
				"params": {
				  "child": "xyz"
				},
				"auth": {
				  "admin": true
				},
				"domain": "firebaseio.com",
				"data": {
				  "data": null,
				  "delta": {
					"grandchild": "other"
				  }
				},
				"resource": "projects/_/instances/my-project-id/refs/gcf-test/xyz",
				"timestamp": "2020-09-29T11:32:00.000Z",
				"eventId": "aaaaaa-1111-bbbb-2222-cccccccccccc"
			  }`,
			wantCE: `{
				"specversion": "1.0",
				"type": "google.firebase.database.ref.v1.written",
				"source": "//firebasedatabase.googleapis.com/projects/_/locations/us-central1/instances/my-project-id",
				"subject": "refs/gcf-test/xyz",
				"id": "aaaaaa-1111-bbbb-2222-cccccccccccc",
				"time": "2020-09-29T11:32:00.000Z",
				"datacontenttype": "application/json",
				"data": {
				  "data": null,
				  "delta": {
					"grandchild": "other"
				  }
				}
			  }`,
		},
		{
			name: "firebase db event upcast localized domain",
			reqBody: `{
				"eventType": "providers/google.firebase.database/eventTypes/ref.write",
				"params": {
				  "child": "xyz"
				},
				"auth": {
				  "admin": true
				},
				"domain":"europe-west1.firebasedatabase.app",
				"data": {
				  "data": {
					"grandchild": "other"
				  },
				  "delta": {
					"grandchild": "other changed"
				  }
				},
				"resource": "projects/_/instances/my-project-id/refs/gcf-test/xyz",
				"timestamp": "2020-09-29T11:32:00.000Z",
				"eventId": "aaaaaa-1111-bbbb-2222-cccccccccccc"
			  }`,
			wantCE: `{
				"specversion": "1.0",
				"type": "google.firebase.database.ref.v1.written",
				"source": "//firebasedatabase.googleapis.com/projects/_/locations/europe-west1/instances/my-project-id",
				"subject": "refs/gcf-test/xyz",
				"id": "aaaaaa-1111-bbbb-2222-cccccccccccc",
				"time": "2020-09-29T11:32:00.000Z",
				"datacontenttype": "application/json",
				"data": {
				  "data": {
					"grandchild": "other"
				  },
				  "delta": {
					"grandchild": "other changed"
				  }
				}
			  }`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "example.com", bytes.NewBufferString(tc.reqBody))
			if err != nil {
				t.Fatalf("unable to create test request data: %v", err)
			}

			rc, err := convertBackgroundToCloudEventRequest(req)
			if err != nil {
				t.Fatalf("unexpected error creating CloudEvent request: %v", err)
			}
			if rc != http.StatusOK {
				t.Errorf("incorrect response code, got %d, want %d", rc, http.StatusOK)
			}

			gotBody, err := ioutil.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("unable to read got request body: %v", err)
			}

			// Convert human-readable string into an easily comparable object
			// so cmp.Diff output is easier to read
			var wantObj map[string]interface{}
			if err := json.Unmarshal([]byte(tc.wantCE), &wantObj); err != nil {
				t.Fatalf("test wantCE is invalid JSON: %q, err: %v", tc.wantCE, err)
			}
			var gotObj map[string]interface{}
			if err := json.Unmarshal(gotBody, &gotObj); err != nil {
				t.Fatalf("createCloudEventRequest() created invalid JSON: %q, err: %v", string(gotBody), err)
			}

			if diff := cmp.Diff(wantObj, gotObj); diff != "" {
				t.Errorf("createCloudEventRequest() mismatch (-want +got):\n%s", diff)
			}

			if got := req.Header.Get(contentTypeHeader); got != jsonContentType {
				t.Errorf("incorrect request content type header, got %s, want %s", got, jsonContentType)
			}

			if got := req.Header.Get(contentLengthHeader); got != fmt.Sprint(len(gotBody)) {
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
			wantResource: "instances/my-instance",
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

package pubsub

import (
	"encoding/json"
	"testing"
	"time"

	"cloud.google.com/go/functions/metadata"
	"github.com/GoogleCloudPlatform/functions-framework-go/internal/fftypes"
	"github.com/google/go-cmp/cmp"
)

func TestExtractTopicFromRequestPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "localhost",
			path: "http://localhost:8080/projects/abc/topics/topic",
			want: "projects/abc/topics/topic",
		}, {
			name: "just topic",
			path: "projects/abc/topics/topic",
			want: "projects/abc/topics/topic",
		}, {
			name: "extra suffix",
			path: "http://localhost:8080/projects/abc/topics/topic/extra/suffix/",
			want: "projects/abc/topics/topic",
		}, {
			name: "extra prefix",
			path: "http://localhost:8080/extra/prefix/projects/abc/topics/topic",
			want: "projects/abc/topics/topic",
		}, {
			name: "from pubsub",
			path: "https://fake-tp.appspot.com/_ah/push-handlers/pubsub/projects/abc/topics/topic",
			want: "projects/abc/topics/topic",
		}, {
			name: "with parameters",
			path: "https://fake-tp.appspot.com/_ah/push-handlers/pubsub/projects/abc/topics/topic?pubsub_trigger=true",
			want: "projects/abc/topics/topic",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ExtractTopicFromRequestPath(test.path)
			if err != nil {
				t.Fatalf("ExtractTopicFromRequestPath(%s) got unexpected error: %s", test.path, err)
			}

			if got != test.want {
				t.Errorf("ExtractTopicFromRequestPath(%s) = %s, want %s", test.path, got, test.want)
			}
		})
	}
}

func TestExtractTopicFromRequestPath_failure(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "missing project",
			path: "https://fake-tp.appspot.com/_ah/push-handlers/pubsub/projects//topics/topic",
		}, {
			name: "missing topic",
			path: "https://fake-tp.appspot.com/_ah/push-handlers/pubsub/projects/abc/topics/",
		}, {
			name: "random",
			path: "fail/to/parse/this",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ExtractTopicFromRequestPath(test.path)
			if err == nil {
				t.Fatalf("ExtractTopicFromRequestPath(%s) = %s, should have returned error", test.path, got)
			}
		})
	}
}

func TestConvertLegacyEventToBackgroundEvent(t *testing.T) {
	timestamp, err := time.Parse(time.RFC3339, "2020-05-18T12:13:19.209Z")
	if err != nil {
		t.Fatalf("unable to parse time: %v", err)
	}

	tests := []struct {
		name  string
		body  string
		topic string
		want  *fftypes.BackgroundEvent
	}{
		{
			name: "legacy pubsub event",
			// eyJmb28iOiJiYXIifQ== is the base64 encoded version of
			// the string '{"foo":"bar"}'
			body: `{
	"subscription": "projects/FOO/subscriptions/BAR_SUB",
	"message": {
		"data": "eyJmb28iOiJiYXIifQ==",
		"messageId": "1",
		"attributes": {
			"test": "123"
		}
	}
}`,
			topic: "projects/FOO/topics/BAR_TOPIC",
			want: &fftypes.BackgroundEvent{
				Metadata: &metadata.Metadata{
					EventID:   "1",
					EventType: "google.pubsub.topic.publish",
					Resource: &metadata.Resource{
						Name:    "projects/FOO/topics/BAR_TOPIC",
						Type:    "type.googleapis.com/google.pubusb.v1.PubsubMessage",
						Service: "pubsub.googleapis.com",
					},
				},
				Data: map[string]interface{}{
					"@type": "type.googleapis.com/google.pubusb.v1.PubsubMessage",
					"data":  []byte(`{"foo":"bar"}`),
					"attributes": map[string]string{
						"test": "123",
					},
				},
			},
		}, {
			name: "missing topic",
			body: `{
	"subscription": "projects/FOO/subscriptions/BAR_SUB",
	"message": {
		"data": "eyJmb28iOiJiYXIifQ==",
		"messageId": "1",
		"attributes": {
			"test": "123"
		}
	}
}`,
			want: &fftypes.BackgroundEvent{
				Metadata: &metadata.Metadata{
					EventID:   "1",
					EventType: "google.pubsub.topic.publish",
					Resource: &metadata.Resource{
						Type:    "type.googleapis.com/google.pubusb.v1.PubsubMessage",
						Service: "pubsub.googleapis.com",
					},
				},
				Data: map[string]interface{}{
					"@type": "type.googleapis.com/google.pubusb.v1.PubsubMessage",
					"data":  []byte(`{"foo":"bar"}`),
					"attributes": map[string]string{
						"test": "123",
					},
				},
			},
		}, {
			name: "no attributes",
			body: `{
					"subscription": "projects/FOO/subscriptions/BAR_SUB",
					"message": {
						"data": "eyJmb28iOiJiYXIifQ==",
						"messageId": "1"
					}
					}`,
			want: &fftypes.BackgroundEvent{
				Metadata: &metadata.Metadata{
					EventID:   "1",
					EventType: "google.pubsub.topic.publish",
					Resource: &metadata.Resource{
						Type:    "type.googleapis.com/google.pubusb.v1.PubsubMessage",
						Service: "pubsub.googleapis.com",
					},
				},
				Data: map[string]interface{}{
					"@type":      "type.googleapis.com/google.pubusb.v1.PubsubMessage",
					"data":       []byte(`{"foo":"bar"}`),
					"attributes": map[string]string(nil),
				},
			},
		}, {
			name: "has timestamp",
			body: `{
							"subscription": "projects/FOO/subscriptions/BAR_SUB",
							"message": {
								"data": "eyJmb28iOiJiYXIifQ==",
								"messageId": "1",
								"publishTime":"2020-05-18T12:13:19.209Z"
							}
							}`,
			want: &fftypes.BackgroundEvent{
				Metadata: &metadata.Metadata{
					EventID:   "1",
					EventType: "google.pubsub.topic.publish",
					Timestamp: timestamp,
					Resource: &metadata.Resource{
						Type:    "type.googleapis.com/google.pubusb.v1.PubsubMessage",
						Service: "pubsub.googleapis.com",
					},
				},
				Data: map[string]interface{}{
					"@type":      "type.googleapis.com/google.pubusb.v1.PubsubMessage",
					"data":       []byte(`{"foo":"bar"}`),
					"attributes": map[string]string(nil),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := LegacyEvent{}
			if err := json.Unmarshal([]byte(test.body), &event); err != nil {
				t.Fatalf("failed to unmarshal test body JSON into a legacy Pub/Sub event: %s", test.body)
			}

			got := ConvertLegacyEventToBackgroundEvent(&event, test.topic)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("ConvertLegacyEventToBackgroundEvent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

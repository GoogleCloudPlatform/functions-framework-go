// Package pubsub contains utilities for handling Google Cloud Pub/Sub events.
package pubsub

import (
	"fmt"
	"regexp"
	"time"

	"cloud.google.com/go/functions/metadata"
	"github.com/GoogleCloudPlatform/functions-framework-go/internal/fftypes"
)

const (
	pubsubEventType   = "google.pubsub.topic.publish"
	pubsubMessageType = "type.googleapis.com/google.pubusb.v1.PubsubMessage"
	pubsubService     = "pubsub.googleapis.com"
)

// LegacyPushSubscriptionEvent is the event payload for legacy Cloud Pub/Sub
// push subscription triggers (https://cloud.google.com/functions/docs/calling/pubsub#legacy_cloud_pubsub_triggers).
// This matched the event payload that is sent by Pub/Sub to HTTP push
// subscription endpoints (https://cloud.google.com/pubsub/docs/push#receiving_messages).
type LegacyPushSubscriptionEvent struct {
	Subscription string `json:"subscription"`
	Message      `json:"message"`
}

// Message represents a Pub/Sub message.
type Message struct {
	// ID identifies this message.
	// This ID is assigned by the server and is populated for Messages obtained from a subscription.
	// This field is read-only.
	ID string `json:"messageId"`

	// Data is the actual data in the message.
	Data []byte `json:"data"`

	// Attributes represents the key-value pairs the current message
	// is labelled with.
	Attributes map[string]string `json:"attributes"`

	// The time at which the message was published.
	// This is populated by the server for Messages obtained from a subscription.
	// This field is read-only.
	PublishTime time.Time `json:"publishTime"`
}

// ExtractTopicFromRequestPath extracts a Pub/Sub topic from a URL request path.
func ExtractTopicFromRequestPath(path string) (string, error) {
	re := regexp.MustCompile(`(projects\/[^/?]+\/topics\/[^/?]+)/*`)
	matches := re.FindStringSubmatch(path)
	if matches == nil {
		return "", fmt.Errorf("failed to extract Pub/Sub topic name from the URL request path: %q, configure your subscription's push endpoint to use the following path pattern: 'projects/PROJECT_NAME/topics/TOPIC_NAME'",
			path)
	}

	// Index 0 is the entire input string matched, index 1 is the first submatch
	return matches[1], nil
}

// ToBackgroundEvent converts the event to the standard BackgroundEvent format
// for Background Functions.
func (e *LegacyPushSubscriptionEvent) ToBackgroundEvent(topic string) *fftypes.BackgroundEvent {
	timestamp := e.Message.PublishTime
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	return &fftypes.BackgroundEvent{
		Metadata: &metadata.Metadata{
			EventID:   e.ID,
			Timestamp: timestamp,
			EventType: pubsubEventType,
			Resource: &metadata.Resource{
				Name:    topic,
				Type:    pubsubMessageType,
				Service: pubsubService,
			},
		},
		Data: map[string]interface{}{
			"@type":      pubsubMessageType,
			"data":       e.Message.Data,
			"attributes": e.Message.Attributes,
		},
	}
}

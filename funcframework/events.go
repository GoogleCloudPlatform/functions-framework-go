package funcframework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/functions/metadata"
)

const (
	ceIDHeader          = "Ce-Id"
	contentTypeHeader   = "Content-Type"
	contentLengthHeader = "Content-Length"

	ceSpecVersion   = "1.0"
	jsonContentType = "application/cloudevents+json"
)

var (
	typeBackgroundToCloudEvent = map[string]string{
		"google.pubsub.topic.publish":                              "google.cloud.pubsub.topic.v1.messagePublished",
		"providers/cloud.pubsub/eventTypes/topic.publish":          "google.cloud.pubsub.topic.v1.messagePublished",
		"google.storage.object.finalize":                           "google.cloud.storage.object.v1.finalized",
		"google.storage.object.delete":                             "google.cloud.storage.object.v1.deleted",
		"google.storage.object.archive":                            "google.cloud.storage.object.v1.archived",
		"google.storage.object.metadataUpdate":                     "google.cloud.storage.object.v1.metadataUpdated",
		"providers/cloud.firestore/eventTypes/document.write":      "google.cloud.firestore.document.v1.written",
		"providers/cloud.firestore/eventTypes/document.create":     "google.cloud.firestore.document.v1.created",
		"providers/cloud.firestore/eventTypes/document.update":     "google.cloud.firestore.document.v1.updated",
		"providers/cloud.firestore/eventTypes/document.delete":     "google.cloud.firestore.document.v1.deleted",
		"providers/firebase.auth/eventTypes/user.create":           "google.firebase.auth.user.v1.created",
		"providers/firebase.auth/eventTypes/user.delete":           "google.firebase.auth.user.v1.deleted",
		"providers/google.firebase.analytics/eventTypes/event.log": "google.firebase.analytics.log.v1.written",
		"providers/google.firebase.database/eventTypes/ref.create": "google.firebase.database.document.v1.created",
		"providers/google.firebase.database/eventTypes/ref.write":  "google.firebase.database.document.v1.written",
		"providers/google.firebase.database/eventTypes/ref.update": "google.firebase.database.document.v1.updated",
		"providers/google.firebase.database/eventTypes/ref.delete": "google.firebase.database.document.v1.deleted",
		"providers/cloud.storage/eventTypes/object.change":         "google.cloud.storage.object.v1.finalized",
	}

	serviceBackgroundToCloudEvent = map[string]string{
		"providers/cloud.firestore/":           "firestore.googleapis.com",
		"providers/google.firebase.analytics/": "firebase.googleapis.com",
		"providers/firebase.auth/":             "firebase.googleapis.com",
		"providers/google.firebase.database/":  "firebase.googleapis.com",
		"providers/cloud.pubsub/":              "pubsub.googleapis.com",
		"providers/cloud.storage/":             "storage.googleapis.com",
		"google.pubsub":                        "pubsub.googleapis.com",
		"google.storage":                       "storage.googleapis.com",
	}
)

func getBackgroundEvent(body []byte) (*metadata.Metadata, interface{}, error) {
	// Handle background events' "data" and "context" fields.
	event := struct {
		Data     interface{}        `json:"data"`
		Metadata *metadata.Metadata `json:"context"`
	}{}
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, nil, err
	}

	// If there is no "data" payload, this isn't a background event, but that's okay.
	if event.Data == nil {
		return nil, nil, nil
	}

	// If the "context" field was present, we have a complete event and so return.
	if event.Metadata != nil {
		return event.Metadata, event.Data, nil
	}

	// Otherwise, try to directly populate a metadata object.
	m := &metadata.Metadata{}
	if err := json.Unmarshal(body, m); err != nil {
		return nil, nil, err
	}

	// Check for event ID to see if this is a background event, but if not that's okay.
	if m.EventID == "" {
		return nil, nil, nil
	}

	return m, event.Data, nil
}

func runBackgroundEvent(w http.ResponseWriter, r *http.Request, m *metadata.Metadata, data, fn interface{}) {
	b, err := encodeData(data)
	if err != nil {
		writeHTTPErrorResponse(w, http.StatusBadRequest, crashStatus, fmt.Sprintf("Unable to encode data %v: %s", data, err.Error()))
		return
	}
	ctx := metadata.NewContext(r.Context(), m)
	runUserFunctionWithContext(ctx, w, r, b, fn)
}

func validateEventFunction(fn interface{}) error {
	ft := reflect.TypeOf(fn)
	if ft.NumIn() != 2 {
		return fmt.Errorf("expected function to have two parameters, found %d", ft.NumIn())
	}
	var err error
	errorType := reflect.TypeOf(&err).Elem()
	if ft.NumOut() != 1 || !ft.Out(0).AssignableTo(errorType) {
		return fmt.Errorf("expected function to return only an error")
	}
	var ctx context.Context
	ctxType := reflect.TypeOf(&ctx).Elem()
	if !ctxType.AssignableTo(ft.In(0)) {
		return fmt.Errorf("expected first parameter to be context.Context")
	}
	return nil
}

func convertBackgroundToCloudEvent(ceHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the incoming request is not CloudEvent, make it so.
		if r.Header.Get(ceIDHeader) == "" && !strings.Contains(r.Header.Get(contentTypeHeader), "cloudevents") {
			rc, err := createCloudEventRequest(r)
			if err != nil {
				writeHTTPErrorResponse(w, rc, crashStatus, fmt.Sprintf("%v", err))
				return
			}
		}
		ceHandler.ServeHTTP(w, r)
	})
}

func encodeData(d interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(d); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func createCloudEventRequest(r *http.Request) (int, error) {
	body, rc, err := readHTTPRequestBody(r)
	if err != nil {
		return rc, err
	}

	md, d, err := getBackgroundEvent(body)
	if err != nil {
		return http.StatusUnsupportedMediaType, fmt.Errorf("parsing background event body %s: %v", string(body), err)
	}

	if md == nil || d == nil {
		return http.StatusUnsupportedMediaType, fmt.Errorf("unable to extract background event from %s", string(body))
	}

	r.Header.Set(contentTypeHeader, jsonContentType)

	t, ok := typeBackgroundToCloudEvent[md.EventType]
	if !ok {
		return http.StatusUnsupportedMediaType, fmt.Errorf("unable to find CloudEvent equivalent event type for %s", md.EventType)
	}

	service := md.Resource.Service
	if service == "" {
		for bService, ceService := range serviceBackgroundToCloudEvent {
			if strings.HasPrefix(md.EventType, bService) {
				service = ceService
			}
		}
		// If service is still empty, we didn't find a match in the map. Return the error.
		if service == "" {
			return http.StatusUnsupportedMediaType, fmt.Errorf("unable to find CloudEvent equivalent service for %s", md.EventType)
		}
	}

	resource := md.Resource.Name
	if resource == "" {
		resource = md.Resource.RawPath
	}

	source := fmt.Sprintf("//%s/%s", service, resource)

	ce := map[string]interface{}{
		"id":              md.EventID,
		"time":            md.Timestamp.Format(time.RFC3339),
		"specversion":     ceSpecVersion,
		"datacontenttype": "application/json",
		"type":            t,
		"source":          source,
		"data":            d,
	}

	encoded, err := json.Marshal(ce)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("Unable to marshal cloudevent %v: %s", ce, err.Error())
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(encoded))
	r.Header.Set(contentLengthHeader, fmt.Sprint(len(encoded)))
	return http.StatusOK, nil
}

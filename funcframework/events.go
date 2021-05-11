package funcframework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/functions/metadata"
	"github.com/GoogleCloudPlatform/functions-framework-go/internal/events/pubsub"
	"github.com/GoogleCloudPlatform/functions-framework-go/internal/fftypes"
)

const (
	ceIDHeader          = "Ce-Id"
	contentTypeHeader   = "Content-Type"
	contentLengthHeader = "Content-Length"

	ceSpecVersion   = "1.0"
	jsonContentType = "application/cloudevents+json"

	firebaseAuthCEService = "firebaseauth.googleapis.com"
	firebaseCEService     = "firebase.googleapis.com"
	firebaseDBCEService   = "firebasedatabase.googleapis.com"
	firestoreCEService    = "firestore.googleapis.com"
	pubSubCEService       = "pubsub.googleapis.com"
	storageCEService      = "storage.googleapis.com"
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
		"providers/cloud.firestore/":           firestoreCEService,
		"providers/google.firebase.analytics/": firebaseCEService,
		"providers/firebase.auth/":             firebaseAuthCEService,
		"providers/google.firebase.database/":  firebaseDBCEService,
		"providers/cloud.pubsub/":              pubSubCEService,
		"providers/cloud.storage/":             storageCEService,
		"google.pubsub":                        pubSubCEService,
		"google.storage":                       storageCEService,
	}

	// ceServiceToResourceRe maps CloudEvent service strings to regexps used to split
	// a background event resource string into CloudEvent resource and subject strings.
	// Each regexp must have exactly two submatches (a.k.a. capture groups): the first
	// for the resource and the second for the subject. See splitResource for more info.
	ceServiceToResourceRe = map[string]*regexp.Regexp{
		firebaseCEService:   regexp.MustCompile("^(projects/[^/]+)/(events/[^/]+)$"),
		firebaseDBCEService: regexp.MustCompile("^(projects/_/instances/[^/]+)/(refs/.+)$"),
		firestoreCEService:  regexp.MustCompile("^(projects/[^/]+/databases/\\(default\\))/(documents/.+)$"),
		storageCEService:    regexp.MustCompile("^(projects/_/buckets/[^/]+)/(objects/.+)$"),
	}

	// firebaseAuthMetadataFieldsBackgroundToCloudEvent maps Firebase Auth background event metadata field
	// names to their equivalent CloudEvent field names.
	firebaseAuthMetadataFieldsBackgroundToCloudEvent = map[string]string{
		"createdAt":      "createTime",
		"lastSignedInAt": "lastSignInTime",
	}
)

func getBackgroundEvent(body []byte, path string) (*metadata.Metadata, interface{}, error) {
	// Known background event types that the incoming request could represent.
	// Event types are mutually exclusive. During unmarshalling, only the field
	// for the matching type is populated.
	type possibleEvents struct {
		*pubsub.LegacyEvent
		*fftypes.BackgroundEvent
	}

	// Attempt to unmarshal into one of the known background event types.
	possible := possibleEvents{}
	if err := json.Unmarshal(body, &possible); err != nil {
		return nil, nil, err
	}

	event := possible.BackgroundEvent
	// If the background event payload is missing, check if it's a legacy
	// Pub/Sub event.
	if possible.BackgroundEvent == nil && possible.LegacyEvent != nil {
		topic, err := pubsub.ExtractTopicFromRequestPath(path)
		if err != nil {
			fmt.Printf("WARNING: %s", err)
		}
		event = pubsub.ConvertLegacyEventToBackgroundEvent(possible.LegacyEvent, topic)
	}

	// If there is no "data" payload, this isn't a background event, but that's okay.
	if event == nil || event.Data == nil {
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

// splitResource takes a background event resource string, which contains the full path to
// a resource, for example:
//
//   - Cloud Storage bucket and object within it
//   - Datastore database and entry within it
//
// and splits those two elements into separate strings; in CloudEvents the former is the
// "resource" and the latter is the "subject". Splitting is performed based on a regexp
// associated with the given CloudEvent service. See ceServiceToResourceRe for the regexp
// mapping. For example,
//
//   "projects/_/buckets/some-bucket/objects/folder/test.txt"
//
// would be split to create the strings "projects/_/buckets/some-bucket"
// and "objects/folder/test.txt". This function returns the resource string, the
// subject string, and an error, which will be non-nil if a regexp failed to match.
// If there is no regexp for the given service then the resource is returned unchanged
// along with a nil error.
func splitResource(service, resource string) (string, string, error) {
	re, ok := ceServiceToResourceRe[service]
	if !ok {
		return resource, "", nil
	}

	match := re.FindStringSubmatch(resource)
	if match == nil {
		return resource, "", fmt.Errorf("resource regexp did not match")
	}

	if len(match) != 3 {
		return resource, "", fmt.Errorf("expected 2 match groups, got %v", len(match)-1)
	}

	return match[1], match[2], nil
}

// convertBackgroundFirebaseAuthMetadata converts Firebase Auth background event metadata to CloudEvent metadata.
// The given data is only modified if it is a map with the requisite keys, so modifications occur in place.
func convertBackgroundFirebaseAuthMetadata(data interface{}) {
	d, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	if _, ok := d["metadata"]; !ok {
		return
	}

	metadata, ok := d["metadata"].(map[string]interface{})
	if !ok {
		return
	}

	for old, new := range firebaseAuthMetadataFieldsBackgroundToCloudEvent {
		if _, ok := metadata[old]; ok {
			metadata[new] = metadata[old]
			delete(metadata, old)
		}
	}
}

// firebaseAuthSubject creates the CloudEvent subject from the "uid" field in the data.
func firebaseAuthSubject(data interface{}) (string, error) {
	d, ok := data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("data is not a map from string to interface")
	}

	if _, ok := d["uid"]; !ok {
		return "", fmt.Errorf("data does not contain field \"uid\"")
	}

	return fmt.Sprintf("users/%v", d["uid"]), nil
}

func createCloudEventRequest(r *http.Request) (int, error) {
	body, rc, err := readHTTPRequestBody(r)
	if err != nil {
		return rc, err
	}

	md, d, err := getBackgroundEvent(body, r.URL.Path)
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

	var subject string
	resource, subject, err = splitResource(service, resource)
	if err != nil {
		return http.StatusUnsupportedMediaType, err
	}

	// Handle Pub/Sub events.
	if service == pubSubCEService {
		// In a CloudEvent "data" is wrapped by "message".
		d = struct {
			Message interface{} `json:"message"`
		}{
			Message: d,
		}
	}

	// Handle Firebase Auth events.
	if service == firebaseAuthCEService {
		convertBackgroundFirebaseAuthMetadata(d)

		if s, err := firebaseAuthSubject(d); err == nil {
			subject = s
		}
	}

	ce := map[string]interface{}{
		"id":              md.EventID,
		"time":            md.Timestamp.Format(time.RFC3339),
		"specversion":     ceSpecVersion,
		"datacontenttype": "application/json",
		"type":            t,
		"source":          fmt.Sprintf("//%s/%s", service, resource),
		"data":            d,
	}

	if subject != "" {
		ce["subject"] = subject
	}

	encoded, err := json.Marshal(ce)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("Unable to marshal CloudEvent %v: %v", ce, err)
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(encoded))
	r.Header.Set(contentLengthHeader, fmt.Sprint(len(encoded)))
	return http.StatusOK, nil
}

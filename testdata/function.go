// Package function contains test functions to validate the framework.
package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"cloud.google.com/go/functions/metadata"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

const (
	outputFile = "function_output.json"
)

// HTTP is a simple HTTP function that writes the request body to the response body.
func HTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := ioutil.WriteFile(outputFile, body, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// Event is a background event function that dumps the data and context to JSON and calls the
// validator script on the result.
func Event(ctx context.Context, data interface{}) error {
	m, err := metadata.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("getting context metadata: %v", err)
	}
	event := struct {
		Data    interface{}        `json:"data"`
		Context *metadata.Metadata `json:"context"`
	}{
		Data:    data,
		Context: m,
	}
	e, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshalling event: %v", err)
	}

	log.Printf("writing %v, %s to file", event, string(e))
	return ioutil.WriteFile(outputFile, e, 0644)
}

// CloudEvent is a cloud event function that dumps the event to JSON and calls the validator script
// on the result.
func CloudEvent(ctx context.Context, ce cloudevents.Event) {
	e, err := json.Marshal(ce)
	if err != nil {
		log.Fatalf("marshalling cloud event: %v", err)
	}

	if err := ioutil.WriteFile(outputFile, e, 0644); err != nil {
		log.Fatalf("writing file: %v", err)
	}
}

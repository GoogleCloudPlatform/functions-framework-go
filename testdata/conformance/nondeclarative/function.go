// Copyright 2022 Google LLC
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

// Package nondeclarative contains non-declarative (unregistered) functions to validate the framework.
package nondeclarative

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
func CloudEvent(ctx context.Context, ce cloudevents.Event) error {
	e, err := json.Marshal(ce)
	if err != nil {
		return fmt.Errorf("marshalling cloud event: %v", err)
	}

	if err := ioutil.WriteFile(outputFile, e, 0644); err != nil {
		return fmt.Errorf("writing file: %v", err)
	}

	return nil
}

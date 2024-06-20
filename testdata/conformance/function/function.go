// Copyright 2021 Google LLC
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

// Package function contains test functions to validate the framework.
package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

const (
	outputFile = "function_output.json"
)

// Register declarative functions
func init() {
	functions.HTTP("declarativeHTTP", HTTP)
	functions.HTTP("concurrentHTTP", concurrentHTTP)
	functions.Typed("declarativeTyped", Typed)
	functions.CloudEvent("declarativeCloudEvent", CloudEvent)
}

func concurrentHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(5 * time.Second)
}

// HTTP is a simple HTTP function that writes the request body to the response body.
func HTTP(w http.ResponseWriter, r *http.Request) {
	l := log.New(funcframework.LogWriter(r.Context()), "", log.Lshortfile)
	l.Println("handling HTTP request")
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

// CloudEvent is a cloud event function that dumps the event to JSON and calls the validator script
func CloudEvent(ctx context.Context, ce cloudevents.Event) error {
	l := log.New(funcframework.LogWriter(ctx), "", log.Lshortfile)
	l.Println("handling CloudEvent request")
	e, err := json.Marshal(ce)
	if err != nil {
		return fmt.Errorf("marshalling cloud event: %v", err)
	}

	if err := ioutil.WriteFile(outputFile, e, 0644); err != nil {
		return fmt.Errorf("writing file: %v", err)
	}

	return nil
}

// Typed is a typed function that dumps the request JSON into the "payload" field of the response i.e. the request {"message":"foo"} becomes {"payload":{"message":"foo"}}}
func Typed(req interface{}) (ConformanceResponse, error) {
	return ConformanceResponse{
		Payload: req,
	}, nil
}

type ConformanceResponse struct {
	Payload interface{} `json:"payload"`
}

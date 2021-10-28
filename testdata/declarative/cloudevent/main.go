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

// Binary that serves the HTTP conformance test function.
package main

import (
	"context"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

func init() {
	log.Print("Listening to function \"cloudevent\" at http://localhost:8080/")
	funcframework.CloudEvent("cloudevent", fn)
}

func fn(ctx context.Context, ce cloudevents.Event) error {
	return nil
}

func main() {
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("Failed to start functions framework: %v", err)
	}
}

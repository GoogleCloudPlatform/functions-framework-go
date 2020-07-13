// This binary starts an HTTP server to serve the Go FF validation test functions.
package main

import (
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/conformance-tests/function"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

func main() {
	funcframework.RegisterHTTPFunction("/", function.LegacyEvent)

	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}

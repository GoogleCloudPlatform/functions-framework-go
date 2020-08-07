// This binary starts an HTTP server to serve the Go FF validation test functions.
package main

import (
	"context"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/conformance-tests/function"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

func main() {
	ctx := context.Background()
	if err := funcframework.RegisterHTTPFunctionContext(ctx, "/", function.HTTP); err != nil {
		log.Fatalf("funcframework.RegisterHTTPFunctionContext: %v\n", err)
	}

	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}

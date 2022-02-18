module github.com/GoogleCloudPlatform/functions-framework-go

go 1.11

require (
	cloud.google.com/go/functions v1.0.0
	github.com/cloudevents/sdk-go/v2 v2.6.1
	github.com/google/go-cmp v0.5.6
)

retract (
	// Declarative registration functions were in the wrong package that
	// caused container build breaks.
	v1.4.0
)
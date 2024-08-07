module github.com/GoogleCloudPlatform/functions-framework-go

go 1.11

require (
	cloud.google.com/go/functions v1.16.2
	cloud.google.com/go/logging v1.10.0 // indirect
	github.com/cloudevents/sdk-go/v2 v2.14.0
	github.com/google/go-cmp v0.6.0
)

// later versions of cloudevents support only go1.18 or later, including some
// vulnerability fixes
require (
	github.com/cloudevents/sdk-go/v2 v2.15.2
) // +build go1.18

package funcframework

import (
	"bytes"
	"fmt"
	"log"
	"net/http/httptest"
	"testing"
)

func TestLoggingIDExtraction(t *testing.T) {
	tcs := []struct {
		name            string
		headers         map[string]string
		wantTraceID     string
		wantSpanID      string
		wantExecutionID string
		randomExecutionIdGenerated	bool
	}{
		{
			name:    "no IDs",
			headers: map[string]string{},
			randomExecutionIdGenerated: true,
		},
		{
			name: "provided execution ID only",
			headers: map[string]string{
				"Function-Execution-Id": "exec id",
			},
			wantExecutionID: "exec id",
		},
		{
			name: "malformatted X-Cloud-Trace-Context",
			headers: map[string]string{
				"X-Cloud-Trace-Context": "$*#$(v434)",
			},
			randomExecutionIdGenerated: true,
		},
		{
			name: "trace ID only",
			headers: map[string]string{
				"X-Cloud-Trace-Context": "0123456789abcdef",
			},
			wantTraceID: "0123456789abcdef",
			randomExecutionIdGenerated: true,
		},
		{
			name: "trace ID and span ID",
			headers: map[string]string{
				"X-Cloud-Trace-Context": "0123456789abcdef/aaaaaa",
			},
			wantTraceID: "0123456789abcdef",
			wantSpanID:  "aaaaaa",
			randomExecutionIdGenerated: true,
		},
		{
			name: "all",
			headers: map[string]string{
				"X-Cloud-Trace-Context": "a/b",
				"Function-Execution-Id": "c",
			},
			wantTraceID:     "a",
			wantSpanID:      "b",
			wantExecutionID: "c",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", bytes.NewReader(nil))
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}
			r = addLoggingIDsToRequest(r)
			ctx := r.Context()

			if tid := TraceIDFromContext(ctx); tid != tc.wantTraceID {
				t.Errorf("expected trace id %q but got %q", tc.wantTraceID, tid)
			}

			if spid := SpanIDFromContext(ctx); spid != tc.wantSpanID {
				t.Errorf("expected span id %q but got %q", tc.wantSpanID, spid)
			}

			eid := ExecutionIDFromContext(ctx); 
			if tc.wantExecutionID != "" && eid != tc.wantExecutionID {
				t.Errorf("expected execution id %q but got %q", tc.wantExecutionID, eid)
			}
			if tc.randomExecutionIdGenerated && eid == "" {
				t.Errorf("expected random execution id generated but got %q", eid)
			}
		})
	}
}

func TestStructuredLogWriter(t *testing.T) {
	output := bytes.NewBuffer(nil)

	w := &structuredLogWriter{
		w: output,
		loggingIDs: loggingIDs{
			spanID:      "a",
			trace:       "b",
			executionID: "c",
		},
	}

	fmt.Fprintf(w, "hello world!\n")
	fmt.Fprintf(w, "this is another log line!\n")

	wantOutput := `{"message":"hello world!","logging.googleapis.com/trace":"b","logging.googleapis.com/spanId":"a","logging.googleapis.com/labels":{"execution_id":"c"}}
{"message":"this is another log line!","logging.googleapis.com/trace":"b","logging.googleapis.com/spanId":"a","logging.googleapis.com/labels":{"execution_id":"c"}}
`
	if output.String() != wantOutput {
		t.Errorf("expected output %q got %q", wantOutput, output.String())
	}
}

func TestLogPackageCompat(t *testing.T) {
	output := bytes.NewBuffer(nil)
	w := &structuredLogWriter{
		w: output,
		loggingIDs: loggingIDs{
			spanID:      "a",
			trace:       "b",
			executionID: "c",
		},
	}

	l := log.New(w, "", 0)
	l.Print("go logger line")
	l.Print("a second log line")
	l.Print("a multiline\nstring in a single log\ncall")

	wantOutput := `{"message":"go logger line","logging.googleapis.com/trace":"b","logging.googleapis.com/spanId":"a","logging.googleapis.com/labels":{"execution_id":"c"}}
{"message":"a second log line","logging.googleapis.com/trace":"b","logging.googleapis.com/spanId":"a","logging.googleapis.com/labels":{"execution_id":"c"}}
{"message":"a multiline","logging.googleapis.com/trace":"b","logging.googleapis.com/spanId":"a","logging.googleapis.com/labels":{"execution_id":"c"}}
{"message":"string in a single log","logging.googleapis.com/trace":"b","logging.googleapis.com/spanId":"a","logging.googleapis.com/labels":{"execution_id":"c"}}
{"message":"call","logging.googleapis.com/trace":"b","logging.googleapis.com/spanId":"a","logging.googleapis.com/labels":{"execution_id":"c"}}
`
	if output.String() != wantOutput {
		t.Errorf("expected output %q got %q", wantOutput, output.String())
	}
}

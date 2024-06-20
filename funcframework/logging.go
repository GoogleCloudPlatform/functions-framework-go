package funcframework

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"sync"
)

var (
	loggingIDsContextKey    contextKey = "loggingIDs"
	validXCloudTraceContext            = regexp.MustCompile(
		// Matches on "TRACE_ID"
		`([a-f\d]+)?` +
			// Matches on "/SPAN_ID"
			`(?:/([a-f\d]+))?` +
			// Matches on ";0=TRACE_TRUE"
			`(?:;o=(\d))?`)
)

type loggingIDs struct {
	trace       string
	spanID      string
	executionID string
}

type contextKey string

func addLoggingIDsToRequest(r *http.Request) *http.Request {
	executionID := r.Header.Get("Function-Execution-Id")
	traceID, spanID, _ := deconstructXCloudTraceContext(r.Header.Get("X-Cloud-Trace-Context"))

	if executionID == "" && traceID == "" && spanID == "" {
		return r
	}

	r = r.WithContext(contextWithLoggingIDs(r.Context(), &loggingIDs{
		trace:       traceID,
		spanID:      spanID,
		executionID: executionID,
	}))

	return r
}

func contextWithLoggingIDs(ctx context.Context, loggingIDs *loggingIDs) context.Context {
	return context.WithValue(ctx, loggingIDsContextKey, loggingIDs)
}

func loggingIDsFromContext(ctx context.Context) *loggingIDs {
	val := ctx.Value(loggingIDsContextKey)
	if val == nil {
		return nil
	}
	return val.(*loggingIDs)
}

func TraceIDFromContext(ctx context.Context) string {
	ids := loggingIDsFromContext(ctx)
	if ids == nil {
		return ""
	}
	return ids.trace
}

func ExecutionIDFromContext(ctx context.Context) string {
	ids := loggingIDsFromContext(ctx)
	if ids == nil {
		return ""
	}
	return ids.executionID
}

func SpanIDFromContext(ctx context.Context) string {
	ids := loggingIDsFromContext(ctx)
	if ids == nil {
		return ""
	}
	return ids.spanID
}

func deconstructXCloudTraceContext(s string) (traceID, spanID string, traceSampled bool) {
	// As per the format described at https://cloud.google.com/trace/docs/setup#force-trace
	//    "X-Cloud-Trace-Context: TRACE_ID/SPAN_ID;o=TRACE_TRUE"
	// for example:
	//    "X-Cloud-Trace-Context: 105445aa7843bc8bf206b120001000/1;o=1"
	matches := validXCloudTraceContext.FindStringSubmatch(s)
	if matches != nil {
		traceID, spanID, traceSampled = matches[1], matches[2], matches[3] == "1"
	}
	if spanID == "0" {
		spanID = ""
	}
	return
}

// structuredLogEvent declares a subset of the fields supported by cloudlogging structured log events.
// See https://cloud.google.com/logging/docs/structured-logging.
type structuredLogEvent struct {
	Message string            `json:"message"`
	Trace   string            `json:"logging.googleapis.com/trace,omitempty"`
	SpanID  string            `json:"logging.googleapis.com/spanId,omitempty"`
	Labels  map[string]string `json:"logging.googleapis.com/labels,omitempty"`
}

// structuredLogWriter writes structured logs
type structuredLogWriter struct {
	mu         sync.Mutex
	w          io.Writer
	loggingIDs loggingIDs
	buf        []byte
}

func (w *structuredLogWriter) writeStructuredLog(loggingIDs loggingIDs, message string) (int, error) {
	event := structuredLogEvent{
		Message: message,
		Trace:   loggingIDs.trace,
		SpanID:  loggingIDs.spanID,
	}
	if loggingIDs.executionID != "" {
		event.Labels = map[string]string{
			"execution_id": loggingIDs.executionID,
		}
	}

	marshalled, err := json.Marshal(event)
	if err != nil {
		return 0, err
	}
	marshalled = append(marshalled, '\n')
	return w.w.Write(marshalled)
}

func (w *structuredLogWriter) Write(output []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, output...)
	buf := w.buf
	wroteLines := 0
	for {
		advance, token, err := bufio.ScanLines(buf, false)
		if token == nil || err != nil {
			break
		}
		buf = buf[advance:]
		if _, err := w.writeStructuredLog(w.loggingIDs, string(token)); err != nil {
			return 0, err
		}
		wroteLines += 1
	}

	if wroteLines > 0 {
		// Compact the buffer by copying remaining bytes to the start.
		w.buf = append(w.buf[:0], buf...)
	}

	return len(output), nil
}

func (w *structuredLogWriter) Close() error {
	if len(w.buf) == 0 {
		return nil
	}
	_, err := w.writeStructuredLog(w.loggingIDs, string(w.buf))
	return err
}

// LogWriter returns an io.Writer as a log sink for the request context.
// One log event is generated for each new line terminated byte sequence
// written to the io.Writer.
//
// This can be used with common logging frameworks, for example:
//
//	import (
//	  "log"
//	  "github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
//	)
//	...
//	func helloWorld(w http.ResponseWriter, r *http.Request) {
//	  l := logger.New(funcframework.LogWriter(r.Context()))
//	  l.Println("hello world!")
//	}
func LogWriter(ctx context.Context) io.WriteCloser {
	loggingIDs := loggingIDsFromContext(ctx)
	if loggingIDs == nil {
		return os.Stderr
	}

	return &structuredLogWriter{
		w:          os.Stderr,
		loggingIDs: *loggingIDs,
	}
}

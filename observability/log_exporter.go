package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

const (
	logFormatJSON = "json"
	logFormatText = "text"
)

type localLogExporter struct {
	mu     sync.Mutex
	stdout io.Writer
	stderr io.Writer
	closer io.Closer
	format string
}

type localLogEntry struct {
	Timestamp         string         `json:"timestamp,omitempty"`
	ObservedTimestamp string         `json:"observed_timestamp,omitempty"`
	Severity          string         `json:"severity,omitempty"`
	SeverityNumber    int            `json:"severity_number,omitempty"`
	EventName         string         `json:"event_name,omitempty"`
	Body              any            `json:"body,omitempty"`
	TraceID           string         `json:"trace_id,omitempty"`
	SpanID            string         `json:"span_id,omitempty"`
	TraceSampled      bool           `json:"trace_sampled,omitempty"`
	Resource          map[string]any `json:"resource,omitempty"`
	Attributes        map[string]any `json:"attributes,omitempty"`
}

func newLocalLogExporter(stdout io.Writer, stderr io.Writer, closer io.Closer, format string) *localLogExporter {
	return &localLogExporter{
		stdout: stdout,
		stderr: stderr,
		closer: closer,
		format: normalizeLogFormat(format),
	}
}

func (e *localLogExporter) Export(ctx context.Context, records []sdklog.Record) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, record := range records {
		writer := e.writerFor(record)
		if writer == nil {
			continue
		}
		line, err := e.formatRecord(record)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(writer, line+"\n"); err != nil {
			return err
		}
	}
	return nil
}

func (e *localLogExporter) Shutdown(context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closer == nil {
		return nil
	}
	return e.closer.Close()
}

func (e *localLogExporter) ForceFlush(context.Context) error {
	return nil
}

func (e *localLogExporter) writerFor(record sdklog.Record) io.Writer {
	if record.Severity() >= otellog.SeverityError && e.stderr != nil {
		return e.stderr
	}
	return e.stdout
}

func (e *localLogExporter) formatRecord(record sdklog.Record) (string, error) {
	if e.format == logFormatText {
		return formatTextLogRecord(record), nil
	}

	entry := localLogEntry{
		Timestamp:         formatLogTime(record.Timestamp()),
		ObservedTimestamp: formatLogTime(record.ObservedTimestamp()),
		Severity:          record.SeverityText(),
		SeverityNumber:    int(record.Severity()),
		EventName:         record.EventName(),
		Body:              logValueToAny(record.Body()),
		Resource:          resourceAttributes(record),
		Attributes:        logRecordAttributes(record),
	}
	if traceID := record.TraceID(); traceID.IsValid() {
		entry.TraceID = traceID.String()
	}
	if spanID := record.SpanID(); spanID.IsValid() {
		entry.SpanID = spanID.String()
	}
	if entry.TraceID != "" || entry.SpanID != "" {
		entry.TraceSampled = record.TraceFlags().IsSampled()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func formatTextLogRecord(record sdklog.Record) string {
	parts := []string{}
	if ts := formatLogTime(record.Timestamp()); ts != "" {
		parts = append(parts, ts)
	}
	if severity := record.SeverityText(); severity != "" {
		parts = append(parts, severity)
	}
	if eventName := record.EventName(); eventName != "" {
		parts = append(parts, eventName)
	}
	if body := strings.TrimSpace(record.Body().String()); body != "" && body != "<nil>" {
		parts = append(parts, body)
	}
	record.WalkAttributes(func(kv otellog.KeyValue) bool {
		parts = append(parts, fmt.Sprintf("%s=%v", kv.Key, logValueToAny(kv.Value)))
		return true
	})
	return strings.Join(parts, " ")
}

func logRecordAttributes(record sdklog.Record) map[string]any {
	attrs := map[string]any{}
	record.WalkAttributes(func(kv otellog.KeyValue) bool {
		attrs[kv.Key] = logValueToAny(kv.Value)
		return true
	})
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

func resourceAttributes(record sdklog.Record) map[string]any {
	res := record.Resource()
	if res == nil {
		return nil
	}
	attrs := res.Attributes()
	if len(attrs) == 0 {
		return nil
	}
	values := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		values[string(attr.Key)] = attributeValueToAny(attr.Value)
	}
	return values
}

func logValueToAny(value otellog.Value) any {
	switch value.Kind() {
	case otellog.KindString:
		return value.AsString()
	case otellog.KindInt64:
		return value.AsInt64()
	case otellog.KindFloat64:
		return value.AsFloat64()
	case otellog.KindBool:
		return value.AsBool()
	case otellog.KindBytes:
		return string(value.AsBytes())
	case otellog.KindSlice:
		values := value.AsSlice()
		result := make([]any, 0, len(values))
		for _, item := range values {
			result = append(result, logValueToAny(item))
		}
		return result
	case otellog.KindMap:
		values := value.AsMap()
		result := make(map[string]any, len(values))
		for _, item := range values {
			result[item.Key] = logValueToAny(item.Value)
		}
		return result
	default:
		return nil
	}
}

func attributeValueToAny(value attribute.Value) any {
	return value.AsInterface()
}

func formatLogTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func normalizeLogFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case logFormatText:
		return logFormatText
	default:
		return logFormatJSON
	}
}

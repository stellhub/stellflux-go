package observability

import (
	"context"
	"time"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

func newRecord(eventName string, message string, status int, duration float64, err error) otellog.Record {
	var record otellog.Record
	now := time.Now()
	record.SetTimestamp(now)
	record.SetObservedTimestamp(now)
	record.SetEventName(eventName)
	record.SetBody(otellog.StringValue(message))
	record.AddAttributes(
		otellog.Int("status.code", status),
		otellog.Float64("duration.s", duration),
	)
	if err != nil || status >= 500 {
		record.SetSeverity(otellog.SeverityError)
		record.SetSeverityText("ERROR")
		if err != nil {
			record.SetErr(err)
			record.AddAttributes(otellog.String("error.message", err.Error()))
		}
		return record
	}
	if status >= 400 {
		record.SetSeverity(otellog.SeverityWarn)
		record.SetSeverityText("WARN")
		return record
	}
	record.SetSeverity(otellog.SeverityInfo)
	record.SetSeverityText("INFO")
	return record
}

func addTraceAttributes(ctx context.Context, record *otellog.Record) {
	spanContext := trace.SpanFromContext(ctx).SpanContext()
	if !spanContext.IsValid() {
		return
	}
	record.AddAttributes(
		otellog.String("trace_id", spanContext.TraceID().String()),
		otellog.String("span_id", spanContext.SpanID().String()),
		otellog.Bool("trace_sampled", spanContext.TraceFlags().IsSampled()),
	)
}

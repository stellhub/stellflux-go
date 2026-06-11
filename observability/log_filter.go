package observability

import (
	"context"
	"strings"

	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type severityFilterProcessor struct {
	min  otellog.Severity
	next sdklog.Processor
}

func newSeverityFilterProcessor(min otellog.Severity, next sdklog.Processor) sdklog.Processor {
	return severityFilterProcessor{min: min, next: next}
}

func (p severityFilterProcessor) OnEmit(ctx context.Context, record *sdklog.Record) error {
	if !p.enabled(record.Severity()) {
		return nil
	}
	return p.next.OnEmit(ctx, record)
}

func (p severityFilterProcessor) Shutdown(ctx context.Context) error {
	return p.next.Shutdown(ctx)
}

func (p severityFilterProcessor) ForceFlush(ctx context.Context) error {
	return p.next.ForceFlush(ctx)
}

func (p severityFilterProcessor) Enabled(_ context.Context, params sdklog.EnabledParameters) bool {
	return p.enabled(params.Severity)
}

func (p severityFilterProcessor) enabled(severity otellog.Severity) bool {
	if severity == 0 {
		return true
	}
	return severity >= p.min
}

func normalizeLogLevel(value string) otellog.Severity {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "trace":
		return otellog.SeverityTrace
	case "debug":
		return otellog.SeverityDebug
	case "", "info":
		return otellog.SeverityInfo
	case "warn", "warning":
		return otellog.SeverityWarn
	case "error":
		return otellog.SeverityError
	case "fatal":
		return otellog.SeverityFatal
	default:
		return otellog.SeverityInfo
	}
}

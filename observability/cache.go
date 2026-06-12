package observability

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	CacheResultOK    = "ok"
	CacheResultHit   = "hit"
	CacheResultMiss  = "miss"
	CacheResultError = "error"
)

type CacheClientRequest struct {
	Adapter   string
	Operation string
}

type CacheClientResult struct {
	Result        string
	SizeBytes     int
	Entries       int64
	CapacityBytes int64
	Err           error
}

func (p *Provider) StartCacheClient(ctx context.Context, request CacheClientRequest) (context.Context, func(CacheClientResult)) {
	if p == nil {
		p = New()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(request.Adapter) == "" {
		request.Adapter = "unknown"
	}
	if strings.TrimSpace(request.Operation) == "" {
		request.Operation = "unknown"
	}

	attrs := append(p.commonAttrs(),
		attribute.String("cache.system.name", request.Adapter),
		attribute.String("cache.operation.name", request.Operation),
	)
	span := trace.SpanFromContext(ctx)
	if p.cacheClientTrace {
		ctx, span = p.Tracer().Start(
			ctx,
			"cache "+request.Operation,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attrs...),
		)
	}
	start := time.Now()

	return ctx, func(result CacheClientResult) {
		duration := durationSeconds(start)
		resultName := normalizeCacheResult(result)
		resultAttrs := append(attrs, attribute.String("cache.result", resultName))
		if result.SizeBytes >= 0 {
			resultAttrs = append(resultAttrs, attribute.Int("cache.value.size", result.SizeBytes))
		}

		if p.cacheClientTrace {
			span.SetAttributes(resultAttrs...)
			if result.Err != nil {
				span.RecordError(result.Err)
				span.SetStatus(codes.Error, result.Err.Error())
			}
			span.End()
		}
		if p.cacheClientMetrics {
			p.recordCacheMetrics(ctx, request, result, resultName, resultAttrs, duration)
		}
		if p.cacheClientLogs {
			p.emitCacheClientLog(ctx, request, result, resultName, duration)
		}
	}
}

func (p *Provider) initCacheMetrics() {
	if p == nil || p.meter == nil {
		return
	}
	requests, err := p.meter.Int64Counter(
		"cache.client.request.count",
		metric.WithDescription("Number of cache client operations."),
	)
	if err == nil {
		p.cacheClientRequests = requests
	}
	duration, err := p.meter.Float64Histogram(
		"cache.client.request.duration",
		metric.WithDescription("Duration of cache client operations."),
		metric.WithUnit("s"),
	)
	if err == nil {
		p.cacheClientDuration = duration
	}
	valueSize, err := p.meter.Int64Histogram(
		"cache.client.value.size",
		metric.WithDescription("Cache value size observed by client operations."),
		metric.WithUnit("By"),
	)
	if err == nil {
		p.cacheClientValueSize = valueSize
	}
	entries, err := p.meter.Int64Gauge(
		"cache.client.entries",
		metric.WithDescription("Current cache entry count observed after client operations."),
		metric.WithUnit("{entry}"),
	)
	if err == nil {
		p.cacheClientEntries = entries
	}
	capacity, err := p.meter.Int64Gauge(
		"cache.client.capacity",
		metric.WithDescription("Current cache capacity observed after client operations."),
		metric.WithUnit("By"),
	)
	if err == nil {
		p.cacheClientCapacity = capacity
	}
}

func (p *Provider) recordCacheMetrics(
	ctx context.Context,
	request CacheClientRequest,
	result CacheClientResult,
	resultName string,
	attrs []attribute.KeyValue,
	duration float64,
) {
	if p.cacheClientRequests != nil {
		p.cacheClientRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if p.cacheClientDuration != nil {
		p.cacheClientDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
	}
	if p.cacheClientValueSize != nil && result.SizeBytes >= 0 {
		p.cacheClientValueSize.Record(ctx, int64(result.SizeBytes), metric.WithAttributes(attrs...))
	}

	gaugeAttrs := append(p.commonAttrs(),
		attribute.String("cache.system.name", request.Adapter),
		attribute.String("cache.result", resultName),
	)
	if p.cacheClientEntries != nil && result.Entries >= 0 {
		p.cacheClientEntries.Record(ctx, result.Entries, metric.WithAttributes(gaugeAttrs...))
	}
	if p.cacheClientCapacity != nil && result.CapacityBytes > 0 {
		p.cacheClientCapacity.Record(ctx, result.CapacityBytes, metric.WithAttributes(gaugeAttrs...))
	}
}

func (p *Provider) emitCacheClientLog(ctx context.Context, request CacheClientRequest, result CacheClientResult, resultName string, duration float64) {
	record := newRecord("cache.client.request", "Cache client request completed", statusFromError(result.Err), duration, result.Err)
	record.AddAttributes(
		otellog.String("cache.system.name", request.Adapter),
		otellog.String("cache.operation.name", request.Operation),
		otellog.String("cache.result", resultName),
		otellog.Int64("cache.entries", result.Entries),
	)
	if result.SizeBytes >= 0 {
		record.AddAttributes(otellog.Int("cache.value.size", result.SizeBytes))
	}
	if result.CapacityBytes > 0 {
		record.AddAttributes(otellog.Int64("cache.capacity", result.CapacityBytes))
	}
	addTraceAttributes(ctx, &record)
	p.cacheClientLogger.Emit(ctx, record)
}

func normalizeCacheResult(result CacheClientResult) string {
	if result.Err != nil {
		return CacheResultError
	}
	switch result.Result {
	case CacheResultOK, CacheResultHit, CacheResultMiss:
		return result.Result
	default:
		return CacheResultOK
	}
}

package observability

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

type HTTPServerRequest struct {
	Method string
	Route  string
	Path   string
	Header http.Header
}

type HTTPServerResult struct {
	StatusCode int
	Err        error
}

func (p *Provider) StartHTTPServer(ctx context.Context, request HTTPServerRequest) (context.Context, trace.Span, func(HTTPServerResult)) {
	if p == nil {
		p = New()
	}
	ctx = p.ExtractHTTP(ctx, request.Header)
	attrs := append(p.commonAttrs(),
		attribute.String("http.request.method", request.Method),
		attribute.String("http.route", request.Route),
		attribute.String("url.path", request.Path),
	)
	span := trace.SpanFromContext(ctx)
	if p.httpServerTrace {
		ctx, span = p.Tracer().Start(
			ctx,
			"HTTP "+request.Method+" "+request.Route,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attrs...),
		)
	}
	start := time.Now()

	return ctx, span, func(result HTTPServerResult) {
		status := result.StatusCode
		if status == 0 {
			status = http.StatusOK
		}
		resultAttrs := append(attrs, attribute.Int("http.response.status_code", status))
		if p.httpServerTrace {
			span.SetAttributes(resultAttrs...)
			if result.Err != nil || status >= http.StatusInternalServerError {
				if result.Err != nil {
					span.RecordError(result.Err)
				}
				span.SetStatus(codes.Error, http.StatusText(status))
			}
		}

		if p.httpServerMetrics && p.httpServerRequests != nil {
			p.httpServerRequests.Add(ctx, 1, metric.WithAttributes(resultAttrs...))
		}
		if p.httpServerMetrics && p.httpServerDuration != nil {
			p.httpServerDuration.Record(ctx, durationSeconds(start), metric.WithAttributes(resultAttrs...))
		}
		if p.httpServerLogs {
			p.emitHTTPServerLog(ctx, request, status, durationSeconds(start), result.Err)
		}
		if p.httpServerTrace {
			span.End()
		}
	}
}

func (p *Provider) HTTPClientTransport(base http.RoundTripper) http.RoundTripper {
	if p == nil {
		p = New()
	}
	if base == nil {
		base = http.DefaultTransport
	}
	if !p.httpClientTrace && !p.httpClientMetrics && !p.httpClientLogsEnabled {
		return base
	}

	wrapped := base
	if p.httpClientLogsEnabled {
		wrapped = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			start := time.Now()
			resp, err := base.RoundTrip(req)
			status := 0
			if resp != nil {
				status = resp.StatusCode
			}
			p.emitHTTPClientLog(req.Context(), req, status, durationSeconds(start), err)
			return resp, err
		})
	}
	tracerProvider := p.TracerProvider()
	if !p.httpClientTrace {
		tracerProvider = tracenoop.NewTracerProvider()
	}
	meterProvider := p.MeterProvider()
	if !p.httpClientMetrics {
		meterProvider = metricnoop.NewMeterProvider()
	}
	return otelhttp.NewTransport(
		wrapped,
		otelhttp.WithTracerProvider(tracerProvider),
		otelhttp.WithMeterProvider(meterProvider),
		otelhttp.WithPropagators(p.Propagator()),
	)
}

func (p *Provider) emitHTTPServerLog(ctx context.Context, request HTTPServerRequest, status int, duration float64, err error) {
	record := newRecord("http.server.request", "HTTP server request completed", status, duration, err)
	record.AddAttributes(
		otellog.String("http.request.method", request.Method),
		otellog.String("http.route", request.Route),
		otellog.String("url.path", request.Path),
		otellog.Int("http.response.status_code", status),
	)
	addTraceAttributes(ctx, &record)
	p.Logger().Emit(ctx, record)
}

func (p *Provider) emitHTTPClientLog(ctx context.Context, request *http.Request, status int, duration float64, err error) {
	record := newRecord("http.client.request", "HTTP client request completed", status, duration, err)
	record.AddAttributes(
		otellog.String("http.request.method", request.Method),
		otellog.String("url.full", request.URL.String()),
		otellog.Int("http.response.status_code", status),
	)
	addTraceAttributes(ctx, &record)
	p.httpClientLogger.Emit(ctx, record)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

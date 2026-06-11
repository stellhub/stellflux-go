package observability

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	stellarconfig "github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/internal/rollingfile"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	DefaultOTLPEndpoint = "localhost:4317"
	outputStdout        = "stdout"
	outputStderr        = "stderr"
	outputConsole       = "console"
	outputFile          = "file"
	outputOTLP          = "otlp"
	outputPrometheus    = "prometheus"
	outputNone          = "none"
)

func NewFromConfig(ctx context.Context, cfg stellarconfig.Config) (*Provider, error) {
	otelCfg := cfg.Starter.OpenTelemetry
	if otelCfg == nil {
		otelCfg = &stellarconfig.OpenTelemetryStarterConfig{}
	}

	res, err := buildResource(cfg)
	if err != nil {
		return nil, err
	}

	providerCfg := []Option{
		WithServiceName(cfg.AppName),
		WithPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})),
	}
	if cfg.HTTP.Server != nil {
		signals := cfg.HTTP.Server.Observability
		providerCfg = append(providerCfg, WithHTTPServerObservability(signals.Trace, signals.Metrics, signals.Logs))
	}
	if cfg.HTTP.Client != nil {
		signals := cfg.HTTP.Client.Observability
		providerCfg = append(providerCfg, WithHTTPClientObservability(signals.Trace, signals.Metrics, signals.Logs))
	}
	var shutdowns []func(context.Context) error
	var metricsHandler http.Handler

	if otelCfg.Trace {
		traceProvider, shutdown, err := buildTraceProvider(ctx, cfg, res)
		if err != nil {
			return nil, err
		}
		providerCfg = append(providerCfg, WithTracerProvider(traceProvider))
		shutdowns = append(shutdowns, shutdown)
	}

	if otelCfg.Metrics {
		meterProvider, handler, shutdown, err := buildMeterProvider(ctx, cfg, res)
		if err != nil {
			return nil, err
		}
		providerCfg = append(providerCfg, WithMeterProvider(meterProvider))
		if handler != nil {
			metricsHandler = handler
		}
		shutdowns = append(shutdowns, shutdown)
	}

	loggerProvider, shutdown, err := buildLoggerProvider(ctx, cfg, res)
	if err != nil {
		return nil, err
	}
	providerCfg = append(providerCfg, WithLoggerProvider(loggerProvider))
	shutdowns = append(shutdowns, shutdown)

	provider := New(providerCfg...)
	provider.shutdowns = shutdowns
	if metricsHandler != nil {
		provider.metricsHandler = metricsHandler
	}
	return provider, nil
}

func buildTraceProvider(ctx context.Context, cfg stellarconfig.Config, res *resource.Resource) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	otelCfg := cfg.Starter.OpenTelemetry
	output := normalizeOutput(otelCfg.TraceOutput, outputNone)
	if output != outputOTLP {
		traceProvider := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
		return traceProvider, traceProvider.Shutdown, nil
	}

	exporter, err := otlptracegrpc.New(ctx, traceOptions(endpointFor(otelCfg.TraceEndpoint, otelCfg.Endpoint), insecure(otelCfg.Insecure))...)
	if err != nil {
		return nil, nil, err
	}
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)
	return traceProvider, traceProvider.Shutdown, nil
}

func buildMeterProvider(ctx context.Context, cfg stellarconfig.Config, res *resource.Resource) (*metric.MeterProvider, http.Handler, func(context.Context) error, error) {
	otelCfg := cfg.Starter.OpenTelemetry
	output := normalizeOutput(otelCfg.MetricsOutput, outputPrometheus)
	if output == outputOTLP {
		exporter, err := otlpmetricgrpc.New(ctx, metricOptions(endpointFor(otelCfg.MetricsEndpoint, otelCfg.Endpoint), insecure(otelCfg.Insecure))...)
		if err != nil {
			return nil, nil, nil, err
		}
		reader := metric.NewPeriodicReader(exporter)
		meterProvider := metric.NewMeterProvider(metric.WithResource(res), metric.WithReader(reader))
		return meterProvider, nil, meterProvider.Shutdown, nil
	}

	exporter, err := prometheus.New(prometheus.WithResourceAsConstantLabels(func(attribute.KeyValue) bool {
		return true
	}))
	if err != nil {
		return nil, nil, nil, err
	}
	meterProvider := metric.NewMeterProvider(metric.WithResource(res), metric.WithReader(exporter))
	return meterProvider, promhttp.Handler(), meterProvider.Shutdown, nil
}

func buildLoggerProvider(ctx context.Context, cfg stellarconfig.Config, res *resource.Resource) (otellog.LoggerProvider, func(context.Context) error, error) {
	otelCfg := cfg.Starter.OpenTelemetry
	if otelCfg == nil {
		otelCfg = &stellarconfig.OpenTelemetryStarterConfig{}
	}
	logCfg := otelCfg.Log
	output := normalizeLogOutput(logCfg)
	level := normalizeLogLevel(logCfg.Level)

	var exporter sdklog.Exporter
	var err error
	if output == outputOTLP {
		exporter, err = otlploggrpc.New(ctx, logOptions(endpointFor(logCfg.Endpoint, otelCfg.Endpoint), insecure(otelCfg.Insecure))...)
		if err != nil {
			return nil, nil, err
		}
	} else {
		exporter, err = buildLocalLogExporter(logCfg, output)
		if err != nil {
			return nil, nil, err
		}
	}

	processor := sdklog.Processor(sdklog.NewSimpleProcessor(exporter))
	if output == outputOTLP {
		processor = sdklog.NewBatchProcessor(exporter)
	}
	if level > 0 {
		processor = newSeverityFilterProcessor(level, processor)
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(processor),
	)
	return provider, provider.Shutdown, nil
}

func buildLocalLogExporter(logCfg stellarconfig.OpenTelemetryLogConfig, output string) (sdklog.Exporter, error) {
	format := normalizeLogFormat(logCfg.Format)
	if output == outputFile {
		writer, err := rollingfile.New(logCfg.Dir, logCfg.FileName, logCfg.MaxSizeBytes, logCfg.MaxBackups)
		if err != nil {
			return nil, err
		}
		return newLocalLogExporter(writer, writer, writer, format), nil
	}

	switch output {
	case outputStderr:
		return newLocalLogExporter(os.Stderr, os.Stderr, nil, format), nil
	case outputStdout:
		return newLocalLogExporter(os.Stdout, os.Stdout, nil, format), nil
	case outputConsole, "":
		return newLocalLogExporter(os.Stdout, os.Stderr, nil, format), nil
	default:
		return newLocalLogExporter(os.Stdout, os.Stderr, nil, format), nil
	}
}

func buildResource(cfg stellarconfig.Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		attribute.String("service.name", cfg.AppName),
		attribute.String("deployment.environment.name", string(cfg.Environment)),
	}
	if cfg.Version != "" {
		attrs = append(attrs, attribute.String("service.version", cfg.Version))
	}
	if cfg.Zone != "" {
		attrs = append(attrs, attribute.String("cloud.availability_zone", cfg.Zone))
	}
	for key, value := range cfg.Metadata {
		if strings.TrimSpace(key) == "" {
			continue
		}
		attrs = append(attrs, attribute.String(key, value))
	}
	return resource.Merge(resource.Default(), resource.NewSchemaless(attrs...))
}

func traceOptions(endpoint string, insecure bool) []otlptracegrpc.Option {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithTimeout(3 * time.Second),
	}
	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	return opts
}

func metricOptions(endpoint string, insecure bool) []otlpmetricgrpc.Option {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithTimeout(3 * time.Second),
	}
	if insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	return opts
}

func logOptions(endpoint string, insecure bool) []otlploggrpc.Option {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithTimeout(3 * time.Second),
	}
	if insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	return opts
}

func endpointFor(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return DefaultOTLPEndpoint
}

func insecure(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func normalizeOutput(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeLogOutput(logCfg stellarconfig.OpenTelemetryLogConfig) string {
	if logCfg.Enabled {
		return outputOTLP
	}
	output := normalizeOutput(logCfg.Output, outputConsole)
	if output == outputOTLP {
		return outputConsole
	}
	return output
}

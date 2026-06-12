package observability

import (
	"context"
	stderrors "errors"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	logglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const ScopeName = "github.com/stellhub/stellar"

type Provider struct {
	serviceName             string
	tracerProvider          trace.TracerProvider
	meterProvider           metric.MeterProvider
	loggerProvider          log.LoggerProvider
	tracer                  trace.Tracer
	meter                   metric.Meter
	logger                  log.Logger
	propagator              propagation.TextMapPropagator
	httpServerTrace         bool
	httpServerMetrics       bool
	httpServerLogs          bool
	httpClientTrace         bool
	httpClientMetrics       bool
	httpClientLogsEnabled   bool
	redisClientTrace        bool
	redisClientMetrics      bool
	redisClientLogs         bool
	mysqlClientTrace        bool
	mysqlClientMetrics      bool
	mysqlClientLogs         bool
	postgresqlClientTrace   bool
	postgresqlClientMetrics bool
	postgresqlClientLogs    bool
	cacheClientTrace        bool
	cacheClientMetrics      bool
	cacheClientLogs         bool

	httpServerRequests     metric.Int64Counter
	httpServerDuration     metric.Float64Histogram
	cacheClientRequests    metric.Int64Counter
	cacheClientDuration    metric.Float64Histogram
	cacheClientValueSize   metric.Int64Histogram
	cacheClientEntries     metric.Int64Gauge
	cacheClientCapacity    metric.Int64Gauge
	httpClientLogger       log.Logger
	redisClientLogger      log.Logger
	mysqlClientLogger      log.Logger
	postgresqlClientLogger log.Logger
	cacheClientLogger      log.Logger
	metricsHandler         http.Handler
	shutdowns              []func(context.Context) error
}

type Option func(*providerConfig)

type providerConfig struct {
	serviceName             string
	tracerProvider          trace.TracerProvider
	meterProvider           metric.MeterProvider
	loggerProvider          log.LoggerProvider
	propagator              propagation.TextMapPropagator
	httpServerTrace         *bool
	httpServerMetrics       *bool
	httpServerLogs          *bool
	httpClientTrace         *bool
	httpClientMetrics       *bool
	httpClientLogs          *bool
	redisClientTrace        *bool
	redisClientMetrics      *bool
	redisClientLogs         *bool
	mysqlClientTrace        *bool
	mysqlClientMetrics      *bool
	mysqlClientLogs         *bool
	postgresqlClientTrace   *bool
	postgresqlClientMetrics *bool
	postgresqlClientLogs    *bool
	cacheClientTrace        *bool
	cacheClientMetrics      *bool
	cacheClientLogs         *bool
}

func New(options ...Option) *Provider {
	cfg := providerConfig{
		tracerProvider: otel.GetTracerProvider(),
		meterProvider:  otel.GetMeterProvider(),
		loggerProvider: logglobal.GetLoggerProvider(),
		propagator:     otel.GetTextMapPropagator(),
	}
	for _, option := range options {
		option(&cfg)
	}
	if cfg.propagator == nil {
		cfg.propagator = propagation.TraceContext{}
	}

	provider := &Provider{
		serviceName:             cfg.serviceName,
		tracerProvider:          cfg.tracerProvider,
		meterProvider:           cfg.meterProvider,
		loggerProvider:          cfg.loggerProvider,
		tracer:                  cfg.tracerProvider.Tracer(ScopeName),
		meter:                   cfg.meterProvider.Meter(ScopeName),
		logger:                  cfg.loggerProvider.Logger(ScopeName),
		httpServerTrace:         boolValue(cfg.httpServerTrace, true),
		httpServerMetrics:       boolValue(cfg.httpServerMetrics, true),
		httpServerLogs:          boolValue(cfg.httpServerLogs, true),
		httpClientTrace:         boolValue(cfg.httpClientTrace, true),
		httpClientMetrics:       boolValue(cfg.httpClientMetrics, true),
		httpClientLogsEnabled:   boolValue(cfg.httpClientLogs, true),
		redisClientTrace:        boolValue(cfg.redisClientTrace, true),
		redisClientMetrics:      boolValue(cfg.redisClientMetrics, true),
		redisClientLogs:         boolValue(cfg.redisClientLogs, true),
		mysqlClientTrace:        boolValue(cfg.mysqlClientTrace, true),
		mysqlClientMetrics:      boolValue(cfg.mysqlClientMetrics, true),
		mysqlClientLogs:         boolValue(cfg.mysqlClientLogs, true),
		postgresqlClientTrace:   boolValue(cfg.postgresqlClientTrace, true),
		postgresqlClientMetrics: boolValue(cfg.postgresqlClientMetrics, true),
		postgresqlClientLogs:    boolValue(cfg.postgresqlClientLogs, true),
		cacheClientTrace:        boolValue(cfg.cacheClientTrace, true),
		cacheClientMetrics:      boolValue(cfg.cacheClientMetrics, true),
		cacheClientLogs:         boolValue(cfg.cacheClientLogs, true),
		httpClientLogger: cfg.loggerProvider.Logger(
			ScopeName + "/http-client",
		),
		redisClientLogger: cfg.loggerProvider.Logger(
			ScopeName + "/redis-client",
		),
		mysqlClientLogger: cfg.loggerProvider.Logger(
			ScopeName + "/mysql-client",
		),
		postgresqlClientLogger: cfg.loggerProvider.Logger(
			ScopeName + "/postgresql-client",
		),
		cacheClientLogger: cfg.loggerProvider.Logger(
			ScopeName + "/cache-client",
		),
		propagator: cfg.propagator,
	}
	provider.initHTTPMetrics()
	provider.initCacheMetrics()
	return provider
}

func WithServiceName(serviceName string) Option {
	return func(cfg *providerConfig) {
		cfg.serviceName = serviceName
	}
}

func WithTracerProvider(provider trace.TracerProvider) Option {
	return func(cfg *providerConfig) {
		if provider != nil {
			cfg.tracerProvider = provider
		}
	}
}

func WithMeterProvider(provider metric.MeterProvider) Option {
	return func(cfg *providerConfig) {
		if provider != nil {
			cfg.meterProvider = provider
		}
	}
}

func WithLoggerProvider(provider log.LoggerProvider) Option {
	return func(cfg *providerConfig) {
		if provider != nil {
			cfg.loggerProvider = provider
		}
	}
}

func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return func(cfg *providerConfig) {
		if propagator != nil {
			cfg.propagator = propagator
		}
	}
}

func WithHTTPServerObservability(trace *bool, metrics *bool, logs *bool) Option {
	return func(cfg *providerConfig) {
		cfg.httpServerTrace = trace
		cfg.httpServerMetrics = metrics
		cfg.httpServerLogs = logs
	}
}

func WithHTTPClientObservability(trace *bool, metrics *bool, logs *bool) Option {
	return func(cfg *providerConfig) {
		cfg.httpClientTrace = trace
		cfg.httpClientMetrics = metrics
		cfg.httpClientLogs = logs
	}
}

func WithRedisClientObservability(trace *bool, metrics *bool, logs *bool) Option {
	return func(cfg *providerConfig) {
		cfg.redisClientTrace = trace
		cfg.redisClientMetrics = metrics
		cfg.redisClientLogs = logs
	}
}

func WithMySQLClientObservability(trace *bool, metrics *bool, logs *bool) Option {
	return func(cfg *providerConfig) {
		cfg.mysqlClientTrace = trace
		cfg.mysqlClientMetrics = metrics
		cfg.mysqlClientLogs = logs
	}
}

func WithPostgreSQLClientObservability(trace *bool, metrics *bool, logs *bool) Option {
	return func(cfg *providerConfig) {
		cfg.postgresqlClientTrace = trace
		cfg.postgresqlClientMetrics = metrics
		cfg.postgresqlClientLogs = logs
	}
}

func WithCacheClientObservability(trace *bool, metrics *bool, logs *bool) Option {
	return func(cfg *providerConfig) {
		cfg.cacheClientTrace = trace
		cfg.cacheClientMetrics = metrics
		cfg.cacheClientLogs = logs
	}
}

func (p *Provider) ServiceName() string {
	if p == nil {
		return ""
	}
	return p.serviceName
}

func (p *Provider) Tracer() trace.Tracer {
	if p == nil {
		return New().Tracer()
	}
	return p.tracer
}

func (p *Provider) TracerProvider() trace.TracerProvider {
	if p == nil || p.tracerProvider == nil {
		return otel.GetTracerProvider()
	}
	return p.tracerProvider
}

func (p *Provider) Meter() metric.Meter {
	if p == nil {
		return New().Meter()
	}
	return p.meter
}

func (p *Provider) MeterProvider() metric.MeterProvider {
	if p == nil || p.meterProvider == nil {
		return otel.GetMeterProvider()
	}
	return p.meterProvider
}

func (p *Provider) Logger() log.Logger {
	if p == nil {
		return New().Logger()
	}
	return p.logger
}

func (p *Provider) LoggerProvider() log.LoggerProvider {
	if p == nil || p.loggerProvider == nil {
		return logglobal.GetLoggerProvider()
	}
	return p.loggerProvider
}

func (p *Provider) Propagator() propagation.TextMapPropagator {
	if p == nil || p.propagator == nil {
		return propagation.TraceContext{}
	}
	return p.propagator
}

func (p *Provider) RedisClientTraceEnabled() bool {
	if p == nil {
		return true
	}
	return p.redisClientTrace
}

func (p *Provider) RedisClientMetricsEnabled() bool {
	if p == nil {
		return true
	}
	return p.redisClientMetrics
}

func (p *Provider) RedisClientLogsEnabled() bool {
	if p == nil {
		return true
	}
	return p.redisClientLogs
}

func (p *Provider) MySQLClientTraceEnabled() bool {
	if p == nil {
		return true
	}
	return p.mysqlClientTrace
}

func (p *Provider) MySQLClientMetricsEnabled() bool {
	if p == nil {
		return true
	}
	return p.mysqlClientMetrics
}

func (p *Provider) MySQLClientLogsEnabled() bool {
	if p == nil {
		return true
	}
	return p.mysqlClientLogs
}

func (p *Provider) PostgreSQLClientTraceEnabled() bool {
	if p == nil {
		return true
	}
	return p.postgresqlClientTrace
}

func (p *Provider) PostgreSQLClientMetricsEnabled() bool {
	if p == nil {
		return true
	}
	return p.postgresqlClientMetrics
}

func (p *Provider) PostgreSQLClientLogsEnabled() bool {
	if p == nil {
		return true
	}
	return p.postgresqlClientLogs
}

func (p *Provider) CacheClientTraceEnabled() bool {
	if p == nil {
		return true
	}
	return p.cacheClientTrace
}

func (p *Provider) CacheClientMetricsEnabled() bool {
	if p == nil {
		return true
	}
	return p.cacheClientMetrics
}

func (p *Provider) CacheClientLogsEnabled() bool {
	if p == nil {
		return true
	}
	return p.cacheClientLogs
}

func (p *Provider) MetricsHandler() http.Handler {
	if p == nil {
		return nil
	}
	return p.metricsHandler
}

func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	var joined error
	for i := len(p.shutdowns) - 1; i >= 0; i-- {
		if err := p.shutdowns[i](ctx); err != nil {
			joined = stderrors.Join(joined, err)
		}
	}
	return joined
}

func (p *Provider) ExtractHTTP(ctx context.Context, header http.Header) context.Context {
	if p == nil {
		return ctx
	}
	return p.Propagator().Extract(ctx, propagation.HeaderCarrier(header))
}

func (p *Provider) InjectHTTP(ctx context.Context, header http.Header) {
	if p == nil {
		return
	}
	p.Propagator().Inject(ctx, propagation.HeaderCarrier(header))
}

func (p *Provider) initHTTPMetrics() {
	if p == nil || p.meter == nil {
		return
	}
	requests, err := p.meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Number of inbound HTTP server requests."),
	)
	if err == nil {
		p.httpServerRequests = requests
	}
	duration, err := p.meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of inbound HTTP server requests."),
		metric.WithUnit("s"),
	)
	if err == nil {
		p.httpServerDuration = duration
	}
}

func (p *Provider) commonAttrs() []attribute.KeyValue {
	if p == nil || p.serviceName == "" {
		return nil
	}
	return []attribute.KeyValue{attribute.String("service.name", p.serviceName)}
}

func durationSeconds(start time.Time) float64 {
	return time.Since(start).Seconds()
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

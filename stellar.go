package stellar

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stellhub/stellar/boot"
	mysqlclient "github.com/stellhub/stellar/clients/mysql"
	redisclient "github.com/stellhub/stellar/clients/redis"
	"github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/lifecycle"
	"github.com/stellhub/stellar/observability"
	transportgrpc "github.com/stellhub/stellar/transport/grpc"
	transporthttp "github.com/stellhub/stellar/transport/http"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type App = boot.App

type Config = config.Config

type Environment = config.Environment

type HTTPConfig = config.HTTPConfig

type HTTPServerConfig = config.HTTPServerConfig

type HTTPClientConfig = config.HTTPClientConfig

type HTTPNamedClientConfig = config.HTTPNamedClientConfig

type ObservabilitySignalConfig = config.ObservabilitySignalConfig

type GRPCConfig = config.GRPCConfig

type GRPCServerConfig = config.GRPCServerConfig

type GRPCClientConfig = config.GRPCClientConfig

type GRPCNamedClientConfig = config.GRPCNamedClientConfig

type RedisConfig = config.RedisConfig

type MySQLConfig = config.MySQLConfig

type DebugAPIConfig = config.DebugAPIConfig

type RedisClient = goredis.Client

type MySQLDB = mysqlclient.DB

type Runtime = boot.Runtime

type Module = boot.Module

type Starter = boot.Starter

type StarterContext = boot.StarterContext

type Transport = boot.Transport

type HTTPAdapter = transporthttp.Adapter

type RPCAdapter = transportgrpc.Adapter

type Option = boot.Option

type Registry = boot.Registry

type Hook = lifecycle.Hook

type ObservabilityProvider = observability.Provider

type ObservabilityOption = observability.Option

type HealthStatus = boot.HealthStatus

type HealthCheck = boot.HealthCheck

type HealthReport = boot.HealthReport

type StatusResponse = boot.StatusResponse

type HealthResponse = boot.HealthResponse

type MiddlewareStandard = boot.MiddlewareStandard

type StandardModule = boot.StandardModule

const (
	EnvDev  = config.EnvDev
	EnvUAT  = config.EnvUAT
	EnvPre  = config.EnvPre
	EnvProd = config.EnvProd
)

const (
	HealthStatusUp      = boot.HealthStatusUp
	HealthStatusDown    = boot.HealthStatusDown
	HealthStatusSkipped = boot.HealthStatusSkipped
)

var ErrAppNameRequired = boot.ErrAppNameRequired

const (
	RedisClientName = redisclient.DefaultClientName
	MySQLDBName     = mysqlclient.DefaultDBName
)

func New(cfg Config, options ...Option) *App {
	return boot.New(cfg, options...)
}

func Start(options ...Option) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return StartContext(ctx, options...)
}

func StartContext(ctx context.Context, options ...Option) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	app, err := boot.NewConfigured(ctx, cfg, options...)
	if err != nil {
		return err
	}
	return app.Run(ctx)
}

func FromEnv(base Config) Config {
	return config.FromEnv(base)
}

func GetAs[T any](registry *Registry, name string) (T, bool) {
	return boot.GetAs[T](registry, name)
}

func NewObservability(options ...ObservabilityOption) *ObservabilityProvider {
	return observability.New(options...)
}

func WithObservability(provider *ObservabilityProvider) Option {
	return boot.WithObservability(provider)
}

func WithObservabilityServiceName(serviceName string) ObservabilityOption {
	return observability.WithServiceName(serviceName)
}

func WithObservabilityTracerProvider(provider trace.TracerProvider) ObservabilityOption {
	return observability.WithTracerProvider(provider)
}

func WithObservabilityMeterProvider(provider metric.MeterProvider) ObservabilityOption {
	return observability.WithMeterProvider(provider)
}

func WithObservabilityLoggerProvider(provider otellog.LoggerProvider) ObservabilityOption {
	return observability.WithLoggerProvider(provider)
}

func WithObservabilityPropagator(propagator propagation.TextMapPropagator) ObservabilityOption {
	return observability.WithPropagator(propagator)
}

func WithLogger(logger *slog.Logger) Option {
	return boot.WithLogger(logger)
}

func WithEnvConfig() Option {
	return boot.WithEnvConfig()
}

func WithModule(modules ...Module) Option {
	return boot.WithModule(modules...)
}

func WithStarter(starters ...Starter) Option {
	return boot.WithStarter(starters...)
}

func WithTransport(transports ...Transport) Option {
	return boot.WithTransport(transports...)
}

func WithHTTPAdapter(adapter HTTPAdapter) Option {
	return boot.WithHTTPAdapter(adapter)
}

func WithHook(hooks ...lifecycle.Hook) Option {
	return boot.WithHook(hooks...)
}

func WithHTTPServer(addr string) Option {
	return boot.WithHTTPServer(addr)
}

func WithRPCAdapter(adapter RPCAdapter) Option {
	return boot.WithRPCAdapter(adapter)
}

func WithRPCServer(addr string) Option {
	return boot.WithRPCServer(addr)
}

func NewStandardModule(standard MiddlewareStandard) *StandardModule {
	return boot.NewStandardModule(standard)
}

func StandardModules() []Module {
	return boot.StandardModules()
}

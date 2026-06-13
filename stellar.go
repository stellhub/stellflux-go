package stellar

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stellhub/stellar/boot"
	cacheclient "github.com/stellhub/stellar/clients/cache"
	mysqlclient "github.com/stellhub/stellar/clients/mysql"
	postgresqlclient "github.com/stellhub/stellar/clients/postgresql"
	redisclient "github.com/stellhub/stellar/clients/redis"
	"github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/discovery"
	"github.com/stellhub/stellar/governance"
	"github.com/stellhub/stellar/interceptor"
	"github.com/stellhub/stellar/lifecycle"
	"github.com/stellhub/stellar/observability"
	serviceregistry "github.com/stellhub/stellar/registry"
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

type PostgreSQLConfig = config.PostgreSQLConfig

type CacheConfig = config.CacheConfig

type RegistryConfig = config.RegistryConfig

type RegistryServiceEndpointConfig = config.RegistryServiceEndpointConfig

type DiscoveryConfig = config.DiscoveryConfig

type DebugAPIConfig = config.DebugAPIConfig

type RedisClient = goredis.Client

type MySQLDB = mysqlclient.DB

type PostgreSQLDB = postgresqlclient.DB

type Cache = cacheclient.Cache

type CacheAdapter = cacheclient.Adapter

type ServiceRegistry = serviceregistry.Registry

type ServiceRegistryAdapter = serviceregistry.Adapter

type ServiceInstance = serviceregistry.Instance

type ServiceEndpoint = serviceregistry.Endpoint

type ServiceQuery = serviceregistry.Query

type ServiceRegistryEvent = serviceregistry.Event

type ServiceRegistryWatcher = serviceregistry.Watcher

type DiscoveryResolver = discovery.Resolver

type DiscoveryCachedResolver = discovery.CachedResolver

type DiscoveryTarget = discovery.Target

type DiscoveryEndpoint = discovery.Endpoint

type DiscoveryPicker = discovery.Picker

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

type Interceptor = interceptor.Interceptor

type InterceptorFunc = interceptor.Func

type InterceptorHandler = interceptor.Handler

type InterceptorDefinition = interceptor.Definition

type InterceptorInvocation = interceptor.Invocation

type InterceptorKind = interceptor.Kind

type InterceptorStage = interceptor.Stage

type GovernanceStore = governance.Store

type GovernanceSnapshot = governance.Snapshot

type GovernanceRule = governance.Rule

type GovernanceRuleKind = governance.RuleKind

type GovernanceScope = governance.Scope

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

const (
	InterceptorHTTPServer = interceptor.KindHTTPServer
	InterceptorHTTPClient = interceptor.KindHTTPClient
	InterceptorGRPCServer = interceptor.KindGRPCServer
	InterceptorGRPCClient = interceptor.KindGRPCClient
)

const (
	InterceptorStageRecovery       = interceptor.StageRecovery
	InterceptorStageRouteResolve   = interceptor.StageRouteResolve
	InterceptorStageObserve        = interceptor.StageObserve
	InterceptorStageDeadline       = interceptor.StageDeadline
	InterceptorStageAdmission      = interceptor.StageAdmission
	InterceptorStageSecurity       = interceptor.StageSecurity
	InterceptorStageDecodeValidate = interceptor.StageDecodeValidate
	InterceptorStageRetry          = interceptor.StageRetry
	InterceptorStageBusiness       = interceptor.StageBusiness
)

const (
	GovernanceRuleRoute            = governance.RuleKindRoute
	GovernanceRuleRateLimit        = governance.RuleKindRateLimit
	GovernanceRuleCircuitBreaker   = governance.RuleKindCircuitBreaker
	GovernanceRuleLoadShedding     = governance.RuleKindLoadShedding
	GovernanceRuleConcurrencyLimit = governance.RuleKindConcurrencyLimit
	GovernanceRuleAuthentication   = governance.RuleKindAuthentication
	GovernanceRuleAuthorization    = governance.RuleKindAuthorization
	GovernanceRuleRetry            = governance.RuleKindRetry
	GovernanceRuleSigning          = governance.RuleKindSigning
	GovernanceRuleQuota            = governance.RuleKindQuota
)

var ErrAppNameRequired = boot.ErrAppNameRequired

const (
	RedisClientName             = redisclient.DefaultClientName
	MySQLDBName                 = mysqlclient.DefaultDBName
	PostgreSQLDBName            = postgresqlclient.DefaultDBName
	CacheName                   = cacheclient.DefaultName
	CacheBigCache               = cacheclient.AdapterBigCache
	CacheFreeCache              = cacheclient.AdapterFreeCache
	RegistryName                = serviceregistry.DefaultName
	RegistryEtcd                = serviceregistry.AdapterEtcd
	RegistryConsul              = serviceregistry.AdapterConsul
	RegistryNacos               = serviceregistry.AdapterNacos
	RegistryStellMap            = serviceregistry.AdapterStellMap
	DiscoveryRoundRobin         = discovery.LoadBalanceRoundRobin
	DiscoveryRandom             = discovery.LoadBalanceRandom
	DiscoveryWeightedRoundRobin = discovery.LoadBalanceWeightedRound
)

func New(cfg Config, options ...Option) *App {
	return boot.New(cfg, options...)
}

// Start loads application.yml/application.yaml, configures the app, and starts it with context.Background.
func Start(options ...Option) (*App, error) {
	return StartWithContext(context.Background(), options...)
}

// StartWithContext loads application.yml/application.yaml, configures the app, and starts it.
func StartWithContext(ctx context.Context, options ...Option) (*App, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	app, err := boot.NewConfigured(ctx, cfg, options...)
	if err != nil {
		return nil, err
	}
	if err := app.Start(ctx); err != nil {
		return nil, err
	}
	return app, nil
}

// Run starts the configured app, waits for an interrupt signal, and then stops it.
func Run(options ...Option) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := StartWithContext(ctx, options...)
	if err != nil {
		return err
	}
	<-ctx.Done()
	return app.Stop(context.Background())
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

func NewGovernanceStore(initial ...GovernanceSnapshot) *GovernanceStore {
	return governance.NewStore(initial...)
}

func NewCache(adapter CacheAdapter, provider *ObservabilityProvider) (*Cache, error) {
	return cacheclient.New(adapter, provider)
}

func NewServiceRegistry(adapter ServiceRegistryAdapter) (*ServiceRegistry, error) {
	return serviceregistry.New(adapter)
}

func NewDiscoveryPicker(policy string) DiscoveryPicker {
	return discovery.NewPicker(policy)
}

func WithObservability(provider *ObservabilityProvider) Option {
	return boot.WithObservability(provider)
}

func WithInterceptor(definitions ...InterceptorDefinition) Option {
	return boot.WithInterceptor(definitions...)
}

func WithGovernanceStore(store *GovernanceStore) Option {
	return boot.WithGovernanceStore(store)
}

func HTTPServerInterceptor(name string, order int, fn InterceptorFunc) InterceptorDefinition {
	return interceptor.Business(interceptor.KindHTTPServer, name, order, interceptor.New(name, fn))
}

func HTTPClientInterceptor(name string, order int, fn InterceptorFunc) InterceptorDefinition {
	return interceptor.Business(interceptor.KindHTTPClient, name, order, interceptor.New(name, fn))
}

func GRPCServerInterceptor(name string, order int, fn InterceptorFunc) InterceptorDefinition {
	return interceptor.Business(interceptor.KindGRPCServer, name, order, interceptor.New(name, fn))
}

func GRPCClientInterceptor(name string, order int, fn InterceptorFunc) InterceptorDefinition {
	return interceptor.Business(interceptor.KindGRPCClient, name, order, interceptor.New(name, fn))
}

func FrameworkInterceptor(kind InterceptorKind, stage InterceptorStage, name string, fn InterceptorFunc) InterceptorDefinition {
	return interceptor.Framework(kind, stage, name, interceptor.New(name, fn))
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

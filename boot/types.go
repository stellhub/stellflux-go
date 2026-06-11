package boot

import (
	"context"
	"log/slog"

	"github.com/stellhub/stellar/config"
	apperrors "github.com/stellhub/stellar/errors"
	"github.com/stellhub/stellar/observability"
	bootgrpc "github.com/stellhub/stellar/transport/grpc"
)

const FrameworkName = "stellar"

type Config = config.Config

type Environment = config.Environment

const (
	EnvDev  = config.EnvDev
	EnvUAT  = config.EnvUAT
	EnvPre  = config.EnvPre
	EnvProd = config.EnvProd
)

var ErrAppNameRequired = apperrors.ErrAppNameRequired

type Option func(*App)

type Runtime interface {
	Config() config.Config
	Logger() *slog.Logger
	Registry() *Registry
	Observability() *observability.Provider
}

type Module interface {
	Name() string
	Start(context.Context, Runtime) error
	Stop(context.Context) error
}

type Starter interface {
	Name() string
	Condition(StarterContext) bool
	Init(context.Context, *App) error
	Start(context.Context) error
	Stop(context.Context) error
}

type StarterContext interface {
	Config() config.Config
	Logger() *slog.Logger
	Registry() *Registry
}

type Transport interface {
	Name() string
	Start(context.Context) error
	Stop(context.Context) error
}

type RPCAdapter = bootgrpc.Adapter

type HealthIndicator interface {
	Health(context.Context) HealthCheck
}

type observabilityConsumer interface {
	UseObservability(*observability.Provider)
}

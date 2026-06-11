package boot

import (
	"context"
	"strings"

	"github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/lifecycle"
	"github.com/stellhub/stellar/observability"
	grpcgoadapter "github.com/stellhub/stellar/transport/grpc/adapters/grpcgo"
	stellarhttp "github.com/stellhub/stellar/transport/http"
	chiadapter "github.com/stellhub/stellar/transport/http/adapters/chi"
	ginadapter "github.com/stellhub/stellar/transport/http/adapters/gin"
	hertzadapter "github.com/stellhub/stellar/transport/http/adapters/hertz"
)

func NewConfigured(ctx context.Context, cfg config.Config, options ...Option) (*App, error) {
	cfg = cfg.Normalize()
	observer, err := observability.NewFromConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	baseOptions := []Option{
		WithObservability(observer),
		WithHook(lifecycle.Hook{
			Name: "opentelemetry",
			OnStop: func(ctx context.Context) error {
				return observer.Shutdown(ctx)
			},
		}),
	}
	baseOptions = append(baseOptions, options...)
	app := New(cfg, baseOptions...)

	metricsHandler := observer.MetricsHandler()
	httpStarted := configureHTTPStarter(app, cfg)
	if metricsHandler != nil {
		path := "/metrics"
		if cfg.Starter.OpenTelemetry != nil && cfg.Starter.OpenTelemetry.MetricsPath != "" {
			path = cfg.Starter.OpenTelemetry.MetricsPath
		}
		app.HTTP().GET(path, stellarhttp.StdHandlerEndpoint(metricsHandler))
		if !httpStarted {
			configureDefaultHTTPStarter(app, cfg)
		}
	}
	configureGRPCStarter(app, cfg)
	return app, nil
}

func configureHTTPStarter(app *App, cfg config.Config) bool {
	server := cfg.HTTP.Server
	if server == nil {
		return false
	}
	if server.Enabled != nil && !*server.Enabled {
		return false
	}

	addr := cfg.HTTP.ServerAddr()
	adapter := "gin"
	if server.Adapter != "" {
		adapter = strings.ToLower(strings.TrimSpace(server.Adapter))
	}
	switch adapter {
	case "chi":
		app.setHTTPAdapter(chiadapter.New(addr), true)
	case "hertz":
		app.setHTTPAdapter(hertzadapter.New(addr), true)
	default:
		app.setHTTPAdapter(ginadapter.New(addr), true)
	}
	return true
}

func configureDefaultHTTPStarter(app *App, cfg config.Config) {
	app.setHTTPAdapter(ginadapter.New(cfg.HTTP.ServerAddr()), true)
}

func configureGRPCStarter(app *App, cfg config.Config) bool {
	server := cfg.GRPC.Server
	if server == nil {
		return false
	}
	if server.Enabled != nil && !*server.Enabled {
		return false
	}

	adapter := "grpc-go"
	if server.Adapter != "" {
		adapter = strings.ToLower(strings.TrimSpace(server.Adapter))
	}
	switch adapter {
	case "grpc", "grpc-go", "grpcgo", "go-grpc":
		app.setRPCAdapter(grpcgoadapter.New(cfg.GRPC.ServerAddr()), true)
	default:
		app.setRPCAdapter(grpcgoadapter.New(cfg.GRPC.ServerAddr()), true)
	}
	return true
}

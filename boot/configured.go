package boot

import (
	"context"
	"fmt"
	"strings"

	mysqlclient "github.com/stellhub/stellar/clients/mysql"
	redisclient "github.com/stellhub/stellar/clients/redis"
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
	if err := configureRedisStarter(app, cfg); err != nil {
		return nil, err
	}
	if err := configureMySQLStarter(ctx, app, cfg); err != nil {
		return nil, err
	}
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

func configureRedisStarter(app *App, cfg config.Config) error {
	if cfg.Redis == nil {
		return nil
	}
	if cfg.Redis.Enabled != nil && !*cfg.Redis.Enabled {
		return nil
	}

	client, err := redisclient.NewClientFromConfig(cfg.Redis, app.observability)
	if err != nil {
		return fmt.Errorf("configure redis client: %w", err)
	}
	app.registry.Set(redisclient.DefaultClientName, client)
	registerRedisDebugAPI(app, cfg.Redis)
	app.lifecycle.Append(lifecycle.Hook{
		Name: redisclient.DefaultClientName,
		OnStop: func(context.Context) error {
			return client.Close()
		},
	})
	return nil
}

func configureMySQLStarter(ctx context.Context, app *App, cfg config.Config) error {
	if cfg.MySQL == nil {
		return nil
	}
	if cfg.MySQL.Enabled != nil && !*cfg.MySQL.Enabled {
		return nil
	}

	db, err := mysqlclient.NewDBFromConfig(ctx, cfg.MySQL, app.observability)
	if err != nil {
		return fmt.Errorf("configure mysql client: %w", err)
	}
	app.registry.Set(mysqlclient.DefaultDBName, db)
	registerMySQLDebugAPI(app, cfg.MySQL)
	app.lifecycle.Append(lifecycle.Hook{
		Name: mysqlclient.DefaultDBName,
		OnStop: func(context.Context) error {
			return db.Close()
		},
	})
	return nil
}

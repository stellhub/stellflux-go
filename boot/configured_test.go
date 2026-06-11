package boot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stellhub/stellar/config"
	stellarhttp "github.com/stellhub/stellar/transport/http"
)

func TestNewConfiguredStartsConfiguredTransports(t *testing.T) {
	cfg := config.Config{
		AppName:     "configured-service",
		Environment: config.EnvDev,
		HTTP: config.HTTPConfig{
			Server: &config.HTTPServerConfig{Port: 18080},
		},
		GRPC: config.GRPCConfig{
			Server: &config.GRPCServerConfig{Port: 19090},
		},
		Starter: config.StarterConfig{
			OpenTelemetry: &config.OpenTelemetryStarterConfig{
				Metrics:       true,
				MetricsOutput: "prometheus",
			},
		},
	}.Normalize()

	app, err := NewConfigured(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new configured app: %v", err)
	}
	if got := app.Transports(); len(got) != 2 {
		t.Fatalf("expected http and grpc transports, got %#v", got)
	}
	routes := app.HTTP().Routes()
	foundMetrics := false
	for _, route := range routes {
		if route.Path == "/metrics" {
			foundMetrics = true
			break
		}
	}
	if !foundMetrics {
		t.Fatalf("expected /metrics route")
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	app.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected /metrics status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestNewConfiguredDoesNotStartGRPCForClientOnlyConfig(t *testing.T) {
	enabled := true
	cfg := config.Config{
		AppName:     "configured-service",
		Environment: config.EnvDev,
		GRPC: config.GRPCConfig{
			Client: &config.GRPCClientConfig{
				Enabled: &enabled,
				Clients: map[string]config.GRPCNamedClientConfig{
					"user-service": {Target: "dns:///localhost:19090"},
				},
			},
		},
	}.Normalize()

	app, err := NewConfigured(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new configured app: %v", err)
	}
	if got := app.Transports(); len(got) != 0 {
		t.Fatalf("expected no server transports for grpc client-only config, got %#v", got)
	}
}

func TestNewConfiguredRegistersRedisAndMySQLClients(t *testing.T) {
	cfg := config.Config{
		AppName:     "configured-service",
		Environment: config.EnvDev,
		Redis: &config.RedisConfig{
			Addr: "localhost:6379",
		},
		MySQL: &config.MySQLConfig{
			DSN: "user:pass@tcp(localhost:3306)/app?parseTime=true",
		},
	}.Normalize()

	app, err := NewConfigured(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new configured app: %v", err)
	}
	redisClient, ok := app.RedisClient()
	if !ok || redisClient == nil {
		t.Fatalf("expected redis client to be registered")
	}
	mysqlDB, ok := app.MySQLDB()
	if !ok || mysqlDB == nil {
		t.Fatalf("expected mysql db to be registered")
	}
	if err := redisClient.Close(); err != nil {
		t.Fatalf("close redis client: %v", err)
	}
	if err := mysqlDB.Close(); err != nil {
		t.Fatalf("close mysql db: %v", err)
	}
}

func TestNewConfiguredRegistersDataDebugAPIsFromConfig(t *testing.T) {
	enabled := true
	cfg := config.Config{
		AppName:     "configured-service",
		Environment: config.EnvDev,
		HTTP: config.HTTPConfig{
			Server: &config.HTTPServerConfig{Port: 18080},
		},
		Redis: &config.RedisConfig{
			Addr:     "localhost:6379",
			DebugAPI: &config.DebugAPIConfig{Enabled: &enabled},
		},
		MySQL: &config.MySQLConfig{
			DSN:      "user:pass@tcp(localhost:3306)/app?parseTime=true",
			DebugAPI: &config.DebugAPIConfig{Enabled: &enabled},
		},
	}.Normalize()

	app, err := NewConfigured(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new configured app: %v", err)
	}
	defer func() {
		if redisClient, ok := app.RedisClient(); ok {
			_ = redisClient.Close()
		}
		if mysqlDB, ok := app.MySQLDB(); ok {
			_ = mysqlDB.Close()
		}
	}()

	routes := app.HTTP().Routes()
	assertRouteExists(t, routes, http.MethodPost, "/redis/items")
	assertRouteExists(t, routes, http.MethodGet, "/redis/items")
	assertRouteExists(t, routes, http.MethodPost, "/mysql/items")
	assertRouteExists(t, routes, http.MethodGet, "/mysql/items")
}

func TestPortFromAddr(t *testing.T) {
	tests := map[string]string{
		":8080":          "8080",
		"127.0.0.1:9090": "9090",
		"localhost:7070": "7070",
		"6060":           "6060",
	}
	for addr, want := range tests {
		if got := portFromAddr(addr); got != want {
			t.Fatalf("portFromAddr(%q) = %q, want %q", addr, got, want)
		}
	}
}

func assertRouteExists(t *testing.T, routes []stellarhttp.Route, method string, path string) {
	t.Helper()
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return
		}
	}
	t.Fatalf("expected route %s %s", method, path)
}

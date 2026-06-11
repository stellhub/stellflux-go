package boot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stellhub/stellar/config"
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

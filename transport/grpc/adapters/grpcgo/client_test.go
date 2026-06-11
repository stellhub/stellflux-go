package grpcgoadapter

import (
	"context"
	"testing"

	"github.com/stellhub/stellar/config"
)

func TestNewNamedClientConnFromConfig(t *testing.T) {
	cfg := &config.GRPCClientConfig{
		Timeout:  "3s",
		Insecure: boolPtr(true),
		Clients: map[string]config.GRPCNamedClientConfig{
			"user-service": {
				Target:  "dns:///localhost:19091",
				Timeout: "2s",
			},
		},
	}

	conn, target, err := NewNamedClientConnFromConfig(context.Background(), cfg, "user-service", nil)
	if err != nil {
		t.Fatalf("new grpc client conn: %v", err)
	}
	defer conn.Close()

	if target != "dns:///localhost:19091" {
		t.Fatalf("unexpected target %q", target)
	}
}

func TestNewNamedClientConnFromConfigRejectsDisabledClient(t *testing.T) {
	cfg := &config.GRPCClientConfig{
		Enabled: boolPtr(false),
		Clients: map[string]config.GRPCNamedClientConfig{
			"user-service": {Target: "dns:///localhost:19091"},
		},
	}

	if _, _, err := NewNamedClientConnFromConfig(context.Background(), cfg, "user-service", nil); err == nil {
		t.Fatalf("expected disabled grpc client error")
	}
}

func TestNewNamedClientConnFromConfigRequiresNamedClient(t *testing.T) {
	cfg := &config.GRPCClientConfig{
		Clients: map[string]config.GRPCNamedClientConfig{},
	}

	if _, _, err := NewNamedClientConnFromConfig(context.Background(), cfg, "user-service", nil); err == nil {
		t.Fatalf("expected missing grpc client error")
	}
}

func boolPtr(value bool) *bool {
	return &value
}

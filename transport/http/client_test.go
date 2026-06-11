package http_test

import (
	"testing"
	"time"

	"github.com/stellhub/stellar/config"
	stellarhttp "github.com/stellhub/stellar/transport/http"
)

func TestNewNamedClientFromConfig(t *testing.T) {
	cfg := &config.HTTPClientConfig{
		Timeout:             "3s",
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     "90s",
		Clients: map[string]config.HTTPNamedClientConfig{
			"user-service": {
				BaseURL: "http://localhost:8081",
				Timeout: "2s",
			},
		},
	}

	client, baseURL, err := stellarhttp.NewNamedClientFromConfig(cfg, "user-service", nil)
	if err != nil {
		t.Fatalf("new named client: %v", err)
	}
	if baseURL != "http://localhost:8081" {
		t.Fatalf("unexpected base url %q", baseURL)
	}
	if client.Timeout != 2*time.Second {
		t.Fatalf("unexpected timeout %s", client.Timeout)
	}
}

func TestNewNamedClientFromConfigRejectsUnknownName(t *testing.T) {
	_, _, err := stellarhttp.NewNamedClientFromConfig(&config.HTTPClientConfig{}, "missing-service", nil)
	if err == nil {
		t.Fatalf("expected unknown named client error")
	}
}

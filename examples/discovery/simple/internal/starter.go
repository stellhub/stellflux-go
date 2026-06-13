package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stellhub/stellar"
	stellarhttp "github.com/stellhub/stellar/transport/http"
)

type discoveryStarter struct {
	app *stellar.App
}

func NewDiscoveryStarter() *discoveryStarter {
	return &discoveryStarter{}
}

func (s *discoveryStarter) Name() string {
	return "discovery-simple-example"
}

func (s *discoveryStarter) Condition(stellar.StarterContext) bool {
	return true
}

func (s *discoveryStarter) Init(_ context.Context, app *stellar.App) error {
	s.app = app
	api := app.HTTP().Group("/api/v1")
	api.GET("/ping", handlePing)
	api.GET("/discovery/call", s.handleDiscoveryCall)
	return nil
}

func (s *discoveryStarter) Start(context.Context) error {
	return nil
}

func (s *discoveryStarter) Stop(context.Context) error {
	return nil
}

func handlePing(context.Context, *stellarhttp.Request) (*stellarhttp.Response, error) {
	return stellarhttp.JSON(http.StatusOK, map[string]any{
		"message": "pong",
		"source":  "stellar-discovery-simple-example",
	}), nil
}

func (s *discoveryStarter) handleDiscoveryCall(ctx context.Context, _ *stellarhttp.Request) (*stellarhttp.Response, error) {
	client, baseURL, err := s.app.NewHTTPClient("self-service")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/ping", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("self-service returned status %d", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode self-service response: %w", err)
	}
	return stellarhttp.JSON(http.StatusOK, map[string]any{
		"client":  "self-service",
		"baseURL": baseURL,
		"result":  payload,
	}), nil
}

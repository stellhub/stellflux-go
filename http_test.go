package stellar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusHandler(t *testing.T) {
	app := New(Config{
		AppName:     "test-service",
		Environment: EnvUAT,
		Zone:        "zone-a",
	})
	app.Use(StandardModules()...)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/stellar/status", nil)

	app.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response StatusResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Framework != "stellar" {
		t.Fatalf("expected framework stellar, got %q", response.Framework)
	}
	if len(response.Modules) != 7 {
		t.Fatalf("expected 7 modules, got %d", len(response.Modules))
	}
}

func TestHealthHandler(t *testing.T) {
	app := New(Config{
		AppName:     "test-service",
		Environment: EnvDev,
	})
	app.Use(StandardModules()...)

	if err := app.Start(testingContext(t)); err != nil {
		t.Fatalf("start app: %v", err)
	}
	defer app.Stop(testingContext(t))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	app.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != HealthStatusUp {
		t.Fatalf("expected health %s, got %s", HealthStatusUp, response.Status)
	}
}

func testingContext(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}

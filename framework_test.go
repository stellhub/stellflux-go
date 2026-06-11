package stellar

import (
	"context"
	"testing"
)

func TestAppLifecycle(t *testing.T) {
	app := New(Config{
		AppName:     "test-service",
		Environment: EnvDev,
		Zone:        "local",
	})
	app.Use(StandardModules()...)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	if got := len(app.Modules()); got != 7 {
		t.Fatalf("expected 7 standard modules, got %d", got)
	}
	if got := app.Health(context.Background()).Status; got != HealthStatusUp {
		t.Fatalf("expected health %s, got %s", HealthStatusUp, got)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatalf("stop app: %v", err)
	}
}

func TestAppNameRequired(t *testing.T) {
	app := New(Config{})

	if err := app.Start(context.Background()); err != ErrAppNameRequired {
		t.Fatalf("expected ErrAppNameRequired, got %v", err)
	}
}

package stellar

import (
	"context"
	"os"
	"path/filepath"
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

func TestStartLoadsApplicationConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.yml")
	if err := os.WriteFile(path, []byte("app:\n  name: configured-service\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("STELLAR_CONFIG_FILE", path)
	t.Setenv("STELLAR_CONFIG", "")
	t.Setenv("STELLAR_APPLICATION_CONFIG", "")

	app, err := Start()
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	defer func() {
		if err := app.Stop(context.Background()); err != nil {
			t.Fatalf("stop app: %v", err)
		}
	}()

	if !app.Started() {
		t.Fatalf("expected app to be started")
	}
	if app.Config().AppName != "configured-service" {
		t.Fatalf("unexpected app name %q", app.Config().AppName)
	}
}

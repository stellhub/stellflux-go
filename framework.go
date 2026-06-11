package stellflux

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
)

var ErrAppNameRequired = errors.New("stellflux: app name is required")

type Runtime interface {
	Config() Config
	Logger() *slog.Logger
}

type Module interface {
	Name() string
	Start(context.Context, Runtime) error
	Stop(context.Context) error
}

type Option func(*App)

type App struct {
	config  Config
	logger  *slog.Logger
	modules []Module
	started bool
	mu      sync.Mutex
}

func New(config Config, options ...Option) *App {
	app := &App{
		config: config.Normalize(),
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
	for _, option := range options {
		option(app)
	}
	return app
}

func WithLogger(logger *slog.Logger) Option {
	return func(app *App) {
		if logger != nil {
			app.logger = logger
		}
	}
}

func (a *App) Use(modules ...Module) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.modules = append(a.modules, modules...)
}

func (a *App) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.config.AppName == "" {
		return ErrAppNameRequired
	}
	if a.started {
		return nil
	}
	if a.config.Disabled {
		a.started = true
		return nil
	}

	runtime := appRuntime{config: a.config, logger: a.logger}
	for _, module := range a.modules {
		if err := module.Start(ctx, runtime); err != nil {
			return err
		}
		a.logger.Info("stellflux module started", "module", module.Name())
	}

	a.started = true
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.started {
		return nil
	}

	for i := len(a.modules) - 1; i >= 0; i-- {
		module := a.modules[i]
		if err := module.Stop(ctx); err != nil {
			return err
		}
		a.logger.Info("stellflux module stopped", "module", module.Name())
	}

	a.started = false
	return nil
}

func (a *App) Config() Config {
	return a.config
}

func (a *App) Logger() *slog.Logger {
	return a.logger
}

func (a *App) Modules() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	names := make([]string, 0, len(a.modules))
	for _, module := range a.modules {
		names = append(names, module.Name())
	}
	return names
}

type appRuntime struct {
	config Config
	logger *slog.Logger
}

func (r appRuntime) Config() Config {
	return r.config
}

func (r appRuntime) Logger() *slog.Logger {
	return r.logger
}

package boot

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/stellhub/stellar/config"
	apperrors "github.com/stellhub/stellar/errors"
	"github.com/stellhub/stellar/lifecycle"
	"github.com/stellhub/stellar/observability"
	boothttp "github.com/stellhub/stellar/transport/http"
)

type App struct {
	mu                sync.Mutex
	config            config.Config
	logger            *slog.Logger
	registry          *Registry
	lifecycle         *lifecycle.Manager
	observability     *observability.Provider
	httpRouter        *boothttp.Router
	httpAdapter       boothttp.Adapter
	rpcAdapter        RPCAdapter
	modules           []Module
	starters          []Starter
	activeStarters    []Starter
	transports        []Transport
	startedModules    []Module
	startedStarters   []Starter
	startedTransports []Transport
	started           bool
}

func New(cfg config.Config, options ...Option) *App {
	cfg = cfg.Normalize()
	observer := observability.New(observability.WithServiceName(cfg.AppName))
	app := &App{
		config:        cfg,
		logger:        slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		registry:      NewRegistry(),
		lifecycle:     lifecycle.NewManager(),
		observability: observer,
		httpRouter:    boothttp.NewRouter(boothttp.WithObservability(observer)),
	}
	app.registerManagementRoutes()
	for _, option := range options {
		option(app)
	}
	app.config = app.config.Normalize()
	return app
}

func WithLogger(logger *slog.Logger) Option {
	return func(app *App) {
		if logger != nil {
			app.logger = logger
		}
	}
}

func WithObservability(provider *observability.Provider) Option {
	return func(app *App) {
		if provider == nil {
			return
		}
		app.observability = provider
		app.httpRouter.SetObservability(provider)
		if app.rpcAdapter != nil {
			if consumer, ok := app.rpcAdapter.(observabilityConsumer); ok {
				consumer.UseObservability(provider)
			}
		}
	}
}

func WithEnvConfig() Option {
	return func(app *App) {
		app.config = config.FromEnv(app.config)
	}
}

func WithModule(modules ...Module) Option {
	return func(app *App) {
		app.modules = append(app.modules, modules...)
	}
}

func WithStarter(starters ...Starter) Option {
	return func(app *App) {
		app.starters = append(app.starters, starters...)
	}
}

func WithTransport(transports ...Transport) Option {
	return func(app *App) {
		for _, transport := range transports {
			app.addTransport(transport)
		}
	}
}

func WithHook(hooks ...lifecycle.Hook) Option {
	return func(app *App) {
		app.lifecycle.Append(hooks...)
	}
}

func (a *App) Use(modules ...Module) {
	a.RegisterModules(modules...)
}

func (a *App) RegisterModules(modules ...Module) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.modules = append(a.modules, modules...)
}

func (a *App) RegisterStarters(starters ...Starter) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.starters = append(a.starters, starters...)
}

func (a *App) RegisterTransports(transports ...Transport) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, transport := range transports {
		a.addTransport(transport)
	}
}

func (a *App) RegisterHooks(hooks ...lifecycle.Hook) {
	a.lifecycle.Append(hooks...)
}

func (a *App) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.validate(); err != nil {
		return err
	}
	if a.started {
		return nil
	}
	a.resetStarted()
	if a.config.Disabled {
		a.started = true
		return nil
	}

	starterCtx := appStarterContext{app: a}
	for _, starter := range a.starters {
		if starter == nil {
			continue
		}
		if !starter.Condition(starterCtx) {
			continue
		}
		if err := starter.Init(ctx, a); err != nil {
			_ = a.stopStarted(ctx)
			return fmt.Errorf("init starter %s: %w", starter.Name(), err)
		}
		a.activeStarters = append(a.activeStarters, starter)
	}

	if err := a.lifecycle.Start(ctx); err != nil {
		_ = a.stopStarted(ctx)
		return fmt.Errorf("start lifecycle: %w", err)
	}

	runtime := appRuntime{app: a}
	for _, module := range a.modules {
		if module == nil {
			continue
		}
		if err := module.Start(ctx, runtime); err != nil {
			_ = a.stopStarted(ctx)
			return fmt.Errorf("start module %s: %w", module.Name(), err)
		}
		a.startedModules = append(a.startedModules, module)
		a.logger.Info("stellar module started", "module", module.Name())
	}

	for _, starter := range a.activeStarters {
		if err := starter.Start(ctx); err != nil {
			_ = a.stopStarted(ctx)
			return fmt.Errorf("start starter %s: %w", starter.Name(), err)
		}
		a.startedStarters = append(a.startedStarters, starter)
		a.logger.Info("stellar starter started", "starter", starter.Name())
	}

	for _, transport := range a.transports {
		if transport == nil {
			continue
		}
		if err := transport.Start(ctx); err != nil {
			_ = a.stopStarted(ctx)
			return fmt.Errorf("start transport %s: %w", transport.Name(), err)
		}
		a.startedTransports = append(a.startedTransports, transport)
		a.logTransportStarted(transport)
	}

	a.started = true
	return nil
}

func (a *App) Run(ctx context.Context) error {
	if err := a.Start(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	if err := a.Stop(context.Background()); err != nil {
		return err
	}
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.started {
		return nil
	}
	err := a.stopStarted(ctx)
	a.started = false
	a.resetStarted()
	return err
}

func (a *App) Config() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.config.Normalize()
}

func (a *App) Logger() *slog.Logger {
	return a.logger
}

func (a *App) Registry() *Registry {
	return a.registry
}

func (a *App) Observability() *observability.Provider {
	return a.observability
}

func (a *App) Lifecycle() *lifecycle.Manager {
	return a.lifecycle
}

func (a *App) HTTP() *boothttp.Router {
	return a.httpRouter
}

func (a *App) RPC() RPCAdapter {
	return a.rpcAdapter
}

func (a *App) Started() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.started
}

func (a *App) Modules() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return moduleNames(a.modules)
}

func (a *App) Starters() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return starterNames(a.activeStarters)
}

func (a *App) Transports() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return transportNames(a.transports)
}

func (a *App) validate() error {
	if strings.TrimSpace(a.config.AppName) == "" {
		return ErrAppNameRequired
	}
	if !a.config.Environment.Valid() {
		return apperrors.New(apperrors.CodeInvalidConfig, "stellar: environment must be one of dev, uat, pre, prod", 400)
	}
	return nil
}

func (a *App) stopStarted(ctx context.Context) error {
	var joined error
	for i := len(a.startedTransports) - 1; i >= 0; i-- {
		transport := a.startedTransports[i]
		if err := transport.Stop(ctx); err != nil {
			joined = stderrors.Join(joined, fmt.Errorf("stop transport %s: %w", transport.Name(), err))
		}
	}
	for i := len(a.startedStarters) - 1; i >= 0; i-- {
		starter := a.startedStarters[i]
		if err := starter.Stop(ctx); err != nil {
			joined = stderrors.Join(joined, fmt.Errorf("stop starter %s: %w", starter.Name(), err))
		}
	}
	for i := len(a.startedModules) - 1; i >= 0; i-- {
		module := a.startedModules[i]
		if err := module.Stop(ctx); err != nil {
			joined = stderrors.Join(joined, fmt.Errorf("stop module %s: %w", module.Name(), err))
		}
	}
	if err := a.lifecycle.Stop(ctx); err != nil {
		joined = stderrors.Join(joined, fmt.Errorf("stop lifecycle: %w", err))
	}
	return joined
}

func (a *App) resetStarted() {
	a.activeStarters = nil
	a.startedModules = nil
	a.startedStarters = nil
	a.startedTransports = nil
}

func moduleNames(values []Module) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		names = append(names, value.Name())
	}
	return names
}

func starterNames(values []Starter) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		names = append(names, value.Name())
	}
	return names
}

func transportNames(values []Transport) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		names = append(names, value.Name())
	}
	return names
}

func (a *App) logTransportStarted(transport Transport) {
	attrs := []any{"transport", transport.Name()}
	if addr := transportAddr(transport); addr != "" {
		attrs = append(attrs, "addr", addr)
		if port := portFromAddr(addr); port != "" {
			attrs = append(attrs, "port", port)
		}
	}
	a.logger.Info("stellar transport started", attrs...)
}

func transportAddr(transport Transport) string {
	reporter, ok := transport.(interface{ Addr() string })
	if !ok {
		return ""
	}
	return reporter.Addr()
}

func portFromAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	_, port, err := net.SplitHostPort(addr)
	if err == nil {
		return port
	}
	if strings.HasPrefix(addr, ":") {
		return strings.TrimPrefix(addr, ":")
	}
	if !strings.Contains(addr, ":") {
		return addr
	}
	index := strings.LastIndex(addr, ":")
	if index >= 0 && index < len(addr)-1 {
		return addr[index+1:]
	}
	return ""
}

type appRuntime struct {
	app *App
}

func (r appRuntime) Config() config.Config {
	return r.app.config.Normalize()
}

func (r appRuntime) Logger() *slog.Logger {
	return r.app.logger
}

func (r appRuntime) Registry() *Registry {
	return r.app.registry
}

func (r appRuntime) Observability() *observability.Provider {
	return r.app.observability
}

type appStarterContext struct {
	app *App
}

func (c appStarterContext) Config() config.Config {
	return c.app.config.Normalize()
}

func (c appStarterContext) Logger() *slog.Logger {
	return c.app.logger
}

func (c appStarterContext) Registry() *Registry {
	return c.app.registry
}

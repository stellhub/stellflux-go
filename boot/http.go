package boot

import (
	"context"
	stdhttp "net/http"

	boothttp "github.com/stellhub/stellar/transport/http"
)

type StatusResponse struct {
	Service     string   `json:"service"`
	Framework   string   `json:"framework"`
	Environment string   `json:"environment"`
	Zone        string   `json:"zone,omitempty"`
	Version     string   `json:"version,omitempty"`
	Started     bool     `json:"started"`
	Modules     []string `json:"modules"`
	Starters    []string `json:"starters"`
	Transports  []string `json:"transports"`
}

type HealthResponse struct {
	Status    HealthStatus  `json:"status"`
	Framework string        `json:"framework"`
	Checks    []HealthCheck `json:"checks"`
}

func (a *App) Handler() stdhttp.Handler {
	return handlerOf(a.ensureHTTPAdapter())
}

func (a *App) NewHTTPClient(name string) (*stdhttp.Client, string, error) {
	cfg := a.Config()
	return boothttp.NewNamedClientFromConfig(cfg.HTTP.Client, name, a.observability)
}

func (a *App) registerManagementRoutes() {
	a.httpRouter.GET("/health", a.handleHealth)
	a.httpRouter.GET("/stellar/status", a.handleStatus)
}

func (a *App) handleHealth(ctx context.Context, _ *boothttp.Request) (*boothttp.Response, error) {
	report := a.Health(ctx)
	status := stdhttp.StatusOK
	if report.Status == HealthStatusDown {
		status = stdhttp.StatusServiceUnavailable
	}
	return boothttp.JSON(status, HealthResponse{
		Status:    report.Status,
		Framework: FrameworkName,
		Checks:    report.Checks,
	}), nil
}

func (a *App) handleStatus(context.Context, *boothttp.Request) (*boothttp.Response, error) {
	cfg := a.Config()
	return boothttp.JSON(stdhttp.StatusOK, StatusResponse{
		Service:     cfg.AppName,
		Framework:   FrameworkName,
		Environment: string(cfg.Environment),
		Zone:        cfg.Zone,
		Version:     cfg.Version,
		Started:     a.Started(),
		Modules:     a.Modules(),
		Starters:    a.Starters(),
		Transports:  a.Transports(),
	}), nil
}

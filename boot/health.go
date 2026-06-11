package boot

import "context"

type HealthStatus string

const (
	HealthStatusUp      HealthStatus = "UP"
	HealthStatusDown    HealthStatus = "DOWN"
	HealthStatusSkipped HealthStatus = "SKIPPED"
)

type HealthCheck struct {
	Name    string       `json:"name"`
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
}

type HealthReport struct {
	Status HealthStatus  `json:"status"`
	Checks []HealthCheck `json:"checks"`
}

func (a *App) Health(ctx context.Context) HealthReport {
	a.mu.Lock()
	defer a.mu.Unlock()

	checks := make([]HealthCheck, 0, len(a.modules)+len(a.activeStarters)+len(a.transports))
	for _, module := range a.modules {
		checks = append(checks, healthOf(ctx, module.Name(), module))
	}
	for _, starter := range a.activeStarters {
		checks = append(checks, healthOf(ctx, starter.Name(), starter))
	}
	for _, transport := range a.transports {
		checks = append(checks, healthOf(ctx, transport.Name(), transport))
	}

	status := HealthStatusUp
	for _, check := range checks {
		if check.Status == HealthStatusDown {
			status = HealthStatusDown
			break
		}
	}
	return HealthReport{Status: status, Checks: checks}
}

func healthOf(ctx context.Context, name string, value any) HealthCheck {
	indicator, ok := value.(HealthIndicator)
	if !ok {
		return HealthCheck{Name: name, Status: HealthStatusUp}
	}
	return indicator.Health(ctx)
}

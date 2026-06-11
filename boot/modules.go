package boot

import "context"

type MiddlewareStandard struct {
	Name        string
	Capability  string
	Description string
}

type StandardModule struct {
	standard MiddlewareStandard
	started  bool
}

func NewStandardModule(standard MiddlewareStandard) *StandardModule {
	return &StandardModule{standard: standard}
}

func (m *StandardModule) Name() string {
	return m.standard.Name
}

func (m *StandardModule) Standard() MiddlewareStandard {
	return m.standard
}

func (m *StandardModule) Start(context.Context, Runtime) error {
	m.started = true
	return nil
}

func (m *StandardModule) Stop(context.Context) error {
	m.started = false
	return nil
}

func (m *StandardModule) Health(context.Context) HealthCheck {
	if !m.started {
		return HealthCheck{Name: m.Name(), Status: HealthStatusSkipped}
	}
	return HealthCheck{Name: m.Name(), Status: HealthStatusUp}
}

func StandardModules() []Module {
	standards := []MiddlewareStandard{
		{Name: "stellmap", Capability: "service-discovery", Description: "Service discovery and registry integration."},
		{Name: "stellflow", Capability: "messaging", Description: "Messaging and event streaming integration."},
		{Name: "stellnula", Capability: "configuration", Description: "Configuration center integration."},
		{Name: "stellspec", Capability: "observability", Description: "Observability, log query, and telemetry standard integration."},
		{Name: "stellorbit", Capability: "traffic-governance", Description: "Routing, retries, traffic switching, and lifecycle policy integration."},
		{Name: "stellgate", Capability: "api-gateway", Description: "API gateway and ingress standard integration."},
		{Name: "stellatlas", Capability: "cmdb", Description: "Asset inventory, topology, relationship, and lifecycle metadata integration."},
	}

	modules := make([]Module, 0, len(standards))
	for _, standard := range standards {
		modules = append(modules, NewStandardModule(standard))
	}
	return modules
}

package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stellhub/stellar/config"
)

func TestHTTPConfigForNamedUsesNamedDiscovery(t *testing.T) {
	enabled := true
	cfg := &config.HTTPClientConfig{
		Discovery: &config.DiscoveryConfig{
			Adapter:   "stellmap",
			Namespace: "default",
			Endpoints: []string{"http://localhost:18090"},
		},
	}
	named := config.HTTPNamedClientConfig{
		Discovery: &config.DiscoveryConfig{
			Enabled:      &enabled,
			Service:      "user-service",
			Protocol:     "http",
			EndpointName: "http",
		},
	}

	discoveryCfg, target, ok, err := HTTPConfigForNamed(cfg, named, "user-service")
	if err != nil {
		t.Fatalf("http discovery config: %v", err)
	}
	if !ok {
		t.Fatalf("expected discovery to be enabled")
	}
	if discoveryCfg.Adapter != "stellmap" || target.Service != "user-service" || target.Protocol != "http" {
		t.Fatalf("unexpected discovery config %#v target %#v", discoveryCfg, target)
	}
}

func TestHTTPConfigForNamedKeepsStaticBaseURL(t *testing.T) {
	cfg := &config.HTTPClientConfig{
		Discovery: &config.DiscoveryConfig{
			Adapter:   "stellmap",
			Namespace: "default",
			Endpoints: []string{"http://localhost:18090"},
		},
	}
	named := config.HTTPNamedClientConfig{
		BaseURL: "http://localhost:8081",
	}

	_, _, ok, err := HTTPConfigForNamed(cfg, named, "user-service")
	if err != nil {
		t.Fatalf("http discovery config: %v", err)
	}
	if ok {
		t.Fatalf("expected static base_url to disable inherited discovery")
	}
}

func TestCachedResolverPickUsesRefreshedEndpoints(t *testing.T) {
	base := &fakeResolver{
		endpoints: []Endpoint{
			{Name: "http", Protocol: "http", Host: "127.0.0.1", Port: 8081},
			{Name: "http", Protocol: "http", Host: "127.0.0.1", Port: 8082},
		},
	}
	cached := NewCachedResolver(base, Target{
		Namespace:   "default",
		Service:     "user-service",
		Protocol:    "http",
		PassingOnly: true,
	}, WithRefreshInterval(time.Hour))
	defer func() { _ = cached.Close(context.Background()) }()

	first, err := cached.Pick(context.Background(), &RoundRobinPicker{})
	if err != nil {
		t.Fatalf("pick first endpoint: %v", err)
	}
	second, err := cached.Pick(context.Background(), &RoundRobinPicker{})
	if err != nil {
		t.Fatalf("pick second endpoint: %v", err)
	}
	if first.Port == 0 || second.Port == 0 {
		t.Fatalf("expected discovered endpoints, got %#v %#v", first, second)
	}
}

type fakeResolver struct {
	endpoints []Endpoint
}

func (r *fakeResolver) Resolve(context.Context, Target) ([]Endpoint, error) {
	return cloneEndpoints(r.endpoints), nil
}

func (r *fakeResolver) Watch(context.Context, Target) (Watcher, error) {
	return &fakeWatcher{events: make(chan Event)}, nil
}

func (r *fakeResolver) Close(context.Context) error {
	return nil
}

type fakeWatcher struct {
	events chan Event
}

func (w *fakeWatcher) Events() <-chan Event {
	return w.events
}

func (w *fakeWatcher) Close() error {
	close(w.events)
	return nil
}

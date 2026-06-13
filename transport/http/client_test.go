package http_test

import (
	"context"
	stdhttp "net/http"
	"testing"
	"time"

	"github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/discovery"
	"github.com/stellhub/stellar/interceptor"
	stellarhttp "github.com/stellhub/stellar/transport/http"
)

func TestNewNamedClientFromConfig(t *testing.T) {
	cfg := &config.HTTPClientConfig{
		Timeout:             "3s",
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     "90s",
		Clients: map[string]config.HTTPNamedClientConfig{
			"user-service": {
				BaseURL: "http://localhost:8081",
				Timeout: "2s",
			},
		},
	}

	client, baseURL, err := stellarhttp.NewNamedClientFromConfig(cfg, "user-service", nil)
	if err != nil {
		t.Fatalf("new named client: %v", err)
	}
	if baseURL != "http://localhost:8081" {
		t.Fatalf("unexpected base url %q", baseURL)
	}
	if client.Timeout != 2*time.Second {
		t.Fatalf("unexpected timeout %s", client.Timeout)
	}
}

func TestNewNamedClientFromConfigRejectsUnknownName(t *testing.T) {
	_, _, err := stellarhttp.NewNamedClientFromConfig(&config.HTTPClientConfig{}, "missing-service", nil)
	if err == nil {
		t.Fatalf("expected unknown named client error")
	}
}

func TestHTTPClientUsesInterceptors(t *testing.T) {
	registry := interceptor.NewRegistry()
	called := false
	registry.Register(interceptor.Business(interceptor.KindHTTPClient, "business", 10, interceptor.New("business", func(ctx context.Context, inv *interceptor.Invocation, req any, next interceptor.Handler) (any, error) {
		called = true
		if inv.Service != "user-service" {
			t.Fatalf("unexpected service %q", inv.Service)
		}
		return next(ctx, inv, req)
	})))

	client := stellarhttp.NewClient(
		stellarhttp.WithClientName("user-service"),
		stellarhttp.WithClientInterceptors(registry),
		stellarhttp.WithTransport(roundTripperFunc(func(req *stdhttp.Request) (*stdhttp.Response, error) {
			return &stdhttp.Response{
				StatusCode: stdhttp.StatusOK,
				Body:       stdhttp.NoBody,
				Header:     stdhttp.Header{},
				Request:    req,
			}, nil
		})),
	)

	req, err := stdhttp.NewRequest(stdhttp.MethodGet, "http://example.com/users/42", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if !called {
		t.Fatalf("expected interceptor to be called")
	}
}

func TestHTTPClientUsesDiscoveryEndpoint(t *testing.T) {
	resolver := discovery.NewCachedResolver(&staticResolver{
		endpoints: []discovery.Endpoint{{
			Name:     "http",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     18081,
		}},
	}, discovery.Target{
		Namespace:   "default",
		Service:     "user-service",
		Protocol:    "http",
		PassingOnly: true,
	})
	defer func() { _ = resolver.Close(context.Background()) }()

	client := stellarhttp.NewClient(
		stellarhttp.WithClientDiscovery(resolver, discovery.NewPicker(discovery.LoadBalanceRoundRobin)),
		stellarhttp.WithTransport(roundTripperFunc(func(req *stdhttp.Request) (*stdhttp.Response, error) {
			if req.URL.Host != "127.0.0.1:18081" {
				t.Fatalf("unexpected discovered host %q", req.URL.Host)
			}
			return &stdhttp.Response{
				StatusCode: stdhttp.StatusOK,
				Body:       stdhttp.NoBody,
				Header:     stdhttp.Header{},
				Request:    req,
			}, nil
		})),
	)

	req, err := stdhttp.NewRequest(stdhttp.MethodGet, "http://user-service/users/42", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
}

type roundTripperFunc func(*stdhttp.Request) (*stdhttp.Response, error)

func (f roundTripperFunc) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	return f(req)
}

type staticResolver struct {
	endpoints []discovery.Endpoint
}

func (r *staticResolver) Resolve(context.Context, discovery.Target) ([]discovery.Endpoint, error) {
	return append([]discovery.Endpoint(nil), r.endpoints...), nil
}

func (r *staticResolver) Watch(context.Context, discovery.Target) (discovery.Watcher, error) {
	return &staticWatcher{events: make(chan discovery.Event)}, nil
}

func (r *staticResolver) Close(context.Context) error {
	return nil
}

type staticWatcher struct {
	events chan discovery.Event
}

func (w *staticWatcher) Events() <-chan discovery.Event {
	return w.events
}

func (w *staticWatcher) Close() error {
	close(w.events)
	return nil
}

package discovery

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/stellhub/stellar/config"
	serviceregistry "github.com/stellhub/stellar/registry"
)

type RegistryResolver struct {
	registry *serviceregistry.Registry
}

func NewRegistryResolver(registry *serviceregistry.Registry) (*RegistryResolver, error) {
	if registry == nil {
		return nil, fmt.Errorf("stellar: discovery registry is required")
	}
	return &RegistryResolver{registry: registry}, nil
}

func NewRegistryResolverFromConfig(ctx context.Context, cfg *config.DiscoveryConfig) (*RegistryResolver, error) {
	registry, err := serviceregistry.NewFromConfig(ctx, RegistryConfigFromDiscovery(cfg))
	if err != nil {
		return nil, err
	}
	return NewRegistryResolver(registry)
}

func (r *RegistryResolver) Resolve(ctx context.Context, target Target) ([]Endpoint, error) {
	target = NormalizeTarget(target)
	instances, err := r.registry.Discover(ctx, serviceregistry.Query{
		Namespace:   target.Namespace,
		Service:     target.Service,
		Zone:        target.Zone,
		Labels:      target.Labels,
		PassingOnly: target.PassingOnly,
	})
	if err != nil {
		return nil, err
	}
	return endpointsFromInstances(instances, target), nil
}

func (r *RegistryResolver) Watch(ctx context.Context, target Target) (Watcher, error) {
	target = NormalizeTarget(target)
	watcher, err := r.registry.Watch(ctx, serviceregistry.Query{
		Namespace:   target.Namespace,
		Service:     target.Service,
		Zone:        target.Zone,
		Labels:      target.Labels,
		PassingOnly: target.PassingOnly,
	})
	if err != nil {
		return nil, err
	}
	return &registryWatcher{
		watcher: watcher,
		target:  target,
		events:  make(chan Event, 128),
		done:    make(chan struct{}),
	}, nil
}

func (r *RegistryResolver) Close(ctx context.Context) error {
	if r == nil || r.registry == nil {
		return nil
	}
	return r.registry.Close(ctx)
}

type registryWatcher struct {
	watcher serviceregistry.Watcher
	target  Target
	events  chan Event
	done    chan struct{}
	once    sync.Once
}

func (w *registryWatcher) Events() <-chan Event {
	w.once.Do(func() {
		go w.run()
	})
	return w.events
}

func (w *registryWatcher) Close() error {
	if w == nil || w.watcher == nil {
		return nil
	}
	return w.watcher.Close()
}

func (w *registryWatcher) run() {
	defer close(w.events)
	defer close(w.done)
	for event := range w.watcher.Events() {
		converted := Event{Type: event.Type}
		if converted.Type == "" {
			converted.Type = EventSnapshot
		}
		if event.Instance != nil {
			endpoints := endpointsFromInstances([]serviceregistry.Instance{*event.Instance}, w.target)
			if len(endpoints) > 0 {
				converted.Endpoint = &endpoints[0]
			}
		}
		if len(event.Instances) > 0 {
			converted.Endpoints = endpointsFromInstances(event.Instances, w.target)
		}
		w.events <- converted
	}
}

func endpointsFromInstances(instances []serviceregistry.Instance, target Target) []Endpoint {
	target = NormalizeTarget(target)
	endpoints := make([]Endpoint, 0)
	for _, instance := range instances {
		for _, endpoint := range instance.Endpoints {
			if target.Protocol != "" && !strings.EqualFold(endpoint.Protocol, target.Protocol) {
				continue
			}
			if target.EndpointName != "" && endpoint.Name != "" && !strings.EqualFold(endpoint.Name, target.EndpointName) {
				continue
			}
			endpoints = append(endpoints, NormalizeEndpoint(Endpoint{
				Name:       endpoint.Name,
				Protocol:   endpoint.Protocol,
				Host:       endpoint.Host,
				Port:       endpoint.Port,
				Path:       endpoint.Path,
				Weight:     endpoint.Weight,
				InstanceID: instance.InstanceID,
				Zone:       instance.Zone,
				Labels:     instance.Labels,
				Metadata:   instance.Metadata,
			}))
		}
	}
	return endpoints
}

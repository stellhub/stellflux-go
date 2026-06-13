package discovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

const (
	DefaultLoadBalance       = "round_robin"
	LoadBalanceRoundRobin    = "round_robin"
	LoadBalanceRandom        = "random"
	LoadBalanceWeightedRound = "weighted_round_robin"

	EventSnapshot = "snapshot"
	EventUpsert   = "upsert"
	EventDelete   = "delete"
)

var ErrNoAvailableEndpoint = errors.New("stellar: no available discovery endpoint")

type Target struct {
	Namespace    string
	Service      string
	Zone         string
	Protocol     string
	EndpointName string
	Labels       []string
	PassingOnly  bool
}

type Endpoint struct {
	Name       string
	Protocol   string
	Host       string
	Port       int
	Path       string
	Weight     int
	InstanceID string
	Zone       string
	Labels     map[string]string
	Metadata   map[string]string
}

func (e Endpoint) Address() string {
	if e.Port <= 0 {
		return strings.TrimSpace(e.Host)
	}
	return net.JoinHostPort(strings.TrimSpace(e.Host), fmt.Sprintf("%d", e.Port))
}

type Event struct {
	Type      string
	Endpoint  *Endpoint
	Endpoints []Endpoint
}

type Watcher interface {
	Events() <-chan Event
	Close() error
}

type Resolver interface {
	Resolve(context.Context, Target) ([]Endpoint, error)
	Watch(context.Context, Target) (Watcher, error)
	Close(context.Context) error
}

func NormalizeTarget(target Target) Target {
	target.Namespace = valueOrDefault(target.Namespace, "default")
	target.Service = strings.TrimSpace(target.Service)
	target.Zone = strings.TrimSpace(target.Zone)
	target.Protocol = strings.ToLower(valueOrDefault(target.Protocol, "http"))
	target.EndpointName = strings.TrimSpace(target.EndpointName)
	target.Labels = append([]string(nil), target.Labels...)
	return target
}

func NormalizeEndpoint(endpoint Endpoint) Endpoint {
	endpoint.Name = strings.TrimSpace(endpoint.Name)
	endpoint.Protocol = strings.ToLower(valueOrDefault(endpoint.Protocol, "http"))
	endpoint.Host = strings.TrimSpace(endpoint.Host)
	endpoint.Path = strings.TrimSpace(endpoint.Path)
	endpoint.InstanceID = strings.TrimSpace(endpoint.InstanceID)
	endpoint.Zone = strings.TrimSpace(endpoint.Zone)
	if endpoint.Weight <= 0 {
		endpoint.Weight = 100
	}
	endpoint.Labels = cloneStringMap(endpoint.Labels)
	endpoint.Metadata = cloneStringMap(endpoint.Metadata)
	return endpoint
}

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

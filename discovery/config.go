package discovery

import (
	"fmt"
	"strings"
	"time"

	"github.com/stellhub/stellar/config"
)

const (
	defaultRefreshInterval = 10 * time.Second
	defaultStaleTTL        = time.Minute
)

func FromRegistryConfig(registryCfg *config.RegistryConfig) *config.DiscoveryConfig {
	if registryCfg == nil {
		return nil
	}
	return normalizeDiscoveryConfig(&config.DiscoveryConfig{
		Enabled:         registryCfg.Enabled,
		Adapter:         registryCfg.Adapter,
		Endpoints:       append([]string(nil), registryCfg.Endpoints...),
		Endpoint:        registryCfg.Endpoint,
		Namespace:       registryCfg.Namespace,
		Group:           registryCfg.Group,
		Cluster:         registryCfg.Cluster,
		Zone:            registryCfg.Zone,
		Timeout:         registryCfg.Timeout,
		Username:        registryCfg.Username,
		Password:        registryCfg.Password,
		Token:           registryCfg.Token,
		Scheme:          registryCfg.Scheme,
		Datacenter:      registryCfg.Datacenter,
		Prefix:          registryCfg.Prefix,
		LoadBalance:     DefaultLoadBalance,
		RefreshInterval: defaultRefreshInterval.String(),
		StaleTTL:        defaultStaleTTL.String(),
		PassingOnly:     boolPtr(true),
		Labels:          cloneStringMap(registryCfg.Labels),
		Metadata:        cloneStringMap(registryCfg.Metadata),
	})
}

func MergeConfig(base *config.DiscoveryConfig, override *config.DiscoveryConfig) *config.DiscoveryConfig {
	if base == nil && override == nil {
		return nil
	}
	merged := config.DiscoveryConfig{}
	if base != nil {
		merged = *base
		merged.Endpoints = append([]string(nil), base.Endpoints...)
		merged.Labels = cloneStringMap(base.Labels)
		merged.Metadata = cloneStringMap(base.Metadata)
	}
	if override == nil {
		return normalizeDiscoveryConfig(&merged)
	}
	if override.Enabled != nil {
		merged.Enabled = override.Enabled
	}
	if strings.TrimSpace(override.Adapter) != "" {
		merged.Adapter = override.Adapter
	}
	if len(override.Endpoints) > 0 {
		merged.Endpoints = append([]string(nil), override.Endpoints...)
	}
	if strings.TrimSpace(override.Endpoint) != "" {
		merged.Endpoint = override.Endpoint
	}
	if strings.TrimSpace(override.Namespace) != "" {
		merged.Namespace = override.Namespace
	}
	if strings.TrimSpace(override.Group) != "" {
		merged.Group = override.Group
	}
	if strings.TrimSpace(override.Cluster) != "" {
		merged.Cluster = override.Cluster
	}
	if strings.TrimSpace(override.Service) != "" {
		merged.Service = override.Service
	}
	if strings.TrimSpace(override.Zone) != "" {
		merged.Zone = override.Zone
	}
	if strings.TrimSpace(override.Protocol) != "" {
		merged.Protocol = override.Protocol
	}
	if strings.TrimSpace(override.EndpointName) != "" {
		merged.EndpointName = override.EndpointName
	}
	if strings.TrimSpace(override.LoadBalance) != "" {
		merged.LoadBalance = override.LoadBalance
	}
	if strings.TrimSpace(override.RefreshInterval) != "" {
		merged.RefreshInterval = override.RefreshInterval
	}
	if strings.TrimSpace(override.StaleTTL) != "" {
		merged.StaleTTL = override.StaleTTL
	}
	if strings.TrimSpace(override.Timeout) != "" {
		merged.Timeout = override.Timeout
	}
	if strings.TrimSpace(override.Username) != "" {
		merged.Username = override.Username
	}
	if strings.TrimSpace(override.Password) != "" {
		merged.Password = override.Password
	}
	if strings.TrimSpace(override.Token) != "" {
		merged.Token = override.Token
	}
	if strings.TrimSpace(override.Scheme) != "" {
		merged.Scheme = override.Scheme
	}
	if strings.TrimSpace(override.Datacenter) != "" {
		merged.Datacenter = override.Datacenter
	}
	if strings.TrimSpace(override.Prefix) != "" {
		merged.Prefix = override.Prefix
	}
	if override.PassingOnly != nil {
		merged.PassingOnly = override.PassingOnly
	}
	if override.Labels != nil {
		merged.Labels = cloneStringMap(override.Labels)
	}
	if override.Metadata != nil {
		merged.Metadata = cloneStringMap(override.Metadata)
	}
	return normalizeDiscoveryConfig(&merged)
}

func normalizeDiscoveryConfig(value *config.DiscoveryConfig) *config.DiscoveryConfig {
	if value == nil {
		return nil
	}
	discovery := *value
	if strings.TrimSpace(discovery.Adapter) == "" {
		discovery.Adapter = "stellmap"
	}
	if strings.TrimSpace(discovery.Namespace) == "" {
		discovery.Namespace = "default"
	}
	if strings.TrimSpace(discovery.Endpoint) != "" && len(discovery.Endpoints) == 0 {
		discovery.Endpoints = []string{discovery.Endpoint}
	}
	if strings.TrimSpace(discovery.LoadBalance) == "" {
		discovery.LoadBalance = DefaultLoadBalance
	}
	if strings.TrimSpace(discovery.RefreshInterval) == "" {
		discovery.RefreshInterval = defaultRefreshInterval.String()
	}
	if strings.TrimSpace(discovery.StaleTTL) == "" {
		discovery.StaleTTL = defaultStaleTTL.String()
	}
	if discovery.Endpoints != nil {
		discovery.Endpoints = append([]string(nil), discovery.Endpoints...)
	}
	discovery.Labels = cloneStringMap(discovery.Labels)
	discovery.Metadata = cloneStringMap(discovery.Metadata)
	return &discovery
}

func HTTPConfigForNamed(clientCfg *config.HTTPClientConfig, named config.HTTPNamedClientConfig, name string) (*config.DiscoveryConfig, Target, bool, error) {
	staticConfigured := strings.TrimSpace(named.BaseURL) != ""
	return clientConfigForNamed(clientCfgDiscovery(clientCfg), named.Discovery, name, "http", staticConfigured)
}

func GRPCConfigForNamed(clientCfg *config.GRPCClientConfig, named config.GRPCNamedClientConfig, name string) (*config.DiscoveryConfig, Target, bool, error) {
	staticConfigured := strings.TrimSpace(named.Target) != ""
	return clientConfigForNamed(grpcClientCfgDiscovery(clientCfg), named.Discovery, name, "grpc", staticConfigured)
}

func clientConfigForNamed(base *config.DiscoveryConfig, named *config.DiscoveryConfig, name string, protocol string, staticConfigured bool) (*config.DiscoveryConfig, Target, bool, error) {
	var cfg *config.DiscoveryConfig
	if named != nil {
		if named.Enabled != nil && !*named.Enabled {
			return nil, Target{}, false, nil
		}
		cfg = MergeConfig(base, named)
	} else {
		if base == nil || (base.Enabled != nil && !*base.Enabled) || staticConfigured {
			return nil, Target{}, false, nil
		}
		cfg = MergeConfig(nil, base)
	}
	if cfg == nil {
		return nil, Target{}, false, nil
	}
	if strings.TrimSpace(cfg.Service) == "" {
		cfg.Service = name
	}
	if strings.TrimSpace(cfg.Protocol) == "" {
		cfg.Protocol = protocol
	}
	if strings.TrimSpace(cfg.EndpointName) == "" {
		cfg.EndpointName = cfg.Protocol
	}
	if strings.TrimSpace(cfg.Service) == "" {
		return nil, Target{}, false, fmt.Errorf("stellar: discovery service is required")
	}
	target := TargetFromConfig(cfg)
	return cfg, target, true, nil
}

func TargetFromConfig(cfg *config.DiscoveryConfig) Target {
	if cfg == nil {
		return Target{}
	}
	passingOnly := true
	if cfg.PassingOnly != nil {
		passingOnly = *cfg.PassingOnly
	}
	labels := make([]string, 0, len(cfg.Labels))
	for key, value := range cfg.Labels {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if strings.TrimSpace(value) == "" {
			labels = append(labels, strings.TrimSpace(key))
			continue
		}
		labels = append(labels, strings.TrimSpace(key)+"="+value)
	}
	return NormalizeTarget(Target{
		Namespace:    cfg.Namespace,
		Service:      cfg.Service,
		Zone:         cfg.Zone,
		Protocol:     cfg.Protocol,
		EndpointName: cfg.EndpointName,
		Labels:       labels,
		PassingOnly:  passingOnly,
	})
}

func RegistryConfigFromDiscovery(cfg *config.DiscoveryConfig) *config.RegistryConfig {
	if cfg == nil {
		return nil
	}
	return &config.RegistryConfig{
		Enabled:    cfg.Enabled,
		Adapter:    cfg.Adapter,
		Endpoints:  append([]string(nil), cfg.Endpoints...),
		Endpoint:   cfg.Endpoint,
		Namespace:  cfg.Namespace,
		Group:      cfg.Group,
		Cluster:    cfg.Cluster,
		Service:    cfg.Service,
		Zone:       cfg.Zone,
		Username:   cfg.Username,
		Password:   cfg.Password,
		Token:      cfg.Token,
		Scheme:     cfg.Scheme,
		Datacenter: cfg.Datacenter,
		Prefix:     cfg.Prefix,
		Timeout:    cfg.Timeout,
		Labels:     cloneStringMap(cfg.Labels),
		Metadata:   cloneStringMap(cfg.Metadata),
	}
}

func RefreshInterval(cfg *config.DiscoveryConfig) (time.Duration, error) {
	return durationFromConfig(cfg, "refresh_interval", defaultRefreshInterval)
}

func StaleTTL(cfg *config.DiscoveryConfig) (time.Duration, error) {
	return durationFromConfig(cfg, "stale_ttl", defaultStaleTTL)
}

func durationFromConfig(cfg *config.DiscoveryConfig, name string, fallback time.Duration) (time.Duration, error) {
	if cfg == nil {
		return fallback, nil
	}
	value := ""
	switch name {
	case "refresh_interval":
		value = cfg.RefreshInterval
	case "stale_ttl":
		value = cfg.StaleTTL
	}
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("stellar: invalid discovery %s %q: %w", name, value, err)
	}
	if duration <= 0 {
		return fallback, nil
	}
	return duration, nil
}

func clientCfgDiscovery(cfg *config.HTTPClientConfig) *config.DiscoveryConfig {
	if cfg == nil {
		return nil
	}
	return cfg.Discovery
}

func grpcClientCfgDiscovery(cfg *config.GRPCClientConfig) *config.DiscoveryConfig {
	if cfg == nil {
		return nil
	}
	return cfg.Discovery
}

func boolPtr(value bool) *bool {
	return &value
}

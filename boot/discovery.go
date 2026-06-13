package boot

import (
	"github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/discovery"
)

func httpClientConfigWithDiscovery(cfg config.Config) *config.HTTPClientConfig {
	if cfg.HTTP.Client == nil {
		return nil
	}
	client := *cfg.HTTP.Client
	if client.Clients != nil {
		clients := make(map[string]config.HTTPNamedClientConfig, len(client.Clients))
		for key, value := range client.Clients {
			clients[key] = value
		}
		client.Clients = clients
	}
	if client.Discovery == nil {
		client.Discovery = inheritedDiscoveryConfig(cfg)
	}
	return &client
}

func grpcClientConfigWithDiscovery(cfg config.Config) *config.GRPCClientConfig {
	if cfg.GRPC.Client == nil {
		return nil
	}
	client := *cfg.GRPC.Client
	if client.Clients != nil {
		clients := make(map[string]config.GRPCNamedClientConfig, len(client.Clients))
		for key, value := range client.Clients {
			clients[key] = value
		}
		client.Clients = clients
	}
	if client.Discovery == nil {
		client.Discovery = inheritedDiscoveryConfig(cfg)
	}
	return &client
}

func inheritedDiscoveryConfig(cfg config.Config) *config.DiscoveryConfig {
	if cfg.Discovery != nil {
		return discovery.MergeConfig(nil, cfg.Discovery)
	}
	return discovery.FromRegistryConfig(cfg.Registry)
}

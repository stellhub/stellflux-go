package http

import (
	"fmt"
	stdhttp "net/http"
	"time"

	stellarconfig "github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/observability"
)

type ClientOption func(*clientConfig)

type clientConfig struct {
	base       *stdhttp.Client
	transport  stdhttp.RoundTripper
	observer   *observability.Provider
	timeout    time.Duration
	copyClient bool
}

func WithClient(base *stdhttp.Client) ClientOption {
	return func(cfg *clientConfig) {
		if base != nil {
			cfg.base = base
			cfg.copyClient = true
		}
	}
}

func WithTransport(transport stdhttp.RoundTripper) ClientOption {
	return func(cfg *clientConfig) {
		if transport != nil {
			cfg.transport = transport
		}
	}
}

func WithClientTimeout(timeout time.Duration) ClientOption {
	return func(cfg *clientConfig) {
		cfg.timeout = timeout
	}
}

func WithClientObservability(provider *observability.Provider) ClientOption {
	return func(cfg *clientConfig) {
		if provider != nil {
			cfg.observer = provider
		}
	}
}

func NewClient(options ...ClientOption) *stdhttp.Client {
	cfg := clientConfig{
		base:     stdhttp.DefaultClient,
		observer: observability.New(),
	}
	for _, option := range options {
		option(&cfg)
	}

	client := cfg.base
	if cfg.copyClient {
		copied := *cfg.base
		client = &copied
	} else {
		client = &stdhttp.Client{}
	}
	if cfg.timeout > 0 {
		client.Timeout = cfg.timeout
	}
	baseTransport := cfg.transport
	if baseTransport == nil && cfg.base != nil {
		baseTransport = cfg.base.Transport
	}
	if baseTransport == nil {
		baseTransport = stdhttp.DefaultTransport
	}
	client.Transport = cfg.observer.HTTPClientTransport(baseTransport)
	return client
}

func NewClientFromConfig(cfg *stellarconfig.HTTPClientConfig, provider *observability.Provider, options ...ClientOption) (*stdhttp.Client, error) {
	client, _, err := NewNamedClientFromConfig(cfg, "", provider, options...)
	return client, err
}

func NewNamedClientFromConfig(cfg *stellarconfig.HTTPClientConfig, name string, provider *observability.Provider, options ...ClientOption) (*stdhttp.Client, string, error) {
	if cfg != nil && cfg.Enabled != nil && !*cfg.Enabled {
		return nil, "", fmt.Errorf("stellar: http client is disabled")
	}

	named := stellarconfig.HTTPNamedClientConfig{}
	if name != "" {
		if cfg == nil || cfg.Clients == nil {
			return nil, "", fmt.Errorf("stellar: http client %q is not configured", name)
		}
		var ok bool
		named, ok = cfg.Clients[name]
		if !ok {
			return nil, "", fmt.Errorf("stellar: http client %q is not configured", name)
		}
	}

	cfgOptions, err := clientOptionsFromConfig(cfg, named)
	if err != nil {
		return nil, "", err
	}
	if provider != nil {
		cfgOptions = append(cfgOptions, WithClientObservability(provider))
	}
	cfgOptions = append(cfgOptions, options...)
	return NewClient(cfgOptions...), named.BaseURL, nil
}

func clientOptionsFromConfig(cfg *stellarconfig.HTTPClientConfig, named stellarconfig.HTTPNamedClientConfig) ([]ClientOption, error) {
	if cfg == nil {
		return nil, nil
	}

	options := make([]ClientOption, 0, 2)
	timeoutValue := cfg.Timeout
	if named.Timeout != "" {
		timeoutValue = named.Timeout
	}
	if timeoutValue != "" {
		timeout, err := time.ParseDuration(timeoutValue)
		if err != nil {
			return nil, fmt.Errorf("stellar: invalid http client timeout %q: %w", timeoutValue, err)
		}
		options = append(options, WithClientTimeout(timeout))
	}

	transport, err := transportFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	if transport != nil {
		options = append(options, WithTransport(transport))
	}
	return options, nil
}

func transportFromConfig(cfg *stellarconfig.HTTPClientConfig) (stdhttp.RoundTripper, error) {
	if cfg.MaxIdleConns <= 0 && cfg.MaxIdleConnsPerHost <= 0 && cfg.IdleConnTimeout == "" {
		return nil, nil
	}

	baseTransport, ok := stdhttp.DefaultTransport.(*stdhttp.Transport)
	if !ok {
		return stdhttp.DefaultTransport, nil
	}
	transport := baseTransport.Clone()
	if cfg.MaxIdleConns > 0 {
		transport.MaxIdleConns = cfg.MaxIdleConns
	}
	if cfg.MaxIdleConnsPerHost > 0 {
		transport.MaxIdleConnsPerHost = cfg.MaxIdleConnsPerHost
	}
	if cfg.IdleConnTimeout != "" {
		timeout, err := time.ParseDuration(cfg.IdleConnTimeout)
		if err != nil {
			return nil, fmt.Errorf("stellar: invalid http client idle_conn_timeout %q: %w", cfg.IdleConnTimeout, err)
		}
		transport.IdleConnTimeout = timeout
	}
	return transport, nil
}

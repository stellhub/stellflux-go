package http

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"strings"
	"time"

	stellarconfig "github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/discovery"
	"github.com/stellhub/stellar/interceptor"
	"github.com/stellhub/stellar/observability"
)

type ClientOption func(*clientConfig)

type clientConfig struct {
	base         *stdhttp.Client
	transport    stdhttp.RoundTripper
	observer     *observability.Provider
	interceptors *interceptor.Registry
	discovery    *discovery.CachedResolver
	picker       discovery.Picker
	clientName   string
	timeout      time.Duration
	copyClient   bool
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

func WithClientInterceptors(registry *interceptor.Registry) ClientOption {
	return func(cfg *clientConfig) {
		cfg.interceptors = registry
	}
}

func WithClientName(name string) ClientOption {
	return func(cfg *clientConfig) {
		cfg.clientName = name
	}
}

func WithClientDiscovery(resolver *discovery.CachedResolver, picker discovery.Picker) ClientOption {
	return func(cfg *clientConfig) {
		if resolver != nil {
			cfg.discovery = resolver
			cfg.picker = picker
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
	if cfg.discovery != nil {
		baseTransport = &discoveryRoundTripper{
			base:     baseTransport,
			resolver: cfg.discovery,
			picker:   cfg.picker,
		}
	}
	if cfg.interceptors != nil {
		baseTransport = &interceptorRoundTripper{
			base:       baseTransport,
			registry:   cfg.interceptors,
			clientName: cfg.clientName,
		}
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
	if name != "" {
		cfgOptions = append(cfgOptions, WithClientName(name))
	}
	if name != "" {
		if discoveryCfg, target, ok, err := discovery.HTTPConfigForNamed(cfg, named, name); err != nil {
			return nil, "", err
		} else if ok {
			resolver, err := discovery.NewCachedResolverFromConfig(context.Background(), discoveryCfg, target)
			if err != nil {
				return nil, "", err
			}
			cfgOptions = append(cfgOptions, WithClientDiscovery(resolver, discovery.NewPicker(discoveryCfg.LoadBalance)))
			if strings.TrimSpace(named.BaseURL) == "" {
				named.BaseURL = logicalHTTPBaseURL(target)
			}
		}
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

type discoveryRoundTripper struct {
	base     stdhttp.RoundTripper
	resolver *discovery.CachedResolver
	picker   discovery.Picker
}

func (t *discoveryRoundTripper) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	base := t.base
	if base == nil {
		base = stdhttp.DefaultTransport
	}
	if t.resolver == nil {
		return base.RoundTrip(req)
	}
	endpoint, err := t.resolver.Pick(req.Context(), t.picker)
	if err != nil {
		return nil, err
	}
	next := req.Clone(req.Context())
	next.URL = cloneURL(req.URL)
	rewriteHTTPURL(next.URL, endpoint)
	next.Host = next.URL.Host
	return base.RoundTrip(next)
}

func logicalHTTPBaseURL(target discovery.Target) string {
	service := strings.TrimSpace(target.Service)
	if service == "" {
		service = "service"
	}
	return "http://" + service
}

func rewriteHTTPURL(value *url.URL, endpoint discovery.Endpoint) {
	scheme := "http"
	switch strings.ToLower(endpoint.Protocol) {
	case "https":
		scheme = "https"
	}
	value.Scheme = scheme
	value.Host = endpoint.Address()
	if endpoint.Path != "" && endpoint.Path != "/" {
		value.Path = joinURLPath(endpoint.Path, value.Path)
	}
}

func cloneURL(value *url.URL) *url.URL {
	if value == nil {
		return &url.URL{}
	}
	copied := *value
	return &copied
}

func joinURLPath(prefix string, path string) string {
	prefix = strings.TrimRight(prefix, "/")
	path = strings.TrimLeft(path, "/")
	if prefix == "" {
		return "/" + path
	}
	if path == "" {
		return prefix
	}
	return prefix + "/" + path
}

type interceptorRoundTripper struct {
	base       stdhttp.RoundTripper
	registry   *interceptor.Registry
	clientName string
}

func (t *interceptorRoundTripper) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	base := t.base
	if base == nil {
		base = stdhttp.DefaultTransport
	}
	if t.registry == nil {
		return base.RoundTrip(req)
	}
	inv := &interceptor.Invocation{
		Kind:      interceptor.KindHTTPClient,
		Protocol:  "http",
		Service:   t.clientName,
		Operation: req.Method + " " + req.URL.Path,
		Method:    req.Method,
		Path:      req.URL.Path,
		Target:    req.URL.String(),
		Headers:   interceptor.HeaderFromHTTP(req.Header),
		Raw:       req,
	}
	handler := t.registry.Chain(interceptor.KindHTTPClient, func(ctx context.Context, _ *interceptor.Invocation, payload any) (any, error) {
		request, ok := payload.(*stdhttp.Request)
		if !ok {
			return nil, fmt.Errorf("stellar: unexpected HTTP client request type %T", payload)
		}
		return base.RoundTrip(request.WithContext(ctx))
	})
	resp, err := handler(req.Context(), inv, req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	httpResp, ok := resp.(*stdhttp.Response)
	if !ok {
		return nil, fmt.Errorf("stellar: unexpected HTTP client response type %T", resp)
	}
	return httpResp, nil
}

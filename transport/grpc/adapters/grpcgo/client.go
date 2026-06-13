package grpcgoadapter

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	stellarconfig "github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/discovery"
	"github.com/stellhub/stellar/interceptor"
	"github.com/stellhub/stellar/observability"
	stellargrpc "github.com/stellhub/stellar/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	grpcresolver "google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
)

type ClientOption func(*clientConfig)

type clientConfig struct {
	dialOptions    []grpc.DialOption
	observer       *observability.Provider
	interceptors   *interceptor.Registry
	defaultTimeout time.Duration
}

func WithDialOption(options ...grpc.DialOption) ClientOption {
	return func(cfg *clientConfig) {
		cfg.dialOptions = append(cfg.dialOptions, options...)
	}
}

func WithClientObservability(provider *observability.Provider) ClientOption {
	return func(cfg *clientConfig) {
		if provider != nil {
			cfg.observer = provider
		}
	}
}

func WithInterceptors(registry *interceptor.Registry) ClientOption {
	return func(cfg *clientConfig) {
		cfg.interceptors = registry
	}
}

func WithDefaultTimeout(timeout time.Duration) ClientOption {
	return func(cfg *clientConfig) {
		if timeout > 0 {
			cfg.defaultTimeout = timeout
		}
	}
}

func WithInsecure() ClientOption {
	return WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func NewClientConn(ctx context.Context, target string, options ...ClientOption) (*grpc.ClientConn, error) {
	cfg := clientConfig{
		observer: observability.New(),
	}
	for _, option := range options {
		option(&cfg)
	}

	dialOptions := []grpc.DialOption{
		grpc.WithStatsHandler(cfg.observer.GRPCClientStatsHandler()),
	}
	unaryInterceptors := []grpc.UnaryClientInterceptor{cfg.observer.UnaryClientInterceptor()}
	streamInterceptors := []grpc.StreamClientInterceptor{cfg.observer.StreamClientInterceptor()}
	if cfg.defaultTimeout > 0 {
		unaryInterceptors = append(unaryInterceptors, defaultTimeoutUnaryInterceptor(cfg.defaultTimeout))
		streamInterceptors = append(streamInterceptors, defaultTimeoutStreamInterceptor(cfg.defaultTimeout))
	}
	if cfg.interceptors != nil {
		unaryInterceptors = append(unaryInterceptors, unaryClientInterceptor(cfg.interceptors))
		streamInterceptors = append(streamInterceptors, streamClientInterceptor(cfg.interceptors))
	}
	dialOptions = append(dialOptions,
		grpc.WithChainUnaryInterceptor(unaryInterceptors...),
		grpc.WithChainStreamInterceptor(streamInterceptors...),
	)
	dialOptions = append(dialOptions, cfg.dialOptions...)
	conn, err := grpc.NewClient(target, dialOptions...)
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		conn.Connect()
	}
	return conn, nil
}

func NewClientConnFromConfig(ctx context.Context, cfg *stellarconfig.GRPCClientConfig, provider *observability.Provider, options ...ClientOption) (*grpc.ClientConn, error) {
	conn, _, err := NewNamedClientConnFromConfig(ctx, cfg, "", provider, options...)
	return conn, err
}

func NewNamedClientConnFromConfig(ctx context.Context, cfg *stellarconfig.GRPCClientConfig, name string, provider *observability.Provider, options ...ClientOption) (*grpc.ClientConn, string, error) {
	if cfg != nil && cfg.Enabled != nil && !*cfg.Enabled {
		return nil, "", fmt.Errorf("stellar: grpc client is disabled")
	}

	named := stellarconfig.GRPCNamedClientConfig{}
	if name != "" {
		if cfg == nil || cfg.Clients == nil {
			return nil, "", fmt.Errorf("stellar: grpc client %q is not configured", name)
		}
		var ok bool
		named, ok = cfg.Clients[name]
		if !ok {
			return nil, "", fmt.Errorf("stellar: grpc client %q is not configured", name)
		}
	}

	target, cfgOptions, err := clientOptionsFromConfig(cfg, named)
	if err != nil {
		return nil, "", err
	}
	if name != "" {
		if discoveryCfg, discoveryTarget, ok, err := discovery.GRPCConfigForNamed(cfg, named, name); err != nil {
			return nil, "", err
		} else if ok {
			discoveredTarget, discoveryOptions, err := discoveryDialOptions(ctx, discoveryCfg, discoveryTarget)
			if err != nil {
				return nil, "", err
			}
			target = discoveredTarget
			cfgOptions = append(cfgOptions, discoveryOptions...)
		}
	}
	if strings.TrimSpace(target) == "" {
		return nil, "", fmt.Errorf("stellar: grpc client target is required")
	}
	if provider != nil {
		cfgOptions = append(cfgOptions, WithClientObservability(provider))
	}
	cfgOptions = append(cfgOptions, options...)
	conn, err := NewClientConn(ctx, target, cfgOptions...)
	if err != nil {
		return nil, "", err
	}
	return conn, target, nil
}

type ClientFactory struct {
	observer     *observability.Provider
	interceptors *interceptor.Registry
	options      []grpc.DialOption
}

func NewClientFactory(options ...ClientOption) *ClientFactory {
	cfg := clientConfig{
		observer: observability.New(),
	}
	for _, option := range options {
		option(&cfg)
	}
	return &ClientFactory{
		observer:     cfg.observer,
		interceptors: cfg.interceptors,
		options:      cfg.dialOptions,
	}
}

func (f *ClientFactory) NewClient(ctx context.Context, target string, _ ...stellargrpc.ClientOption) (any, error) {
	return NewClientConn(ctx, target, WithClientObservability(f.observer), WithInterceptors(f.interceptors), WithDialOption(f.options...))
}

func clientOptionsFromConfig(cfg *stellarconfig.GRPCClientConfig, named stellarconfig.GRPCNamedClientConfig) (string, []ClientOption, error) {
	if cfg == nil {
		return "", nil, nil
	}

	target := cfg.Target
	timeoutValue := cfg.Timeout
	authority := cfg.Authority
	insecureEnabled := boolValue(cfg.Insecure, true)

	if named.Target != "" {
		target = named.Target
	}
	if named.Timeout != "" {
		timeoutValue = named.Timeout
	}
	if named.Authority != "" {
		authority = named.Authority
	}
	if named.Insecure != nil {
		insecureEnabled = *named.Insecure
	}

	options := make([]ClientOption, 0, 3)
	if timeoutValue != "" {
		timeout, err := time.ParseDuration(timeoutValue)
		if err != nil {
			return "", nil, fmt.Errorf("stellar: invalid grpc client timeout %q: %w", timeoutValue, err)
		}
		options = append(options, WithDefaultTimeout(timeout))
	}
	if authority != "" {
		options = append(options, WithDialOption(grpc.WithAuthority(authority)))
	}
	if insecureEnabled {
		options = append(options, WithInsecure())
	}
	return target, options, nil
}

func defaultTimeoutUnaryInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req any, reply any, conn *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := contextWithDefaultTimeout(ctx, timeout)
		defer cancel()
		return invoker(ctx, method, req, reply, conn, opts...)
	}
}

func defaultTimeoutStreamInterceptor(timeout time.Duration) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, conn *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx, _ = contextWithDefaultTimeout(ctx, timeout)
		return streamer(ctx, desc, conn, method, opts...)
	}
}

func contextWithDefaultTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		return ctx, func() {}
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func unaryClientInterceptor(registry *interceptor.Registry) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req any, reply any, conn *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		inv := grpcClientInvocation(ctx, method, conn)
		chain := registry.Chain(interceptor.KindGRPCClient, func(ctx context.Context, _ *interceptor.Invocation, payload any) (any, error) {
			return nil, invoker(ctx, method, payload, reply, conn, opts...)
		})
		_, err := chain(ctx, inv, req)
		return err
	}
}

func streamClientInterceptor(registry *interceptor.Registry) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, conn *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		inv := grpcClientInvocation(ctx, method, conn)
		chain := registry.Chain(interceptor.KindGRPCClient, func(ctx context.Context, _ *interceptor.Invocation, payload any) (any, error) {
			return streamer(ctx, desc, conn, method, opts...)
		})
		stream, err := chain(ctx, inv, nil)
		if err != nil {
			return nil, err
		}
		clientStream, ok := stream.(grpc.ClientStream)
		if !ok {
			return nil, fmt.Errorf("stellar: unexpected grpc client stream type %T", stream)
		}
		return clientStream, nil
	}
}

func grpcClientInvocation(ctx context.Context, fullMethod string, conn *grpc.ClientConn) *interceptor.Invocation {
	service, method := splitFullMethod(fullMethod)
	target := ""
	if conn != nil {
		target = conn.Target()
	}
	return &interceptor.Invocation{
		Kind:      interceptor.KindGRPCClient,
		Protocol:  "grpc",
		Service:   service,
		Operation: fullMethod,
		Method:    method,
		Path:      fullMethod,
		Target:    target,
		Headers:   headersFromOutgoingContext(ctx),
		Raw:       conn,
	}
}

var discoveryResolverID atomic.Uint64

func discoveryDialOptions(ctx context.Context, cfg *stellarconfig.DiscoveryConfig, target discovery.Target) (string, []ClientOption, error) {
	cached, err := discovery.NewCachedResolverFromConfig(ctx, cfg, target)
	if err != nil {
		return "", nil, err
	}
	endpoints, err := cached.Resolve(ctx, target)
	if err != nil {
		_ = cached.Close(context.Background())
		return "", nil, err
	}
	scheme := fmt.Sprintf("stellar-discovery-%d", discoveryResolverID.Add(1))
	builder := manual.NewBuilderWithScheme(scheme)
	builder.InitialState(grpcResolverState(endpoints))

	updateCtx, cancel := context.WithCancel(context.Background())
	builder.CloseCallback = func() {
		cancel()
		_ = cached.Close(context.Background())
	}
	refreshInterval, err := discovery.RefreshInterval(cfg)
	if err != nil {
		cancel()
		_ = cached.Close(context.Background())
		return "", nil, err
	}
	go updateGRPCResolverLoop(updateCtx, cached, target, builder, refreshInterval)

	serviceConfig := fmt.Sprintf(`{"loadBalancingConfig":[{"%s":{}}]}`, roundrobin.Name)
	return builder.Scheme() + ":///" + target.Service, []ClientOption{
		WithDialOption(
			grpc.WithResolvers(builder),
			grpc.WithDefaultServiceConfig(serviceConfig),
		),
	}, nil
}

func updateGRPCResolverLoop(ctx context.Context, cached *discovery.CachedResolver, target discovery.Target, builder *manual.Resolver, interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			endpoints, err := cached.Resolve(ctx, target)
			if err == nil {
				builder.UpdateState(grpcResolverState(endpoints))
			}
		case <-ctx.Done():
			return
		}
	}
}

func grpcResolverState(endpoints []discovery.Endpoint) grpcresolver.State {
	addresses := make([]grpcresolver.Address, 0, len(endpoints))
	for _, endpoint := range endpoints {
		address := endpoint.Address()
		if strings.TrimSpace(address) == "" {
			continue
		}
		addresses = append(addresses, grpcresolver.Address{Addr: address})
	}
	return grpcresolver.State{Addresses: addresses}
}

func headersFromOutgoingContext(ctx context.Context) interceptor.Header {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return nil
	}
	headers := make(interceptor.Header, len(md))
	for key, values := range md {
		headers[key] = append([]string(nil), values...)
	}
	return headers
}

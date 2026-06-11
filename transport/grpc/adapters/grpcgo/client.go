package grpcgoadapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	stellarconfig "github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/observability"
	stellargrpc "github.com/stellhub/stellar/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientOption func(*clientConfig)

type clientConfig struct {
	dialOptions    []grpc.DialOption
	observer       *observability.Provider
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

	dialOptions := append(cfg.observer.GRPCClientOptions(), cfg.dialOptions...)
	if cfg.defaultTimeout > 0 {
		dialOptions = append(dialOptions,
			grpc.WithChainUnaryInterceptor(defaultTimeoutUnaryInterceptor(cfg.defaultTimeout)),
			grpc.WithChainStreamInterceptor(defaultTimeoutStreamInterceptor(cfg.defaultTimeout)),
		)
	}
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
	observer *observability.Provider
	options  []grpc.DialOption
}

func NewClientFactory(options ...ClientOption) *ClientFactory {
	cfg := clientConfig{
		observer: observability.New(),
	}
	for _, option := range options {
		option(&cfg)
	}
	return &ClientFactory{
		observer: cfg.observer,
		options:  cfg.dialOptions,
	}
}

func (f *ClientFactory) NewClient(ctx context.Context, target string, _ ...stellargrpc.ClientOption) (any, error) {
	return NewClientConn(ctx, target, WithClientObservability(f.observer), WithDialOption(f.options...))
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

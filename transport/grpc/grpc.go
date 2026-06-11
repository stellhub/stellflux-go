package grpc

import "context"

const TransportName = "grpc"

type Adapter interface {
	Name() string
	Register(Service) error
	Start(context.Context) error
	Stop(context.Context) error
}

type AddrSetter interface {
	SetAddr(string)
}

type Service struct {
	Description    any
	Implementation any
}

type ClientFactory interface {
	NewClient(ctx context.Context, target string, opts ...ClientOption) (any, error)
}

type ClientOption func(*ClientConfig)

type ClientConfig struct {
	Authority string
	Headers   map[string]string
}

func WithAuthority(authority string) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.Authority = authority
	}
}

func WithHeader(key string, value string) ClientOption {
	return func(cfg *ClientConfig) {
		if cfg.Headers == nil {
			cfg.Headers = map[string]string{}
		}
		cfg.Headers[key] = value
	}
}

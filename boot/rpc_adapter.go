package boot

import (
	"context"

	bootgrpc "github.com/stellhub/stellar/transport/grpc"
	grpcgoadapter "github.com/stellhub/stellar/transport/grpc/adapters/grpcgo"
	"google.golang.org/grpc"
)

func WithRPCAdapter(adapter bootgrpc.Adapter) Option {
	return func(app *App) {
		app.setRPCAdapter(adapter, true)
	}
}

func WithRPCServer(addr string) Option {
	return func(app *App) {
		adapter := app.rpcAdapter
		if adapter == nil {
			adapter = grpcgoadapter.New(addr)
		}
		if setter, ok := adapter.(bootgrpc.AddrSetter); ok {
			setter.SetAddr(addr)
		}
		app.setRPCAdapter(adapter, true)
	}
}

func (a *App) NewGRPCClient(ctx context.Context, name string) (*grpc.ClientConn, string, error) {
	cfg := a.Config()
	return grpcgoadapter.NewNamedClientConnFromConfig(ctx, cfg.GRPC.Client, name, a.observability)
}

func (a *App) setRPCAdapter(adapter bootgrpc.Adapter, registerTransport bool) {
	if adapter == nil {
		return
	}
	if consumer, ok := adapter.(observabilityConsumer); ok {
		consumer.UseObservability(a.observability)
	}
	a.rpcAdapter = adapter
	if registerTransport {
		a.addTransport(adapter)
	}
}

package grpcgoadapter

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/stellhub/stellar/observability"
	stellargrpc "github.com/stellhub/stellar/transport/grpc"
	"google.golang.org/grpc"
)

const Name = "rpc-grpc-go"

type Adapter struct {
	addr     string
	server   *grpc.Server
	options  []grpc.ServerOption
	observer *observability.Provider
	services []stellargrpc.Service
	listener net.Listener
	errCh    chan error
	mu       sync.Mutex
}

type Option func(*Adapter)

func New(addr string, opts ...Option) *Adapter {
	if addr == "" {
		addr = ":9090"
	}
	adapter := &Adapter{
		addr:     addr,
		observer: observability.New(),
		errCh:    make(chan error, 1),
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

func WithServerOption(options ...grpc.ServerOption) Option {
	return func(adapter *Adapter) {
		adapter.options = append(adapter.options, options...)
	}
}

func (a *Adapter) Name() string {
	return Name
}

func (a *Adapter) Addr() string {
	return a.addr
}

func (a *Adapter) SetAddr(addr string) {
	if addr != "" {
		a.addr = addr
	}
}

func (a *Adapter) UseObservability(provider *observability.Provider) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if provider != nil {
		a.observer = provider
	}
}

func (a *Adapter) Register(service stellargrpc.Service) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if service.Description == nil || service.Implementation == nil {
		return fmt.Errorf("grpc-go: service description and implementation are required")
	}
	desc, ok := service.Description.(*grpc.ServiceDesc)
	if !ok {
		return fmt.Errorf("grpc-go: service description must be *grpc.ServiceDesc")
	}
	if a.server != nil {
		a.server.RegisterService(desc, service.Implementation)
	}
	a.services = append(a.services, service)
	return nil
}

func (a *Adapter) Server() *grpc.Server {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.server == nil {
		a.buildServerLocked()
	}
	return a.server
}

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil {
		a.buildServerLocked()
	}

	listener, err := net.Listen("tcp", a.addr)
	if err != nil {
		return err
	}
	a.listener = listener

	go func() {
		if err := a.server.Serve(listener); err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
			a.errCh <- err
		}
	}()

	select {
	case err := <-a.errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.server == nil {
		return nil
	}

	done := make(chan struct{})
	go func() {
		a.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		a.server.Stop()
		return ctx.Err()
	}
}

func (a *Adapter) buildServerLocked() {
	options := append(a.observer.GRPCServerOptions(), a.options...)
	a.server = grpc.NewServer(options...)
	for _, service := range a.services {
		desc := service.Description.(*grpc.ServiceDesc)
		a.server.RegisterService(desc, service.Implementation)
	}
}

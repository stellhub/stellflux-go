package internal

import (
	"context"
	"errors"

	"github.com/stellhub/stellar"
	stellargrpc "github.com/stellhub/stellar/transport/grpc"
)

type customRouterStarter struct{}

func NewCustomRouterStarter() *customRouterStarter {
	return &customRouterStarter{}
}

func (s *customRouterStarter) Name() string {
	return "grpc-custom-router-example"
}

func (s *customRouterStarter) Condition(stellar.StarterContext) bool {
	return true
}

func (s *customRouterStarter) Init(_ context.Context, app *stellar.App) error {
	rpc := app.RPC()
	if rpc == nil {
		return errors.New("grpc custom router example requires grpc.server.enabled=true")
	}

	return rpc.Register(stellargrpc.Service{
		Description:    customRouterServiceDesc(),
		Implementation: &customRouterService{},
	})
}

func (s *customRouterStarter) Start(context.Context) error {
	return nil
}

func (s *customRouterStarter) Stop(context.Context) error {
	return nil
}

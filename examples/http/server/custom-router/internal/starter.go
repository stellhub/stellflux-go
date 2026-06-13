package internal

import (
	"context"
	"net/http"

	"github.com/stellhub/stellar"
	stellarhttp "github.com/stellhub/stellar/transport/http"
)

type customRouterStarter struct{}

func NewCustomRouterStarter() *customRouterStarter {
	return &customRouterStarter{}
}

func (s *customRouterStarter) Name() string {
	return "http-custom-router-example"
}

func (s *customRouterStarter) Condition(stellar.StarterContext) bool {
	return true
}

func (s *customRouterStarter) Init(_ context.Context, app *stellar.App) error {
	api := app.HTTP().Group("/api/v1")

	api.GET("/ping", handlePing)
	api.GET("/hello", handleHello)
	stellarhttp.Handle(
		api,
		http.MethodPost,
		"/items",
		stellarhttp.JSONBinder[createItemRequest](),
		createItem,
		stellarhttp.JSONEncoder[createItemResponse],
	)

	return nil
}

func (s *customRouterStarter) Start(context.Context) error {
	return nil
}

func (s *customRouterStarter) Stop(context.Context) error {
	return nil
}

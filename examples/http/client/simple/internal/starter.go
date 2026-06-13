package internal

import (
	"context"
	"log/slog"
	"time"

	"github.com/stellhub/stellar"
	stellarhttp "github.com/stellhub/stellar/transport/http"
)

const mockUserServiceAddr = "127.0.0.1:18085"

type clientStarter struct {
	config stellar.Config
	logger *slog.Logger
	client *UserClient
	mock   *MockUserService
}

func NewClientStarter() *clientStarter {
	return &clientStarter{}
}

func (s *clientStarter) Name() string {
	return "http-client-simple-example"
}

func (s *clientStarter) Condition(ctx stellar.StarterContext) bool {
	s.config = ctx.Config()
	return true
}

func (s *clientStarter) Init(_ context.Context, app *stellar.App) error {
	s.logger = app.Logger()

	httpClient, baseURL, err := stellarhttp.NewNamedClientFromConfig(s.config.HTTP.Client, "user-service", app.Observability())
	if err != nil {
		return err
	}
	userClient, err := newUserClient(httpClient, baseURL)
	if err != nil {
		return err
	}

	s.client = userClient
	s.mock = NewMockUserService(mockUserServiceAddr, s.logger)
	return nil
}

func (s *clientStarter) Start(ctx context.Context) error {
	if err := s.mock.Start(ctx); err != nil {
		return err
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.mock.Stop(stopCtx); err != nil {
			s.logger.Error("mock user service shutdown failed", "error", err)
		}
	}()

	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user, err := s.client.GetUser(requestCtx, "42")
	if err != nil {
		return err
	}
	s.logger.Info("user loaded",
		"id", user.ID,
		"name", user.Name,
		"source", user.Source,
	)
	return nil
}

func (s *clientStarter) Stop(context.Context) error {
	return nil
}

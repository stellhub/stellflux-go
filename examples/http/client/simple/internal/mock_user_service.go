package internal

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
)

type MockUserService struct {
	addr   string
	logger *slog.Logger
	server *http.Server
}

func NewMockUserService(addr string, logger *slog.Logger) *MockUserService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MockUserService{
		addr:   addr,
		logger: logger,
	}
}

func (s *MockUserService) Start(context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/users/42", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(User{
			ID:     "42",
			Name:   "Stellar User",
			Source: "mock-user-service",
		})
	})

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.server = &http.Server{
		Handler: mux,
	}
	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("mock user service stopped", "error", err)
		}
	}()
	s.logger.Info("mock user service started", "addr", s.addr)
	return nil
}

func (s *MockUserService) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

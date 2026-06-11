package http_test

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stellhub/stellar/middleware"
	stellarhttp "github.com/stellhub/stellar/transport/http"
	ginadapter "github.com/stellhub/stellar/transport/http/adapters/gin"
)

type pingRequest struct{}

type pingResponse struct {
	Message string `json:"message"`
}

func TestTypedRouteAndMiddleware(t *testing.T) {
	router := stellarhttp.NewRouter()
	order := make([]string, 0, 2)
	first := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			order = append(order, "first")
			return next(ctx, req)
		}
	}
	second := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			order = append(order, "second")
			return next(ctx, req)
		}
	}
	router.Use(first)

	stellarhttp.Handle(router, stdhttp.MethodGet, "/ping", stellarhttp.EmptyBinder[pingRequest](), func(context.Context, *pingRequest) (*pingResponse, error) {
		return &pingResponse{Message: "pong"}, nil
	}, stellarhttp.JSONEncoder[pingResponse], second)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/ping", nil)

	adapter := ginadapter.New("")
	adapter.UseRouter(router)
	adapter.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, recorder.Code)
	}
	if !reflect.DeepEqual(order, []string{"first", "second"}) {
		t.Fatalf("unexpected middleware order: %#v", order)
	}

	var response pingResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Message != "pong" {
		t.Fatalf("expected pong, got %q", response.Message)
	}
}

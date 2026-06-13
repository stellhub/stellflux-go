package internal

import (
	"context"
	"net/http"
	"strings"

	stellarhttp "github.com/stellhub/stellar/transport/http"
)

type pingResponse struct {
	Message string `json:"message"`
}

type helloResponse struct {
	Message string `json:"message"`
}

type createItemRequest struct {
	Name string `json:"name"`
}

type createItemResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func handlePing(context.Context, *stellarhttp.Request) (*stellarhttp.Response, error) {
	return stellarhttp.JSON(http.StatusOK, pingResponse{Message: "pong"}), nil
}

func handleHello(_ context.Context, req *stellarhttp.Request) (*stellarhttp.Response, error) {
	name := strings.TrimSpace(req.Query.Get("name"))
	if name == "" {
		name = "stellar"
	}
	return stellarhttp.JSON(http.StatusOK, helloResponse{
		Message: "hello, " + name,
	}), nil
}

func createItem(_ context.Context, req *createItemRequest) (*createItemResponse, error) {
	return &createItemResponse{
		ID:   "item-001",
		Name: req.Name,
	}, nil
}

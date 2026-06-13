package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type User struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
}

type UserClient struct {
	client  *http.Client
	baseURL string
}

type DownstreamError struct {
	Service string
	Status  int
	Body    string
}

func (e *DownstreamError) Error() string {
	return e.Service + " returned " + strconv.Itoa(e.Status) + ": " + e.Body
}

func newUserClient(client *http.Client, baseURL string) (*UserClient, error) {
	if client == nil {
		return nil, errors.New("http client is required")
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("user-service base_url is required")
	}
	return &UserClient{
		client:  client,
		baseURL: baseURL,
	}, nil
}

func (c *UserClient) GetUser(ctx context.Context, id string) (*User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("user id is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/users/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, decodeDownstreamError("user-service", resp)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode user-service response: %w", err)
	}
	return &user, nil
}

func decodeDownstreamError(service string, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read %s error response: %w", service, err)
	}
	return &DownstreamError{
		Service: service,
		Status:  resp.StatusCode,
		Body:    string(body),
	}
}

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"

	apperrors "github.com/stellhub/stellar/errors"
	"github.com/stellhub/stellar/middleware"
	"github.com/stellhub/stellar/observability"
)

const TransportName = "http"

type Handler[Req any, Resp any] func(ctx context.Context, req *Req) (*Resp, error)

type Binder[Req any] func(*Request) (*Req, error)

type Encoder[Resp any] func(*Resp) *Response

type Endpoint func(context.Context, *Request) (*Response, error)

type Adapter interface {
	Name() string
	UseRouter(*Router)
	Start(context.Context) error
	Stop(context.Context) error
}

type HandlerProvider interface {
	Handler() stdhttp.Handler
}

type AddrSetter interface {
	SetAddr(string)
}

type Request struct {
	Method string
	Path   string
	Header stdhttp.Header
	Query  url.Values
	Body   io.ReadCloser
	Raw    any
}

type Response struct {
	Status int
	Header stdhttp.Header
	Body   any
}

type ErrorResponse struct {
	Code    apperrors.Code `json:"code"`
	Message string         `json:"message"`
}

type Route struct {
	Method      string
	Path        string
	Endpoint    Endpoint
	Middlewares []middleware.Middleware
}

type Snapshot struct {
	Middlewares []middleware.Middleware
	Routes      []Route
	Observer    *observability.Provider
}

type GroupOption func(*groupConfig)

type Option func(*Router)

type Router struct {
	mu           sync.RWMutex
	middlewares  []middleware.Middleware
	routes       []Route
	prefix       string
	groupMws     []middleware.Middleware
	parent       *Router
	errorHandler ErrorHandler
	observer     *observability.Provider
}

type groupConfig struct {
	middlewares []middleware.Middleware
}

type ErrorHandler func(error) *Response

func NewRouter(opts ...Option) *Router {
	router := &Router{
		errorHandler: DefaultErrorHandler,
		observer:     observability.New(),
	}
	for _, opt := range opts {
		opt(router)
	}
	return router
}

func WithObservability(provider *observability.Provider) Option {
	return func(router *Router) {
		if provider != nil {
			router.observer = provider
		}
	}
}

func WithMiddleware(mws ...middleware.Middleware) Option {
	return func(router *Router) {
		router.middlewares = append(router.middlewares, mws...)
	}
}

func WithErrorHandler(handler ErrorHandler) Option {
	return func(router *Router) {
		if handler != nil {
			router.errorHandler = handler
		}
	}
}

func WithGroupMiddleware(mws ...middleware.Middleware) GroupOption {
	return func(cfg *groupConfig) {
		cfg.middlewares = append(cfg.middlewares, mws...)
	}
}

func (r *Router) Use(mws ...middleware.Middleware) {
	root := r.root()
	root.mu.Lock()
	defer root.mu.Unlock()
	root.middlewares = append(root.middlewares, mws...)
}

func (r *Router) GET(path string, endpoint Endpoint, mws ...middleware.Middleware) {
	r.Handle(stdhttp.MethodGet, path, endpoint, mws...)
}

func (r *Router) POST(path string, endpoint Endpoint, mws ...middleware.Middleware) {
	r.Handle(stdhttp.MethodPost, path, endpoint, mws...)
}

func (r *Router) PUT(path string, endpoint Endpoint, mws ...middleware.Middleware) {
	r.Handle(stdhttp.MethodPut, path, endpoint, mws...)
}

func (r *Router) DELETE(path string, endpoint Endpoint, mws ...middleware.Middleware) {
	r.Handle(stdhttp.MethodDelete, path, endpoint, mws...)
}

func (r *Router) Handle(method string, path string, endpoint Endpoint, mws ...middleware.Middleware) {
	root := r.root()
	fullPath := r.fullPath(path)
	routeMiddlewares := append(r.groupMiddlewares(), mws...)

	root.mu.Lock()
	defer root.mu.Unlock()
	root.routes = append(root.routes, Route{
		Method:      method,
		Path:        fullPath,
		Endpoint:    endpoint,
		Middlewares: routeMiddlewares,
	})
}

func (r *Router) Group(prefix string, opts ...GroupOption) *Router {
	cfg := groupConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Router{
		parent:   r.root(),
		prefix:   joinPath(r.prefix, prefix),
		groupMws: append(r.groupMiddlewares(), cfg.middlewares...),
	}
}

func (r *Router) Routes() []Route {
	return r.Snapshot().Routes
}

func (r *Router) Snapshot() Snapshot {
	root := r.root()
	root.mu.RLock()
	defer root.mu.RUnlock()

	mws := make([]middleware.Middleware, len(root.middlewares))
	copy(mws, root.middlewares)

	routes := make([]Route, len(root.routes))
	for i, route := range root.routes {
		routes[i] = route
		routes[i].Middlewares = append([]middleware.Middleware(nil), route.Middlewares...)
	}
	return Snapshot{Middlewares: mws, Routes: routes, Observer: root.observer}
}

func (r *Router) ErrorHandler() ErrorHandler {
	root := r.root()
	if root.errorHandler == nil {
		return DefaultErrorHandler
	}
	return root.errorHandler
}

func Handle[Req any, Resp any](router *Router, method string, path string, binder Binder[Req], handler Handler[Req, Resp], encoder Encoder[Resp], mws ...middleware.Middleware) {
	if binder == nil {
		binder = EmptyBinder[Req]()
	}
	if encoder == nil {
		encoder = JSONEncoder[Resp]
	}

	router.Handle(method, path, func(ctx context.Context, request *Request) (*Response, error) {
		req, err := binder(request)
		if err != nil {
			return nil, err
		}

		final := middleware.Handler(func(ctx context.Context, payload any) (any, error) {
			typedReq := payload.(*Req)
			return handler(ctx, typedReq)
		})
		resp, err := middleware.Chain(mws...)(final)(ctx, req)
		if err != nil {
			return nil, err
		}
		if isNil(resp) {
			return &Response{Status: stdhttp.StatusNoContent}, nil
		}
		typedResp, ok := resp.(*Resp)
		if !ok {
			return nil, apperrors.New(apperrors.CodeInternal, "unexpected response type", stdhttp.StatusInternalServerError)
		}
		return encoder(typedResp), nil
	})
}

func Execute(ctx context.Context, request *Request, snapshot Snapshot, route Route) (*Response, error) {
	observer := snapshot.Observer
	if observer == nil {
		observer = observability.New()
	}
	ctx, _, finish := observer.StartHTTPServer(ctx, observability.HTTPServerRequest{
		Method: request.Method,
		Route:  route.Path,
		Path:   request.Path,
		Header: request.Header,
	})
	defer func() {
		if finish != nil {
			finish(observability.HTTPServerResult{StatusCode: stdhttp.StatusInternalServerError})
		}
	}()

	final := middleware.Handler(func(ctx context.Context, req any) (any, error) {
		httpReq := req.(*Request)
		return route.Endpoint(ctx, httpReq)
	})
	chain := middleware.Chain(append(snapshot.Middlewares, route.Middlewares...)...)(final)
	resp, err := chain(ctx, request)
	if err != nil {
		finish(observability.HTTPServerResult{StatusCode: apperrors.HTTPStatusOf(err), Err: err})
		finish = nil
		return nil, err
	}
	if isNil(resp) {
		response := &Response{Status: stdhttp.StatusNoContent}
		finish(observability.HTTPServerResult{StatusCode: response.Status})
		finish = nil
		return response, nil
	}
	typedResp, ok := resp.(*Response)
	if !ok {
		err := apperrors.New(apperrors.CodeInternal, "unexpected response type", stdhttp.StatusInternalServerError)
		finish(observability.HTTPServerResult{StatusCode: stdhttp.StatusInternalServerError, Err: err})
		finish = nil
		return nil, err
	}
	status := typedResp.Status
	if status == 0 {
		status = stdhttp.StatusOK
	}
	finish(observability.HTTPServerResult{StatusCode: status})
	finish = nil
	return typedResp, nil
}

func (r *Router) SetObservability(provider *observability.Provider) {
	root := r.root()
	root.mu.Lock()
	defer root.mu.Unlock()
	if provider != nil {
		root.observer = provider
	}
}

func EmptyBinder[Req any]() Binder[Req] {
	return func(*Request) (*Req, error) {
		return new(Req), nil
	}
}

func JSONBinder[Req any]() Binder[Req] {
	return func(request *Request) (*Req, error) {
		if request.Body == nil {
			return nil, apperrors.New(apperrors.CodeInvalidArgument, "request body is required", stdhttp.StatusBadRequest)
		}
		defer request.Body.Close()

		req := new(Req)
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(req); err != nil {
			return nil, apperrors.Wrap(apperrors.CodeInvalidArgument, "invalid JSON request body", err, stdhttp.StatusBadRequest)
		}
		return req, nil
	}
}

func JSONEncoder[Resp any](resp *Resp) *Response {
	return &Response{
		Status: stdhttp.StatusOK,
		Header: stdhttp.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: resp,
	}
}

func JSON(status int, payload any) *Response {
	return &Response{
		Status: status,
		Header: stdhttp.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: payload,
	}
}

func StdHandlerEndpoint(handler stdhttp.Handler) Endpoint {
	return func(ctx context.Context, request *Request) (*Response, error) {
		if handler == nil {
			return &Response{Status: stdhttp.StatusNotFound}, nil
		}
		stdReq := httptest.NewRequest(request.Method, request.Path, request.Body)
		stdReq = stdReq.WithContext(ctx)
		stdReq.Header = request.Header.Clone()
		stdReq.URL.RawQuery = request.Query.Encode()

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, stdReq)
		result := recorder.Result()
		defer result.Body.Close()

		body, err := io.ReadAll(result.Body)
		if err != nil {
			return nil, err
		}
		return &Response{
			Status: result.StatusCode,
			Header: result.Header,
			Body:   body,
		}, nil
	}
}

func DefaultErrorHandler(err error) *Response {
	status := apperrors.HTTPStatusOf(err)
	return JSON(status, ErrorResponse{
		Code:    apperrors.CodeOf(err),
		Message: apperrors.MessageOf(err),
	})
}

func NewRequestFromHTTP(r *stdhttp.Request, raw any) *Request {
	return &Request{
		Method: r.Method,
		Path:   r.URL.Path,
		Header: r.Header.Clone(),
		Query:  r.URL.Query(),
		Body:   r.Body,
		Raw:    raw,
	}
}

func WriteHTTPResponse(w stdhttp.ResponseWriter, response *Response) {
	if response == nil {
		response = &Response{Status: stdhttp.StatusNoContent}
	}
	status := response.Status
	if status == 0 {
		status = stdhttp.StatusOK
	}
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if response.Body == nil {
		w.WriteHeader(status)
		return
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(status)
	if strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		if err := json.NewEncoder(w).Encode(response.Body); err != nil {
			stdhttp.Error(w, stdhttp.StatusText(stdhttp.StatusInternalServerError), stdhttp.StatusInternalServerError)
		}
		return
	}
	if data, ok := response.Body.([]byte); ok {
		_, _ = w.Write(data)
		return
	}
	_, _ = fmt.Fprint(w, response.Body)
}

func (r *Router) root() *Router {
	if r.parent == nil {
		return r
	}
	return r.parent.root()
}

func (r *Router) fullPath(path string) string {
	return joinPath(r.prefix, path)
}

func (r *Router) groupMiddlewares() []middleware.Middleware {
	if len(r.groupMws) == 0 {
		return nil
	}
	mws := make([]middleware.Middleware, len(r.groupMws))
	copy(mws, r.groupMws)
	return mws
}

func joinPath(prefix string, path string) string {
	if prefix == "" {
		prefix = "/"
	}
	if path == "" {
		path = "/"
	}
	joined := strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
	if joined == "" {
		return "/"
	}
	if !strings.HasPrefix(joined, "/") {
		return "/" + joined
	}
	return joined
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

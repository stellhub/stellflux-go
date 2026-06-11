package hertzadapter

import (
	"bytes"
	"context"
	"io"
	stdhttp "net/http"
	"net/url"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	stellarhttp "github.com/stellhub/stellar/transport/http"
)

const Name = "http-hertz"

type Adapter struct {
	addr   string
	router *stellarhttp.Router
	engine *server.Hertz
	errCh  chan error
	mu     sync.Mutex
}

func New(addr string) *Adapter {
	if addr == "" {
		addr = ":8080"
	}
	return &Adapter{
		addr:   addr,
		router: stellarhttp.NewRouter(),
		errCh:  make(chan error, 1),
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

func (a *Adapter) UseRouter(router *stellarhttp.Router) {
	if router != nil {
		a.router = router
	}
}

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.bindRoutesLocked()
	go func() {
		a.engine.Spin()
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
	if a.engine == nil {
		return nil
	}
	return a.engine.Shutdown(ctx)
}

func (a *Adapter) bindRoutesLocked() {
	a.engine = server.Default(server.WithHostPorts(a.addr))
	snapshot := a.router.Snapshot()
	errorHandler := a.router.ErrorHandler()
	for _, route := range snapshot.Routes {
		route := route
		a.engine.Handle(route.Method, route.Path, func(ctx context.Context, c *app.RequestContext) {
			request := newHertzRequest(c)
			response, err := stellarhttp.Execute(ctx, request, snapshot, route)
			if err != nil {
				response = errorHandler(err)
			}
			writeHertzResponse(c, response)
		})
	}
}

func newHertzRequest(c *app.RequestContext) *stellarhttp.Request {
	query := url.Values{}
	c.QueryArgs().VisitAll(func(key []byte, value []byte) {
		query.Add(string(key), string(value))
	})

	header := stdhttp.Header{}
	c.Request.Header.VisitAll(func(key []byte, value []byte) {
		header.Add(string(key), string(value))
	})

	return &stellarhttp.Request{
		Method: string(c.Method()),
		Path:   string(c.Path()),
		Header: header,
		Query:  query,
		Body:   io.NopCloser(bytes.NewReader(c.Request.Body())),
		Raw:    c,
	}
}

func writeHertzResponse(c *app.RequestContext, response *stellarhttp.Response) {
	if response == nil {
		c.Status(consts.StatusNoContent)
		return
	}
	status := response.Status
	if status == 0 {
		status = stdhttp.StatusOK
	}
	for key, values := range response.Header {
		for _, value := range values {
			c.Response.Header.Add(key, value)
		}
	}
	if response.Body == nil {
		c.Status(status)
		return
	}
	contentType := response.Header.Get("Content-Type")
	if data, ok := response.Body.([]byte); ok && contentType != "" && contentType != "application/json" {
		c.Data(status, contentType, data)
		return
	}
	if text, ok := response.Body.(string); ok && contentType != "" && contentType != "application/json" {
		c.Data(status, contentType, []byte(text))
		return
	}
	c.JSON(status, response.Body)
}

package ginadapter

import (
	"context"
	stderrors "errors"
	stdhttp "net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	stellarhttp "github.com/stellhub/stellar/transport/http"
)

const Name = "http-gin"

type Adapter struct {
	addr   string
	router *stellarhttp.Router
	engine *gin.Engine
	server *stdhttp.Server
	errCh  chan error
	mu     sync.Mutex
}

type Option func(*Adapter)

func New(addr string, opts ...Option) *Adapter {
	if addr == "" {
		addr = ":8080"
	}
	gin.SetMode(gin.ReleaseMode)
	adapter := &Adapter{
		addr:   addr,
		router: stellarhttp.NewRouter(),
		engine: gin.New(),
		errCh:  make(chan error, 1),
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

func WithEngine(engine *gin.Engine) Option {
	return func(adapter *Adapter) {
		if engine != nil {
			adapter.engine = engine
		}
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

func (a *Adapter) Handler() stdhttp.Handler {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.bindRoutesLocked()
	return a.engine
}

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.bindRoutesLocked()
	a.server = &stdhttp.Server{
		Addr:              a.addr,
		Handler:           a.engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := a.server.ListenAndServe(); err != nil && !stderrors.Is(err, stdhttp.ErrServerClosed) {
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
	return a.server.Shutdown(ctx)
}

func (a *Adapter) bindRoutesLocked() {
	a.engine = gin.New()
	snapshot := a.router.Snapshot()
	errorHandler := a.router.ErrorHandler()
	for _, route := range snapshot.Routes {
		route := route
		a.engine.Handle(route.Method, route.Path, func(c *gin.Context) {
			request := stellarhttp.NewRequestFromHTTP(c.Request, c)
			response, err := stellarhttp.Execute(c.Request.Context(), request, snapshot, route)
			if err != nil {
				response = errorHandler(err)
			}
			writeGinResponse(c, response)
		})
	}
}

func writeGinResponse(c *gin.Context, response *stellarhttp.Response) {
	if response == nil {
		c.Status(stdhttp.StatusNoContent)
		return
	}
	status := response.Status
	if status == 0 {
		status = stdhttp.StatusOK
	}
	for key, values := range response.Header {
		for _, value := range values {
			c.Header(key, value)
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

package chiadapter

import (
	"context"
	stderrors "errors"
	stdhttp "net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	stellarhttp "github.com/stellhub/stellar/transport/http"
)

const Name = "http-chi"

type Adapter struct {
	addr   string
	router *stellarhttp.Router
	mux    chi.Router
	server *stdhttp.Server
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
		mux:    chi.NewRouter(),
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

func (a *Adapter) Handler() stdhttp.Handler {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.bindRoutesLocked()
	return a.mux
}

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.bindRoutesLocked()
	a.server = &stdhttp.Server{
		Addr:              a.addr,
		Handler:           a.mux,
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
	a.mux = chi.NewRouter()
	snapshot := a.router.Snapshot()
	errorHandler := a.router.ErrorHandler()
	for _, route := range snapshot.Routes {
		route := route
		a.mux.MethodFunc(route.Method, route.Path, func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			request := stellarhttp.NewRequestFromHTTP(r, r)
			response, err := stellarhttp.Execute(r.Context(), request, snapshot, route)
			if err != nil {
				response = errorHandler(err)
			}
			stellarhttp.WriteHTTPResponse(w, response)
		})
	}
}

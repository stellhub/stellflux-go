package boot

import (
	"context"
	stdhttp "net/http"

	ginadapter "github.com/stellhub/stellar/transport/http/adapters/gin"

	boothttp "github.com/stellhub/stellar/transport/http"
)

func WithHTTPAdapter(adapter boothttp.Adapter) Option {
	return func(app *App) {
		app.setHTTPAdapter(adapter, true)
	}
}

func WithHTTPServer(addr string) Option {
	return func(app *App) {
		adapter := app.httpAdapter
		if adapter == nil {
			adapter = ginadapter.New(addr)
		}
		if setter, ok := adapter.(boothttp.AddrSetter); ok {
			setter.SetAddr(addr)
		}
		app.setHTTPAdapter(adapter, true)
	}
}

func (a *App) ensureHTTPAdapter() boothttp.Adapter {
	if a.httpAdapter == nil {
		a.setHTTPAdapter(ginadapter.New(a.config.HTTP.ServerAddr()), false)
	}
	return a.httpAdapter
}

func (a *App) setHTTPAdapter(adapter boothttp.Adapter, registerTransport bool) {
	if adapter == nil {
		return
	}
	adapter.UseRouter(a.httpRouter)
	a.httpAdapter = adapter
	if registerTransport {
		a.addTransport(adapter)
	}
}

func (a *App) addTransport(transport Transport) {
	if transport == nil {
		return
	}
	for _, existing := range a.transports {
		if existing == transport {
			return
		}
	}
	a.transports = append(a.transports, transport)
}

func handlerOf(adapter boothttp.Adapter) stdhttp.Handler {
	provider, ok := adapter.(boothttp.HandlerProvider)
	if !ok {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			stdhttp.Error(w, "http adapter does not expose net/http handler", stdhttp.StatusServiceUnavailable)
		})
	}
	return provider.Handler()
}

func (a *App) httpHealth(context.Context) HealthCheck {
	if a.httpAdapter == nil {
		return HealthCheck{Name: "http", Status: HealthStatusSkipped}
	}
	return HealthCheck{Name: a.httpAdapter.Name(), Status: HealthStatusUp}
}

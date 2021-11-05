package router

import (
	"net/http"

	"github.com/wraix/device-flow-proxy/endpoint"
	"github.com/wraix/device-flow-proxy/endpoint/browser"
	"github.com/wraix/device-flow-proxy/endpoint/device"
	"github.com/wraix/device-flow-proxy/endpoint/docs"
	"github.com/wraix/device-flow-proxy/endpoint/health"
	"github.com/wraix/device-flow-proxy/endpoint/metrics"
	"github.com/wraix/device-flow-proxy/middleware"

	"github.com/charmixer/oas/api"

	"github.com/julienschmidt/httprouter"

	"github.com/rs/zerolog/log"
)

type Router struct {
	httprouter.Router
	OpenAPI    api.Api
	Middleware []middleware.MiddlewareHandler
}

func (r *Router) NewRoute(method string, uri string, ep endpoint.EndpointHandler, handlers ...middleware.MiddlewareHandler) {
	log.Debug().
		Str("method", method).
		Str("endpoint", uri).
		Msg("Setting up endpoint")

	r.OpenAPI.NewEndpoint(method, uri, ep.Specification())

	middlewareHandlers := append(handlers, ep.Middleware()...)
	r.Handler(method, uri, middleware.New(ep.(http.Handler), middlewareHandlers...))
}
func (r *Router) Use(h ...middleware.MiddlewareHandler) {
	r.Middleware = append(r.Middleware, h...)
}
func (r *Router) Handle() http.Handler {
	return middleware.New(r, r.Middleware...)
}

func NewRouter(name string, description string, version string) *Router {
	r := &Router{
		OpenAPI: api.Api{
			Title:       name,
			Description: description,
			Version:     version,
		},
	}

	// Ordering matters
	r.Use(
		middleware.WithInitialization(),
		middleware.WithContext(),
		middleware.WithTracing(name),
		middleware.WithMetrics(),
		middleware.WithLogging(),
	)

	r.NewRoute("GET", "/health", health.NewGetHealthEndpoint())

	r.NewRoute("GET", "/docs", docs.NewGetDocsEndpoint())
	r.NewRoute("GET", "/docs/openapi", docs.NewGetOpenapiEndpoint())

	r.NewRoute("GET", "/metrics", metrics.NewGetMetricsEndpoint())

	// Device API
	r.NewRoute("POST", "/device/code", device.NewPostCodeEndpoint())
	r.NewRoute("POST", "/device/token", device.NewPostTokenEndpoint())

	// Browser routes
	r.NewRoute("GET", "/device", browser.NewGetDeviceEndpoint())
	r.NewRoute("GET", "/auth/verify_code", browser.NewGetVerifyCodeEndpoint())
	r.NewRoute("GET", "/auth/redirect", browser.NewGetRedirectEndpoint())

	return r
}

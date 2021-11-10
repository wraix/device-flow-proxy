package endpoint

import (
	"net/http"

	"github.com/charmixer/oas/api"
	"github.com/wraix/device-flow-proxy/middleware"
)

type EndpointHandler interface {
	http.Handler
	Specification() api.Path
	Middleware() []middleware.MiddlewareHandler
}
type Endpoint struct {
	EndpointHandler
	specification api.Path
	middleware    []middleware.MiddlewareHandler
}
type EndpointOption func(e *Endpoint)

func (ep *Endpoint) Setup(options ...EndpointOption) {
	for _, opt := range options {
		opt(ep)
	}
}

func (ep Endpoint) Specification() api.Path {
	return ep.specification
}
func (ep Endpoint) Middleware() []middleware.MiddlewareHandler {
	return ep.middleware
}

func WithSpecification(spec api.Path) EndpointOption {
	return func(e *Endpoint) {
		e.specification = spec
	}
}

func WithMiddleware(handlers ...middleware.MiddlewareHandler) EndpointOption {
	return func(e *Endpoint) {
		e.middleware = append(e.middleware, handlers...)
	}
}

// https://pkg.go.dev/github.com/alexliesenfeld/health#WithCacheDuration
// https://github.com/alexliesenfeld/health/blob/v0.6.0/config.go#L159

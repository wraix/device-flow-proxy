package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"

	"github.com/wraix/device-flow-proxy/endpoint"

	"github.com/charmixer/oas/api"
)

var (
	OpenApiTags = []api.Tag{
		{Name: "Metrics", Description: ""},
	}
)

type GetMetricsRequest struct{}
type GetMetricsResponse struct{}

// GetMetricsEndpoint is using embedding, see https://golang.org/doc/effective_go#embedding
type GetMetricsEndpoint struct {
	endpoint.Endpoint
}

func (ep GetMetricsEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}).(http.HandlerFunc)
	t.ServeHTTP(w, r)
}

func NewGetMetricsEndpoint() endpoint.EndpointHandler {
	ep := GetMetricsEndpoint{}

	ep.Setup(
		endpoint.WithSpecification(api.Path{
			Summary:     "Get metrics from the application",
			Description: `Get metrics from the application`,
			Tags:        OpenApiTags,

			Request: api.Request{
				Description: `Request metrics`,
				//Schema: GetMetricsRequest{},
			},

			Responses: []api.Response{{
				Description: `Metrics from prometheus`,
				Code:        200,
				ContentType: []string{"application/text"},
				//Schema: GetMetricsResponse{},
			}},
		}),
	)

	// Must be pointer to allow ServeHTTP method to be used with *Endpoint
	return ep
}

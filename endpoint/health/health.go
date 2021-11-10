package health

import (
	"fmt"
	"net/http"

	"github.com/charmixer/oas/api"

	"github.com/wraix/device-flow-proxy/endpoint"
	"github.com/wraix/device-flow-proxy/endpoint/problem"

	"go.opentelemetry.io/otel"
)

var (
	OpenAPITags = []api.Tag{
		{Name: "Health", Description: "Endpoints reporting the health of the application"},
	}
)

type GetHealthRequest struct{}
type GetHealthResponse struct {
	Alive bool `json:"alive_json" oas-desc:"Tells if the service is alive (ping)"`
	Ready bool `json:"ready_json" oas-desc:"Tells if the service is ready to accept requests"`
}

// https://golang.org/doc/effective_go#embedding
type GetHealthEndpoint struct {
	endpoint.Endpoint
}

func (ep GetHealthEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, fmt.Sprintf("%s execution", r.URL.Path))
	defer span.End()

	request := GetHealthRequest{}
	if err := endpoint.WithRequestQueryParser(ctx, r, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	if err := endpoint.WithRequestValidation(ctx, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	response := GetHealthResponse{
		Alive: true,
		Ready: true,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := endpoint.WithResponseValidation(ctx, response); err != nil {
		problem.MustWrite(w, err)
		return
	}

	if err := endpoint.WithJsonResponseWriter(ctx, w, response); err != nil {
		problem.MustWrite(w, err)
		return
	}
}

func NewGetHealthEndpoint() endpoint.EndpointHandler {
	ep := GetHealthEndpoint{}

	/*
		ep.Setup(
			endpoint.WithSpecification(api.NewPath(
				api.WithSummary("Get health information about the service"),
				api.WithDescription(``),
				api.WithTags(OPENAPI_TAGS),

				api.WithRequest(
					api.WithRequestScheme(GetHealthRequest{}),
				),

				api.WithResponse(
					api.WithResponseDescription(http.StatusText(http.StatusOK)),
					api.WithResponseCode(http.StatusOK),
					api.WithResponseScheme(GetHealthResponse{}),
				),
			)),
	*/

	ep.Setup(
		endpoint.WithSpecification(api.Path{
			Summary:     "Get health information about the service",
			Description: ``,
			Tags:        OpenAPITags,

			Request: api.Request{
				Description: ``,
				Schema:      GetHealthRequest{},
			},

			Responses: []api.Response{{
				Description: http.StatusText(http.StatusOK),
				Code:        http.StatusOK,
				Schema:      GetHealthResponse{},
			}, {
				Description: http.StatusText(http.StatusBadRequest),
				Code:        http.StatusBadRequest,
				Schema:      problem.ValidationProblem{}, // TODO fix oas to work with: problem.ValidationError{},
			}},
		}),
	)

	return ep
}

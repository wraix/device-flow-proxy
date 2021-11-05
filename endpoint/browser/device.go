package browser

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/wraix/device-flow-proxy/endpoint"
	"github.com/wraix/device-flow-proxy/endpoint/problem"

  "github.com/charmixer/oas/api"

	"go.opentelemetry.io/otel"
)

type GetDeviceRequest struct {
	Code string `query:"code"`
}

type GetDeviceEndpoint struct {
	endpoint.Endpoint
}

type DevicePageData struct {
	Code       string
	FormAction string
	PageTitle  string
}

func (ep GetDeviceEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, fmt.Sprintf("%s execution", r.URL.Path))
	defer span.End()

	request := GetDeviceRequest{}
	if err := endpoint.WithRequestQueryParser(ctx, r, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	if err := endpoint.WithRequestValidation(ctx, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	tmpl := template.Must(template.ParseFiles("./endpoint/browser/device.html"))

	data := DevicePageData{
		Code:       request.Code,
		FormAction: "/auth/verify_code",
		PageTitle:  "Enter Device Code",
	}

	tmpl.Execute(w, data)
}

func NewGetDeviceEndpoint() endpoint.EndpointHandler {
	ep := GetDeviceEndpoint{}

	ep.Setup(
		endpoint.WithSpecification(api.Path{
			Summary:     "Device",
			Description: ``,
			Tags:        OPENAPI_TAGS,

			Request: api.Request{
				Description: ``,
				Schema:      GetDeviceRequest{},
			},
		}),
	)

	return ep
}

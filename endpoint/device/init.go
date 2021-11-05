package device

import (
	"github.com/charmixer/oas/api"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var (
	OPENAPI_TAGS = []api.Tag{
		{Name: "Device Code Grant", Description: "The device code grant flow"},
	}
	tr trace.Tracer
)

func init() {
	tr = otel.Tracer("request")
}

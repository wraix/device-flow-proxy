package tracing

import (
	"context"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/rs/zerolog/log"
)

const (
	DEFAULT_JAEGER_ENDPOINT = "http://localhost:14268/api/traces"
)

type errorHandler struct{}

func (e errorHandler) Handle(err error) {
	log.Error().Err(err).Msg("Tracing failed")
}

// Nil version of SpanExporter to prevent import of otel in other package for type
func SetupNilExporter() sdktrace.SpanExporter {
	return nil
}
func SetupJaegerExporter(url string) (sdktrace.SpanExporter, error) {
	if url == "" {
		url = DEFAULT_JAEGER_ENDPOINT
	}

	log.Info().Msgf("Tracing will be exported to jaeger @ %s", url)
	return jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint(url)),
	)
}

func SetupTracing(exp sdktrace.SpanExporter, service string, environment string, version string) (func(), error) {
	ctx := context.Background()

	eHandler := errorHandler{}
	otel.SetErrorHandler(eHandler)

	tp := sdktrace.NewTracerProvider(
		// Always be sure to batch in production.
		sdktrace.WithBatcher(exp),
		// Record information about this application in an Resource.
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(service),
			attribute.String("environment", environment),
			attribute.String("version", version),
		)),
	)

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)

	return func() {
		err := tp.Shutdown(ctx)

		if err != nil {
			log.Error().Err(err).Msg("Failed to shutdown Trace Provider")
		}
	}, nil
}

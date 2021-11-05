package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const requestIdKey = attribute.Key("request_id")
const spanKindKey = attribute.Key("span.kind")

func WithTracing(appName string) (MiddlewareHandler) {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

    	ctxTraced, span := otel.Tracer("request").Start( ctx, r.URL.Path )
			defer span.End()

			span.SetAttributes(
				spanKindKey.String("server"),
				requestIdKey.String(ctxTraced.Value("req_id").(string)), // The infamous request id!
		  )

			// OpenTelemetry semantic convention tracing
			span.SetAttributes(semconv.HTTPServerAttributesFromHTTPRequest(appName, r.URL.Path, r)...)

    	// Wrap response so we can trace it
			res := w.(*responseWriter)

			// Delegate down the middleware layer chain
			next.ServeHTTP(res, r.WithContext(ctxTraced))

			// Trace response upon chain completion
			span.SetAttributes(semconv.HTTPAttributesFromHTTPStatusCode(res.Status)...)
			span.SetStatus(semconv.SpanStatusFromHTTPStatusCode(res.Status))
		})
	}
}

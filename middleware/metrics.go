package middleware

import (
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var totalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Number of get requests.",
	},
	[]string{"path", "method", "status"},
)

var httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "http_response_time_seconds",
	Help: "Duration of HTTP requests.",
}, []string{"path", "method"})

func WithMetrics() MiddlewareHandler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			tr := otel.Tracer("request")
			ctx, span := tr.Start(ctx, "middleware.metrics")
			defer span.End()

			timer := prometheus.NewTimer(httpDuration.WithLabelValues(r.URL.Path, r.Method))

			wrapped := w.(*responseWriter)
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			ctx, span = tr.Start(ctx, "record metrics")
			defer span.End()

			totalRequests.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(wrapped.Status)).Inc()

			timer.ObserveDuration()
		})
	}
}

func init() {
	prometheus.Register(totalRequests)
	prometheus.Register(httpDuration)
}

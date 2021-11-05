package middleware

import (
	"net/http"
)

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP status code to be captured from middleware, like logging.
type responseWriter struct {
	http.ResponseWriter
	Status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.Status = code
	rw.ResponseWriter.WriteHeader(code)
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, Status: http.StatusOK}
}

func WithInitialization() (MiddlewareHandler) {
  return func(next http.Handler) http.Handler {
  	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  		wrapped := wrapResponseWriter(w)
  		next.ServeHTTP(wrapped, r)
  	})
  }
}

type MiddlewareHandler func(http.Handler) http.Handler
func New(h http.Handler, handlers ...MiddlewareHandler ) http.Handler {

	if h == nil {
		h = http.DefaultServeMux
	}

	for i := range handlers {
		h = handlers[len(handlers)-1-i](h)
	}

	return h
}

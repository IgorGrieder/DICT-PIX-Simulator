package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// CORSMiddleware adds CORS headers to responses using rs/cors library
func CORSMiddleware(next http.Handler) http.Handler {
	c := cors.New(cors.Options{
		// Allow all origins dynamically to support credentials
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
			"X-Idempotency-Key",
			"X-User-Id",
			"Accept",
			"Origin",
			"X-Requested-With",
			// OpenTelemetry headers
			"traceparent",
			"tracestate",
			"baggage",
			"sentry-trace",
		},
		AllowCredentials: true,
	})

	return c.Handler(next)
}

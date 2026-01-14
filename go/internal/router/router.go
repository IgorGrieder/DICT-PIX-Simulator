package router

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/dict-simulator/go/internal/middleware"
	"github.com/dict-simulator/go/internal/modules/auth"
	"github.com/dict-simulator/go/internal/modules/entries"
	"github.com/dict-simulator/go/internal/modules/health"
	"github.com/dict-simulator/go/internal/ratelimit"
	"github.com/dict-simulator/go/internal/telemetry"
)

// spanNames maps route patterns to custom span names (preserving current naming convention)
var spanNames = map[string]string{
	"GET /health":           "health",
	"POST /auth/register":   "auth.register",
	"POST /auth/login":      "auth.login",
	"POST /entries":         "entries.create",
	"GET /entries/{key}":    "entries.get",
	"PUT /entries/{key}":    "entries.update",
	"DELETE /entries/{key}": "entries.delete",
}

// Setup creates and configures the HTTP router with all routes
func Setup(
	authHandler *auth.Handler,
	entriesHandler *entries.Handler,
	mwManager *middleware.Manager,
) http.Handler {
	mux := http.NewServeMux()

	// Initialize health handler (stateless)
	healthHandler := health.NewHandler()

	// Get rate limiting policies
	policies := ratelimit.DefaultPolicies()

	// Health and metrics endpoints (no tracing wrapper needed - otelhttp will handle it)
	mux.HandleFunc("GET /health", healthHandler.Health)
	mux.Handle("GET /metrics", healthHandler.Metrics())

	// Auth routes (no auth middleware)
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)

	// Entries routes with per-method rate limiting policies
	// POST /entries - createEntry uses ENTRIES_WRITE policy (1200/min, 36000 bucket)
	mux.Handle("POST /entries", middleware.Chain(
		http.HandlerFunc(entriesHandler.Create),
		middleware.AuthMiddleware,
		mwManager.RateLimiterWithPolicy(policies[ratelimit.PolicyEntriesWrite]),
		mwManager.Idempotency,
	))

	// GET /entries/{key} - getEntry uses ENTRIES_READ_PARTICIPANT_ANTISCAN policy
	// Category H: 2/min, 50 bucket, 404 costs 3 tokens
	mux.Handle("GET /entries/{key}", middleware.Chain(
		http.HandlerFunc(entriesHandler.Get),
		middleware.AuthMiddleware,
		mwManager.RateLimiterWithPolicy(policies[ratelimit.PolicyEntriesReadParticipant]),
	))

	// PUT /entries/{key} - updateEntry uses ENTRIES_UPDATE policy (600/min, 600 bucket)
	mux.Handle("PUT /entries/{key}", middleware.Chain(
		http.HandlerFunc(entriesHandler.Update),
		middleware.AuthMiddleware,
		mwManager.RateLimiterWithPolicy(policies[ratelimit.PolicyEntriesUpdate]),
	))

	// DELETE /entries/{key} - deleteEntry uses ENTRIES_WRITE policy (same as create)
	mux.Handle("DELETE /entries/{key}", middleware.Chain(
		http.HandlerFunc(entriesHandler.Delete),
		middleware.AuthMiddleware,
		mwManager.RateLimiterWithPolicy(policies[ratelimit.PolicyEntriesWrite]),
	))

	// Wrap with global middlewares: metrics -> logging -> CORS -> routes
	innerHandler := middleware.MetricsMiddleware(
		middleware.LoggingMiddleware(
			middleware.CORSMiddleware(mux),
		),
	)

	// Wrap with otelhttp for automatic tracing with custom span names
	handler := otelhttp.NewHandler(
		innerHandler,
		"dict-simulator",
		otelhttp.WithTracerProvider(telemetry.TracerProvider),
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			// Use Go 1.22+ pattern matching to get the route pattern
			// r.Pattern contains the matched pattern like "GET /entries/{key}"
			key := r.Method + " " + r.Pattern
			if name, ok := spanNames[key]; ok {
				return name
			}
			// Fallback: use pattern if available, otherwise path
			if r.Pattern != "" {
				return r.Pattern
			}
			return r.URL.Path
		}),
	)

	return handler
}

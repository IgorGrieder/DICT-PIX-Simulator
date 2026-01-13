package router

import (
	"net/http"

	"github.com/dict-simulator/go/internal/middleware"
	"github.com/dict-simulator/go/internal/modules/auth"
	"github.com/dict-simulator/go/internal/modules/entries"
	"github.com/dict-simulator/go/internal/modules/health"
	"github.com/dict-simulator/go/internal/ratelimit"
	"github.com/dict-simulator/go/internal/telemetry"
)

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

	// Health and metrics endpoints
	mux.HandleFunc("GET /health", telemetry.WithTracing("health", healthHandler.Health))
	mux.Handle("GET /metrics", healthHandler.Metrics())

	// Auth routes (no auth middleware)
	mux.HandleFunc("POST /auth/register", telemetry.WithTracing("auth.register", authHandler.Register))
	mux.HandleFunc("POST /auth/login", telemetry.WithTracing("auth.login", authHandler.Login))

	// Entries routes with per-method rate limiting policies
	// POST /entries - createEntry uses ENTRIES_WRITE policy (1200/min, 36000 bucket)
	mux.Handle("POST /entries", middleware.Chain(
		http.HandlerFunc(telemetry.WithTracing("entries.create", entriesHandler.Create)),
		middleware.AuthMiddleware,
		mwManager.RateLimiterWithPolicy(policies[ratelimit.PolicyEntriesWrite]),
		mwManager.Idempotency,
	))

	// GET /entries/{key} - getEntry uses ENTRIES_READ_PARTICIPANT_ANTISCAN policy
	// Category H: 2/min, 50 bucket, 404 costs 3 tokens
	mux.Handle("GET /entries/{key}", middleware.Chain(
		http.HandlerFunc(telemetry.WithTracing("entries.get", entriesHandler.Get)),
		middleware.AuthMiddleware,
		mwManager.RateLimiterWithPolicy(policies[ratelimit.PolicyEntriesReadParticipant]),
	))

	// PUT /entries/{key} - updateEntry uses ENTRIES_UPDATE policy (600/min, 600 bucket)
	mux.Handle("PUT /entries/{key}", middleware.Chain(
		http.HandlerFunc(telemetry.WithTracing("entries.update", entriesHandler.Update)),
		middleware.AuthMiddleware,
		mwManager.RateLimiterWithPolicy(policies[ratelimit.PolicyEntriesUpdate]),
	))

	// DELETE /entries/{key} - deleteEntry uses ENTRIES_WRITE policy (same as create)
	mux.Handle("DELETE /entries/{key}", middleware.Chain(
		http.HandlerFunc(telemetry.WithTracing("entries.delete", entriesHandler.Delete)),
		middleware.AuthMiddleware,
		mwManager.RateLimiterWithPolicy(policies[ratelimit.PolicyEntriesWrite]),
	))

	// Wrap with global middlewares: metrics -> logging -> CORS -> routes
	handler := middleware.MetricsMiddleware(
		middleware.LoggingMiddleware(
			middleware.CORSMiddleware(mux),
		),
	)

	return handler
}

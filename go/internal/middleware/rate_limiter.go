package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/dict-simulator/go/internal/httputil"
	"github.com/dict-simulator/go/internal/ratelimit"
)

// responseCapture wraps http.ResponseWriter to capture the status code
type responseCapture struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (r *responseCapture) WriteHeader(code int) {
	if !r.written {
		r.statusCode = code
		r.written = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseCapture) Write(b []byte) (int, error) {
	if !r.written {
		r.statusCode = http.StatusOK
		r.written = true
	}
	return r.ResponseWriter.Write(b)
}

// RateLimiterWithPolicy creates a rate limiting middleware for a specific policy
// This middleware:
// 1. Checks if the request is allowed before processing
// 2. Captures the response status code
// 3. Deducts tokens based on the response (error-based counting)
func (m *Manager) RateLimiterWithPolicy(policy ratelimit.Policy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting if disabled
			if !m.rateLimitEnabled {
				next.ServeHTTP(w, r)
				return
			}

			if m.rateLimiter == nil {
				// Fail open if bucket not initialized
				next.ServeHTTP(w, r)
				return
			}

			// Get identifier (participant ID from header, fallback to user ID)
			identifier := r.Header.Get("X-Participant-Id")
			if identifier == "" {
				identifier = r.Header.Get("X-User-Id")
			}
			if identifier == "" {
				identifier = "anonymous"
			}

			ctx := r.Context()

			// Pre-check: verify there's capacity in the bucket
			state, err := m.rateLimiter.Check(ctx, policy, identifier)
			if err != nil {
				// Fail open on Redis errors
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			setRateLimitHeaders(w, policy, state)

			// If no tokens available, return 429
			if !state.Allowed {
				writeRateLimitError(w, r)
				return
			}

			// Wrap response writer to capture status code
			capture := &responseCapture{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Process the request
			next.ServeHTTP(capture, r)

			// Post-response: deduct tokens based on actual status code
			// This implements the DICT spec error-based counting:
			// - 2xx: subtract SuccessCost (usually 1)
			// - 404: subtract NotFoundCost (can be 3 for antiscan)
			// - 5xx: skip deduction if IgnoreOn5xx is true
			_ = m.rateLimiter.Consume(ctx, policy, identifier, capture.statusCode)
		})
	}
}

// setRateLimitHeaders adds standard rate limit headers to the response
func setRateLimitHeaders(w http.ResponseWriter, policy ratelimit.Policy, state *ratelimit.BucketState) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(policy.BucketSize))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(state.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(state.Reset, 10))
	w.Header().Set("X-RateLimit-Policy", string(policy.Name))
}

// writeRateLimitError writes a 429 Too Many Requests response with DICT-compliant format
func writeRateLimitError(w http.ResponseWriter, r *http.Request) {
	correlationID := httputil.GetCorrelationID(r)

	response := httputil.APIResponse{
		ResponseTime:  time.Now().UTC(),
		CorrelationId: correlationID,
		Error:         "TOO_MANY_REQUESTS",
		Message:       "Rate limit exceeded. Please try again later.",
	}

	w.Header().Set(httputil.CorrelationIDHeader, correlationID)
	httputil.WriteJSON(w, http.StatusTooManyRequests, response)
}

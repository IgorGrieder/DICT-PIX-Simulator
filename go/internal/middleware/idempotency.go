package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

const IdempotencyKeyHeader = "X-Idempotency-Key"

// responseRecorder captures the response for idempotency storage
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.body.Write(b)
	return rr.ResponseWriter.Write(b)
}

// Idempotency handles idempotent requests
func (m *Manager) Idempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idempotencyKey := r.Header.Get(IdempotencyKeyHeader)

		// If no idempotency key, proceed normally
		if idempotencyKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()

		// Try to atomically insert a "processing" record to claim this key
		// This prevents race conditions between concurrent requests
		claimed, record, err := m.idempotencyRepo.ClaimKey(ctx, idempotencyKey)
		if err != nil {
			// On error, proceed with the request
			next.ServeHTTP(w, r)
			return
		}

		// If we didn't claim the key, return the existing response
		if !claimed && record != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.StatusCode)
			w.Write([]byte(record.Response))
			return
		}

		// We claimed the key, process the request
		recorder := newResponseRecorder(w)
		next.ServeHTTP(recorder, r)

		// Store the response as raw JSON string (fire and forget, but synchronous to avoid data races)
		responseBody := recorder.body.String()
		if json.Valid([]byte(responseBody)) {
			m.idempotencyRepo.Save(context.Background(), idempotencyKey, responseBody, recorder.statusCode)
		}
	})
}

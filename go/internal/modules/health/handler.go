package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler handles health and metrics endpoints
type Handler struct{}

// NewHandler creates a new health handler
func NewHandler() *Handler {
	return &Handler{}
}

// Health returns the health status of the service
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Metrics returns Prometheus metrics
func (h *Handler) Metrics() http.Handler {
	return promhttp.Handler()
}

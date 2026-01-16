package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status" example:"ok"`
	Timestamp string `json:"timestamp" example:"2024-01-15T10:30:00Z"`
}

// Handler handles health and metrics endpoints
type Handler struct{}

// NewHandler creates a new health handler
func NewHandler() *Handler {
	return &Handler{}
}

// Health returns the health status of the service
//
//	@Summary		Health check
//	@Description	Returns the health status of the service
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	HealthResponse	"Service is healthy"
//	@Router			/health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Metrics returns Prometheus metrics
//
//	@Summary		Prometheus metrics
//	@Description	Returns Prometheus metrics for monitoring
//	@Tags			health
//	@Produce		text/plain
//	@Success		200	{string}	string	"Prometheus metrics in text format"
//	@Router			/metrics [get]
func (h *Handler) Metrics() http.Handler {
	return promhttp.Handler()
}

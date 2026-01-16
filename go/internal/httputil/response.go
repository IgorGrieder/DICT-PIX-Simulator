package httputil

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/dict-simulator/go/internal/constants"
)

// CorrelationIDHeader is the header name for correlation ID
const CorrelationIDHeader = "X-Correlation-Id"

// APIResponse wraps all API responses with DICT-compliant metadata
// Per DICT spec, responses include ResponseTime and CorrelationId
type APIResponse struct {
	ResponseTime  time.Time `json:"responseTime" example:"2024-01-15T10:30:00Z"`
	CorrelationId string    `json:"correlationId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code          string    `json:"code,omitempty" example:"ENTRY_CREATED"`
	Data          any       `json:"data,omitempty"`
	Error         string    `json:"error,omitempty" example:"INVALID_REQUEST"`
	Message       string    `json:"message,omitempty" example:"Request processed successfully"`
}

// ErrorResponse represents a standard error response (for backwards compatibility)
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// GetCorrelationID extracts the correlation ID from the request header
// If not present, generates a new UUID v4
func GetCorrelationID(r *http.Request) string {
	correlationID := r.Header.Get(CorrelationIDHeader)
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	return correlationID
}

// WriteJSON writes a JSON response with the given status code
// This is the legacy function for backwards compatibility
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteAPIResponse writes a DICT-compliant API response with metadata
// Includes ResponseTime and CorrelationId from request header
func WriteAPIResponse(w http.ResponseWriter, r *http.Request, status int, data any) {
	correlationID := GetCorrelationID(r)

	// Set correlation ID in response header as well
	w.Header().Set(CorrelationIDHeader, correlationID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := APIResponse{
		ResponseTime:  time.Now().UTC(),
		CorrelationId: correlationID,
		Data:          data,
	}

	json.NewEncoder(w).Encode(response)
}

// WriteAPIError writes a DICT-compliant error response with metadata using a predefined APIError.
// Includes ResponseTime and CorrelationId from request header.
func WriteAPIError(w http.ResponseWriter, r *http.Request, apiErr constants.APIError) {
	correlationID := GetCorrelationID(r)

	// Set correlation ID in response header as well
	w.Header().Set(CorrelationIDHeader, correlationID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)

	response := APIResponse{
		ResponseTime:  time.Now().UTC(),
		CorrelationId: correlationID,
		Error:         apiErr.Code,
		Message:       apiErr.Message,
	}

	json.NewEncoder(w).Encode(response)
}

// WriteError writes a DICT-compliant error response without requiring an http.Request.
// Useful for middleware that may not have access to the full request context.
// Note: Does not include CorrelationId since there's no request to extract it from.
func WriteError(w http.ResponseWriter, apiErr constants.APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)

	response := APIResponse{
		ResponseTime: time.Now().UTC(),
		Error:        apiErr.Code,
		Message:      apiErr.Message,
	}

	json.NewEncoder(w).Encode(response)
}

// WriteAPISuccess writes a DICT-compliant success response with metadata using a predefined APISuccess.
// Includes ResponseTime, CorrelationId, success code, and data.
func WriteAPISuccess(w http.ResponseWriter, r *http.Request, apiSuccess constants.APISuccess, data any) {
	correlationID := GetCorrelationID(r)

	// Set correlation ID in response header as well
	w.Header().Set(CorrelationIDHeader, correlationID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiSuccess.Status)

	response := APIResponse{
		ResponseTime:  time.Now().UTC(),
		CorrelationId: correlationID,
		Code:          apiSuccess.Code,
		Data:          data,
	}

	json.NewEncoder(w).Encode(response)
}

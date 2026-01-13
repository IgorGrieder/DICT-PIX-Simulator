package httputil

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WriteJSON writes a JSON response with the given status code
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError writes an error response with the given status code
func WriteError(w http.ResponseWriter, status int, errCode, message string) {
	WriteJSON(w, status, ErrorResponse{
		Error:   errCode,
		Message: message,
	})
}

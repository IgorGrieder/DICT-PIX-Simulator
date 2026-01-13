package entries

import (
	"encoding/json"
	"net/http"

	"github.com/dict-simulator/go/internal/httputil"
	"github.com/dict-simulator/go/internal/models"
	"github.com/dict-simulator/go/internal/validation"
)

// Handler handles entry-related HTTP requests
type Handler struct{}

// NewHandler creates a new entries handler
func NewHandler() *Handler {
	return &Handler{}
}

// Create handles creating a new entry
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate request using validator library
	if err := validation.Validate(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Validate key format based on key type
	validationResult := ValidateKey(req.Key, req.KeyType)
	if !validationResult.Success {
		httputil.WriteJSON(w, http.StatusBadRequest, validationResult.Error)
		return
	}

	ctx := r.Context()

	// Check if key already exists
	existing, err := models.FindEntryByKey(ctx, req.Key)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check existing entry")
		return
	}

	if existing != nil {
		httputil.WriteError(w, http.StatusConflict, "KEY_ALREADY_EXISTS", "This key is already registered in the directory")
		return
	}

	// Create entry
	entry, err := models.CreateEntry(ctx, &req)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create entry")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, entry.ToResponse())
}

// Get handles getting an entry by key
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Key is required")
		return
	}

	ctx := r.Context()

	entry, err := models.FindEntryByKey(ctx, key)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to find entry")
		return
	}

	if entry == nil {
		httputil.WriteError(w, http.StatusNotFound, "ENTRY_NOT_FOUND", "No entry found for this key")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, entry.ToResponse())
}

// Delete handles deleting an entry by key
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Key is required")
		return
	}

	ctx := r.Context()

	entry, err := models.DeleteEntryByKey(ctx, key)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete entry")
		return
	}

	if entry == nil {
		httputil.WriteError(w, http.StatusNotFound, "ENTRY_NOT_FOUND", "No entry found for this key")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, models.DeleteEntryResponse{
		Message: "Entry deleted successfully",
		Key:     entry.Key,
	})
}

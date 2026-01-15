package entries

import (
	"encoding/json"
	"net/http"

	"github.com/dict-simulator/go/internal/httputil"
	"github.com/dict-simulator/go/internal/models"
	"github.com/dict-simulator/go/internal/validation"
)

// Handler handles entry-related HTTP requests
type Handler struct {
	repo *models.EntryRepository
}

// NewHandler creates a new entries handler
func NewHandler(repo *models.EntryRepository) *Handler {
	return &Handler{
		repo: repo,
	}
}

// Create handles creating a new entry
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate request using validator library
	if err := validation.Validate(&req); err != nil {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Validate key format based on key type
	validationResult := ValidateKey(req.Key, req.KeyType)
	if !validationResult.Success {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, validationResult.Error.Type, validationResult.Error.Message)
		return
	}

	ctx := r.Context()

	// Check if key already exists
	existing, err := h.repo.FindByKey(ctx, req.Key)
	if err != nil {
		httputil.WriteAPIError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check existing entry")
		return
	}

	if existing != nil {
		httputil.WriteAPIError(w, r, http.StatusConflict, "KEY_ALREADY_EXISTS", "This key is already registered in the directory")
		return
	}

	// Create entry
	entry, err := h.repo.Create(ctx, &req)
	if err != nil {
		httputil.WriteAPIError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create entry")
		return
	}

	httputil.WriteAPIResponse(w, r, http.StatusCreated, entry.ToResponse())
}

// Get handles getting an entry by key
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Key is required")
		return
	}

	ctx := r.Context()

	entry, err := h.repo.FindByKey(ctx, key)
	if err != nil {
		httputil.WriteAPIError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to find entry")
		return
	}

	if entry == nil {
		httputil.WriteAPIError(w, r, http.StatusNotFound, "ENTRY_NOT_FOUND", "No entry found for this key")
		return
	}

	httputil.WriteAPIResponse(w, r, http.StatusOK, entry.ToResponse())
}

// Delete handles deleting an entry by key
// Per DICT spec: POST /entries/{key}/delete with request body
// The participant in the request must match the entry's participant
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Key is required")
		return
	}

	var req models.DeleteEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Ensure key in path matches key in body
	if req.Key != "" && req.Key != key {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Key in path must match key in body")
		return
	}
	req.Key = key

	// Validate request using validator library
	if err := validation.Validate(&req); err != nil {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	ctx := r.Context()

	// Check if entry exists and validate participant
	existing, err := h.repo.FindByKey(ctx, key)
	if err != nil {
		httputil.WriteAPIError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to find entry")
		return
	}

	if existing == nil {
		httputil.WriteAPIError(w, r, http.StatusNotFound, "ENTRY_NOT_FOUND", "No entry found for this key")
		return
	}

	// Verify participant matches the entry's participant (authorization check)
	if existing.Account.Participant != req.Participant {
		httputil.WriteAPIError(w, r, http.StatusForbidden, "FORBIDDEN", "Participant does not match the entry's participant")
		return
	}

	// Delete the entry
	entry, err := h.repo.DeleteByKey(ctx, key)
	if err != nil {
		httputil.WriteAPIError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete entry")
		return
	}

	if entry == nil {
		httputil.WriteAPIError(w, r, http.StatusNotFound, "ENTRY_NOT_FOUND", "No entry found for this key")
		return
	}

	httputil.WriteAPIResponse(w, r, http.StatusOK, models.DeleteEntryResponse{
		Message: "Entry deleted successfully",
		Key:     entry.Key,
	})
}

// Update handles updating an entry by key
// Per DICT spec:
// - EVP keys cannot be updated
// - Only account info, name, and trade name can be updated
// - Valid reasons: USER_REQUESTED, BRANCH_TRANSFER, RECONCILIATION, RFB_VALIDATION
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Key is required")
		return
	}

	var req models.UpdateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Ensure key in path matches key in body
	if req.Key != "" && req.Key != key {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Key in path must match key in body")
		return
	}
	req.Key = key

	// Validate request using validator library
	if err := validation.Validate(&req); err != nil {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	ctx := r.Context()

	// Check if entry exists
	existing, err := h.repo.FindByKey(ctx, key)
	if err != nil {
		httputil.WriteAPIError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to find entry")
		return
	}

	if existing == nil {
		httputil.WriteAPIError(w, r, http.StatusNotFound, "ENTRY_NOT_FOUND", "No entry found for this key")
		return
	}

	// EVP keys cannot be updated per DICT spec
	if existing.KeyType == models.KeyTypeEVP {
		httputil.WriteAPIError(w, r, http.StatusBadRequest, "INVALID_OPERATION", "EVP keys cannot be updated")
		return
	}

	// Update the entry
	entry, err := h.repo.UpdateByKey(ctx, key, &req)
	if err != nil {
		httputil.WriteAPIError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update entry")
		return
	}

	if entry == nil {
		httputil.WriteAPIError(w, r, http.StatusNotFound, "ENTRY_NOT_FOUND", "No entry found for this key")
		return
	}

	httputil.WriteAPIResponse(w, r, http.StatusOK, entry.ToResponse())
}

package entries

import (
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/dict-simulator/go/internal/constants"
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
//
//	@Summary		Create a new DICT entry
//	@Description	Register a new Pix key entry in the DICT system. The key must be unique and valid for its type.
//	@Tags			entries
//	@Accept			json
//	@Produce		json
//	@Param			X-Idempotency-Key	header		string					true	"Idempotency key for request deduplication"
//	@Param			request				body		models.CreateEntryRequest	true	"Entry creation request"
//	@Success		201					{object}	httputil.APIResponse{data=models.EntryResponse}	"Entry created successfully"
//	@Failure		400					{object}	httputil.APIResponse								"Invalid request body or key format"
//	@Failure		401					{object}	httputil.APIResponse								"Unauthorized"
//	@Failure		409					{object}	httputil.APIResponse								"Key already exists"
//	@Failure		429					{object}	httputil.APIResponse								"Rate limit exceeded"
//	@Failure		500					{object}	httputil.APIResponse								"Internal server error"
//	@Security		BearerAuth
//	@Router			/entries [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	var req models.CreateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetStatus(codes.Error, "JSON decode failed")
		span.SetAttributes(
			attribute.String("error.type", "json_decode"),
			attribute.String("error.message", err.Error()),
		)
		span.RecordError(err)
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody.WithMessage("JSON decode error: "+err.Error()))
		return
	}

	// Validate request using validator library
	if err := validation.Validate(&req); err != nil {
		span.SetStatus(codes.Error, "Validation failed")
		span.SetAttributes(
			attribute.String("error.type", "validation"),
			attribute.String("error.message", err.Error()),
		)
		span.RecordError(err)
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody.WithMessage(err.Error()))
		return
	}

	// Validate key format based on key type
	validationResult := ValidateKey(req.Key, req.KeyType)
	if !validationResult.Success {
		span.SetStatus(codes.Error, "Key validation failed")
		span.SetAttributes(
			attribute.String("error.type", "key_validation"),
			attribute.String("error.message", validationResult.Error.Message),
		)
		httputil.WriteAPIError(w, r, constants.APIError{
			Code:    validationResult.Error.Type,
			Message: validationResult.Error.Message,
			Status:  http.StatusBadRequest,
		})
		return
	}

	// Check if key already exists
	existing, err := h.repo.FindByKey(ctx, req.Key)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToCheckEntry)
		return
	}

	if existing != nil {
		httputil.WriteAPIError(w, r, constants.ErrKeyAlreadyExists)
		return
	}

	// Create entry
	entry, err := h.repo.Create(ctx, &req)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToCreateEntry)
		return
	}

	httputil.WriteAPISuccess(w, r, constants.SuccessEntryCreated, entry.ToResponse())
}

// Get handles getting an entry by key
//
//	@Summary		Get a DICT entry by key
//	@Description	Retrieve a Pix key entry from the DICT system using the key value
//	@Tags			entries
//	@Accept			json
//	@Produce		json
//	@Param			key	path		string	true	"The Pix key to retrieve (CPF, CNPJ, EMAIL, PHONE, or EVP)"
//	@Success		200	{object}	httputil.APIResponse{data=models.EntryResponse}	"Entry found"
//	@Failure		400	{object}	httputil.APIResponse								"Key is required"
//	@Failure		401	{object}	httputil.APIResponse								"Unauthorized"
//	@Failure		404	{object}	httputil.APIResponse								"Entry not found"
//	@Failure		429	{object}	httputil.APIResponse								"Rate limit exceeded"
//	@Failure		500	{object}	httputil.APIResponse								"Internal server error"
//	@Security		BearerAuth
//	@Router			/entries/{key} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httputil.WriteAPIError(w, r, constants.ErrKeyRequired)
		return
	}

	ctx := r.Context()

	entry, err := h.repo.FindByKey(ctx, key)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToFindEntry)
		return
	}

	if entry == nil {
		httputil.WriteAPIError(w, r, constants.ErrEntryNotFound)
		return
	}

	httputil.WriteAPISuccess(w, r, constants.SuccessEntryFound, entry.ToResponse())
}

// Delete handles deleting an entry by key
// Per DICT spec: POST /entries/{key}/delete with request body
// The participant in the request must match the entry's participant
//
//	@Summary		Delete a DICT entry
//	@Description	Delete a Pix key entry from the DICT system. The requesting participant must own the entry.
//	@Tags			entries
//	@Accept			json
//	@Produce		json
//	@Param			key		path		string						true	"The Pix key to delete"
//	@Param			request	body		models.DeleteEntryRequest	true	"Delete entry request with participant and reason"
//	@Success		200		{object}	httputil.APIResponse{data=models.DeleteEntryResponse}	"Entry deleted successfully"
//	@Failure		400		{object}	httputil.APIResponse										"Invalid request body or key mismatch"
//	@Failure		401		{object}	httputil.APIResponse										"Unauthorized"
//	@Failure		403		{object}	httputil.APIResponse										"Forbidden - participant mismatch"
//	@Failure		404		{object}	httputil.APIResponse										"Entry not found"
//	@Failure		429		{object}	httputil.APIResponse										"Rate limit exceeded"
//	@Failure		500		{object}	httputil.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/entries/{key}/delete [post]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	key := r.PathValue("key")
	if key == "" {
		httputil.WriteAPIError(w, r, constants.ErrKeyRequired)
		return
	}

	var req models.DeleteEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetStatus(codes.Error, "JSON decode failed")
		span.SetAttributes(
			attribute.String("error.type", "json_decode"),
			attribute.String("error.message", err.Error()),
		)
		span.RecordError(err)
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody.WithMessage("JSON decode error: "+err.Error()))
		return
	}

	// Ensure key in path matches key in body
	if req.Key != "" && req.Key != key {
		httputil.WriteAPIError(w, r, constants.ErrKeyMismatch)
		return
	}
	req.Key = key

	// Validate request using validator library
	if err := validation.Validate(&req); err != nil {
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody.WithMessage(err.Error()))
		return
	}

	// Check if entry exists and validate participant
	existing, err := h.repo.FindByKey(ctx, key)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToFindEntry)
		return
	}

	if existing == nil {
		httputil.WriteAPIError(w, r, constants.ErrEntryNotFound)
		return
	}

	// Verify participant matches the entry's participant (authorization check)
	if existing.Account.Participant != req.Participant {
		httputil.WriteAPIError(w, r, constants.ErrForbiddenParticipant)
		return
	}

	// Delete the entry
	entry, err := h.repo.DeleteByKey(ctx, key)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToDeleteEntry)
		return
	}

	if entry == nil {
		httputil.WriteAPIError(w, r, constants.ErrEntryNotFound)
		return
	}

	httputil.WriteAPISuccess(w, r, constants.SuccessEntryDeleted, models.DeleteEntryResponse{
		Message: "Entry deleted successfully",
		Key:     entry.Key,
	})
}

// Update handles updating an entry by key
// Per DICT spec:
// - EVP keys cannot be updated
// - Only account info, name, and trade name can be updated
// - Valid reasons: USER_REQUESTED, BRANCH_TRANSFER, RECONCILIATION, RFB_VALIDATION
//
//	@Summary		Update a DICT entry
//	@Description	Update an existing Pix key entry. EVP keys cannot be updated. Only account info, name, and trade name can be modified.
//	@Tags			entries
//	@Accept			json
//	@Produce		json
//	@Param			key		path		string						true	"The Pix key to update"
//	@Param			request	body		models.UpdateEntryRequest	true	"Update entry request"
//	@Success		200		{object}	httputil.APIResponse{data=models.EntryResponse}	"Entry updated successfully"
//	@Failure		400		{object}	httputil.APIResponse								"Invalid request body, key mismatch, or EVP key update attempt"
//	@Failure		401		{object}	httputil.APIResponse								"Unauthorized"
//	@Failure		404		{object}	httputil.APIResponse								"Entry not found"
//	@Failure		429		{object}	httputil.APIResponse								"Rate limit exceeded"
//	@Failure		500		{object}	httputil.APIResponse								"Internal server error"
//	@Security		BearerAuth
//	@Router			/entries/{key} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	key := r.PathValue("key")
	if key == "" {
		httputil.WriteAPIError(w, r, constants.ErrKeyRequired)
		return
	}

	var req models.UpdateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetStatus(codes.Error, "JSON decode failed")
		span.SetAttributes(
			attribute.String("error.type", "json_decode"),
			attribute.String("error.message", err.Error()),
		)
		span.RecordError(err)
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody.WithMessage("JSON decode error: "+err.Error()))
		return
	}

	// Ensure key in path matches key in body
	if req.Key != "" && req.Key != key {
		httputil.WriteAPIError(w, r, constants.ErrKeyMismatch)
		return
	}
	req.Key = key

	// Validate request using validator library
	if err := validation.Validate(&req); err != nil {
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody.WithMessage(err.Error()))
		return
	}

	// Check if entry exists
	existing, err := h.repo.FindByKey(ctx, key)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToFindEntry)
		return
	}

	if existing == nil {
		httputil.WriteAPIError(w, r, constants.ErrEntryNotFound)
		return
	}

	// EVP keys cannot be updated per DICT spec
	if existing.KeyType == models.KeyTypeEVP {
		httputil.WriteAPIError(w, r, constants.ErrEVPKeyNotUpdatable)
		return
	}

	// Update the entry
	entry, err := h.repo.UpdateByKey(ctx, key, &req)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToUpdateEntry)
		return
	}

	if entry == nil {
		httputil.WriteAPIError(w, r, constants.ErrEntryNotFound)
		return
	}

	httputil.WriteAPISuccess(w, r, constants.SuccessEntryUpdated, entry.ToResponse())
}

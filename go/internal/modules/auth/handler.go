package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/dict-simulator/go/internal/constants"
	"github.com/dict-simulator/go/internal/httputil"
	"github.com/dict-simulator/go/internal/middleware"
	"github.com/dict-simulator/go/internal/models"
	"github.com/dict-simulator/go/internal/validation"
)

// RegisterRequest represents the register request body
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required,min=6" example:"password123"`
	Name     string `json:"name" validate:"required" example:"John Doe"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"password123"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	Token string              `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  models.UserResponse `json:"user"`
}

// Handler handles auth-related HTTP requests
type Handler struct {
	repo      *models.UserRepository
	jwtSecret string
}

// NewHandler creates a new auth handler
func NewHandler(repo *models.UserRepository, jwtSecret string) *Handler {
	return &Handler{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

// Register handles user registration
//
//	@Summary		Register a new user
//	@Description	Create a new user account with email, password, and name. Returns a JWT token on success.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest									true	"User registration details"
//	@Success		201		{object}	httputil.APIResponse{data=AuthResponse}		"User registered successfully"
//	@Failure		400		{object}	httputil.APIResponse							"Invalid request body"
//	@Failure		409		{object}	httputil.APIResponse							"User already exists"
//	@Failure		500		{object}	httputil.APIResponse							"Internal server error"
//	@Router			/auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	var req RegisterRequest
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
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody.WithMessage(err.Error()))
		return
	}

	// Check if user already exists
	existingUser, err := h.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToCheckUser)
		return
	}

	if existingUser != nil {
		httputil.WriteAPIError(w, r, constants.ErrUserAlreadyExists)
		return
	}

	// Create user
	user, err := h.repo.Create(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToCreateUser)
		return
	}

	// Generate JWT
	token, err := h.generateToken(user)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToGenerateToken)
		return
	}

	httputil.WriteAPISuccess(w, r, constants.SuccessUserRegistered, AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}

// Login handles user login
//
//	@Summary		User login
//	@Description	Authenticate a user with email and password. Returns a JWT token on success.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest								true	"User login credentials"
//	@Success		200		{object}	httputil.APIResponse{data=AuthResponse}	"Login successful"
//	@Failure		400		{object}	httputil.APIResponse						"Invalid request body"
//	@Failure		401		{object}	httputil.APIResponse						"Invalid credentials"
//	@Failure		500		{object}	httputil.APIResponse						"Internal server error"
//	@Router			/auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	var req LoginRequest
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
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody.WithMessage(err.Error()))
		return
	}

	// Find user
	user, err := h.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToFindUser)
		return
	}

	if user == nil {
		httputil.WriteAPIError(w, r, constants.ErrInvalidCredentials)
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		httputil.WriteAPIError(w, r, constants.ErrInvalidCredentials)
		return
	}

	// Generate JWT
	token, err := h.generateToken(user)
	if err != nil {
		httputil.WriteAPIError(w, r, constants.ErrFailedToGenerateToken)
		return
	}

	httputil.WriteAPISuccess(w, r, constants.SuccessLoginSuccess, AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}

func (h *Handler) generateToken(user *models.User) (string, error) {
	claims := middleware.JWTClaims{
		UserID: user.ID.Hex(),
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

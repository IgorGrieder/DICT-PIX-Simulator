package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/dict-simulator/go/internal/constants"
	"github.com/dict-simulator/go/internal/httputil"
	"github.com/dict-simulator/go/internal/middleware"
	"github.com/dict-simulator/go/internal/models"
	"github.com/dict-simulator/go/internal/validation"
)

// RegisterRequest represents the register request body
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	Token string              `json:"token"`
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
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody)
		return
	}

	// Validate request using validator library
	if err := validation.Validate(&req); err != nil {
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody.WithMessage(err.Error()))
		return
	}

	ctx := r.Context()

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
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody)
		return
	}

	// Validate request using validator library
	if err := validation.Validate(&req); err != nil {
		httputil.WriteAPIError(w, r, constants.ErrInvalidRequestBody.WithMessage(err.Error()))
		return
	}

	ctx := r.Context()

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

// Me handles getting current user
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		httputil.WriteAPIError(w, r, constants.ErrUnauthorized)
		return
	}

	// For now, we just return the ID as we don't have FindByID yet
	httputil.WriteAPISuccess(w, r, constants.SuccessUserFound, map[string]string{
		"id": userID,
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

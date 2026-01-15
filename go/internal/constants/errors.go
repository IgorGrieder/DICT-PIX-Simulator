package constants

import "net/http"

// APIError represents a standardized API error with code, message, and HTTP status.
// Use these predefined errors for consistent API responses across the application.
type APIError struct {
	Code    string
	Message string
	Status  int
}

// WithMessage returns a copy of the APIError with a custom message.
// Useful for validation errors or other dynamic messages.
func (e APIError) WithMessage(message string) APIError {
	return APIError{
		Code:    e.Code,
		Message: message,
		Status:  e.Status,
	}
}

// Common errors - shared across multiple modules
var (
	ErrInvalidRequestBody = APIError{
		Code:    CodeInvalidRequest,
		Message: MsgInvalidRequestBody,
		Status:  http.StatusBadRequest,
	}
	ErrKeyRequired = APIError{
		Code:    CodeInvalidRequest,
		Message: MsgKeyRequired,
		Status:  http.StatusBadRequest,
	}
	ErrKeyMismatch = APIError{
		Code:    CodeInvalidRequest,
		Message: MsgKeyMismatch,
		Status:  http.StatusBadRequest,
	}
	ErrInternalError = APIError{
		Code:    CodeInternalError,
		Message: MsgInternalError,
		Status:  http.StatusInternalServerError,
	}
)

// Entry-related errors
var (
	ErrEntryNotFound = APIError{
		Code:    CodeEntryNotFound,
		Message: MsgEntryNotFound,
		Status:  http.StatusNotFound,
	}
	ErrKeyAlreadyExists = APIError{
		Code:    CodeKeyAlreadyExists,
		Message: MsgKeyAlreadyExists,
		Status:  http.StatusConflict,
	}
	ErrFailedToCheckEntry = APIError{
		Code:    CodeInternalError,
		Message: MsgFailedToCheckEntry,
		Status:  http.StatusInternalServerError,
	}
	ErrFailedToFindEntry = APIError{
		Code:    CodeInternalError,
		Message: MsgFailedToFindEntry,
		Status:  http.StatusInternalServerError,
	}
	ErrFailedToCreateEntry = APIError{
		Code:    CodeInternalError,
		Message: MsgFailedToCreateEntry,
		Status:  http.StatusInternalServerError,
	}
	ErrFailedToUpdateEntry = APIError{
		Code:    CodeInternalError,
		Message: MsgFailedToUpdateEntry,
		Status:  http.StatusInternalServerError,
	}
	ErrFailedToDeleteEntry = APIError{
		Code:    CodeInternalError,
		Message: MsgFailedToDeleteEntry,
		Status:  http.StatusInternalServerError,
	}
	ErrEVPKeyNotUpdatable = APIError{
		Code:    CodeInvalidOperation,
		Message: MsgEVPKeyNotUpdatable,
		Status:  http.StatusBadRequest,
	}
	ErrForbiddenParticipant = APIError{
		Code:    CodeForbidden,
		Message: MsgForbiddenParticipant,
		Status:  http.StatusForbidden,
	}
)

// Auth-related errors
var (
	ErrUserAlreadyExists = APIError{
		Code:    CodeUserAlreadyExists,
		Message: MsgUserAlreadyExists,
		Status:  http.StatusConflict,
	}
	ErrInvalidCredentials = APIError{
		Code:    CodeInvalidCredentials,
		Message: MsgInvalidCredentials,
		Status:  http.StatusUnauthorized,
	}
	ErrUnauthorized = APIError{
		Code:    CodeUnauthorized,
		Message: MsgUserNotFound,
		Status:  http.StatusUnauthorized,
	}
	ErrAuthHeaderRequired = APIError{
		Code:    CodeUnauthorized,
		Message: MsgAuthHeaderRequired,
		Status:  http.StatusUnauthorized,
	}
	ErrInvalidToken = APIError{
		Code:    CodeUnauthorized,
		Message: MsgInvalidToken,
		Status:  http.StatusUnauthorized,
	}
	ErrInvalidTokenClaims = APIError{
		Code:    CodeUnauthorized,
		Message: MsgInvalidTokenClaims,
		Status:  http.StatusUnauthorized,
	}
	ErrFailedToCheckUser = APIError{
		Code:    CodeInternalError,
		Message: MsgFailedToCheckUser,
		Status:  http.StatusInternalServerError,
	}
	ErrFailedToFindUser = APIError{
		Code:    CodeInternalError,
		Message: MsgFailedToFindUser,
		Status:  http.StatusInternalServerError,
	}
	ErrFailedToCreateUser = APIError{
		Code:    CodeInternalError,
		Message: MsgFailedToCreateUser,
		Status:  http.StatusInternalServerError,
	}
	ErrFailedToGenerateToken = APIError{
		Code:    CodeInternalError,
		Message: MsgFailedToGenerateToken,
		Status:  http.StatusInternalServerError,
	}
)

// Rate limiting errors
var (
	ErrTooManyRequests = APIError{
		Code:    CodeTooManyRequests,
		Message: MsgTooManyRequests,
		Status:  http.StatusTooManyRequests,
	}
	ErrRateLimitInternal = APIError{
		Code:    CodeInternalError,
		Message: MsgRateLimitInternal,
		Status:  http.StatusInternalServerError,
	}
)

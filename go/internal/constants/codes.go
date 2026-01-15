package constants

// Error codes used in API responses.
// These are the machine-readable codes returned in the "error" field.
const (
	// Common error codes
	CodeInvalidRequest = "INVALID_REQUEST"
	CodeInternalError  = "INTERNAL_ERROR"
	CodeForbidden      = "FORBIDDEN"

	// Entry-specific codes
	CodeEntryNotFound    = "ENTRY_NOT_FOUND"
	CodeKeyAlreadyExists = "KEY_ALREADY_EXISTS"
	CodeInvalidOperation = "INVALID_OPERATION"

	// Auth-specific codes
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeUserAlreadyExists  = "USER_ALREADY_EXISTS"

	// Rate limiting codes
	CodeTooManyRequests = "TOO_MANY_REQUESTS"

	// Success codes - Entry operations
	CodeEntryCreated = "ENTRY_CREATED"
	CodeEntryFound   = "ENTRY_FOUND"
	CodeEntryUpdated = "ENTRY_UPDATED"
	CodeEntryDeleted = "ENTRY_DELETED"

	// Success codes - Auth operations
	CodeUserRegistered = "USER_REGISTERED"
	CodeLoginSuccess   = "LOGIN_SUCCESS"
	CodeUserFound      = "USER_FOUND"
)

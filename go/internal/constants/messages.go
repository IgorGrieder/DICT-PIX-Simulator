package constants

// Error messages used in API responses.
// These are the human-readable messages returned in the "message" field.
const (
	// Common messages
	MsgInvalidRequestBody = "Invalid request body"
	MsgKeyRequired        = "Key is required"
	MsgKeyMismatch        = "Key in path must match key in body"
	MsgInternalError      = "An internal error occurred"

	// Entry-specific messages
	MsgEntryNotFound        = "No entry found for this key"
	MsgKeyAlreadyExists     = "This key is already registered in the directory"
	MsgFailedToCheckEntry   = "Failed to check existing entry"
	MsgFailedToFindEntry    = "Failed to find entry"
	MsgFailedToCreateEntry  = "Failed to create entry"
	MsgFailedToUpdateEntry  = "Failed to update entry"
	MsgFailedToDeleteEntry  = "Failed to delete entry"
	MsgEVPKeyNotUpdatable   = "EVP keys cannot be updated"
	MsgForbiddenParticipant = "Participant does not match the entry's participant"

	// Auth-specific messages
	MsgUserAlreadyExists     = "User with this email already exists"
	MsgInvalidCredentials    = "Invalid email or password"
	MsgUserNotFound          = "User ID not found"
	MsgAuthHeaderRequired    = "Authorization header is required"
	MsgInvalidToken          = "Invalid or expired token"
	MsgInvalidTokenClaims    = "Invalid token claims"
	MsgFailedToCheckUser     = "Failed to check existing user"
	MsgFailedToFindUser      = "Failed to find user"
	MsgFailedToCreateUser    = "Failed to create user"
	MsgFailedToGenerateToken = "Failed to generate token"

	// Rate limiting messages
	MsgTooManyRequests   = "Rate limit exceeded. Please try again later."
	MsgRateLimitInternal = "Rate limit check failed"
)

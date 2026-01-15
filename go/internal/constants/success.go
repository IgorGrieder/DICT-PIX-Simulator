package constants

import "net/http"

// APISuccess represents a standardized API success response with code and HTTP status.
// Use these predefined success constants for consistent API responses across the application.
type APISuccess struct {
	Code   string
	Status int
}

// Entry-related success responses
var (
	SuccessEntryCreated = APISuccess{
		Code:   CodeEntryCreated,
		Status: http.StatusCreated,
	}
	SuccessEntryFound = APISuccess{
		Code:   CodeEntryFound,
		Status: http.StatusOK,
	}
	SuccessEntryUpdated = APISuccess{
		Code:   CodeEntryUpdated,
		Status: http.StatusOK,
	}
	SuccessEntryDeleted = APISuccess{
		Code:   CodeEntryDeleted,
		Status: http.StatusOK,
	}
)

// Auth-related success responses
var (
	SuccessUserRegistered = APISuccess{
		Code:   CodeUserRegistered,
		Status: http.StatusCreated,
	}
	SuccessLoginSuccess = APISuccess{
		Code:   CodeLoginSuccess,
		Status: http.StatusOK,
	}
	SuccessUserFound = APISuccess{
		Code:   CodeUserFound,
		Status: http.StatusOK,
	}
)

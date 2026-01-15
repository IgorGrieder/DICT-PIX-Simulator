package entries

import (
	"regexp"
	"strings"

	"github.com/dict-simulator/go/internal/models"
	"github.com/dict-simulator/go/internal/validation"
)

// ValidationError represents a key validation error
type ValidationError struct {
	Type    string `json:"error"`
	Message string `json:"message"`
}

// ValidationResult represents the result of key validation
type ValidationResult struct {
	Success bool
	Error   *ValidationError
}

// ValidateKey validates a key based on its type
func ValidateKey(key string, keyType models.KeyType) ValidationResult {
	switch keyType {
	case models.KeyTypeCPF:
		return validateCPF(key)
	case models.KeyTypeCNPJ:
		return validateCNPJ(key)
	case models.KeyTypeEMAIL:
		return validateEmail(key)
	case models.KeyTypePHONE:
		return validatePhone(key)
	case models.KeyTypeEVP:
		return validateEVP(key)
	default:
		return ValidationResult{
			Success: false,
			Error: &ValidationError{
				Type:    "INVALID_KEY_TYPE",
				Message: "Invalid key type",
			},
		}
	}
}

// validateCPF validates a CPF using Módulo 11 algorithm
func validateCPF(cpf string) ValidationResult {
	invalidResult := ValidationResult{
		Success: false,
		Error: &ValidationError{
			Type:    "INVALID_CPF",
			Message: "Invalid CPF format",
		},
	}

	// Must be 11 digits
	if matched, _ := regexp.MatchString(`^\d{11}$`, cpf); !matched {
		return invalidResult
	}

	if !validation.IsValidCPF(cpf) {
		return invalidResult
	}

	return ValidationResult{Success: true}
}

// validateCNPJ validates a CNPJ using Módulo 11 algorithm
func validateCNPJ(cnpj string) ValidationResult {
	invalidResult := ValidationResult{
		Success: false,
		Error: &ValidationError{
			Type:    "INVALID_CNPJ",
			Message: "Invalid CNPJ format",
		},
	}

	// Must be 14 digits
	if matched, _ := regexp.MatchString(`^\d{14}$`, cnpj); !matched {
		return invalidResult
	}

	if !validation.IsValidCNPJ(cnpj) {
		return invalidResult
	}

	return ValidationResult{Success: true}
}

// validateEmail validates an email address per DICT spec
// DICT spec: ^[a-z0-9.!#$&'*+/=?^_`{|}~-]+@[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$
// Note: Email must be lowercase and max 77 characters
func validateEmail(email string) ValidationResult {
	invalidResult := ValidationResult{
		Success: false,
		Error: &ValidationError{
			Type:    "INVALID_EMAIL",
			Message: "Invalid email format",
		},
	}

	// Max 77 chars as per DICT spec
	if len(email) > 77 {
		return invalidResult
	}

	// DICT spec requires lowercase emails
	if email != strings.ToLower(email) {
		return ValidationResult{
			Success: false,
			Error: &ValidationError{
				Type:    "INVALID_EMAIL",
				Message: "Email must be lowercase",
			},
		}
	}

	// DICT spec regex - only lowercase allowed
	emailRegex := regexp.MustCompile(`^[a-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$`)
	if !emailRegex.MatchString(email) {
		return invalidResult
	}

	return ValidationResult{Success: true}
}

// validatePhone validates a phone number per DICT spec
// DICT spec: ^\+[1-9]\d{1,14}$
// Supports international E.164 format (not just Brazil)
// E.164 requires minimum 8 digits total for valid phone numbers
func validatePhone(phone string) ValidationResult {
	invalidResult := ValidationResult{
		Success: false,
		Error: &ValidationError{
			Type:    "INVALID_PHONE",
			Message: "Invalid phone format",
		},
	}

	// DICT spec: E.164 international format
	// Must start with + followed by country code (1-9) and up to 14 more digits
	// Minimum length: +XX (country) + NNNNNN (subscriber) = at least 8 chars total
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
	if !phoneRegex.MatchString(phone) {
		return invalidResult
	}

	return ValidationResult{Success: true}
}

// validateEVP validates an EVP (UUID v4)
func validateEVP(evp string) ValidationResult {
	invalidResult := ValidationResult{
		Success: false,
		Error: &ValidationError{
			Type:    "INVALID_EVP",
			Message: "Invalid EVP format",
		},
	}

	evp = strings.ToLower(evp)
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(evp) {
		return invalidResult
	}

	return ValidationResult{Success: true}
}

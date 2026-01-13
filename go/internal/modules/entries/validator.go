package entries

import (
	"regexp"
	"strings"

	"github.com/dict-simulator/go/internal/models"
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

	// All same digits is invalid
	if matched, _ := regexp.MatchString(`^(\d)\1{10}$`, cpf); matched {
		return invalidResult
	}

	digits := make([]int, 11)
	for i, c := range cpf {
		digits[i] = int(c - '0')
	}

	// First check digit
	sum := 0
	for i := 0; i < 9; i++ {
		sum += digits[i] * (10 - i)
	}
	remainder := (sum * 10) % 11
	if remainder == 10 {
		remainder = 0
	}
	if remainder != digits[9] {
		return invalidResult
	}

	// Second check digit
	sum = 0
	for i := 0; i < 10; i++ {
		sum += digits[i] * (11 - i)
	}
	remainder = (sum * 10) % 11
	if remainder == 10 {
		remainder = 0
	}
	if remainder != digits[10] {
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

	// All same digits is invalid
	if matched, _ := regexp.MatchString(`^(\d)\1{13}$`, cnpj); matched {
		return invalidResult
	}

	digits := make([]int, 14)
	for i, c := range cnpj {
		digits[i] = int(c - '0')
	}

	weights1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	weights2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	// First check digit
	sum := 0
	for i := 0; i < 12; i++ {
		sum += digits[i] * weights1[i]
	}
	remainder := sum % 11
	firstCheck := 0
	if remainder >= 2 {
		firstCheck = 11 - remainder
	}
	if firstCheck != digits[12] {
		return invalidResult
	}

	// Second check digit
	sum = 0
	for i := 0; i < 13; i++ {
		sum += digits[i] * weights2[i]
	}
	remainder = sum % 11
	secondCheck := 0
	if remainder >= 2 {
		secondCheck = 11 - remainder
	}
	if secondCheck != digits[13] {
		return invalidResult
	}

	return ValidationResult{Success: true}
}

// validateEmail validates an email address (RFC 5322 simplified)
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

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	if !emailRegex.MatchString(email) {
		return invalidResult
	}

	return ValidationResult{Success: true}
}

// validatePhone validates a phone number (+55 prefix, 10-11 digits)
func validatePhone(phone string) ValidationResult {
	invalidResult := ValidationResult{
		Success: false,
		Error: &ValidationError{
			Type:    "INVALID_PHONE",
			Message: "Invalid phone format",
		},
	}

	phoneRegex := regexp.MustCompile(`^\+55\d{10,11}$`)
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

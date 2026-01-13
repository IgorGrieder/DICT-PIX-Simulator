package validation

import (
	"regexp"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	once     sync.Once
)

// Get returns the singleton validator instance with custom validators registered
func Get() *validator.Validate {
	once.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())

		// Register custom validators for DICT API
		validate.RegisterValidation("participant_id", validateParticipantID)
		validate.RegisterValidation("tax_id", validateTaxID)
		validate.RegisterValidation("evp", validateEVP)
	})
	return validate
}

// Validate validates a struct and returns an error if invalid
func Validate(s any) error {
	return Get().Struct(s)
}

// validateParticipantID validates an 8-digit ISPB participant ID
func validateParticipantID(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	matched, _ := regexp.MatchString(`^[0-9]{8}$`, value)
	return matched
}

// validateTaxID validates a CPF (11 digits) or CNPJ (14 digits)
func validateTaxID(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	// Check if it's a valid CPF or CNPJ format
	if matched, _ := regexp.MatchString(`^[0-9]{11}$`, value); matched {
		return IsValidCPF(value)
	}
	if matched, _ := regexp.MatchString(`^[0-9]{14}$`, value); matched {
		return IsValidCNPJ(value)
	}
	return false
}

// validateEVP validates a UUID v4 format for EVP keys
func validateEVP(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	// EVP must be lowercase UUID v4
	matched, _ := regexp.MatchString(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, value)
	return matched
}

// IsValidCPF validates CPF using Modulo 11 algorithm
func IsValidCPF(cpf string) bool {
	if len(cpf) != 11 {
		return false
	}

	// All same digits is invalid
	if matched, _ := regexp.MatchString(`^(\d)\1{10}$`, cpf); matched {
		return false
	}

	digits := make([]int, 11)
	for i, c := range cpf {
		digits[i] = int(c - '0')
	}

	// First check digit
	sum := 0
	for i := range 9 {
		sum += digits[i] * (10 - i)
	}
	remainder := (sum * 10) % 11
	if remainder == 10 {
		remainder = 0
	}
	if remainder != digits[9] {
		return false
	}

	// Second check digit
	sum = 0
	for i := range 10 {
		sum += digits[i] * (11 - i)
	}
	remainder = (sum * 10) % 11
	if remainder == 10 {
		remainder = 0
	}
	return remainder == digits[10]
}

// IsValidCNPJ validates CNPJ using Modulo 11 algorithm
func IsValidCNPJ(cnpj string) bool {
	if len(cnpj) != 14 {
		return false
	}

	// All same digits is invalid
	if matched, _ := regexp.MatchString(`^(\d)\1{13}$`, cnpj); matched {
		return false
	}

	digits := make([]int, 14)
	for i, c := range cnpj {
		digits[i] = int(c - '0')
	}

	weights1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	weights2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	// First check digit
	sum := 0
	for i := range 12 {
		sum += digits[i] * weights1[i]
	}
	remainder := sum % 11
	firstCheck := 0
	if remainder >= 2 {
		firstCheck = 11 - remainder
	}
	if firstCheck != digits[12] {
		return false
	}

	// Second check digit
	sum = 0
	for i := range 13 {
		sum += digits[i] * weights2[i]
	}
	remainder = sum % 11
	secondCheck := 0
	if remainder >= 2 {
		secondCheck = 11 - remainder
	}
	return secondCheck == digits[13]
}

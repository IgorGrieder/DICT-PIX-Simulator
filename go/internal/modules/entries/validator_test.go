package entries

import (
	"testing"

	"github.com/dict-simulator/go/internal/models"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantOK  bool
		errType string
	}{
		{"valid lowercase email", "test@example.com", true, ""},
		{"valid with dots", "test.user@example.com", true, ""},
		{"valid with plus", "test+tag@example.com", true, ""},
		{"invalid uppercase", "Test@Example.com", false, "INVALID_EMAIL"},
		{"invalid no domain", "test@", false, "INVALID_EMAIL"},
		{"invalid no @", "testexample.com", false, "INVALID_EMAIL"},
		{"too long", string(make([]byte, 78)) + "@a.com", false, "INVALID_EMAIL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateEmail(tt.email)
			if result.Success != tt.wantOK {
				t.Errorf("validateEmail(%q) Success = %v, want %v", tt.email, result.Success, tt.wantOK)
			}
			if !tt.wantOK && result.Error != nil && result.Error.Type != tt.errType {
				t.Errorf("validateEmail(%q) Error.Type = %v, want %v", tt.email, result.Error.Type, tt.errType)
			}
		})
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantOK  bool
		errType string
	}{
		{"valid Brazil mobile", "+5511987654321", true, ""},
		{"valid Brazil landline", "+551134567890", true, ""},
		{"valid US number", "+14155552671", true, ""},
		{"valid international short", "+1123456789", true, ""},
		{"valid international long", "+441onal234567890", false, "INVALID_PHONE"},
		{"missing plus", "5511987654321", false, "INVALID_PHONE"},
		{"starts with zero", "+0511987654321", false, "INVALID_PHONE"},
		{"too short", "+1", false, "INVALID_PHONE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validatePhone(tt.phone)
			if result.Success != tt.wantOK {
				t.Errorf("validatePhone(%q) Success = %v, want %v", tt.phone, result.Success, tt.wantOK)
			}
			if !tt.wantOK && result.Error != nil && result.Error.Type != tt.errType {
				t.Errorf("validatePhone(%q) Error.Type = %v, want %v", tt.phone, result.Error.Type, tt.errType)
			}
		})
	}
}

func TestValidateCPF(t *testing.T) {
	tests := []struct {
		name   string
		cpf    string
		wantOK bool
	}{
		{"valid CPF", "11144477735", true},
		{"valid CPF 2", "52998224725", true},
		{"valid CPF 3", "12345678909", true}, // Valid CPF with check digits
		{"wrong check digit", "12345678901", false},
		{"too short", "1234567890", false},
		{"too long", "123456789012", false},
		{"with letters", "1234567890a", false},
		{"with spaces", "123 456 789 09", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCPF(tt.cpf)
			if result.Success != tt.wantOK {
				t.Errorf("validateCPF(%q) Success = %v, want %v", tt.cpf, result.Success, tt.wantOK)
			}
		})
	}
}

func TestValidateCNPJ(t *testing.T) {
	tests := []struct {
		name   string
		cnpj   string
		wantOK bool
	}{
		{"valid CNPJ", "11222333000181", true},
		{"valid CNPJ 2", "45997418000153", true},
		{"all same digits", "11111111111111", false},
		{"wrong check digit", "12345678000191", false},
		{"too short", "1234567800019", false},
		{"too long", "123456780001912", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCNPJ(tt.cnpj)
			if result.Success != tt.wantOK {
				t.Errorf("validateCNPJ(%q) Success = %v, want %v", tt.cnpj, result.Success, tt.wantOK)
			}
		})
	}
}

func TestValidateEVP(t *testing.T) {
	tests := []struct {
		name   string
		evp    string
		wantOK bool
	}{
		{"valid UUID v4 lowercase", "123e4567-e89b-42d3-a456-426655440000", true},
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"uppercase should convert", "550E8400-E29B-41D4-A716-446655440000", true},
		{"invalid format", "not-a-uuid", false},
		{"missing dashes", "550e8400e29b41d4a716446655440000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateEVP(tt.evp)
			if result.Success != tt.wantOK {
				t.Errorf("validateEVP(%q) Success = %v, want %v", tt.evp, result.Success, tt.wantOK)
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		keyType models.KeyType
		wantOK  bool
	}{
		{"valid CPF", "11144477735", models.KeyTypeCPF, true},
		{"valid CNPJ", "11222333000181", models.KeyTypeCNPJ, true},
		{"valid email", "test@example.com", models.KeyTypeEMAIL, true},
		{"valid phone", "+5511987654321", models.KeyTypePHONE, true},
		{"valid EVP", "550e8400-e29b-41d4-a716-446655440000", models.KeyTypeEVP, true},
		{"invalid key type", "test", "INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateKey(tt.key, tt.keyType)
			if result.Success != tt.wantOK {
				t.Errorf("ValidateKey(%q, %q) Success = %v, want %v", tt.key, tt.keyType, result.Success, tt.wantOK)
			}
		})
	}
}

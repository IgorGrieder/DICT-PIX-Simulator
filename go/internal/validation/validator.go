package validation

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	once     sync.Once
)

// Get returns the singleton validator instance
func Get() *validator.Validate {
	once.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
	})
	return validate
}

// Validate validates a struct and returns an error if invalid
func Validate(s any) error {
	return Get().Struct(s)
}

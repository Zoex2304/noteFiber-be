package serverutils

import (
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateRequest(req any) error {
	err := validate.Struct(req)
	if err == nil {
		return nil
	}

	var details []ValidationErrorDetail
	for _, err := range err.(validator.ValidationErrors) {
		details = append(details, ValidationErrorDetail{
			Field:   err.Field(),
			Message: err.Tag(),
		})
	}

	return NewValidationError(details)
}

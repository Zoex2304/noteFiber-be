package serverutils

type ValidationErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type ValidationError struct {
	Details []ValidationErrorDetail
}

func (v *ValidationError) Error() string {
	return "validation failed"
}

func NewValidationError(details []ValidationErrorDetail) *ValidationError {
	return &ValidationError{Details: details}
}

func (v *ValidationError) ToErrorDetails() []ErrorDetail {
	result := make([]ErrorDetail, len(v.Details))
	for i, d := range v.Details {
		result[i] = ErrorDetail{
			Field:   d.Field,
			Message: d.Message,
		}
	}
	return result
}

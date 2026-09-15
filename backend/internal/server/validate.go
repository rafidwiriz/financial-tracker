package server

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func Validate(v any) *AppError {
	if err := validate.Struct(v); err != nil {
		var details []map[string]string
		for _, fe := range err.(validator.ValidationErrors) {
			details = append(details, map[string]string{
				"field":   fe.Field(),
				"message": fe.Error(),
			})
		}
		return NewAppError(400, "VALIDATION_ERROR", "Validation failed", details)
	}
	return nil
}

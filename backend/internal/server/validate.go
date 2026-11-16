package server

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func init() {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "max":
		return "must be at most " + fe.Param() + " characters"
	case "len":
		return "must be exactly " + fe.Param() + " characters"
	case "oneof":
		return "must be one of: " + fe.Param()
	case "uppercase":
		return "must be uppercase"
	case "hexcolor":
		return "must be a hex color (e.g. #2196F3)"
	case "alpha":
		return "must contain only letters"
	default:
		return "is invalid"
	}
}

func Validate(v any) *AppError {
	if err := validate.Struct(v); err != nil {
		var details []map[string]string
		for _, fe := range err.(validator.ValidationErrors) {
			details = append(details, map[string]string{
				"field":   fe.Field(),
				"message": validationMessage(fe),
			})
		}
		return NewAppError(400, "VALIDATION_ERROR", "Validation failed", details)
	}
	return nil
}

package dto

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ParseValidationError converts a go-validator error into a human-readable message.
// Requires that the validator's RegisterTagNameFunc is set to use json tags (done in router init).
func ParseValidationError(err error) string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return err.Error()
	}

	msgs := make([]string, 0, len(ve))
	for _, fe := range ve {
		msgs = append(msgs, fieldErrorMsg(fe))
	}
	return strings.Join(msgs, "; ")
}

func fieldErrorMsg(fe validator.FieldError) string {
	field := fe.Field() // json tag name thanks to RegisterTagNameFunc

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", field, fe.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", field, fe.Param())
	default:
		return fmt.Sprintf("%s is invalid (%s)", field, fe.Tag())
	}
}

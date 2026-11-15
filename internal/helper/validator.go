package helper

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
)

func FormatValidationError(err error) map[string]string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		out := make(map[string]string)
		for _, fe := range ve {
			field := strings.ToLower(fe.Field())
			switch fe.Tag() {
			case "required":
				out[field] = field + " wajib diisi"
			case "min":
				out[field] = field + " minimal " + fe.Param() + " karakter"
			case "gt":
				out[field] = field + " harus lebih besar dari " + fe.Param()
			case "email":
				out[field] = field + " harus berupa email"
			default:
				out[field] = field + " tidak valid"
			}
		}
		return out
	}
	return nil
}

package watcher

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

func msgForTag(tag string) string {
	switch tag {
	case "required":
		return "is required"
	case "address":
		return "incorrect address"
	}
	return ""
}

func Validate(data interface{}) error {
	validate := validator.New()

	_ = validate.RegisterValidation("address", func(fl validator.FieldLevel) bool {
		address := fl.Field().String()

		addressBytes := HexToBytes(address)
		if addressBytes == nil || len(addressBytes) != 20 {
			return false
		}

		return true
	})

	if err := validate.Struct(data); err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			return errors.New("invalid request body")
		}

		var out []string
		for _, err := range err.(validator.ValidationErrors) {
			out = append(out, fmt.Sprintf("%s - %s", err.Field(), msgForTag(err.Tag())))
		}

		return errors.New(strings.Join(out, ", "))
	}

	return nil
}

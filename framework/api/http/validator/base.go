package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	errors2 "github.com/king-glitch/hexag/framework/api/service/errors"
	"github.com/pkg/errors"
)

type Validatable interface {
	IsValid() bool
}

type Base struct {
	validate *validator.Validate
}

func NewStructValidator() fiber.StructValidator {
	v := validator.New()
	_ = v.RegisterValidation(
		"enum",
		func(fl validator.FieldLevel) bool {
			if val, ok := fl.Field().Interface().(Validatable); ok {
				return val.IsValid()
			}
			return false
		},
	)
	return Base{validate: v}
}

func (v Base) Validate(out any) error {
	err := v.validate.Struct(out)
	if err == nil {
		return nil
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return errors.Wrap(err, "failed to validate request")
	}

	fieldType := reflect.TypeOf(out)
	if fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}

	serviceErr := errors2.NewServiceError(
		errors2.ServiceErrorCodeValidation,
		errors.New("one or more fields failed validation"),
	)

	for _, fieldErr := range validationErrors {
		fieldName := fieldErr.Field()

		if field, ok := fieldType.FieldByName(fieldErr.Field()); ok {
			if tag, _, _ := strings.Cut(field.Tag.Get("json"), ","); tag != "" {
				fieldName = tag
			} else if tag, _, _ := strings.Cut(field.Tag.Get("form"), ","); tag != "" {
				fieldName = tag
			} else if tag, _, _ := strings.Cut(field.Tag.Get("query"), ","); tag != "" {
				fieldName = tag
			}
		}

		serviceErr = serviceErr.AddError(fieldName, validationMessage(fieldErr), nil)
	}

	return serviceErr
}

func validationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "this field is required"
	case "min":
		return fmt.Sprintf("this field is required to be at least %s characters long", fieldErr.Param())
	case "max":
		return fmt.Sprintf("this field is required to be at most %s characters long", fieldErr.Param())
	case "email":
		return "this field is required to be a valid email address"
	case "oneof":
		return fmt.Sprintf("this field must be one of: %s", fieldErr.Param())
	case "enum":
		return "this field contains an invalid value"
	default:
		return fmt.Sprintf("validation failed on %s", fieldErr.Tag())
	}
}

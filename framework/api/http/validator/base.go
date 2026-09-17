package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	errors2 "github.com/king-glitch/hexag/framework/api/service/errors"
	hexports "github.com/king-glitch/hexag/framework/ports"
	"github.com/pkg/errors"
)

type Base struct {
	validate *validator.Validate
}

type validatableFieldError struct {
	field   string
	message string
}

func NewStructValidator() fiber.StructValidator {
	v := validator.New()
	validateValidatable := func(fl validator.FieldLevel) bool {
		if val, ok := fl.Field().Interface().(hexports.Validatable); ok {
			return val.IsValid()
		}
		return false
	}
	_ = v.RegisterValidation("validatable", validateValidatable)
	_ = v.RegisterValidation("enum", validateValidatable)
	return Base{validate: v}
}

func (v Base) Validate(out any) error {
	if out == nil {
		return nil
	}

	val := reflect.ValueOf(out)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	var validationErrors validator.ValidationErrors
	if err := v.validate.Struct(out); err != nil {
		if !errors.As(err, &validationErrors) {
			return errors.Wrap(err, "failed to validate request")
		}
	}

	// Track fields already reported by validator.Validate
	reportedFields := make(map[string]bool)
	fieldType := val.Type()

	serviceErr := errors2.NewServiceError(
		errors2.ServiceErrorCodeValidation,
		errors.New("one or more fields failed validation"),
	)

	for _, fieldErr := range validationErrors {
		fieldName := fieldErr.Field()

		if field, ok := fieldType.FieldByName(fieldErr.Field()); ok {
			fieldName = getFieldName(field)
		}

		reportedFields[fieldName] = true
		serviceErr = serviceErr.AddError(fieldName, validationMessage(fieldErr), nil)
	}

	// Autodetect any fields implementing hexports.Validatable (IsValid() bool)
	var validatableErrors []validatableFieldError
	collectValidatableErrors(val, "", &validatableErrors)

	for _, ve := range validatableErrors {
		if !reportedFields[ve.field] {
			reportedFields[ve.field] = true
			serviceErr = serviceErr.AddError(ve.field, ve.message, nil)
		}
	}

	if len(reportedFields) == 0 {
		return nil
	}

	return serviceErr
}

func collectValidatableErrors(val reflect.Value, prefix string, errs *[]validatableFieldError) {
	if !val.IsValid() {
		return
	}

	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		fieldVal := val.Field(i)
		fieldName := getFieldName(sf)
		if prefix != "" {
			fieldName = prefix + "." + fieldName
		}

		// Direct check for hexports.Validatable implementation (value or pointer receiver)
		if fieldVal.CanInterface() {
			if validatable, ok := fieldVal.Interface().(hexports.Validatable); ok {
				if !fieldVal.IsZero() && !validatable.IsValid() {
					*errs = append(*errs, validatableFieldError{
						field:   fieldName,
						message: "this field contains an invalid value",
					})
					continue
				}
			} else if fieldVal.CanAddr() {
				if validatable, ok := fieldVal.Addr().Interface().(hexports.Validatable); ok {
					if !fieldVal.IsZero() && !validatable.IsValid() {
						*errs = append(*errs, validatableFieldError{
							field:   fieldName,
							message: "this field contains an invalid value",
						})
						continue
					}
				}
			}
		}

		// Pointer to validatable check
		if fieldVal.Kind() == reflect.Pointer && !fieldVal.IsNil() && fieldVal.CanInterface() {
			if validatable, ok := fieldVal.Interface().(hexports.Validatable); ok {
				if !validatable.IsValid() {
					*errs = append(*errs, validatableFieldError{
						field:   fieldName,
						message: "this field contains an invalid value",
					})
					continue
				}
			} else if fieldVal.Elem().CanInterface() {
				if validatable, ok := fieldVal.Elem().Interface().(hexports.Validatable); ok {
					if !validatable.IsValid() {
						*errs = append(*errs, validatableFieldError{
							field:   fieldName,
							message: "this field contains an invalid value",
						})
						continue
					}
				}
			}
		}

		// Slice of validatables
		if (fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array) && fieldVal.CanInterface() {
			for j := 0; j < fieldVal.Len(); j++ {
				elem := fieldVal.Index(j)
				if elem.CanInterface() {
					if validatable, ok := elem.Interface().(hexports.Validatable); ok {
						if !elem.IsZero() && !validatable.IsValid() {
							elemName := fmt.Sprintf("%s[%d]", fieldName, j)
							*errs = append(*errs, validatableFieldError{
								field:   elemName,
								message: "this field contains an invalid value",
							})
						}
					} else if elem.CanAddr() {
						if validatable, ok := elem.Addr().Interface().(hexports.Validatable); ok {
							if !elem.IsZero() && !validatable.IsValid() {
								elemName := fmt.Sprintf("%s[%d]", fieldName, j)
								*errs = append(*errs, validatableFieldError{
									field:   elemName,
									message: "this field contains an invalid value",
								})
							}
						}
					}
				}
			}
			continue
		}

		// Nested struct
		if fieldVal.Kind() == reflect.Struct {
			collectValidatableErrors(fieldVal, fieldName, errs)
		} else if fieldVal.Kind() == reflect.Pointer && !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.Struct {
			collectValidatableErrors(fieldVal.Elem(), fieldName, errs)
		}
	}
}

func getFieldName(sf reflect.StructField) string {
	if tag, _, _ := strings.Cut(sf.Tag.Get("json"), ","); tag != "" && tag != "-" {
		return tag
	}
	if tag, _, _ := strings.Cut(sf.Tag.Get("form"), ","); tag != "" && tag != "-" {
		return tag
	}
	if tag, _, _ := strings.Cut(sf.Tag.Get("query"), ","); tag != "" && tag != "-" {
		return tag
	}
	return sf.Name
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

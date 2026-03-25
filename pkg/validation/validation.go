package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

var (
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	slugRegex  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

func Init() {
	validate = validator.New()
	validate.RegisterValidation("phone", validatePhone)
	validate.RegisterValidation("slug", validateSlug)
}

func Get() *validator.Validate {
	if validate == nil {
		Init()
	}
	return validate
}

func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return true
	}
	return phoneRegex.MatchString(phone)
}

func validateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	if slug == "" {
		return true
	}
	return slugRegex.MatchString(slug)
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors []FieldError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	return ve[0].Message
}

func FormatErrors(err error) ValidationErrors {
	var errors ValidationErrors

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return errors
	}

	for _, err := range validationErrors {
		field := err.Field()
		var message string

		switch err.Tag() {
		case "required":
			message = field + " is required"
		case "email":
			message = field + " must be a valid email address"
		case "phone":
			message = field + " must be a valid phone number"
		case "min":
			message = field + " must be at least " + err.Param() + " characters"
		case "max":
			message = field + " must be at most " + err.Param() + " characters"
		case "e164":
			message = field + " must be a valid E.164 phone number"
		case "slug":
			message = field + " must be a valid slug (lowercase letters, numbers, and hyphens only)"
		case "uuid":
			message = field + " must be a valid UUID"
		case "oneof":
			message = field + " must be one of: " + err.Param()
		default:
			message = field + " is invalid"
		}

		errors = append(errors, FieldError{
			Field:   field,
			Message: message,
		})
	}

	return errors
}

func Validate(s any) ValidationErrors {
	err := Get().Struct(s)
	if err == nil {
		return nil
	}
	return FormatErrors(err)
}

func Var(field any, tag string) error {
	return validate.Var(field, tag)
}

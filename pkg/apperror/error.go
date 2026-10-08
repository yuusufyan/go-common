package apperror

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// AppError is a custom error type that contains HTTP status code and message
type AppError struct {
	Code    int
	Message string
	Errors  interface{}
}

func (e *AppError) Error() string {
	return e.Message
}

// New creates a new AppError
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Common errors
func NotFound(message string) *AppError {
	return New(http.StatusNotFound, message)
}

func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, message)
}

func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, message)
}

func Forbidden(message string) *AppError {
	return New(http.StatusForbidden, message)
}

func Conflict(message string) *AppError {
	return New(http.StatusConflict, message)
}

func InternalServer(message string) *AppError {
	if message == "" {
		message = "Internal Server Error"
	}
	return New(http.StatusInternalServerError, message)
}

// customMessages maps validator tags to messages. The first %s is the field name,
// the optional second %s is the tag param (e.g. min=3).
var customMessages = map[string]string{
	"required": "Field %s must be filled",
	"email":    "Invalid email address for field %s",
	"min":      "Field %s must have a minimum length of %s characters",
	"max":      "Field %s must have a maximum length of %s characters",
	"len":      "Field %s must be exactly %s characters long",
	"number":   "Field %s must be a number",
	"positive": "Field %s must be a positive number",
	"alphanum": "Field %s must contain only alphanumeric characters",
	"oneof":    "Invalid value for field %s",
}

// SetValidationMessage registers or overrides the message for a validator tag,
// e.g. SetValidationMessage("required", "Field %s wajib diisi").
// Call it during application startup.
func SetValidationMessage(tag, message string) {
	customMessages[tag] = message
}

// SetValidationMessages registers or overrides several tag messages at once.
func SetValidationMessages(messages map[string]string) {
	for tag, msg := range messages {
		customMessages[tag] = msg
	}
}

// TranslateValidationError translates validator.ValidationErrors to an AppError with custom messages
func TranslateValidationError(err error) *AppError {
	var ve validator.ValidationErrors
	if ok := (err != nil && (func() bool {
		var castOk bool
		ve, castOk = err.(validator.ValidationErrors)
		return castOk
	}())); ok {
		errorsMap := make(map[string]string)
		for _, e := range ve {
			fieldName := e.Field()
			tag := e.Tag()

			message := customMessages[tag]
			if message != "" {
				if strings.Count(message, "%s") >= 2 {
					errorsMap[fieldName] = fmt.Sprintf(message, fieldName, e.Param())
				} else {
					errorsMap[fieldName] = fmt.Sprintf(message, fieldName)
				}
			} else {
				errorsMap[fieldName] = fmt.Sprintf("Field validation for '%s' failed on the '%s' tag", fieldName, tag)
			}
		}
		return &AppError{
			Code:    http.StatusBadRequest,
			Message: "Validation failed",
			Errors:  errorsMap,
		}
	}
	return BadRequest(fmt.Sprintf("Validation error: %v", err))
}

// Validate helper to perform validation and translate errors in one go
func Validate(v *validator.Validate, s interface{}) error {
	if v == nil {
		return nil
	}
	if err := v.Struct(s); err != nil {
		return TranslateValidationError(err)
	}
	return nil
}

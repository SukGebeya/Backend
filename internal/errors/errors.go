package errors

import (
	"fmt"
	"net/http"
)

type Code string

const (
	CodeInternal           Code = "INTERNAL_ERROR"
	CodeNotFound           Code = "NOT_FOUND"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeForbidden          Code = "FORBIDDEN"
	CodeBadRequest         Code = "BAD_REQUEST"
	CodeConflict           Code = "CONFLICT"
	CodeValidation         Code = "VALIDATION_ERROR"
	CodeRateLimited        Code = "RATE_LIMITED"
	CodeInvalidToken       Code = "INVALID_TOKEN"
	CodeTokenExpired       Code = "TOKEN_EXPIRED"
	CodeInvalidCredentials Code = "INVALID_CREDENTIALS"
	CodeEmailExists        Code = "EMAIL_EXISTS"
	CodeNoteNotFound       Code = "NOTE_NOT_FOUND"
	CodeUserNotFound       Code = "USER_NOT_FOUND"
)

type AppError struct {
	Code       Code
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewError(code Code, message string, httpStatus int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

func IsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

func Internal(message string, err error) *AppError {
	return NewError(CodeInternal, message, http.StatusInternalServerError, err)
}

func NotFound(message string) *AppError {
	return NewError(CodeNotFound, message, http.StatusNotFound, nil)
}

func NotFoundWithErr(message string, err error) *AppError {
	return NewError(CodeNotFound, message, http.StatusNotFound, err)
}

func Unauthorized(message string) *AppError {
	return NewError(CodeUnauthorized, message, http.StatusUnauthorized, nil)
}

func UnauthorizedWithErr(message string, err error) *AppError {
	return NewError(CodeUnauthorized, message, http.StatusUnauthorized, err)
}

func Forbidden(message string) *AppError {
	return NewError(CodeForbidden, message, http.StatusForbidden, nil)
}

func BadRequest(message string) *AppError {
	return NewError(CodeBadRequest, message, http.StatusBadRequest, nil)
}

func BadRequestWithErr(message string, err error) *AppError {
	return NewError(CodeBadRequest, message, http.StatusBadRequest, err)
}

func Conflict(message string) *AppError {
	return NewError(CodeConflict, message, http.StatusConflict, nil)
}

func Validation(message string) *AppError {
	return NewError(CodeValidation, message, http.StatusBadRequest, nil)
}

func RateLimited(message string) *AppError {
	return NewError(CodeRateLimited, message, http.StatusTooManyRequests, nil)
}

func InvalidToken(message string) *AppError {
	return NewError(CodeInvalidToken, message, http.StatusUnauthorized, nil)
}

func TokenExpired(message string) *AppError {
	return NewError(CodeTokenExpired, message, http.StatusUnauthorized, nil)
}

func InvalidCredentials(message string) *AppError {
	return NewError(CodeInvalidCredentials, message, http.StatusUnauthorized, nil)
}

func EmailExists(message string) *AppError {
	return NewError(CodeEmailExists, message, http.StatusConflict, nil)
}

func NoteNotFound(message string) *AppError {
	return NewError(CodeNoteNotFound, message, http.StatusNotFound, nil)
}

func UserNotFound(message string) *AppError {
	return NewError(CodeUserNotFound, message, http.StatusNotFound, nil)
}

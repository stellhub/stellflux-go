package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
)

type Code string

const (
	CodeOK              Code = "OK"
	CodeInvalidArgument Code = "INVALID_ARGUMENT"
	CodeInvalidConfig   Code = "INVALID_CONFIG"
	CodeNotFound        Code = "NOT_FOUND"
	CodeConflict        Code = "CONFLICT"
	CodeUnavailable     Code = "UNAVAILABLE"
	CodeInternal        Code = "INTERNAL"
)

type Error struct {
	Code       Code   `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	cause      error
}

var ErrAppNameRequired = New(CodeInvalidConfig, "stellar: app name is required", http.StatusBadRequest)

func New(code Code, message string, httpStatus int) *Error {
	if httpStatus == 0 {
		httpStatus = http.StatusInternalServerError
	}
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func Wrap(code Code, message string, cause error, httpStatus int) *Error {
	err := New(code, message, httpStatus)
	err.cause = cause
	return err
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.cause == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func HTTPStatusOf(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var appErr *Error
	if stderrors.As(err, &appErr) && appErr.HTTPStatus != 0 {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}

func CodeOf(err error) Code {
	if err == nil {
		return CodeOK
	}
	var appErr *Error
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeInternal
}

func MessageOf(err error) string {
	if err == nil {
		return ""
	}
	var appErr *Error
	if stderrors.As(err, &appErr) {
		return appErr.Message
	}
	return err.Error()
}

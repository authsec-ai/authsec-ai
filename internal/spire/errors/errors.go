package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewUnauthorizedError(message string, err error) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: message, Status: http.StatusUnauthorized, Err: err}
}

func NewForbiddenError(message string, err error) *AppError {
	return &AppError{Code: "FORBIDDEN", Message: message, Status: http.StatusForbidden, Err: err}
}

func NewNotFoundError(message string, err error) *AppError {
	return &AppError{Code: "NOT_FOUND", Message: message, Status: http.StatusNotFound, Err: err}
}

func NewBadRequestError(message string, err error) *AppError {
	return &AppError{Code: "BAD_REQUEST", Message: message, Status: http.StatusBadRequest, Err: err}
}

func NewInternalError(message string, err error) *AppError {
	return &AppError{Code: "INTERNAL_ERROR", Message: message, Status: http.StatusInternalServerError, Err: err}
}

func NewConflictError(message string, err error) *AppError {
	return &AppError{Code: "CONFLICT", Message: message, Status: http.StatusConflict, Err: err}
}

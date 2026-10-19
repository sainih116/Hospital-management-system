// Package apperror defines the application's error taxonomy so that every
// layer (service, repository, handler) can produce errors that map
// predictably onto HTTP responses.
package apperror

import (
	"errors"
	"net/http"
)

// AppError is a structured, API-facing error.
type AppError struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	HTTPStatus int         `json:"-"`
	Details    interface{} `json:"details,omitempty"`
	cause      error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

// Unwrap enables errors.Is / errors.As against the wrapped cause.
func (e *AppError) Unwrap() error {
	return e.cause
}

// WithCause attaches an underlying error for logging/debugging purposes
// without exposing it to API clients.
func (e *AppError) WithCause(err error) *AppError {
	clone := *e
	clone.cause = err
	return &clone
}

// WithDetails attaches structured validation/debug details to the error.
func (e *AppError) WithDetails(details interface{}) *AppError {
	clone := *e
	clone.Details = details
	return &clone
}

func newError(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status}
}

// Predefined, reusable application errors covering the common HTTP cases
// used across every module of the HMS.
var (
	ErrBadRequest      = newError("BAD_REQUEST", "The request could not be processed.", http.StatusBadRequest)
	ErrValidation      = newError("VALIDATION_ERROR", "The request failed validation.", http.StatusUnprocessableEntity)
	ErrUnauthorized    = newError("UNAUTHORIZED", "Authentication is required.", http.StatusUnauthorized)
	ErrForbidden       = newError("FORBIDDEN", "You do not have permission to perform this action.", http.StatusForbidden)
	ErrNotFound        = newError("NOT_FOUND", "The requested resource was not found.", http.StatusNotFound)
	ErrConflict        = newError("CONFLICT", "The request conflicts with the current state of the resource.", http.StatusConflict)
	ErrTooManyRequests = newError("RATE_LIMITED", "Too many requests. Please try again later.", http.StatusTooManyRequests)
	ErrInternal        = newError("INTERNAL_ERROR", "An unexpected error occurred.", http.StatusInternalServerError)
	ErrServiceDown     = newError("SERVICE_UNAVAILABLE", "A dependent service is unavailable.", http.StatusServiceUnavailable)
)

// New creates a new AppError with a custom code, message and HTTP status.
func New(code, message string, status int) *AppError {
	return newError(code, message, status)
}

// From normalizes any error into an *AppError, defaulting to ErrInternal
// when the error does not already carry structured information.
func From(err error) *AppError {
	if err == nil {
		return newError("OK", "", http.StatusOK)
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return ErrInternal.WithCause(err)
}

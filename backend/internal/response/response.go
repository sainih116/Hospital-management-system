// Package response provides the standard JSON envelope used by every API
// endpoint, keeping success and error payload shapes consistent.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
)

// Meta carries request-scoped metadata returned alongside successful
// responses.
type Meta struct {
	RequestID string `json:"requestId,omitempty"`
}

// Envelope is the top-level shape of every JSON response.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
}

// ErrorBody describes a failed request.
type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Success writes a 2xx JSON envelope carrying data.
func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, Envelope{
		Success: true,
		Data:    data,
		Meta:    &Meta{RequestID: requestID(c)},
	})
}

// OK writes a 200 success envelope.
func OK(c *gin.Context, data interface{}) {
	Success(c, http.StatusOK, data)
}

// Created writes a 201 success envelope.
func Created(c *gin.Context, data interface{}) {
	Success(c, http.StatusCreated, data)
}

// Error writes an error envelope. If err is an *apperror.AppError its code
// and HTTP status are used; otherwise it falls back to a generic 500.
func Error(c *gin.Context, err error) {
	appErr := apperror.From(err)

	c.JSON(appErr.HTTPStatus, Envelope{
		Success: false,
		Error: &ErrorBody{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		},
		Meta: &Meta{RequestID: requestID(c)},
	})
}

func requestID(c *gin.Context) string {
	if v, ok := c.Get("requestId"); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

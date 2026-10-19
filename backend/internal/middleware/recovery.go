package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/response"
)

// Recovery converts panics into a structured 500 JSON error response and
// logs the panic (with stack trace) instead of letting Gin crash the
// connection or leak a bare stack trace to the client.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					zap.String("requestId", requestIDFrom(c)),
					zap.Any("panic", r),
					zap.Stack("stack"),
				)

				err := apperror.ErrInternal.WithCause(fmt.Errorf("panic: %v", r))
				response.Error(c, err)
				c.Abort()
			}
		}()
		c.Next()
	}
}

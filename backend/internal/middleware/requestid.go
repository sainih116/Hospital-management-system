package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDHeader is the HTTP header used to propagate/return request IDs.
const RequestIDHeader = "X-Request-Id"

// RequestID assigns a unique ID to every incoming request, reusing an
// inbound X-Request-Id header when the caller already supplied one so that
// requests can be traced across services.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}

		c.Set("requestId", id)
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}

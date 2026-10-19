package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger emits one structured log line per request, tagged with the
// request ID so logs can be correlated with client-reported issues.
// Sensitive data (auth headers, bodies) is intentionally never logged.
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		fields := []zap.Field{
			zap.String("requestId", requestIDFrom(c)),
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("clientIp", c.ClientIP()),
			zap.Duration("latency", time.Since(start)),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		switch {
		case c.Writer.Status() >= 500:
			log.Error("request", fields...)
		case c.Writer.Status() >= 400:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}

func requestIDFrom(c *gin.Context) string {
	if v, ok := c.Get("requestId"); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

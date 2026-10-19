package middleware

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/response"
)

// RequirePermission authorizes a request only if the authenticated user's
// token carries at least one of the given permission codes. Must run after
// Authenticate.
func RequirePermission(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c)
		if !ok {
			response.Error(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		if !hasAny(claims.Permissions, codes) {
			response.Error(c, apperror.ErrForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole authorizes a request only if the authenticated user's role is
// one of the given role codes. Must run after Authenticate.
func RequireRole(roleCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c)
		if !ok {
			response.Error(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		for _, role := range roleCodes {
			if claims.Role == role {
				c.Next()
				return
			}
		}

		response.Error(c, apperror.ErrForbidden)
		c.Abort()
	}
}

func hasAny(have []string, want []string) bool {
	set := make(map[string]struct{}, len(have))
	for _, p := range have {
		set[p] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; ok {
			return true
		}
	}
	return false
}

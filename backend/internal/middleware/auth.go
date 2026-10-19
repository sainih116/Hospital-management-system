package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/response"
	"hms-backend/internal/security"
)

const claimsContextKey = "authClaims"

// ActiveUserChecker reports whether a user id still belongs to an active
// account. It is a minimal interface (rather than a concrete repository
// type) so this middleware has zero compile-time dependency on any
// module's persistence layer.
type ActiveUserChecker func(ctx context.Context, userID string) (bool, error)

// Authenticate validates the bearer access token on every request, and
// re-checks the account is still active so a deactivated user is locked
// out immediately rather than only once their token expires.
func Authenticate(tokens *security.TokenService, isActive ActiveUserChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := bearerToken(c)
		if !ok {
			response.Error(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, err := tokens.ParseAccessToken(tokenString)
		if err != nil {
			response.Error(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		active, err := isActive(c.Request.Context(), claims.UserID)
		if err != nil || !active {
			response.Error(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set(claimsContextKey, claims)
		c.Next()
	}
}

// ClaimsFromContext retrieves the authenticated request's JWT claims, set
// by Authenticate. Handlers use it to identify "the current user".
func ClaimsFromContext(c *gin.Context) (*security.Claims, bool) {
	v, ok := c.Get(claimsContextKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*security.Claims)
	return claims, ok
}

func bearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"hms-backend/internal/config"
	"hms-backend/internal/security"
)

func testTokenService() *security.TokenService {
	return security.NewTokenService(config.JWTConfig{
		AccessSecret:   "test-secret",
		AccessTokenTTL: 15 * time.Minute,
		Issuer:         "hms-backend-test",
	})
}

func alwaysActive(context.Context, string) (bool, error)   { return true, nil }
func alwaysInactive(context.Context, string) (bool, error) { return false, nil }

func requestWithAuthHeader(header string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	if header != "" {
		c.Request.Header.Set("Authorization", header)
	}
	return c, rec
}

func TestAuthenticateRejectsMissingHeader(t *testing.T) {
	c, rec := requestWithAuthHeader("")

	Authenticate(testTokenService(), alwaysActive)(c)

	if !c.IsAborted() || rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a missing Authorization header, got aborted=%v status=%d", c.IsAborted(), rec.Code)
	}
}

func TestAuthenticateRejectsMalformedHeader(t *testing.T) {
	c, rec := requestWithAuthHeader("Token abc.def.ghi")

	Authenticate(testTokenService(), alwaysActive)(c)

	if !c.IsAborted() || rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a non-Bearer scheme, got aborted=%v status=%d", c.IsAborted(), rec.Code)
	}
}

func TestAuthenticateRejectsInvalidToken(t *testing.T) {
	c, rec := requestWithAuthHeader("Bearer not-a-real-jwt")

	Authenticate(testTokenService(), alwaysActive)(c)

	if !c.IsAborted() || rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an invalid token, got aborted=%v status=%d", c.IsAborted(), rec.Code)
	}
}

func TestAuthenticateRejectsInactiveAccount(t *testing.T) {
	tokens := testTokenService()
	token, _, err := tokens.GenerateAccessToken("user-1", "a@b.com", "ADMIN", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	c, rec := requestWithAuthHeader("Bearer " + token)

	Authenticate(tokens, alwaysInactive)(c)

	if !c.IsAborted() || rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a deactivated account, got aborted=%v status=%d", c.IsAborted(), rec.Code)
	}
}

func TestAuthenticateAcceptsValidTokenAndSetsClaims(t *testing.T) {
	tokens := testTokenService()
	token, _, err := tokens.GenerateAccessToken("user-1", "a@b.com", "ADMIN", []string{"user.manage"})
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	c, rec := requestWithAuthHeader("Bearer " + token)

	Authenticate(tokens, alwaysActive)(c)

	if c.IsAborted() {
		t.Fatalf("expected the request not to be aborted, status=%d", rec.Code)
	}

	claims, ok := ClaimsFromContext(c)
	if !ok {
		t.Fatalf("expected claims to be set in context")
	}
	if claims.UserID != "user-1" {
		t.Fatalf("expected UserID user-1, got %s", claims.UserID)
	}
}

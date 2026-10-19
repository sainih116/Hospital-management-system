package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"hms-backend/internal/security"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestContextWithClaims(claims *security.Claims) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if claims != nil {
		c.Set(claimsContextKey, claims)
	}
	return c, rec
}

func TestRequirePermissionAllowsWhenClaimHasPermission(t *testing.T) {
	c, rec := newTestContextWithClaims(&security.Claims{UserID: "u1", Role: "DOCTOR", Permissions: []string{"patient.read", "emr.write"}})

	RequirePermission("emr.write")(c)

	if c.IsAborted() {
		t.Fatalf("expected the request not to be aborted, status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRequirePermissionDeniesWhenClaimLacksPermission(t *testing.T) {
	c, rec := newTestContextWithClaims(&security.Claims{UserID: "u1", Role: "NURSE", Permissions: []string{"patient.read"}})

	RequirePermission("billing.manage")(c)

	if !c.IsAborted() {
		t.Fatalf("expected the request to be aborted")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}

func TestRequirePermissionRejectsUnauthenticatedRequest(t *testing.T) {
	c, rec := newTestContextWithClaims(nil)

	RequirePermission("patient.read")(c)

	if !c.IsAborted() {
		t.Fatalf("expected the request to be aborted")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireRoleAllowsMatchingRole(t *testing.T) {
	c, rec := newTestContextWithClaims(&security.Claims{UserID: "u1", Role: "ADMIN"})

	RequireRole("ADMIN", "DOCTOR")(c)

	if c.IsAborted() {
		t.Fatalf("expected the request not to be aborted, status=%d", rec.Code)
	}
}

func TestRequireRoleDeniesNonMatchingRole(t *testing.T) {
	c, rec := newTestContextWithClaims(&security.Claims{UserID: "u1", Role: "PHARMACIST"})

	RequireRole("ADMIN", "DOCTOR")(c)

	if !c.IsAborted() {
		t.Fatalf("expected the request to be aborted")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}

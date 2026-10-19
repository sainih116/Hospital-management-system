package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/config"
	"hms-backend/internal/modules/auth/domain"
	"hms-backend/internal/modules/auth/dto"
	"hms-backend/internal/security"
)

type harness struct {
	svc           AuthService
	users         *fakeUserRepository
	roles         *fakeRoleRepository
	refreshTokens *fakeRefreshTokenRepository
	resetTokens   *fakePasswordResetRepository
	notifier      *fakeNotifier
}

func newHarness() *harness {
	users := newFakeUserRepository()
	roles := newFakeRoleRepository()
	refreshTokens := newFakeRefreshTokenRepository()
	resetTokens := newFakePasswordResetRepository()
	notif := &fakeNotifier{}

	tokens := security.NewTokenService(config.JWTConfig{
		AccessSecret:   "test-secret",
		AccessTokenTTL: 15 * time.Minute,
		Issuer:         "hms-backend-test",
	})

	svc := NewAuthService(Deps{
		Users:         users,
		Roles:         roles,
		RefreshTokens: refreshTokens,
		ResetTokens:   resetTokens,
		Tokens:        tokens,
		Notifier:      notif,
		Logger:        zap.NewNop(),
		FrontendURL:   "http://localhost:5173",
	})

	return &harness{svc: svc, users: users, roles: roles, refreshTokens: refreshTokens, resetTokens: resetTokens, notifier: notif}
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected an *apperror.AppError, got %T: %v", err, err)
	}
	return appErr.Code
}

func TestRegisterCreatesActiveUserWithHashedPassword(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	profile, err := h.svc.Register(ctx, dto.RegisterRequest{
		Name:     "Dr. Anil Sharma",
		Email:    "anil.sharma@hms.local",
		Password: "StrongPass1!",
		RoleCode: domain.RoleDoctor,
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if profile.Role != domain.RoleDoctor || profile.Status != string(domain.UserStatusActive) {
		t.Fatalf("unexpected profile: %+v", profile)
	}

	stored, err := h.users.FindByEmail(ctx, "anil.sharma@hms.local")
	if err != nil {
		t.Fatalf("expected stored user, got error: %v", err)
	}
	if stored.PasswordHash == "StrongPass1!" {
		t.Fatalf("password must never be stored in plaintext")
	}
	if !security.VerifyPassword(stored.PasswordHash, "StrongPass1!") {
		t.Fatalf("stored hash does not verify against the original password")
	}
}

func TestRegisterRejectsUnknownRole(t *testing.T) {
	h := newHarness()

	_, err := h.svc.Register(context.Background(), dto.RegisterRequest{
		Name:     "Someone",
		Email:    "someone@hms.local",
		Password: "StrongPass1!",
		RoleCode: "SUPERUSER",
	})
	if err == nil {
		t.Fatalf("expected an error for an unknown role code")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	h := newHarness()
	ctx := context.Background()
	req := dto.RegisterRequest{Name: "A", Email: "dupe@hms.local", Password: "StrongPass1!", RoleCode: domain.RoleNurse}

	if _, err := h.svc.Register(ctx, req); err != nil {
		t.Fatalf("first Register returned error: %v", err)
	}

	_, err := h.svc.Register(ctx, req)
	if err == nil {
		t.Fatalf("expected an error for a duplicate email")
	}
	if code := appErrCode(t, err); code != "EMAIL_TAKEN" {
		t.Fatalf("expected EMAIL_TAKEN, got %s", code)
	}
}

func TestLoginSucceedsWithCorrectCredentials(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	_, err := h.svc.Register(ctx, dto.RegisterRequest{
		Name: "Priya", Email: "priya@hms.local", Password: "NursePass1!", RoleCode: domain.RoleNurse,
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	result, err := h.svc.Login(ctx, dto.LoginRequest{Email: "priya@hms.local", Password: "NursePass1!"}, RequestMeta{IP: "127.0.0.1"})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("expected non-empty access and refresh tokens")
	}
	if len(result.User.Permissions) == 0 {
		t.Fatalf("expected the login response to include the role's permissions")
	}
	if result.User.Role != domain.RoleNurse {
		t.Fatalf("expected role NURSE, got %s", result.User.Role)
	}
}

func TestLoginRejectsWrongPasswordWithGenericError(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	_, _ = h.svc.Register(ctx, dto.RegisterRequest{Name: "A", Email: "wrongpass@hms.local", Password: "CorrectPass1!", RoleCode: domain.RoleReceptionist})

	_, err := h.svc.Login(ctx, dto.LoginRequest{Email: "wrongpass@hms.local", Password: "wrong"}, RequestMeta{})
	if err == nil {
		t.Fatalf("expected an error for the wrong password")
	}
	if code := appErrCode(t, err); code != "INVALID_CREDENTIALS" {
		t.Fatalf("expected INVALID_CREDENTIALS, got %s", code)
	}
}

func TestLoginRejectsUnknownEmailWithSameGenericError(t *testing.T) {
	h := newHarness()

	_, err := h.svc.Login(context.Background(), dto.LoginRequest{Email: "nobody@hms.local", Password: "whatever"}, RequestMeta{})
	if err == nil {
		t.Fatalf("expected an error for an unknown email")
	}
	// Same error code as a wrong password: login must not reveal which
	// field was incorrect (prevents user enumeration).
	if code := appErrCode(t, err); code != "INVALID_CREDENTIALS" {
		t.Fatalf("expected INVALID_CREDENTIALS, got %s", code)
	}
}

func TestLoginRejectsInactiveAccount(t *testing.T) {
	h := newHarness()
	passwordHash, _ := security.HashPassword("InactivePass1!")
	h.users.seed(&domain.User{
		Name: "Inactive User", Email: "inactive@hms.local", PasswordHash: passwordHash,
		RoleCode: domain.RoleReceptionist, Status: domain.UserStatusInactive,
	})

	_, err := h.svc.Login(context.Background(), dto.LoginRequest{Email: "inactive@hms.local", Password: "InactivePass1!"}, RequestMeta{})
	if err == nil {
		t.Fatalf("expected an error for an inactive account")
	}
	if code := appErrCode(t, err); code != "ACCOUNT_INACTIVE" {
		t.Fatalf("expected ACCOUNT_INACTIVE, got %s", code)
	}
}

func TestRefreshTokenRotatesAndInvalidatesThePreviousToken(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	_, _ = h.svc.Register(ctx, dto.RegisterRequest{Name: "A", Email: "rotate@hms.local", Password: "RotatePass1!", RoleCode: domain.RoleAdmin})
	login, err := h.svc.Login(ctx, dto.LoginRequest{Email: "rotate@hms.local", Password: "RotatePass1!"}, RequestMeta{})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	refreshed, err := h.svc.RefreshToken(ctx, login.RefreshToken, RequestMeta{})
	if err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}
	if refreshed.RefreshToken == login.RefreshToken {
		t.Fatalf("expected a rotated (different) refresh token")
	}

	// The old refresh token must no longer work.
	if _, err := h.svc.RefreshToken(ctx, login.RefreshToken, RequestMeta{}); err == nil {
		t.Fatalf("expected the rotated-out refresh token to be rejected")
	}
}

func TestRefreshTokenReuseRevokesEverySessionForTheUser(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	_, _ = h.svc.Register(ctx, dto.RegisterRequest{Name: "A", Email: "reuse@hms.local", Password: "ReusePass1!", RoleCode: domain.RoleAdmin})
	login, err := h.svc.Login(ctx, dto.LoginRequest{Email: "reuse@hms.local", Password: "ReusePass1!"}, RequestMeta{})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	// Rotate once (this revokes the original token).
	second, err := h.svc.RefreshToken(ctx, login.RefreshToken, RequestMeta{})
	if err != nil {
		t.Fatalf("first RefreshToken returned error: %v", err)
	}

	// Replay the already-revoked original token — this must be treated as
	// a compromise signal and kill every session for the user, including
	// the legitimate second one.
	if _, err := h.svc.RefreshToken(ctx, login.RefreshToken, RequestMeta{}); err == nil {
		t.Fatalf("expected reuse of a revoked token to be rejected")
	}

	if _, err := h.svc.RefreshToken(ctx, second.RefreshToken, RequestMeta{}); err == nil {
		t.Fatalf("expected the legitimate rotated token to be revoked too after reuse was detected")
	}
}

func TestLogoutIsIdempotentAndRevokesTheToken(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	_, _ = h.svc.Register(ctx, dto.RegisterRequest{Name: "A", Email: "logout@hms.local", Password: "LogoutPass1!", RoleCode: domain.RoleAdmin})
	login, err := h.svc.Login(ctx, dto.LoginRequest{Email: "logout@hms.local", Password: "LogoutPass1!"}, RequestMeta{})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if err := h.svc.Logout(ctx, login.RefreshToken); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
	// Calling logout again with the same (already revoked) token must not error.
	if err := h.svc.Logout(ctx, login.RefreshToken); err != nil {
		t.Fatalf("second Logout call returned error: %v", err)
	}

	if _, err := h.svc.RefreshToken(ctx, login.RefreshToken, RequestMeta{}); err == nil {
		t.Fatalf("expected the logged-out refresh token to be rejected")
	}
}

func TestForgotPasswordDoesNotRevealWhetherEmailExists(t *testing.T) {
	h := newHarness()

	if err := h.svc.ForgotPassword(context.Background(), "nobody@hms.local"); err != nil {
		t.Fatalf("ForgotPassword returned error: %v", err)
	}
	if len(h.notifier.resetEmails) != 0 {
		t.Fatalf("expected no notification for an unregistered email")
	}
}

func TestForgotPasswordAndResetPasswordFullFlow(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	_, _ = h.svc.Register(ctx, dto.RegisterRequest{Name: "A", Email: "reset@hms.local", Password: "OldPass123!", RoleCode: domain.RoleAdmin})
	login, err := h.svc.Login(ctx, dto.LoginRequest{Email: "reset@hms.local", Password: "OldPass123!"}, RequestMeta{})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if err := h.svc.ForgotPassword(ctx, "reset@hms.local"); err != nil {
		t.Fatalf("ForgotPassword returned error: %v", err)
	}

	link := h.notifier.lastResetLink()
	token := link[strings.Index(link, "token=")+len("token="):]
	if token == "" {
		t.Fatalf("expected a reset token embedded in the notification link")
	}

	if err := h.svc.ResetPassword(ctx, token, "NewPass456!"); err != nil {
		t.Fatalf("ResetPassword returned error: %v", err)
	}

	// Old password no longer works, new one does.
	if _, err := h.svc.Login(ctx, dto.LoginRequest{Email: "reset@hms.local", Password: "OldPass123!"}, RequestMeta{}); err == nil {
		t.Fatalf("expected old password to be rejected after reset")
	}
	if _, err := h.svc.Login(ctx, dto.LoginRequest{Email: "reset@hms.local", Password: "NewPass456!"}, RequestMeta{}); err != nil {
		t.Fatalf("expected new password to work after reset: %v", err)
	}

	// The session that existed before the reset must be revoked too.
	if _, err := h.svc.RefreshToken(ctx, login.RefreshToken, RequestMeta{}); err == nil {
		t.Fatalf("expected pre-reset sessions to be revoked")
	}
}

func TestResetPasswordRejectsUnknownToken(t *testing.T) {
	h := newHarness()

	err := h.svc.ResetPassword(context.Background(), "not-a-real-token", "NewPass456!")
	if err == nil {
		t.Fatalf("expected an error for an unknown reset token")
	}
	if code := appErrCode(t, err); code != "INVALID_RESET_TOKEN" {
		t.Fatalf("expected INVALID_RESET_TOKEN, got %s", code)
	}
}

func TestResetPasswordRejectsExpiredToken(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	_, _ = h.svc.Register(ctx, dto.RegisterRequest{Name: "A", Email: "expired@hms.local", Password: "OldPass123!", RoleCode: domain.RoleAdmin})
	if err := h.svc.ForgotPassword(ctx, "expired@hms.local"); err != nil {
		t.Fatalf("ForgotPassword returned error: %v", err)
	}

	link := h.notifier.lastResetLink()
	token := link[strings.Index(link, "token=")+len("token="):]

	h.resetTokens.expireAll()

	err := h.svc.ResetPassword(ctx, token, "NewPass456!")
	if err == nil {
		t.Fatalf("expected an error for an expired reset token")
	}
	if code := appErrCode(t, err); code != "INVALID_RESET_TOKEN" {
		t.Fatalf("expected INVALID_RESET_TOKEN, got %s", code)
	}
}

func TestMeReturnsCurrentUserProfile(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	profile, err := h.svc.Register(ctx, dto.RegisterRequest{Name: "Me User", Email: "me@hms.local", Password: "MePass123!", RoleCode: domain.RoleLabTech})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	fetched, err := h.svc.Me(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Me returned error: %v", err)
	}
	if len(fetched.Permissions) == 0 {
		t.Fatalf("expected Me to include the role's permissions")
	}
	if fetched.Email != "me@hms.local" {
		t.Fatalf("expected email me@hms.local, got %s", fetched.Email)
	}
}

func TestMeRejectsUnknownUserID(t *testing.T) {
	h := newHarness()

	_, err := h.svc.Me(context.Background(), "000000000000000000000000")
	if err == nil {
		t.Fatalf("expected an error for an unknown user id")
	}
	if code := appErrCode(t, err); code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %s", code)
	}
}

func TestListUsersFiltersByRole(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	if _, err := h.svc.Register(ctx, dto.RegisterRequest{Name: "Doc A", Email: "doca@hms.local", Password: "Password1!", RoleCode: domain.RoleDoctor}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if _, err := h.svc.Register(ctx, dto.RegisterRequest{Name: "Nurse A", Email: "nursea@hms.local", Password: "Password1!", RoleCode: domain.RoleNurse}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	result, err := h.svc.List(ctx, dto.ListUsersQuery{Role: domain.RoleDoctor})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected 1 DOCTOR-role user, got %d", result.Total)
	}
	if result.Items[0].Email != "doca@hms.local" {
		t.Fatalf("expected doca@hms.local, got %s", result.Items[0].Email)
	}
}

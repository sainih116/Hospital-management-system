// Package service implements the auth module's business logic: it is the
// only layer allowed to coordinate repositories, security primitives and
// outbound notifications for authentication and RBAC concerns.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/modules/auth/domain"
	"hms-backend/internal/modules/auth/dto"
	"hms-backend/internal/modules/auth/repository"
	"hms-backend/internal/notifier"
	"hms-backend/internal/pkg/pagination"
	"hms-backend/internal/security"
)

const (
	refreshTokenTTL     = 7 * 24 * time.Hour
	passwordResetTTL    = 30 * time.Minute
	passwordResetPath   = "/reset-password"
	passwordResetQParam = "token"
)

// RequestMeta captures request-scoped context worth attaching to a session
// for audit/security purposes.
type RequestMeta struct {
	UserAgent string
	IP        string
}

// AuthService is the auth module's public business-logic contract.
type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserProfile, error)
	Login(ctx context.Context, req dto.LoginRequest, meta RequestMeta) (*dto.LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string, meta RequestMeta) (*dto.RefreshResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	Me(ctx context.Context, userID string) (*dto.UserProfile, error)
	List(ctx context.Context, query dto.ListUsersQuery) (pagination.Result[dto.UserSummary], error)
}

type authService struct {
	users         repository.UserRepository
	roles         repository.RoleRepository
	refreshTokens repository.RefreshTokenRepository
	resetTokens   repository.PasswordResetRepository
	tokens        *security.TokenService
	notifier      notifier.Notifier
	log           *zap.Logger
	frontendURL   string
}

// Deps bundles every collaborator authService needs.
type Deps struct {
	Users         repository.UserRepository
	Roles         repository.RoleRepository
	RefreshTokens repository.RefreshTokenRepository
	ResetTokens   repository.PasswordResetRepository
	Tokens        *security.TokenService
	Notifier      notifier.Notifier
	Logger        *zap.Logger
	FrontendURL   string
}

// NewAuthService builds the auth module's service implementation.
func NewAuthService(d Deps) AuthService {
	return &authService{
		users:         d.Users,
		roles:         d.Roles,
		refreshTokens: d.RefreshTokens,
		resetTokens:   d.ResetTokens,
		tokens:        d.Tokens,
		notifier:      d.Notifier,
		log:           d.Logger,
		frontendURL:   d.FrontendURL,
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserProfile, error) {
	if !domain.IsValidRoleCode(req.RoleCode) {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"roleCode": "unknown role"})
	}

	if _, err := s.users.FindByEmail(ctx, req.Email); err == nil {
		return nil, apperror.New("EMAIL_TAKEN", "An account with this email already exists.", 409)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	passwordHash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	user := &domain.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: passwordHash,
		RoleCode:     req.RoleCode,
		Status:       domain.UserStatusActive,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.log.Info("user registered", zap.String("userId", user.ID.Hex()), zap.String("role", user.RoleCode))

	role, err := s.roles.FindByCode(ctx, user.RoleCode)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(fmt.Errorf("load role for user %s: %w", user.ID.Hex(), err))
	}

	profile := toUserProfile(user, role.Permissions)
	return &profile, nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest, meta RequestMeta) (*dto.LoginResponse, error) {
	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, invalidCredentialsError()
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	if !security.VerifyPassword(user.PasswordHash, req.Password) {
		return nil, invalidCredentialsError()
	}

	if !user.IsActive() {
		return nil, apperror.New("ACCOUNT_INACTIVE", "This account is not active. Contact an administrator.", 403)
	}

	role, err := s.roles.FindByCode(ctx, user.RoleCode)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(fmt.Errorf("load role for user %s: %w", user.ID.Hex(), err))
	}

	accessToken, expiresAt, err := s.tokens.GenerateAccessToken(user.ID.Hex(), user.Email, user.RoleCode, role.Permissions)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	rawRefreshToken, err := s.issueRefreshToken(ctx, user.ID, meta)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	now := time.Now().UTC()
	if err := s.users.UpdateLastLoginAt(ctx, user.ID, now); err != nil {
		s.log.Warn("failed to update last login timestamp", zap.String("userId", user.ID.Hex()), zap.Error(err))
	}

	profile := toUserProfile(user, role.Permissions)
	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresAt:    expiresAt,
		User:         profile,
	}, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string, meta RequestMeta) (*dto.RefreshResponse, error) {
	hash := security.HashOpaqueToken(refreshToken)

	stored, err := s.refreshTokens.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrUnauthorized
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	now := time.Now().UTC()

	if stored.RevokedAt != nil {
		// A revoked refresh token was presented again — either a replay
		// attack or a stale browser tab. Treat it as a compromise signal
		// and kill every session for this user.
		s.log.Warn("revoked refresh token reused; revoking all sessions", zap.String("userId", stored.UserID.Hex()))
		if err := s.refreshTokens.RevokeAllForUser(ctx, stored.UserID); err != nil {
			s.log.Error("failed to revoke sessions after refresh token reuse", zap.Error(err))
		}
		return nil, apperror.ErrUnauthorized
	}

	if !stored.IsActive(now) {
		return nil, apperror.ErrUnauthorized
	}

	user, err := s.users.FindByID(ctx, stored.UserID.Hex())
	if err != nil {
		return nil, apperror.ErrUnauthorized
	}
	if !user.IsActive() {
		return nil, apperror.New("ACCOUNT_INACTIVE", "This account is not active. Contact an administrator.", 403)
	}

	role, err := s.roles.FindByCode(ctx, user.RoleCode)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	newRawToken, err := s.issueRefreshToken(ctx, user.ID, meta)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	if err := s.refreshTokens.Revoke(ctx, stored.ID, security.HashOpaqueToken(newRawToken)); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	accessToken, expiresAt, err := s.tokens.GenerateAccessToken(user.ID.Hex(), user.Email, user.RoleCode, role.Permissions)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	return &dto.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRawToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	hash := security.HashOpaqueToken(refreshToken)

	stored, err := s.refreshTokens.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Already gone; logout is idempotent from the caller's view.
			return nil
		}
		return apperror.ErrInternal.WithCause(err)
	}

	if stored.RevokedAt != nil {
		return nil
	}

	if err := s.refreshTokens.Revoke(ctx, stored.ID, ""); err != nil {
		return apperror.ErrInternal.WithCause(err)
	}
	return nil
}

func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Do not reveal whether the email is registered.
			return nil
		}
		return apperror.ErrInternal.WithCause(err)
	}

	rawToken, err := security.GenerateOpaqueToken()
	if err != nil {
		return apperror.ErrInternal.WithCause(err)
	}

	resetToken := &domain.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: security.HashOpaqueToken(rawToken),
		ExpiresAt: time.Now().UTC().Add(passwordResetTTL),
	}

	if err := s.resetTokens.Create(ctx, resetToken); err != nil {
		return apperror.ErrInternal.WithCause(err)
	}

	resetLink := fmt.Sprintf("%s%s?%s=%s", s.frontendURL, passwordResetPath, passwordResetQParam, rawToken)

	if err := s.notifier.SendPasswordResetEmail(ctx, user.Email, user.Name, resetLink); err != nil {
		s.log.Error("failed to send password reset notification", zap.String("userId", user.ID.Hex()), zap.Error(err))
	}

	return nil
}

func (s *authService) ResetPassword(ctx context.Context, token, newPassword string) error {
	hash := security.HashOpaqueToken(token)

	stored, err := s.resetTokens.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New("INVALID_RESET_TOKEN", "This password reset link is invalid or has expired.", 400)
		}
		return apperror.ErrInternal.WithCause(err)
	}

	if !stored.IsValid(time.Now().UTC()) {
		return apperror.New("INVALID_RESET_TOKEN", "This password reset link is invalid or has expired.", 400)
	}

	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return apperror.ErrInternal.WithCause(err)
	}

	if err := s.users.UpdatePasswordHash(ctx, stored.UserID, passwordHash); err != nil {
		return apperror.ErrInternal.WithCause(err)
	}

	if err := s.resetTokens.MarkUsed(ctx, stored.ID); err != nil {
		s.log.Warn("failed to mark password reset token used", zap.Error(err))
	}

	// Force re-authentication everywhere: a leaked/compromised password
	// means every existing session should be treated as suspect too.
	if err := s.refreshTokens.RevokeAllForUser(ctx, stored.UserID); err != nil {
		s.log.Warn("failed to revoke sessions after password reset", zap.Error(err))
	}

	return nil
}

func (s *authService) Me(ctx context.Context, userID string) (*dto.UserProfile, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrUnauthorized
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	role, err := s.roles.FindByCode(ctx, user.RoleCode)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(fmt.Errorf("load role for user %s: %w", user.ID.Hex(), err))
	}

	profile := toUserProfile(user, role.Permissions)
	return &profile, nil
}

func (s *authService) List(ctx context.Context, query dto.ListUsersQuery) (pagination.Result[dto.UserSummary], error) {
	page := pagination.New(query.Page, query.PageSize)

	users, total, err := s.users.List(ctx, query.Role, page)
	if err != nil {
		return pagination.Result[dto.UserSummary]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.UserSummary, 0, len(users))
	for _, u := range users {
		items = append(items, dto.UserSummary{ID: u.ID.Hex(), Name: u.Name, Email: u.Email, Role: u.RoleCode, Status: string(u.Status)})
	}

	return pagination.NewResult(items, page, total), nil
}

func (s *authService) issueRefreshToken(ctx context.Context, userID primitive.ObjectID, meta RequestMeta) (string, error) {
	rawToken, err := security.GenerateOpaqueToken()
	if err != nil {
		return "", err
	}

	record := &domain.RefreshToken{
		UserID:    userID,
		TokenHash: security.HashOpaqueToken(rawToken),
		ExpiresAt: time.Now().UTC().Add(refreshTokenTTL),
		UserAgent: meta.UserAgent,
		IP:        meta.IP,
	}

	if err := s.refreshTokens.Create(ctx, record); err != nil {
		return "", err
	}

	return rawToken, nil
}

func invalidCredentialsError() *apperror.AppError {
	return apperror.New("INVALID_CREDENTIALS", "Email or password is incorrect.", 401)
}

func toUserProfile(user *domain.User, permissions []string) dto.UserProfile {
	return dto.UserProfile{
		ID:          user.ID.Hex(),
		Name:        user.Name,
		Email:       user.Email,
		Role:        user.RoleCode,
		Permissions: permissions,
		Status:      string(user.Status),
		CreatedAt:   user.CreatedAt,
	}
}

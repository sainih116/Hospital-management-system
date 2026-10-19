// Package handler adapts HTTP requests to the auth service layer: binding
// and shallow validation only — all business rules live in service.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/auth/dto"
	"hms-backend/internal/modules/auth/service"
	"hms-backend/internal/response"
)

// Handler serves the auth module's HTTP endpoints.
type Handler struct {
	service service.AuthService
}

// NewHandler builds an auth Handler.
func NewHandler(svc service.AuthService) *Handler {
	return &Handler{service: svc}
}

// Register godoc
// @Summary      Create a staff account
// @Description  Admin-only. Creates a new user with one of the six operational roles.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.RegisterRequest  true  "New user"
// @Success      201      {object}  response.Envelope
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Router       /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	profile, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, profile)
}

// Login godoc
// @Summary      Authenticate
// @Description  Exchanges email/password for an access token and a refresh token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginRequest  true  "Credentials"
// @Success      200      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      429      {object}  response.Envelope
// @Router       /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.Login(c.Request.Context(), req, requestMeta(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

// Refresh godoc
// @Summary      Rotate a session
// @Description  Exchanges a still-valid refresh token for a new access/refresh token pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RefreshRequest  true  "Refresh token"
// @Success      200      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken, requestMeta(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

// Logout godoc
// @Summary      End a session
// @Description  Revokes the given refresh token. Idempotent.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LogoutRequest  true  "Refresh token to revoke"
// @Success      200  {object}  response.Envelope
// @Router       /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"loggedOut": true})
}

// ForgotPassword godoc
// @Summary      Start a password reset
// @Description  Always returns 200 to avoid revealing whether an email is registered.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ForgotPasswordRequest  true  "Account email"
// @Success      200  {object}  response.Envelope
// @Failure      429  {object}  response.Envelope
// @Router       /auth/password/forgot [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"message": "If an account exists for this email, a reset link has been sent."})
}

// ResetPassword godoc
// @Summary      Complete a password reset
// @Description  Redeems a reset token to set a new password and revokes every existing session.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ResetPasswordRequest  true  "Reset token and new password"
// @Success      200  {object}  response.Envelope
// @Failure      400  {object}  response.Envelope
// @Router       /auth/password/reset [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"message": "Password updated. Please sign in again."})
}

// Me godoc
// @Summary      Current user profile
// @Description  Returns the authenticated user's own profile.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Router       /auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		response.Error(c, apperror.ErrUnauthorized)
		return
	}

	profile, err := h.service.Me(c.Request.Context(), claims.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, profile)
}

// ListUsers godoc
// @Summary      List users
// @Description  Admin-only. Lists staff accounts, optionally filtered by role (e.g. to find a DOCTOR-role account to attach a clinical profile to).
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Param        role      query  string  false  "Filter by role code"
// @Param        page      query  int     false  "Page number (default 1)"
// @Param        pageSize  query  int     false  "Items per page (default 20, max 100)"
// @Success      200  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	var query dto.ListUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

func requestMeta(c *gin.Context) service.RequestMeta {
	return service.RequestMeta{
		UserAgent: c.Request.UserAgent(),
		IP:        c.ClientIP(),
	}
}

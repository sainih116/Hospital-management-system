// Package dto defines the request/response payloads for the auth module's
// HTTP API, decoupled from the internal domain model so persistence
// concerns never leak into the wire format.
package dto

import "time"

// RegisterRequest creates a new staff account. Only Admins may call this
// (enforced via RBAC middleware, not by this struct).
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=120"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	RoleCode string `json:"roleCode" binding:"required,oneof=ADMIN DOCTOR NURSE RECEPTIONIST PHARMACIST LAB_TECHNICIAN"`
}

// LoginRequest authenticates a user with email/password.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse carries the issued session and a sanitized user profile.
type LoginResponse struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	ExpiresAt    time.Time   `json:"expiresAt"`
	User         UserProfile `json:"user"`
}

// RefreshRequest exchanges a still-valid refresh token for a new session.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// RefreshResponse carries the rotated session.
type RefreshResponse struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// LogoutRequest revokes a single refresh token (the session on this
// device/browser).
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// ForgotPasswordRequest starts a password-reset flow.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest completes a password-reset flow.
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=72"`
}

// UserProfile is the public-safe representation of a user (never includes
// the password hash). Permissions are included so the frontend can drive
// its own UI gating (show/hide actions) without decoding the JWT itself;
// the backend's RBAC middleware remains the actual enforcement point.
type UserProfile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ListUsersQuery is bound from query parameters for GET /users.
type ListUsersQuery struct {
	Role     string `form:"role" binding:"omitempty,oneof=ADMIN DOCTOR NURSE RECEPTIONIST PHARMACIST LAB_TECHNICIAN"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// UserSummary is a lightweight user representation for admin listing
// screens (e.g. picking a DOCTOR-role account to attach a clinical
// profile to) — deliberately excludes permissions/timestamps that
// UserProfile carries for the "current user" use case.
type UserSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

// Package dto defines the audit logging module's request/response
// payloads.
package dto

import "time"

// SearchAuditLogsQuery is bound from query parameters for GET /audit-logs.
// From/To are RFC3339 timestamps (parsed by the service, not Gin's form
// binding, since Gin has no built-in time.Time query decoder) bounding
// CreatedAt inclusively.
type SearchAuditLogsQuery struct {
	ActorID    string `form:"actorId"`
	EntityType string `form:"entityType"`
	EntityID   string `form:"entityId"`
	Action     string `form:"action"`
	From       string `form:"from"`
	To         string `form:"to"`
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
}

// AuditLogResponse is the public representation of one audit log entry.
type AuditLogResponse struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actorId,omitempty"`
	ActorEmail string         `json:"actorEmail,omitempty"`
	ActorRole  string         `json:"actorRole,omitempty"`
	Action     string         `json:"action"`
	EntityType string         `json:"entityType"`
	EntityID   string         `json:"entityId,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	IP         string         `json:"ip,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

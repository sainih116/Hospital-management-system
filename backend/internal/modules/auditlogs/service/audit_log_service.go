// Package service implements the audit logging module's business logic:
// parsing/validating search filters and mapping persisted audit_logs
// documents to their public representation. There is deliberately no
// write path here — every other module already writes through
// internal/audit.Recorder as it performs auditable actions.
package service

import (
	"context"
	"time"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"

	"hms-backend/internal/modules/auditlogs/dto"
	"hms-backend/internal/modules/auditlogs/repository"
	"hms-backend/internal/pkg/pagination"
)

// AuditLogService is the audit logging module's public business-logic
// contract.
type AuditLogService interface {
	Search(ctx context.Context, query dto.SearchAuditLogsQuery) (pagination.Result[dto.AuditLogResponse], error)
}

type auditLogService struct {
	logs repository.AuditLogRepository
}

// Deps bundles every collaborator auditLogService needs.
type Deps struct {
	Logs repository.AuditLogRepository
}

// NewAuditLogService builds the audit logging module's service
// implementation.
func NewAuditLogService(d Deps) AuditLogService {
	return &auditLogService{logs: d.Logs}
}

func (s *auditLogService) Search(ctx context.Context, query dto.SearchAuditLogsQuery) (pagination.Result[dto.AuditLogResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.SearchFilter{
		ActorID: query.ActorID, EntityType: query.EntityType, EntityID: query.EntityID, Action: query.Action,
	}

	if query.From != "" {
		from, err := time.Parse(time.RFC3339, query.From)
		if err != nil {
			return pagination.Result[dto.AuditLogResponse]{}, apperror.ErrValidation.WithDetails(map[string]string{"from": "must be an RFC3339 timestamp"})
		}
		filter.From = &from
	}
	if query.To != "" {
		to, err := time.Parse(time.RFC3339, query.To)
		if err != nil {
			return pagination.Result[dto.AuditLogResponse]{}, apperror.ErrValidation.WithDetails(map[string]string{"to": "must be an RFC3339 timestamp"})
		}
		filter.To = &to
	}

	logs, total, err := s.logs.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.AuditLogResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.AuditLogResponse, 0, len(logs))
	for _, l := range logs {
		items = append(items, toAuditLogResponse(&l))
	}

	return pagination.NewResult(items, page, total), nil
}

func toAuditLogResponse(l *audit.Log) dto.AuditLogResponse {
	return dto.AuditLogResponse{
		ID: l.ID.Hex(), ActorID: l.ActorID, ActorEmail: l.ActorEmail, ActorRole: l.ActorRole,
		Action: l.Action, EntityType: l.EntityType, EntityID: l.EntityID,
		Metadata: l.Metadata, IP: l.IP, CreatedAt: l.CreatedAt,
	}
}

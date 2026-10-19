package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"

	"hms-backend/internal/modules/auditlogs/dto"
)

func TestSearchFiltersByActorEntityTypeAndAction(t *testing.T) {
	repo := newFakeAuditLogRepository(
		audit.Log{ID: primitive.NewObjectID(), ActorID: "u1", EntityType: "patient", Action: "PATIENT_CREATED", CreatedAt: time.Now()},
		audit.Log{ID: primitive.NewObjectID(), ActorID: "u2", EntityType: "billing", Action: "BILL_CREATED", CreatedAt: time.Now()},
	)
	svc := NewAuditLogService(Deps{Logs: repo})

	result, err := svc.Search(context.Background(), dto.SearchAuditLogsQuery{ActorID: "u1", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search returned an unexpected error: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ActorID != "u1" {
		t.Fatalf("expected only u1's entry to match, got %+v", result.Items)
	}
}

func TestSearchParsesValidDateRange(t *testing.T) {
	repo := newFakeAuditLogRepository()
	svc := NewAuditLogService(Deps{Logs: repo})

	_, err := svc.Search(context.Background(), dto.SearchAuditLogsQuery{
		From: "2026-01-01T00:00:00Z", To: "2026-01-31T23:59:59Z", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("Search returned an unexpected error: %v", err)
	}
	if repo.lastFilter.From == nil || repo.lastFilter.To == nil {
		t.Fatalf("expected both From and To to be parsed into the repository filter, got %+v", repo.lastFilter)
	}
}

func TestSearchRejectsInvalidFromDate(t *testing.T) {
	repo := newFakeAuditLogRepository()
	svc := NewAuditLogService(Deps{Logs: repo})

	_, err := svc.Search(context.Background(), dto.SearchAuditLogsQuery{From: "not-a-date", Page: 1, PageSize: 20})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrValidation.Code {
		t.Fatalf("expected a validation error for an unparsable from date, got: %v", err)
	}
}

func TestSearchRejectsInvalidToDate(t *testing.T) {
	repo := newFakeAuditLogRepository()
	svc := NewAuditLogService(Deps{Logs: repo})

	_, err := svc.Search(context.Background(), dto.SearchAuditLogsQuery{To: "not-a-date", Page: 1, PageSize: 20})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrValidation.Code {
		t.Fatalf("expected a validation error for an unparsable to date, got: %v", err)
	}
}

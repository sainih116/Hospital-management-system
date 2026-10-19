package service

import (
	"context"
	"sync"

	"hms-backend/internal/audit"
	"hms-backend/internal/pkg/pagination"

	"hms-backend/internal/modules/auditlogs/repository"
)

// fakeAuditLogRepository is a minimal in-memory implementation of
// repository.AuditLogRepository. It records the last filter it was
// searched with, so tests can assert the service built the filter it
// meant to.
type fakeAuditLogRepository struct {
	mu         sync.Mutex
	logs       []audit.Log
	lastFilter repository.SearchFilter
}

func newFakeAuditLogRepository(logs ...audit.Log) *fakeAuditLogRepository {
	return &fakeAuditLogRepository{logs: logs}
}

func (f *fakeAuditLogRepository) Search(_ context.Context, filter repository.SearchFilter, page pagination.Request) ([]audit.Log, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastFilter = filter

	var matched []audit.Log
	for _, l := range f.logs {
		if filter.ActorID != "" && l.ActorID != filter.ActorID {
			continue
		}
		if filter.EntityType != "" && l.EntityType != filter.EntityType {
			continue
		}
		if filter.Action != "" && l.Action != filter.Action {
			continue
		}
		matched = append(matched, l)
	}

	total := int64(len(matched))
	start := int(page.Skip())
	if start > len(matched) {
		start = len(matched)
	}
	end := start + int(page.Limit())
	if end > len(matched) {
		end = len(matched)
	}
	return matched[start:end], total, nil
}

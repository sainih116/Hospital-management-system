package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/audit"
	"hms-backend/internal/modules/departments/domain"
	"hms-backend/internal/modules/departments/repository"
)

type fakeDepartmentRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Department
}

func newFakeDepartmentRepository() *fakeDepartmentRepository {
	return &fakeDepartmentRepository{byID: make(map[string]*domain.Department)}
}

func (f *fakeDepartmentRepository) Create(_ context.Context, d *domain.Department) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, existing := range f.byID {
		if existing.Code == d.Code {
			return repository.ErrCodeTaken
		}
	}

	d.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	d.CreatedAt = now
	d.UpdatedAt = now
	if d.Status == "" {
		d.Status = domain.StatusActive
	}

	clone := *d
	f.byID[d.ID.Hex()] = &clone
	return nil
}

func (f *fakeDepartmentRepository) FindByID(_ context.Context, id string) (*domain.Department, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	d, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *d
	return &clone, nil
}

func (f *fakeDepartmentRepository) FindByCode(_ context.Context, code string) (*domain.Department, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, d := range f.byID {
		if d.Code == code {
			clone := *d
			return &clone, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeDepartmentRepository) List(_ context.Context) ([]domain.Department, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]domain.Department, 0, len(f.byID))
	for _, d := range f.byID {
		out = append(out, *d)
	}
	return out, nil
}

func (f *fakeDepartmentRepository) Update(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.Department, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	d, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if v, ok := set["name"]; ok {
		d.Name = v.(string)
	}
	if v, ok := set["status"]; ok {
		d.Status = domain.Status(v.(string))
	}
	d.UpdatedAt = time.Now().UTC()

	clone := *d
	return &clone, nil
}

type fakeAuditRecorder struct {
	mu      sync.Mutex
	entries []audit.Entry
}

func (f *fakeAuditRecorder) Record(_ context.Context, entry audit.Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, entry)
	return nil
}

func (f *fakeAuditRecorder) actions() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.entries))
	for i, e := range f.entries {
		out[i] = e.Action
	}
	return out
}

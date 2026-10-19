package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/audit"
	"hms-backend/internal/modules/labtests/domain"
	"hms-backend/internal/modules/labtests/repository"
)

type fakeLabTestRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.LabTest
}

func newFakeLabTestRepository() *fakeLabTestRepository {
	return &fakeLabTestRepository{byID: make(map[string]*domain.LabTest)}
}

func (f *fakeLabTestRepository) Create(_ context.Context, t *domain.LabTest) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, existing := range f.byID {
		if existing.Code == t.Code {
			return repository.ErrCodeTaken
		}
	}

	t.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = domain.StatusActive
	}

	clone := *t
	f.byID[t.ID.Hex()] = &clone
	return nil
}

func (f *fakeLabTestRepository) FindByID(_ context.Context, id string) (*domain.LabTest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	t, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *t
	return &clone, nil
}

func (f *fakeLabTestRepository) FindByCode(_ context.Context, code string) (*domain.LabTest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, t := range f.byID {
		if t.Code == code {
			clone := *t
			return &clone, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeLabTestRepository) List(_ context.Context) ([]domain.LabTest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]domain.LabTest, 0, len(f.byID))
	for _, t := range f.byID {
		out = append(out, *t)
	}
	return out, nil
}

func (f *fakeLabTestRepository) Update(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.LabTest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	t, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if v, ok := set["name"]; ok {
		t.Name = v.(string)
	}
	if v, ok := set["cost"]; ok {
		t.Cost = v.(float64)
	}
	if v, ok := set["status"]; ok {
		t.Status = domain.Status(v.(string))
	}
	t.UpdatedAt = time.Now().UTC()

	clone := *t
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

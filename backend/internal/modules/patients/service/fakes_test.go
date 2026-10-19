package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/audit"
	"hms-backend/internal/modules/patients/domain"
	"hms-backend/internal/modules/patients/repository"
	"hms-backend/internal/pkg/pagination"
)

// fakePatientRepository is a minimal in-memory implementation of
// repository.PatientRepository used to unit test patientService's business
// logic without a real MongoDB instance.
type fakePatientRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Patient
}

func newFakePatientRepository() *fakePatientRepository {
	return &fakePatientRepository{byID: make(map[string]*domain.Patient)}
}

func (f *fakePatientRepository) Create(_ context.Context, patient *domain.Patient) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	patient.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	patient.CreatedAt = now
	patient.UpdatedAt = now
	if patient.Status == "" {
		patient.Status = domain.StatusActive
	}

	clone := *patient
	f.byID[patient.ID.Hex()] = &clone
	return nil
}

func (f *fakePatientRepository) FindByID(_ context.Context, id string) (*domain.Patient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.byID[id]
	if !ok || p.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	clone := *p
	return &clone, nil
}

func (f *fakePatientRepository) FindByPatientCode(_ context.Context, code string) (*domain.Patient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, p := range f.byID {
		if p.PatientCode == code && p.DeletedAt == nil {
			clone := *p
			return &clone, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakePatientRepository) Update(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.Patient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.byID[id.Hex()]
	if !ok || p.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}

	if v, ok := set["name"]; ok {
		p.Name = v.(string)
	}
	if v, ok := set["dob"]; ok {
		p.DOB = v.(time.Time)
	}
	if v, ok := set["gender"]; ok {
		p.Gender = domain.Gender(v.(string))
	}
	if v, ok := set["phone"]; ok {
		p.Phone = v.(string)
	}
	if v, ok := set["email"]; ok {
		p.Email = v.(string)
	}
	if v, ok := set["address"]; ok {
		p.Address = v.(string)
	}
	if v, ok := set["bloodGroup"]; ok {
		p.BloodGroup = v.(string)
	}
	if v, ok := set["allergies"]; ok {
		p.Allergies = v.([]string)
	}
	if v, ok := set["emergencyContact"]; ok {
		p.EmergencyContact, _ = v.(*domain.EmergencyContact)
	}
	if v, ok := set["insurance"]; ok {
		p.Insurance, _ = v.(*domain.Insurance)
	}
	if v, ok := set["type"]; ok {
		p.Type = domain.PatientType(v.(string))
	}
	p.UpdatedAt = time.Now().UTC()

	clone := *p
	return &clone, nil
}

func (f *fakePatientRepository) SoftDelete(_ context.Context, id primitive.ObjectID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.byID[id.Hex()]
	if !ok || p.DeletedAt != nil {
		return repository.ErrNotFound
	}
	now := time.Now().UTC()
	p.DeletedAt = &now
	p.Status = domain.StatusInactive
	return nil
}

func (f *fakePatientRepository) Search(_ context.Context, filter repository.SearchFilter, page pagination.Request) ([]domain.Patient, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var matched []domain.Patient
	for _, p := range f.byID {
		if p.DeletedAt != nil {
			continue
		}
		if filter.Gender != "" && string(p.Gender) != filter.Gender {
			continue
		}
		if filter.BloodGroup != "" && p.BloodGroup != filter.BloodGroup {
			continue
		}
		if filter.Type != "" && string(p.Type) != filter.Type {
			continue
		}
		if filter.Query != "" && !containsFold(p.Name, filter.Query) && !containsFold(p.PatientCode, filter.Query) && !containsFold(p.Phone, filter.Query) {
			continue
		}
		matched = append(matched, *p)
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

func containsFold(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	// Simple case-sensitive contains is sufficient for these tests since
	// fixtures use consistent casing; production filtering uses a
	// case-insensitive regex (see repository.patientRepository.Search).
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// fakeSequenceGenerator hands out sequential integers per key, starting at 1.
type fakeSequenceGenerator struct {
	mu     sync.Mutex
	values map[string]int64
}

func newFakeSequenceGenerator() *fakeSequenceGenerator {
	return &fakeSequenceGenerator{values: make(map[string]int64)}
}

func (f *fakeSequenceGenerator) Next(_ context.Context, key string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[key]++
	return f.values[key], nil
}

// fakeAuditRecorder captures every recorded entry for assertions.
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

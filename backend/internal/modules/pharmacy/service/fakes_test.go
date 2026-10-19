package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/audit"
	authDomain "hms-backend/internal/modules/auth/domain"
	authRepo "hms-backend/internal/modules/auth/repository"
	prescriptionsDomain "hms-backend/internal/modules/prescriptions/domain"
	prescriptionsRepo "hms-backend/internal/modules/prescriptions/repository"
	"hms-backend/internal/pkg/pagination"

	"hms-backend/internal/modules/pharmacy/domain"
	"hms-backend/internal/modules/pharmacy/repository"
)

// fakeMedicineRepository is a minimal in-memory implementation of
// repository.MedicineRepository.
type fakeMedicineRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Medicine
}

func newFakeMedicineRepository() *fakeMedicineRepository {
	return &fakeMedicineRepository{byID: make(map[string]*domain.Medicine)}
}

func (f *fakeMedicineRepository) seed(m *domain.Medicine) *domain.Medicine {
	f.mu.Lock()
	defer f.mu.Unlock()
	m.ID = primitive.NewObjectID()
	if m.Status == "" {
		m.Status = domain.StatusActive
	}
	clone := *m
	f.byID[m.ID.Hex()] = &clone
	return m
}

func (f *fakeMedicineRepository) Create(_ context.Context, m *domain.Medicine) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now
	if m.Status == "" {
		m.Status = domain.StatusActive
	}
	clone := *m
	f.byID[m.ID.Hex()] = &clone
	return nil
}

func (f *fakeMedicineRepository) FindByID(_ context.Context, id string) (*domain.Medicine, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *m
	return &clone, nil
}

func (f *fakeMedicineRepository) FindByName(_ context.Context, name string) (*domain.Medicine, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range f.byID {
		if m.Name == name {
			clone := *m
			return &clone, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeMedicineRepository) Update(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.Medicine, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if v, ok := set["status"]; ok {
		m.Status = domain.Status(v.(string))
	}
	m.UpdatedAt = time.Now().UTC()
	clone := *m
	return &clone, nil
}

func (f *fakeMedicineRepository) Search(_ context.Context, filter repository.MedicineSearchFilter, page pagination.Request) ([]domain.Medicine, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var matched []domain.Medicine
	for _, m := range f.byID {
		if filter.LowStock && !m.IsLowStock() {
			continue
		}
		matched = append(matched, *m)
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

func (f *fakeMedicineRepository) AdjustStock(_ context.Context, id primitive.ObjectID, delta int) (*domain.Medicine, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if delta < 0 && m.CurrentStock+delta < 0 {
		return nil, repository.ErrInsufficientStock
	}
	m.CurrentStock += delta
	m.UpdatedAt = time.Now().UTC()
	clone := *m
	return &clone, nil
}

// fakeBatchRepository is a minimal in-memory implementation of
// repository.BatchRepository, sorting by expiry date to honor FEFO.
type fakeBatchRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Batch
}

func newFakeBatchRepository() *fakeBatchRepository {
	return &fakeBatchRepository{byID: make(map[string]*domain.Batch)}
}

func (f *fakeBatchRepository) seed(b *domain.Batch) *domain.Batch {
	f.mu.Lock()
	defer f.mu.Unlock()
	b.ID = primitive.NewObjectID()
	clone := *b
	f.byID[b.ID.Hex()] = &clone
	return b
}

func (f *fakeBatchRepository) Create(_ context.Context, b *domain.Batch) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	b.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now
	clone := *b
	f.byID[b.ID.Hex()] = &clone
	return nil
}

func (f *fakeBatchRepository) FindByID(_ context.Context, id string) (*domain.Batch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *b
	return &clone, nil
}

func (f *fakeBatchRepository) FindAvailableForMedicine(_ context.Context, medicineID primitive.ObjectID, now time.Time) ([]domain.Batch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var matched []domain.Batch
	for _, b := range f.byID {
		if b.MedicineID != medicineID {
			continue
		}
		if b.Quantity <= 0 {
			continue
		}
		if !b.ExpiryDate.After(now) {
			continue
		}
		matched = append(matched, *b)
	}
	// FEFO: sort ascending by expiry date (simple insertion sort, fine for
	// the small fixture sizes these tests use).
	for i := 1; i < len(matched); i++ {
		for j := i; j > 0 && matched[j].ExpiryDate.Before(matched[j-1].ExpiryDate); j-- {
			matched[j], matched[j-1] = matched[j-1], matched[j]
		}
	}
	return matched, nil
}

func (f *fakeBatchRepository) FindByMedicineID(_ context.Context, medicineID primitive.ObjectID) ([]domain.Batch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var matched []domain.Batch
	for _, b := range f.byID {
		if b.MedicineID == medicineID {
			matched = append(matched, *b)
		}
	}
	return matched, nil
}

func (f *fakeBatchRepository) FindExpiringBefore(_ context.Context, cutoff time.Time) ([]domain.Batch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var matched []domain.Batch
	for _, b := range f.byID {
		if b.ExpiryDate.Before(cutoff) {
			matched = append(matched, *b)
		}
	}
	return matched, nil
}

func (f *fakeBatchRepository) DecrementQuantity(_ context.Context, id primitive.ObjectID, amount int) (*domain.Batch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if b.Quantity < amount {
		return nil, repository.ErrInsufficientBatchStock
	}
	b.Quantity -= amount
	b.UpdatedAt = time.Now().UTC()
	clone := *b
	return &clone, nil
}

func (f *fakeBatchRepository) IncrementQuantity(_ context.Context, id primitive.ObjectID, amount int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id.Hex()]
	if !ok {
		return repository.ErrNotFound
	}
	b.Quantity += amount
	b.UpdatedAt = time.Now().UTC()
	return nil
}

// fakeStockMovementRepository is a minimal in-memory implementation of
// repository.StockMovementRepository.
type fakeStockMovementRepository struct {
	mu        sync.Mutex
	movements []domain.StockMovement
}

func newFakeStockMovementRepository() *fakeStockMovementRepository {
	return &fakeStockMovementRepository{}
}

func (f *fakeStockMovementRepository) Create(_ context.Context, m *domain.StockMovement) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m.ID = primitive.NewObjectID()
	m.CreatedAt = time.Now().UTC()
	f.movements = append(f.movements, *m)
	return nil
}

func (f *fakeStockMovementRepository) Search(_ context.Context, filter repository.StockMovementFilter, page pagination.Request) ([]domain.StockMovement, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var matched []domain.StockMovement
	for _, m := range f.movements {
		if filter.MedicineID != "" && m.MedicineID.Hex() != filter.MedicineID {
			continue
		}
		if filter.Type != "" && string(m.Type) != filter.Type {
			continue
		}
		matched = append(matched, m)
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

func (f *fakeStockMovementRepository) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.movements)
}

// fakePrescriptionRepository implements just enough of
// prescriptionsRepo.PrescriptionRepository for pharmacyService's dispense
// flow.
type fakePrescriptionRepository struct {
	mu   sync.Mutex
	byID map[string]*prescriptionsDomain.Prescription
}

func newFakePrescriptionRepository() *fakePrescriptionRepository {
	return &fakePrescriptionRepository{byID: make(map[string]*prescriptionsDomain.Prescription)}
}

func (f *fakePrescriptionRepository) seed(p *prescriptionsDomain.Prescription) *prescriptionsDomain.Prescription {
	f.mu.Lock()
	defer f.mu.Unlock()
	p.ID = primitive.NewObjectID()
	if p.Status == "" {
		p.Status = prescriptionsDomain.StatusActive
	}
	clone := *p
	f.byID[p.ID.Hex()] = &clone
	return p
}

func (f *fakePrescriptionRepository) Create(context.Context, *prescriptionsDomain.Prescription) error {
	return nil
}

func (f *fakePrescriptionRepository) FindByID(_ context.Context, id string) (*prescriptionsDomain.Prescription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.byID[id]
	if !ok {
		return nil, prescriptionsRepo.ErrNotFound
	}
	clone := *p
	return &clone, nil
}

func (f *fakePrescriptionRepository) UpdateFields(_ context.Context, id primitive.ObjectID, set bson.M) (*prescriptionsDomain.Prescription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.byID[id.Hex()]
	if !ok {
		return nil, prescriptionsRepo.ErrNotFound
	}
	if v, ok := set["status"]; ok {
		p.Status = prescriptionsDomain.Status(v.(string))
	}
	p.UpdatedAt = time.Now().UTC()
	clone := *p
	return &clone, nil
}

func (f *fakePrescriptionRepository) Search(context.Context, prescriptionsRepo.SearchFilter, pagination.Request) ([]prescriptionsDomain.Prescription, int64, error) {
	return nil, 0, nil
}

func (f *fakePrescriptionRepository) statusOf(id primitive.ObjectID) prescriptionsDomain.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.byID[id.Hex()].Status
}

// fakeUserRepository implements just enough of authRepo.UserRepository for
// pharmacyService's display-name resolution.
type fakeUserRepository struct {
	byID map[string]*authDomain.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{byID: make(map[string]*authDomain.User)}
}

func (f *fakeUserRepository) Create(context.Context, *authDomain.User) error { return nil }
func (f *fakeUserRepository) FindByEmail(context.Context, string) (*authDomain.User, error) {
	return nil, authRepo.ErrNotFound
}
func (f *fakeUserRepository) FindByID(_ context.Context, id string) (*authDomain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, authRepo.ErrNotFound
	}
	return u, nil
}
func (f *fakeUserRepository) IsActiveUser(context.Context, string) (bool, error) { return true, nil }
func (f *fakeUserRepository) UpdatePasswordHash(context.Context, primitive.ObjectID, string) error {
	return nil
}
func (f *fakeUserRepository) UpdateLastLoginAt(context.Context, primitive.ObjectID, time.Time) error {
	return nil
}
func (f *fakeUserRepository) List(context.Context, string, pagination.Request) ([]authDomain.User, int64, error) {
	return nil, 0, nil
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

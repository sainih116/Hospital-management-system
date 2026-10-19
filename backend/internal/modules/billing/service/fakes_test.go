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
	patientsDomain "hms-backend/internal/modules/patients/domain"
	patientsRepo "hms-backend/internal/modules/patients/repository"
	"hms-backend/internal/pkg/pagination"

	"hms-backend/internal/modules/billing/domain"
	"hms-backend/internal/modules/billing/repository"
)

// fakeBillRepository is a minimal in-memory implementation of
// repository.BillRepository.
type fakeBillRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Bill
}

func newFakeBillRepository() *fakeBillRepository {
	return &fakeBillRepository{byID: make(map[string]*domain.Bill)}
}

func (f *fakeBillRepository) Create(_ context.Context, b *domain.Bill) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	b.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now
	if b.Status == "" {
		b.Status = domain.StatusUnpaid
	}
	clone := *b
	f.byID[b.ID.Hex()] = &clone
	return nil
}

func (f *fakeBillRepository) FindByID(_ context.Context, id string) (*domain.Bill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *b
	return &clone, nil
}

func (f *fakeBillRepository) UpdateFields(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.Bill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if v, ok := set["status"]; ok {
		b.Status = domain.Status(v.(string))
	}
	b.UpdatedAt = time.Now().UTC()
	clone := *b
	return &clone, nil
}

func (f *fakeBillRepository) Search(_ context.Context, filter repository.SearchFilter, page pagination.Request) ([]domain.Bill, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var matched []domain.Bill
	for _, b := range f.byID {
		if filter.OutstandingOnly && b.Status != domain.StatusUnpaid && b.Status != domain.StatusPartiallyPaid {
			continue
		}
		if !filter.OutstandingOnly && filter.Status != "" && string(b.Status) != filter.Status {
			continue
		}
		matched = append(matched, *b)
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

func (f *fakeBillRepository) RecordPayment(_ context.Context, id primitive.ObjectID, amount float64) (*domain.Bill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if b.AmountPaid+amount > b.TotalAmount {
		return nil, repository.ErrOverpayment
	}
	b.AmountPaid += amount
	b.UpdatedAt = time.Now().UTC()
	clone := *b
	return &clone, nil
}

// fakePaymentRepository is a minimal in-memory implementation of
// repository.PaymentRepository.
type fakePaymentRepository struct {
	mu       sync.Mutex
	payments []domain.Payment
}

func newFakePaymentRepository() *fakePaymentRepository {
	return &fakePaymentRepository{}
}

func (f *fakePaymentRepository) Create(_ context.Context, p *domain.Payment) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	p.CreatedAt = now
	if p.PaidAt.IsZero() {
		p.PaidAt = now
	}
	f.payments = append(f.payments, *p)
	return nil
}

func (f *fakePaymentRepository) FindByBillID(_ context.Context, billID primitive.ObjectID) ([]domain.Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var matched []domain.Payment
	for _, p := range f.payments {
		if p.BillID == billID {
			matched = append(matched, p)
		}
	}
	return matched, nil
}

// fakePatientRepository implements just enough of
// patientsRepo.PatientRepository for billingService's lookups.
type fakePatientRepository struct {
	byID map[string]*patientsDomain.Patient
}

func newFakePatientRepository() *fakePatientRepository {
	return &fakePatientRepository{byID: make(map[string]*patientsDomain.Patient)}
}

func (f *fakePatientRepository) seed(p *patientsDomain.Patient) *patientsDomain.Patient {
	p.ID = primitive.NewObjectID()
	f.byID[p.ID.Hex()] = p
	return p
}

func (f *fakePatientRepository) Create(context.Context, *patientsDomain.Patient) error { return nil }
func (f *fakePatientRepository) FindByID(_ context.Context, id string) (*patientsDomain.Patient, error) {
	p, ok := f.byID[id]
	if !ok {
		return nil, patientsRepo.ErrNotFound
	}
	return p, nil
}
func (f *fakePatientRepository) FindByPatientCode(context.Context, string) (*patientsDomain.Patient, error) {
	return nil, patientsRepo.ErrNotFound
}
func (f *fakePatientRepository) Update(context.Context, primitive.ObjectID, bson.M) (*patientsDomain.Patient, error) {
	return nil, patientsRepo.ErrNotFound
}
func (f *fakePatientRepository) SoftDelete(context.Context, primitive.ObjectID) error { return nil }
func (f *fakePatientRepository) Search(context.Context, patientsRepo.SearchFilter, pagination.Request) ([]patientsDomain.Patient, int64, error) {
	return nil, 0, nil
}

// fakeUserRepository implements just enough of authRepo.UserRepository for
// billingService's display-name resolution.
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

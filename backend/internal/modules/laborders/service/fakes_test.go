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
	doctorsDomain "hms-backend/internal/modules/doctors/domain"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	labTestsDomain "hms-backend/internal/modules/labtests/domain"
	labTestsRepo "hms-backend/internal/modules/labtests/repository"
	patientsDomain "hms-backend/internal/modules/patients/domain"
	patientsRepo "hms-backend/internal/modules/patients/repository"
	"hms-backend/internal/pkg/pagination"

	"hms-backend/internal/modules/laborders/domain"
	"hms-backend/internal/modules/laborders/repository"
)

// fakeOrderRepository is a minimal in-memory implementation of
// repository.LabOrderRepository.
type fakeOrderRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.LabOrder
}

func newFakeOrderRepository() *fakeOrderRepository {
	return &fakeOrderRepository{byID: make(map[string]*domain.LabOrder)}
}

func (f *fakeOrderRepository) Create(_ context.Context, o *domain.LabOrder) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	o.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	o.CreatedAt = now
	o.UpdatedAt = now
	if o.Status == "" {
		o.Status = domain.StatusOrdered
	}

	clone := *o
	f.byID[o.ID.Hex()] = &clone
	return nil
}

func (f *fakeOrderRepository) FindByID(_ context.Context, id string) (*domain.LabOrder, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	o, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *o
	return &clone, nil
}

func (f *fakeOrderRepository) UpdateFields(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.LabOrder, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	o, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if v, ok := set["status"]; ok {
		o.Status = domain.Status(v.(string))
	}
	o.UpdatedAt = time.Now().UTC()

	clone := *o
	return &clone, nil
}

func (f *fakeOrderRepository) Search(_ context.Context, filter repository.SearchFilter, page pagination.Request) ([]domain.LabOrder, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var matched []domain.LabOrder
	for _, o := range f.byID {
		if filter.Status != "" && string(o.Status) != filter.Status {
			continue
		}
		matched = append(matched, *o)
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

// fakeResultRepository is a minimal in-memory implementation of
// repository.LabResultRepository.
type fakeResultRepository struct {
	mu      sync.Mutex
	byID    map[string]*domain.LabResult
	byOrder map[string]string // orderID.Hex() -> resultID.Hex()
}

func newFakeResultRepository() *fakeResultRepository {
	return &fakeResultRepository{byID: make(map[string]*domain.LabResult), byOrder: make(map[string]string)}
}

func (f *fakeResultRepository) Create(_ context.Context, r *domain.LabResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.byOrder[r.LabOrderID.Hex()]; exists {
		return repository.ErrAlreadyExists
	}

	r.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.EnteredAt.IsZero() {
		r.EnteredAt = now
	}

	clone := *r
	f.byID[r.ID.Hex()] = &clone
	f.byOrder[r.LabOrderID.Hex()] = r.ID.Hex()
	return nil
}

func (f *fakeResultRepository) FindByID(_ context.Context, id string) (*domain.LabResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	r, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *r
	return &clone, nil
}

func (f *fakeResultRepository) FindByLabOrderID(_ context.Context, labOrderID primitive.ObjectID) (*domain.LabResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	resultID, ok := f.byOrder[labOrderID.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *f.byID[resultID]
	return &clone, nil
}

func (f *fakeResultRepository) UpdateFields(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.LabResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	r, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if v, ok := set["verifiedByName"]; ok {
		r.VerifiedByName = v.(string)
	}
	if v, ok := set["verifiedAt"]; ok {
		t := v.(time.Time)
		r.VerifiedAt = &t
	}
	if v, ok := set["verifiedBy"]; ok {
		oid := v.(primitive.ObjectID)
		r.VerifiedBy = &oid
	}
	r.UpdatedAt = time.Now().UTC()

	clone := *r
	return &clone, nil
}

// fakeLabTestRepository implements just enough of
// labTestsRepo.LabTestRepository for labOrderService's lookups.
type fakeLabTestRepository struct {
	byID map[string]*labTestsDomain.LabTest
}

func newFakeLabTestRepository() *fakeLabTestRepository {
	return &fakeLabTestRepository{byID: make(map[string]*labTestsDomain.LabTest)}
}

func (f *fakeLabTestRepository) seed(t *labTestsDomain.LabTest) {
	t.ID = primitive.NewObjectID()
	f.byID[t.ID.Hex()] = t
}

func (f *fakeLabTestRepository) Create(context.Context, *labTestsDomain.LabTest) error { return nil }
func (f *fakeLabTestRepository) FindByID(_ context.Context, id string) (*labTestsDomain.LabTest, error) {
	t, ok := f.byID[id]
	if !ok {
		return nil, labTestsRepo.ErrNotFound
	}
	return t, nil
}
func (f *fakeLabTestRepository) FindByCode(context.Context, string) (*labTestsDomain.LabTest, error) {
	return nil, labTestsRepo.ErrNotFound
}
func (f *fakeLabTestRepository) List(context.Context) ([]labTestsDomain.LabTest, error) {
	return nil, nil
}
func (f *fakeLabTestRepository) Update(context.Context, primitive.ObjectID, bson.M) (*labTestsDomain.LabTest, error) {
	return nil, labTestsRepo.ErrNotFound
}

// fakePatientRepository implements just enough of
// patientsRepo.PatientRepository for labOrderService's lookups.
type fakePatientRepository struct {
	byID map[string]*patientsDomain.Patient
}

func newFakePatientRepository() *fakePatientRepository {
	return &fakePatientRepository{byID: make(map[string]*patientsDomain.Patient)}
}

func (f *fakePatientRepository) seed(p *patientsDomain.Patient) {
	p.ID = primitive.NewObjectID()
	f.byID[p.ID.Hex()] = p
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

// fakeDoctorRepository implements just enough of
// doctorsRepo.DoctorRepository for labOrderService's lookups.
type fakeDoctorRepository struct {
	byID map[string]*doctorsDomain.Doctor
}

func newFakeDoctorRepository() *fakeDoctorRepository {
	return &fakeDoctorRepository{byID: make(map[string]*doctorsDomain.Doctor)}
}

func (f *fakeDoctorRepository) seed(d *doctorsDomain.Doctor) {
	d.ID = primitive.NewObjectID()
	f.byID[d.ID.Hex()] = d
}

func (f *fakeDoctorRepository) Create(context.Context, *doctorsDomain.Doctor) error { return nil }
func (f *fakeDoctorRepository) FindByID(_ context.Context, id string) (*doctorsDomain.Doctor, error) {
	d, ok := f.byID[id]
	if !ok {
		return nil, doctorsRepo.ErrNotFound
	}
	return d, nil
}
func (f *fakeDoctorRepository) FindByUserID(context.Context, primitive.ObjectID) (*doctorsDomain.Doctor, error) {
	return nil, doctorsRepo.ErrNotFound
}
func (f *fakeDoctorRepository) Update(context.Context, primitive.ObjectID, bson.M) (*doctorsDomain.Doctor, error) {
	return nil, doctorsRepo.ErrNotFound
}
func (f *fakeDoctorRepository) SoftDelete(context.Context, primitive.ObjectID) error { return nil }
func (f *fakeDoctorRepository) Search(context.Context, doctorsRepo.SearchFilter, pagination.Request) ([]doctorsDomain.Doctor, int64, error) {
	return nil, 0, nil
}

// fakeUserRepository implements just enough of authRepo.UserRepository for
// labOrderService's display-name resolution.
type fakeUserRepository struct {
	byID map[string]*authDomain.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{byID: make(map[string]*authDomain.User)}
}

func (f *fakeUserRepository) seed(u *authDomain.User) {
	u.ID = primitive.NewObjectID()
	f.byID[u.ID.Hex()] = u
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

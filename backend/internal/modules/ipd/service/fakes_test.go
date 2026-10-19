package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/audit"
	doctorsDomain "hms-backend/internal/modules/doctors/domain"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	patientsDomain "hms-backend/internal/modules/patients/domain"
	patientsRepo "hms-backend/internal/modules/patients/repository"
	"hms-backend/internal/pkg/pagination"

	"hms-backend/internal/modules/ipd/domain"
	"hms-backend/internal/modules/ipd/repository"
)

// fakeWardRepository is a minimal in-memory implementation of
// repository.WardRepository.
type fakeWardRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Ward
}

func newFakeWardRepository() *fakeWardRepository {
	return &fakeWardRepository{byID: make(map[string]*domain.Ward)}
}

func (f *fakeWardRepository) seed(w *domain.Ward) *domain.Ward {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.ID = primitive.NewObjectID()
	if w.Status == "" {
		w.Status = domain.StatusActive
	}
	clone := *w
	f.byID[w.ID.Hex()] = &clone
	return w
}

func (f *fakeWardRepository) Create(_ context.Context, w *domain.Ward) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	w.CreatedAt, w.UpdatedAt = now, now
	if w.Status == "" {
		w.Status = domain.StatusActive
	}
	clone := *w
	f.byID[w.ID.Hex()] = &clone
	return nil
}

func (f *fakeWardRepository) FindByID(_ context.Context, id string) (*domain.Ward, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *w
	return &clone, nil
}

func (f *fakeWardRepository) List(context.Context) ([]domain.Ward, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.Ward, 0, len(f.byID))
	for _, w := range f.byID {
		out = append(out, *w)
	}
	return out, nil
}

func (f *fakeWardRepository) Update(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.Ward, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if v, ok := set["status"]; ok {
		w.Status = domain.Status(v.(string))
	}
	clone := *w
	return &clone, nil
}

// fakeBedRepository is a minimal in-memory implementation of
// repository.BedRepository, including the atomic Occupy guard the real
// Mongo implementation provides.
type fakeBedRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Bed
}

func newFakeBedRepository() *fakeBedRepository {
	return &fakeBedRepository{byID: make(map[string]*domain.Bed)}
}

func (f *fakeBedRepository) seed(b *domain.Bed) *domain.Bed {
	f.mu.Lock()
	defer f.mu.Unlock()
	b.ID = primitive.NewObjectID()
	if b.Status == "" {
		b.Status = domain.BedAvailable
	}
	clone := *b
	f.byID[b.ID.Hex()] = &clone
	return b
}

func (f *fakeBedRepository) Create(_ context.Context, b *domain.Bed) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	b.ID = primitive.NewObjectID()
	if b.Status == "" {
		b.Status = domain.BedAvailable
	}
	clone := *b
	f.byID[b.ID.Hex()] = &clone
	return nil
}

func (f *fakeBedRepository) FindByID(_ context.Context, id string) (*domain.Bed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *b
	return &clone, nil
}

func (f *fakeBedRepository) List(_ context.Context, filter repository.BedFilter) ([]domain.Bed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Bed
	for _, b := range f.byID {
		if filter.WardID != "" && b.WardID.Hex() != filter.WardID {
			continue
		}
		if filter.Status != "" && string(b.Status) != filter.Status {
			continue
		}
		out = append(out, *b)
	}
	return out, nil
}

func (f *fakeBedRepository) Occupy(_ context.Context, id primitive.ObjectID) (*domain.Bed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if b.Status != domain.BedAvailable {
		return nil, repository.ErrBedNotAvailable
	}
	b.Status = domain.BedOccupied
	clone := *b
	return &clone, nil
}

func (f *fakeBedRepository) Release(_ context.Context, id primitive.ObjectID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id.Hex()]
	if !ok {
		return repository.ErrNotFound
	}
	b.Status = domain.BedAvailable
	return nil
}

func (f *fakeBedRepository) SetStatus(_ context.Context, id primitive.ObjectID, status domain.BedStatus) (*domain.Bed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	b.Status = status
	clone := *b
	return &clone, nil
}

// fakeAdmissionRepository is a minimal in-memory implementation of
// repository.AdmissionRepository.
type fakeAdmissionRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Admission
}

func newFakeAdmissionRepository() *fakeAdmissionRepository {
	return &fakeAdmissionRepository{byID: make(map[string]*domain.Admission)}
}

func (f *fakeAdmissionRepository) Create(_ context.Context, a *domain.Admission) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	a.CreatedAt, a.UpdatedAt = now, now
	if a.Status == "" {
		a.Status = domain.AdmissionActive
	}
	if a.AdmittedAt.IsZero() {
		a.AdmittedAt = now
	}
	clone := *a
	f.byID[a.ID.Hex()] = &clone
	return nil
}

func (f *fakeAdmissionRepository) FindByID(_ context.Context, id string) (*domain.Admission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *a
	return &clone, nil
}

func (f *fakeAdmissionRepository) UpdateFields(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.Admission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if v, ok := set["status"]; ok {
		a.Status = domain.AdmissionStatus(v.(string))
	}
	if v, ok := set["dischargedAt"]; ok {
		t := v.(time.Time)
		a.DischargedAt = &t
	}
	if v, ok := set["dischargeNote"]; ok {
		a.DischargeNote = v.(string)
	}
	clone := *a
	return &clone, nil
}

func (f *fakeAdmissionRepository) Search(_ context.Context, filter repository.AdmissionFilter, page pagination.Request) ([]domain.Admission, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var matched []domain.Admission
	for _, a := range f.byID {
		if filter.PatientID != "" && a.PatientID.Hex() != filter.PatientID {
			continue
		}
		if filter.Status != "" && string(a.Status) != filter.Status {
			continue
		}
		matched = append(matched, *a)
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

// fakePatientRepository implements just enough of
// patientsRepo.PatientRepository for ipdService's lookups.
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

// fakeDoctorRepository implements just enough of
// doctorsRepo.DoctorRepository for ipdService's lookups.
type fakeDoctorRepository struct {
	byID map[string]*doctorsDomain.Doctor
}

func newFakeDoctorRepository() *fakeDoctorRepository {
	return &fakeDoctorRepository{byID: make(map[string]*doctorsDomain.Doctor)}
}

func (f *fakeDoctorRepository) seed(d *doctorsDomain.Doctor) *doctorsDomain.Doctor {
	d.ID = primitive.NewObjectID()
	f.byID[d.ID.Hex()] = d
	return d
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

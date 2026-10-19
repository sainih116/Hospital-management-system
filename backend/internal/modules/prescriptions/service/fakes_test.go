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
	medicalRecordsDomain "hms-backend/internal/modules/medicalrecords/domain"
	medicalRecordsRepo "hms-backend/internal/modules/medicalrecords/repository"
	patientsDomain "hms-backend/internal/modules/patients/domain"
	patientsRepo "hms-backend/internal/modules/patients/repository"
	"hms-backend/internal/pkg/pagination"

	"hms-backend/internal/modules/prescriptions/domain"
	"hms-backend/internal/modules/prescriptions/repository"
)

// fakePrescriptionRepository is a minimal in-memory implementation of
// repository.PrescriptionRepository.
type fakePrescriptionRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Prescription
}

func newFakePrescriptionRepository() *fakePrescriptionRepository {
	return &fakePrescriptionRepository{byID: make(map[string]*domain.Prescription)}
}

func (f *fakePrescriptionRepository) Create(_ context.Context, p *domain.Prescription) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	p.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = domain.StatusActive
	}
	if p.IssuedAt.IsZero() {
		p.IssuedAt = now
	}

	clone := *p
	f.byID[p.ID.Hex()] = &clone
	return nil
}

func (f *fakePrescriptionRepository) FindByID(_ context.Context, id string) (*domain.Prescription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *p
	return &clone, nil
}

func (f *fakePrescriptionRepository) UpdateFields(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.Prescription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if v, ok := set["status"]; ok {
		p.Status = domain.Status(v.(string))
	}
	p.UpdatedAt = time.Now().UTC()

	clone := *p
	return &clone, nil
}

func (f *fakePrescriptionRepository) Search(_ context.Context, filter repository.SearchFilter, page pagination.Request) ([]domain.Prescription, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var matched []domain.Prescription
	for _, p := range f.byID {
		if filter.PatientID != "" {
			oid, err := primitive.ObjectIDFromHex(filter.PatientID)
			if err == nil && p.PatientID != oid {
				continue
			}
		}
		if filter.Status != "" && string(p.Status) != filter.Status {
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

// fakePatientRepository implements just enough of
// patientsRepo.PatientRepository for prescriptionService's lookups.
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
// doctorsRepo.DoctorRepository for prescriptionService's lookups.
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

// fakeMedicalRecordRepository implements just enough of
// medicalRecordsRepo.MedicalRecordRepository for prescriptionService's
// optional medicalRecordId link validation.
type fakeMedicalRecordRepository struct {
	byID map[string]*medicalRecordsDomain.MedicalRecord
}

func newFakeMedicalRecordRepository() *fakeMedicalRecordRepository {
	return &fakeMedicalRecordRepository{byID: make(map[string]*medicalRecordsDomain.MedicalRecord)}
}

func (f *fakeMedicalRecordRepository) seed(r *medicalRecordsDomain.MedicalRecord) {
	r.ID = primitive.NewObjectID()
	f.byID[r.ID.Hex()] = r
}

func (f *fakeMedicalRecordRepository) Create(context.Context, *medicalRecordsDomain.MedicalRecord) error {
	return nil
}
func (f *fakeMedicalRecordRepository) FindByID(_ context.Context, id string) (*medicalRecordsDomain.MedicalRecord, error) {
	r, ok := f.byID[id]
	if !ok {
		return nil, medicalRecordsRepo.ErrNotFound
	}
	return r, nil
}
func (f *fakeMedicalRecordRepository) Update(context.Context, primitive.ObjectID, bson.M) (*medicalRecordsDomain.MedicalRecord, error) {
	return nil, medicalRecordsRepo.ErrNotFound
}
func (f *fakeMedicalRecordRepository) Search(context.Context, medicalRecordsRepo.SearchFilter, pagination.Request) ([]medicalRecordsDomain.MedicalRecord, int64, error) {
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

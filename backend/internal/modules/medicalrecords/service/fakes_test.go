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

	"hms-backend/internal/modules/medicalrecords/domain"
	"hms-backend/internal/modules/medicalrecords/repository"
)

// fakeRecordRepository is a minimal in-memory implementation of
// repository.MedicalRecordRepository.
type fakeRecordRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.MedicalRecord
}

func newFakeRecordRepository() *fakeRecordRepository {
	return &fakeRecordRepository{byID: make(map[string]*domain.MedicalRecord)}
}

func (f *fakeRecordRepository) Create(_ context.Context, r *domain.MedicalRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	r.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now

	clone := *r
	f.byID[r.ID.Hex()] = &clone
	return nil
}

func (f *fakeRecordRepository) FindByID(_ context.Context, id string) (*domain.MedicalRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	r, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *r
	return &clone, nil
}

func (f *fakeRecordRepository) Update(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.MedicalRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	r, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}

	if v, ok := set["complaint"]; ok {
		r.Complaint = v.(string)
	}
	if v, ok := set["hpi"]; ok {
		r.HPI = v.(string)
	}
	if v, ok := set["history"]; ok {
		r.History = v.(string)
	}
	if v, ok := set["examination"]; ok {
		r.Examination = v.(string)
	}
	if v, ok := set["diagnosis"]; ok {
		r.Diagnosis = v.(string)
	}
	if v, ok := set["treatment"]; ok {
		r.Treatment = v.(string)
	}
	if v, ok := set["followUp"]; ok {
		r.FollowUp = v.(string)
	}
	r.UpdatedAt = time.Now().UTC()

	clone := *r
	return &clone, nil
}

func (f *fakeRecordRepository) Search(_ context.Context, filter repository.SearchFilter, page pagination.Request) ([]domain.MedicalRecord, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var matched []domain.MedicalRecord
	for _, r := range f.byID {
		if filter.PatientID != "" {
			oid, err := primitive.ObjectIDFromHex(filter.PatientID)
			if err == nil && r.PatientID != oid {
				continue
			}
		}
		if filter.DoctorID != "" {
			oid, err := primitive.ObjectIDFromHex(filter.DoctorID)
			if err == nil && r.DoctorID != oid {
				continue
			}
		}
		matched = append(matched, *r)
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
// patientsRepo.PatientRepository for medicalRecordService's lookups.
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
// doctorsRepo.DoctorRepository for medicalRecordService's lookups.
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

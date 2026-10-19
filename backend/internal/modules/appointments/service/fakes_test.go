package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/audit"
	"hms-backend/internal/modules/appointments/domain"
	"hms-backend/internal/modules/appointments/repository"
	doctorsDomain "hms-backend/internal/modules/doctors/domain"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	patientsDomain "hms-backend/internal/modules/patients/domain"
	patientsRepo "hms-backend/internal/modules/patients/repository"
	"hms-backend/internal/pkg/pagination"
)

// fakeAppointmentRepository is a minimal in-memory implementation of
// repository.AppointmentRepository, faithful enough to exercise the
// module's double-booking defenses (see repository package doc) in tests.
type fakeAppointmentRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Appointment
}

func newFakeAppointmentRepository() *fakeAppointmentRepository {
	return &fakeAppointmentRepository{byID: make(map[string]*domain.Appointment)}
}

func holdsSlot(status domain.Status) bool {
	return status != domain.StatusCancelled && status != domain.StatusNoShow && status != domain.StatusRescheduled
}

func (f *fakeAppointmentRepository) Create(_ context.Context, appt *domain.Appointment) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if appt.Status == "" {
		appt.Status = domain.StatusScheduled
	}

	// Mirrors the partial unique index on (doctorId, startAt) scoped to
	// calendar-holding statuses.
	if holdsSlot(appt.Status) {
		for _, existing := range f.byID {
			if holdsSlot(existing.Status) && existing.DoctorID == appt.DoctorID && existing.StartAt.Equal(appt.StartAt) {
				return repository.ErrSlotTaken
			}
		}
	}

	appt.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	appt.CreatedAt = now
	appt.UpdatedAt = now

	clone := *appt
	f.byID[appt.ID.Hex()] = &clone
	return nil
}

func (f *fakeAppointmentRepository) FindByID(_ context.Context, id string) (*domain.Appointment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	a, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *a
	return &clone, nil
}

func (f *fakeAppointmentRepository) HasConflict(_ context.Context, doctorID primitive.ObjectID, start, end time.Time, excludeID *primitive.ObjectID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, a := range f.byID {
		if !holdsSlot(a.Status) || a.DoctorID != doctorID {
			continue
		}
		if excludeID != nil && a.ID == *excludeID {
			continue
		}
		if a.StartAt.Before(end) && a.EndAt.After(start) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeAppointmentRepository) UpdateFields(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.Appointment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	a, ok := f.byID[id.Hex()]
	if !ok {
		return nil, repository.ErrNotFound
	}

	if v, ok := set["status"]; ok {
		a.Status = domain.Status(v.(string))
	}
	if v, ok := set["cancelReason"]; ok {
		a.CancelReason = v.(string)
	}
	if v, ok := set["rescheduledToId"]; ok {
		oid := v.(primitive.ObjectID)
		a.RescheduledToID = &oid
	}
	a.UpdatedAt = time.Now().UTC()

	clone := *a
	return &clone, nil
}

func (f *fakeAppointmentRepository) Search(_ context.Context, filter repository.SearchFilter, page pagination.Request) ([]domain.Appointment, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var matched []domain.Appointment
	for _, a := range f.byID {
		if filter.DoctorID != "" {
			oid, err := primitive.ObjectIDFromHex(filter.DoctorID)
			if err == nil && a.DoctorID != oid {
				continue
			}
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
// patientsRepo.PatientRepository for appointmentService's lookups.
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
// doctorsRepo.DoctorRepository for appointmentService's lookups.
type fakeDoctorRepository struct {
	byID     map[string]*doctorsDomain.Doctor
	byUserID map[primitive.ObjectID]*doctorsDomain.Doctor
}

func newFakeDoctorRepository() *fakeDoctorRepository {
	return &fakeDoctorRepository{byID: make(map[string]*doctorsDomain.Doctor), byUserID: make(map[primitive.ObjectID]*doctorsDomain.Doctor)}
}

func (f *fakeDoctorRepository) seed(d *doctorsDomain.Doctor) {
	d.ID = primitive.NewObjectID()
	if d.UserID.IsZero() {
		d.UserID = primitive.NewObjectID()
	}
	f.byID[d.ID.Hex()] = d
	f.byUserID[d.UserID] = d
}

func (f *fakeDoctorRepository) Create(context.Context, *doctorsDomain.Doctor) error { return nil }
func (f *fakeDoctorRepository) FindByID(_ context.Context, id string) (*doctorsDomain.Doctor, error) {
	d, ok := f.byID[id]
	if !ok {
		return nil, doctorsRepo.ErrNotFound
	}
	return d, nil
}
func (f *fakeDoctorRepository) FindByUserID(_ context.Context, userID primitive.ObjectID) (*doctorsDomain.Doctor, error) {
	d, ok := f.byUserID[userID]
	if !ok {
		return nil, doctorsRepo.ErrNotFound
	}
	return d, nil
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

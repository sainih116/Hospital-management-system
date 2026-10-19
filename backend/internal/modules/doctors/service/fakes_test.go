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
	deptDomain "hms-backend/internal/modules/departments/domain"
	deptRepo "hms-backend/internal/modules/departments/repository"
	"hms-backend/internal/modules/doctors/domain"
	"hms-backend/internal/modules/doctors/repository"
	"hms-backend/internal/pkg/pagination"
)

// fakeDoctorRepository is a minimal in-memory implementation of
// repository.DoctorRepository used to unit test doctorService's business
// logic without a real MongoDB instance.
type fakeDoctorRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Doctor
}

func newFakeDoctorRepository() *fakeDoctorRepository {
	return &fakeDoctorRepository{byID: make(map[string]*domain.Doctor)}
}

func (f *fakeDoctorRepository) Create(_ context.Context, doctor *domain.Doctor) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, existing := range f.byID {
		if existing.DeletedAt != nil {
			continue
		}
		if existing.UserID == doctor.UserID {
			return repository.ErrUserAlreadyLinked
		}
		if existing.LicenseNo == doctor.LicenseNo {
			return repository.ErrLicenseTaken
		}
	}

	doctor.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	doctor.CreatedAt = now
	doctor.UpdatedAt = now
	if doctor.Status == "" {
		doctor.Status = domain.StatusActive
	}

	clone := *doctor
	f.byID[doctor.ID.Hex()] = &clone
	return nil
}

func (f *fakeDoctorRepository) FindByID(_ context.Context, id string) (*domain.Doctor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	d, ok := f.byID[id]
	if !ok || d.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	clone := *d
	return &clone, nil
}

func (f *fakeDoctorRepository) FindByUserID(_ context.Context, userID primitive.ObjectID) (*domain.Doctor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, d := range f.byID {
		if d.UserID == userID && d.DeletedAt == nil {
			clone := *d
			return &clone, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeDoctorRepository) Update(_ context.Context, id primitive.ObjectID, set bson.M) (*domain.Doctor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	d, ok := f.byID[id.Hex()]
	if !ok || d.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}

	if v, ok := set["licenseNo"]; ok {
		newLicense := v.(string)
		for otherID, other := range f.byID {
			if otherID != id.Hex() && other.DeletedAt == nil && other.LicenseNo == newLicense {
				return nil, repository.ErrLicenseTaken
			}
		}
		d.LicenseNo = newLicense
	}
	if v, ok := set["departmentId"]; ok {
		d.DepartmentID = v.(primitive.ObjectID)
	}
	if v, ok := set["departmentName"]; ok {
		d.DepartmentName = v.(string)
	}
	if v, ok := set["specialization"]; ok {
		d.Specialization = v.(string)
	}
	if v, ok := set["fee"]; ok {
		d.Fee = v.(float64)
	}
	if v, ok := set["bio"]; ok {
		d.Bio = v.(string)
	}
	if v, ok := set["availability"]; ok {
		d.Availability = v.([]domain.AvailabilitySlot)
	}
	d.UpdatedAt = time.Now().UTC()

	clone := *d
	return &clone, nil
}

func (f *fakeDoctorRepository) SoftDelete(_ context.Context, id primitive.ObjectID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	d, ok := f.byID[id.Hex()]
	if !ok || d.DeletedAt != nil {
		return repository.ErrNotFound
	}
	now := time.Now().UTC()
	d.DeletedAt = &now
	d.Status = domain.StatusInactive
	return nil
}

func (f *fakeDoctorRepository) Search(_ context.Context, filter repository.SearchFilter, page pagination.Request) ([]domain.Doctor, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var matched []domain.Doctor
	for _, d := range f.byID {
		if d.DeletedAt != nil {
			continue
		}
		if filter.DepartmentID != "" {
			oid, err := primitive.ObjectIDFromHex(filter.DepartmentID)
			if err == nil && d.DepartmentID != oid {
				continue
			}
		}
		matched = append(matched, *d)
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

// fakeUserRepository implements just enough of authRepo.UserRepository for
// doctorService's cross-module lookups.
type fakeUserRepository struct {
	byID map[string]*authDomain.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{byID: make(map[string]*authDomain.User)}
}

func (f *fakeUserRepository) seed(user *authDomain.User) {
	user.ID = primitive.NewObjectID()
	f.byID[user.ID.Hex()] = user
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

// fakeDepartmentRepository implements just enough of
// deptRepo.DepartmentRepository for doctorService's cross-module lookups.
type fakeDepartmentRepository struct {
	byID map[string]*deptDomain.Department
}

func newFakeDepartmentRepository() *fakeDepartmentRepository {
	return &fakeDepartmentRepository{byID: make(map[string]*deptDomain.Department)}
}

func (f *fakeDepartmentRepository) seed(d *deptDomain.Department) {
	d.ID = primitive.NewObjectID()
	f.byID[d.ID.Hex()] = d
}

func (f *fakeDepartmentRepository) Create(context.Context, *deptDomain.Department) error { return nil }
func (f *fakeDepartmentRepository) FindByID(_ context.Context, id string) (*deptDomain.Department, error) {
	d, ok := f.byID[id]
	if !ok {
		return nil, deptRepo.ErrNotFound
	}
	return d, nil
}
func (f *fakeDepartmentRepository) FindByCode(context.Context, string) (*deptDomain.Department, error) {
	return nil, deptRepo.ErrNotFound
}
func (f *fakeDepartmentRepository) List(context.Context) ([]deptDomain.Department, error) {
	return nil, nil
}
func (f *fakeDepartmentRepository) Update(_ context.Context, id primitive.ObjectID, set bson.M) (*deptDomain.Department, error) {
	d, ok := f.byID[id.Hex()]
	if !ok {
		return nil, deptRepo.ErrNotFound
	}
	return d, nil
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

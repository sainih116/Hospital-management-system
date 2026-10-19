package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	authDomain "hms-backend/internal/modules/auth/domain"
	deptDomain "hms-backend/internal/modules/departments/domain"
	"hms-backend/internal/modules/doctors/dto"
)

type harness struct {
	svc         DoctorService
	doctors     *fakeDoctorRepository
	users       *fakeUserRepository
	departments *fakeDepartmentRepository
	audit       *fakeAuditRecorder
}

func newHarness() *harness {
	doctors := newFakeDoctorRepository()
	users := newFakeUserRepository()
	departmentsRepo := newFakeDepartmentRepository()
	auditRecorder := &fakeAuditRecorder{}

	svc := NewDoctorService(Deps{
		Doctors:     doctors,
		Users:       users,
		Departments: departmentsRepo,
		Audit:       auditRecorder,
		Logger:      zap.NewNop(),
	})

	return &harness{svc: svc, doctors: doctors, users: users, departments: departmentsRepo, audit: auditRecorder}
}

func seedDoctorUser(h *harness, status authDomain.UserStatus) *authDomain.User {
	user := &authDomain.User{Name: "Dr. Anil Sharma", Email: "anil.sharma@hms.local", RoleCode: authDomain.RoleDoctor, Status: status}
	h.users.seed(user)
	return user
}

func seedDepartment(h *harness, status deptDomain.Status) *deptDomain.Department {
	dept := &deptDomain.Department{Name: "Cardiology", Code: "CARD", Status: status}
	h.departments.seed(dept)
	return dept
}

func validCreateRequest(userID, deptID string) dto.CreateDoctorRequest {
	return dto.CreateDoctorRequest{
		UserID:         userID,
		DepartmentID:   deptID,
		LicenseNo:      "LIC-001",
		Specialization: "Cardiology",
		Fee:            500,
		Availability: []dto.AvailabilitySlotRequest{
			{DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00"},
		},
	}
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected an *apperror.AppError, got %T: %v", err, err)
	}
	return appErr.Code
}

func TestCreateDoctorSnapshotsUserAndDepartment(t *testing.T) {
	h := newHarness()
	user := seedDoctorUser(h, authDomain.UserStatusActive)
	dept := seedDepartment(h, deptDomain.StatusActive)

	profile, err := h.svc.Create(context.Background(), validCreateRequest(user.ID.Hex(), dept.ID.Hex()), audit.Actor{UserID: "admin1"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if profile.Name != user.Name || profile.Email != user.Email {
		t.Errorf("expected doctor to snapshot the linked user's name/email, got name=%s email=%s", profile.Name, profile.Email)
	}
	if profile.DepartmentName != dept.Name {
		t.Errorf("expected doctor to snapshot the department name, got %s", profile.DepartmentName)
	}
	if len(profile.Availability) != 1 {
		t.Errorf("expected 1 availability slot, got %d", len(profile.Availability))
	}

	if actions := h.audit.actions(); len(actions) != 1 || actions[0] != "DOCTOR_CREATED" {
		t.Fatalf("expected a DOCTOR_CREATED audit entry, got %v", actions)
	}
}

func TestCreateDoctorRejectsNonDoctorUser(t *testing.T) {
	h := newHarness()
	nurse := &authDomain.User{Name: "Nurse Priya", Email: "priya@hms.local", RoleCode: authDomain.RoleNurse, Status: authDomain.UserStatusActive}
	h.users.seed(nurse)
	dept := seedDepartment(h, deptDomain.StatusActive)

	_, err := h.svc.Create(context.Background(), validCreateRequest(nurse.ID.Hex(), dept.ID.Hex()), audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error linking a non-DOCTOR user")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreateDoctorRejectsInactiveUser(t *testing.T) {
	h := newHarness()
	user := seedDoctorUser(h, authDomain.UserStatusInactive)
	dept := seedDepartment(h, deptDomain.StatusActive)

	_, err := h.svc.Create(context.Background(), validCreateRequest(user.ID.Hex(), dept.ID.Hex()), audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an inactive user account")
	}
}

func TestCreateDoctorRejectsUnknownDepartment(t *testing.T) {
	h := newHarness()
	user := seedDoctorUser(h, authDomain.UserStatusActive)

	_, err := h.svc.Create(context.Background(), validCreateRequest(user.ID.Hex(), "000000000000000000000000"), audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown department")
	}
}

func TestCreateDoctorRejectsInactiveDepartment(t *testing.T) {
	h := newHarness()
	user := seedDoctorUser(h, authDomain.UserStatusActive)
	dept := seedDepartment(h, deptDomain.StatusInactive)

	_, err := h.svc.Create(context.Background(), validCreateRequest(user.ID.Hex(), dept.ID.Hex()), audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an inactive department")
	}
}

func TestCreateDoctorRejectsDuplicateUserLink(t *testing.T) {
	h := newHarness()
	user := seedDoctorUser(h, authDomain.UserStatusActive)
	dept := seedDepartment(h, deptDomain.StatusActive)

	if _, err := h.svc.Create(context.Background(), validCreateRequest(user.ID.Hex(), dept.ID.Hex()), audit.Actor{}); err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}

	req := validCreateRequest(user.ID.Hex(), dept.ID.Hex())
	req.LicenseNo = "LIC-002"
	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error linking a user who already has a doctor profile")
	}
	if code := appErrCode(t, err); code != "DOCTOR_PROFILE_EXISTS" {
		t.Fatalf("expected DOCTOR_PROFILE_EXISTS, got %s", code)
	}
}

func TestCreateDoctorRejectsDuplicateLicense(t *testing.T) {
	h := newHarness()
	userA := seedDoctorUser(h, authDomain.UserStatusActive)
	userB := &authDomain.User{Name: "Dr. Neha Verma", Email: "neha.verma@hms.local", RoleCode: authDomain.RoleDoctor, Status: authDomain.UserStatusActive}
	h.users.seed(userB)
	dept := seedDepartment(h, deptDomain.StatusActive)

	if _, err := h.svc.Create(context.Background(), validCreateRequest(userA.ID.Hex(), dept.ID.Hex()), audit.Actor{}); err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}

	_, err := h.svc.Create(context.Background(), validCreateRequest(userB.ID.Hex(), dept.ID.Hex()), audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for a duplicate license number")
	}
	if code := appErrCode(t, err); code != "LICENSE_TAKEN" {
		t.Fatalf("expected LICENSE_TAKEN, got %s", code)
	}
}

func TestCreateDoctorRejectsInvalidAvailability(t *testing.T) {
	h := newHarness()
	user := seedDoctorUser(h, authDomain.UserStatusActive)
	dept := seedDepartment(h, deptDomain.StatusActive)

	req := validCreateRequest(user.ID.Hex(), dept.ID.Hex())
	req.Availability = []dto.AvailabilitySlotRequest{{DayOfWeek: 1, StartTime: "17:00", EndTime: "09:00"}}

	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error when startTime is after endTime")
	}
}

func TestUpdateDoctorReSnapshotsDepartmentName(t *testing.T) {
	h := newHarness()
	user := seedDoctorUser(h, authDomain.UserStatusActive)
	dept := seedDepartment(h, deptDomain.StatusActive)

	created, err := h.svc.Create(context.Background(), validCreateRequest(user.ID.Hex(), dept.ID.Hex()), audit.Actor{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	newDept := &deptDomain.Department{Name: "Pediatrics", Code: "PED", Status: deptDomain.StatusActive}
	h.departments.seed(newDept)
	newDeptID := newDept.ID.Hex()

	updated, err := h.svc.Update(context.Background(), created.ID, dto.UpdateDoctorRequest{DepartmentID: &newDeptID}, audit.Actor{})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if updated.DepartmentName != "Pediatrics" {
		t.Errorf("expected department name to be re-snapshotted to Pediatrics, got %s", updated.DepartmentName)
	}

	if actions := h.audit.actions(); len(actions) != 2 || actions[1] != "DOCTOR_UPDATED" {
		t.Fatalf("expected a DOCTOR_UPDATED audit entry, got %v", actions)
	}
}

func TestUpdateDoctorRejectsEmptyBody(t *testing.T) {
	h := newHarness()
	user := seedDoctorUser(h, authDomain.UserStatusActive)
	dept := seedDepartment(h, deptDomain.StatusActive)
	created, _ := h.svc.Create(context.Background(), validCreateRequest(user.ID.Hex(), dept.ID.Hex()), audit.Actor{})

	_, err := h.svc.Update(context.Background(), created.ID, dto.UpdateDoctorRequest{}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an update with no fields")
	}
}

func TestDeleteDoctorSoftDeletes(t *testing.T) {
	h := newHarness()
	user := seedDoctorUser(h, authDomain.UserStatusActive)
	dept := seedDepartment(h, deptDomain.StatusActive)
	created, _ := h.svc.Create(context.Background(), validCreateRequest(user.ID.Hex(), dept.ID.Hex()), audit.Actor{})

	if err := h.svc.Delete(context.Background(), created.ID, audit.Actor{}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if _, err := h.svc.GetByID(context.Background(), created.ID); err == nil {
		t.Fatalf("expected a deleted doctor to no longer be retrievable")
	}
}

func TestScheduleReturnsAvailability(t *testing.T) {
	h := newHarness()
	user := seedDoctorUser(h, authDomain.UserStatusActive)
	dept := seedDepartment(h, deptDomain.StatusActive)
	created, _ := h.svc.Create(context.Background(), validCreateRequest(user.ID.Hex(), dept.ID.Hex()), audit.Actor{})

	schedule, err := h.svc.Schedule(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Schedule returned error: %v", err)
	}
	if len(schedule.Availability) != 1 || schedule.Availability[0].StartTime != "09:00" {
		t.Fatalf("unexpected schedule: %+v", schedule)
	}
}

func TestSearchFiltersByDepartment(t *testing.T) {
	h := newHarness()
	deptA := seedDepartment(h, deptDomain.StatusActive)
	deptB := &deptDomain.Department{Name: "Pediatrics", Code: "PED", Status: deptDomain.StatusActive}
	h.departments.seed(deptB)

	userA := seedDoctorUser(h, authDomain.UserStatusActive)
	reqA := validCreateRequest(userA.ID.Hex(), deptA.ID.Hex())
	if _, err := h.svc.Create(context.Background(), reqA, audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	userB := &authDomain.User{Name: "Dr. Rahul Mehta", Email: "rahul.mehta@hms.local", RoleCode: authDomain.RoleDoctor, Status: authDomain.UserStatusActive}
	h.users.seed(userB)
	reqB := validCreateRequest(userB.ID.Hex(), deptB.ID.Hex())
	reqB.LicenseNo = "LIC-002"
	if _, err := h.svc.Create(context.Background(), reqB, audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	result, err := h.svc.Search(context.Background(), dto.SearchDoctorsQuery{DepartmentID: deptA.ID.Hex()})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected 1 doctor in department A, got %d", result.Total)
	}
}

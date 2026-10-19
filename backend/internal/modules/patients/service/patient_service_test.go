package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/modules/patients/dto"
)

type harness struct {
	svc      PatientService
	patients *fakePatientRepository
	audit    *fakeAuditRecorder
}

func newHarness() *harness {
	patients := newFakePatientRepository()
	auditRecorder := &fakeAuditRecorder{}

	svc := NewPatientService(Deps{
		Patients:  patients,
		Sequences: newFakeSequenceGenerator(),
		Audit:     auditRecorder,
		Logger:    zap.NewNop(),
	})

	return &harness{svc: svc, patients: patients, audit: auditRecorder}
}

func validCreateRequest() dto.CreatePatientRequest {
	return dto.CreatePatientRequest{
		Name:   "Rohan Kumar",
		DOB:    "1992-04-15",
		Gender: "M",
		Phone:  "9000000001",
		Type:   "OPD",
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

func TestCreatePatientGeneratesSequentialPatientCodes(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	first, err := h.svc.Create(ctx, validCreateRequest(), audit.Actor{UserID: "", Role: "RECEPTIONIST"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	second, err := h.svc.Create(ctx, validCreateRequest(), audit.Actor{Role: "RECEPTIONIST"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if first.PatientCode != "PAT-10001" {
		t.Errorf("expected first patient code PAT-10001, got %s", first.PatientCode)
	}
	if second.PatientCode != "PAT-10002" {
		t.Errorf("expected second patient code PAT-10002, got %s", second.PatientCode)
	}
}

func TestCreatePatientRejectsFutureDOB(t *testing.T) {
	h := newHarness()
	req := validCreateRequest()
	req.DOB = time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for a future date of birth")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreatePatientRejectsImplausibleDOB(t *testing.T) {
	h := newHarness()
	req := validCreateRequest()
	req.DOB = "1850-01-01"

	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an implausible date of birth")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreatePatientRejectsMalformedDOB(t *testing.T) {
	h := newHarness()
	req := validCreateRequest()
	req.DOB = "15-04-1992"

	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for a malformed date of birth")
	}
}

func TestCreatePatientRejectsInvalidPhone(t *testing.T) {
	h := newHarness()
	req := validCreateRequest()
	req.Phone = "call-me"

	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an invalid phone number")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreatePatientRejectsInvalidEmergencyContactPhone(t *testing.T) {
	h := newHarness()
	req := validCreateRequest()
	req.EmergencyContact = &dto.EmergencyContactRequest{Name: "Next of Kin", Phone: "not-a-phone"}

	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an invalid emergency contact phone number")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreatePatientAcceptsValidEmergencyContactPhone(t *testing.T) {
	h := newHarness()
	req := validCreateRequest()
	req.EmergencyContact = &dto.EmergencyContactRequest{Name: "Next of Kin", Phone: "9123456780"}

	profile, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if profile.EmergencyContact == nil || profile.EmergencyContact.Phone != "9123456780" {
		t.Fatalf("expected the emergency contact to be saved, got %+v", profile.EmergencyContact)
	}
}

func TestUpdatePatientRejectsInvalidEmergencyContactPhone(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	created, _ := h.svc.Create(ctx, validCreateRequest(), audit.Actor{})

	badContact := &dto.EmergencyContactRequest{Name: "Next of Kin", Phone: "xx"}
	_, err := h.svc.Update(ctx, created.ID, dto.UpdatePatientRequest{EmergencyContact: badContact}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an invalid emergency contact phone number")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreatePatientRecordsAuditEntry(t *testing.T) {
	h := newHarness()

	profile, err := h.svc.Create(context.Background(), validCreateRequest(), audit.Actor{UserID: "u1", Role: "RECEPTIONIST"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	actions := h.audit.actions()
	if len(actions) != 1 || actions[0] != "PATIENT_CREATED" {
		t.Fatalf("expected exactly one PATIENT_CREATED audit entry, got %v", actions)
	}
	if h.audit.entries[0].EntityID != profile.ID {
		t.Fatalf("expected audit entry entityId to match the created patient")
	}
}

func TestUpdatePatientAppliesPartialChangesAndAudits(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	created, err := h.svc.Create(ctx, validCreateRequest(), audit.Actor{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	newPhone := "9111111111"
	updated, err := h.svc.Update(ctx, created.ID, dto.UpdatePatientRequest{Phone: &newPhone}, audit.Actor{UserID: "u1"})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if updated.Phone != newPhone {
		t.Errorf("expected phone to be updated to %s, got %s", newPhone, updated.Phone)
	}
	if updated.Name != created.Name {
		t.Errorf("expected name to remain unchanged, got %s", updated.Name)
	}

	actions := h.audit.actions()
	if len(actions) != 2 || actions[1] != "PATIENT_UPDATED" {
		t.Fatalf("expected a PATIENT_UPDATED audit entry to follow PATIENT_CREATED, got %v", actions)
	}
}

func TestUpdatePatientRejectsEmptyBody(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	created, _ := h.svc.Create(ctx, validCreateRequest(), audit.Actor{})

	_, err := h.svc.Update(ctx, created.ID, dto.UpdatePatientRequest{}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an update with no fields")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestUpdatePatientRejectsInvalidPhone(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	created, _ := h.svc.Create(ctx, validCreateRequest(), audit.Actor{})

	badPhone := "abc"
	_, err := h.svc.Update(ctx, created.ID, dto.UpdatePatientRequest{Phone: &badPhone}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an invalid phone number")
	}
}

func TestUpdatePatientReturnsNotFoundForUnknownID(t *testing.T) {
	h := newHarness()
	name := "Someone"

	_, err := h.svc.Update(context.Background(), "000000000000000000000000", dto.UpdatePatientRequest{Name: &name}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown patient id")
	}
	if code := appErrCode(t, err); code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %s", code)
	}
}

func TestDeletePatientSoftDeletesAndHidesFromLookups(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	created, _ := h.svc.Create(ctx, validCreateRequest(), audit.Actor{})

	if err := h.svc.Delete(ctx, created.ID, audit.Actor{UserID: "admin1"}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if _, err := h.svc.GetByID(ctx, created.ID); err == nil {
		t.Fatalf("expected a deleted patient to no longer be retrievable")
	}

	actions := h.audit.actions()
	if len(actions) != 2 || actions[1] != "PATIENT_DELETED" {
		t.Fatalf("expected a PATIENT_DELETED audit entry, got %v", actions)
	}
}

func TestDeletePatientTwiceReturnsNotFound(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	created, _ := h.svc.Create(ctx, validCreateRequest(), audit.Actor{})
	if err := h.svc.Delete(ctx, created.ID, audit.Actor{}); err != nil {
		t.Fatalf("first Delete returned error: %v", err)
	}

	err := h.svc.Delete(ctx, created.ID, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error deleting an already-deleted patient")
	}
	if code := appErrCode(t, err); code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %s", code)
	}
}

func TestSearchFiltersByTypeAndPaginates(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		req := validCreateRequest()
		req.Name = fmt.Sprintf("OPD Patient %d", i)
		req.Type = "OPD"
		if _, err := h.svc.Create(ctx, req, audit.Actor{}); err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
	}
	ipdReq := validCreateRequest()
	ipdReq.Name = "IPD Patient"
	ipdReq.Type = "IPD"
	if _, err := h.svc.Create(ctx, ipdReq, audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	result, err := h.svc.Search(ctx, dto.SearchPatientsQuery{Type: "OPD", Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("expected 3 total OPD matches, got %d", result.Total)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected page size of 2 items, got %d", len(result.Items))
	}
	if result.TotalPages != 2 {
		t.Errorf("expected 2 total pages, got %d", result.TotalPages)
	}
}

func TestSearchExcludesDeletedPatients(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	created, _ := h.svc.Create(ctx, validCreateRequest(), audit.Actor{})
	if err := h.svc.Delete(ctx, created.ID, audit.Actor{}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	result, err := h.svc.Search(ctx, dto.SearchPatientsQuery{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("expected deleted patients to be excluded from search, got total=%d", result.Total)
	}
}

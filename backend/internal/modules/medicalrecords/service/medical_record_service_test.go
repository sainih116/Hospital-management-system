package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	doctorsDomain "hms-backend/internal/modules/doctors/domain"
	patientsDomain "hms-backend/internal/modules/patients/domain"

	"hms-backend/internal/modules/medicalrecords/dto"
)

type harness struct {
	svc      MedicalRecordService
	records  *fakeRecordRepository
	patients *fakePatientRepository
	doctors  *fakeDoctorRepository
	audit    *fakeAuditRecorder
}

func newHarness() *harness {
	records := newFakeRecordRepository()
	patients := newFakePatientRepository()
	doctors := newFakeDoctorRepository()
	auditRecorder := &fakeAuditRecorder{}

	svc := NewMedicalRecordService(Deps{
		Records: records, Patients: patients, Doctors: doctors, Audit: auditRecorder, Logger: zap.NewNop(),
	})

	return &harness{svc: svc, records: records, patients: patients, doctors: doctors, audit: auditRecorder}
}

func seedPatient(h *harness) *patientsDomain.Patient {
	p := &patientsDomain.Patient{Name: "Rohan Kumar", PatientCode: "PAT-10001", Status: patientsDomain.StatusActive}
	h.patients.seed(p)
	return p
}

func seedDoctor(h *harness) *doctorsDomain.Doctor {
	d := &doctorsDomain.Doctor{Name: "Dr. Anil Sharma", Specialization: "Cardiology", Status: doctorsDomain.StatusActive}
	h.doctors.seed(d)
	return d
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected an *apperror.AppError, got %T: %v", err, err)
	}
	return appErr.Code
}

func validCreateRequest(patient *patientsDomain.Patient, doctor *doctorsDomain.Doctor) dto.CreateMedicalRecordRequest {
	return dto.CreateMedicalRecordRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(),
		Complaint: "Chest pain", Diagnosis: "Suspected angina", Treatment: "Aspirin, follow-up ECG",
	}
}

func TestCreateMedicalRecordSnapshotsNamesAndAudits(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)

	resp, err := h.svc.Create(context.Background(), validCreateRequest(patient, doctor), audit.Actor{UserID: "u1", Role: "DOCTOR"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if resp.PatientName != patient.Name || resp.DoctorName != doctor.Name {
		t.Errorf("expected snapshotted names, got patient=%s doctor=%s", resp.PatientName, resp.DoctorName)
	}
	if resp.Diagnosis != "Suspected angina" {
		t.Errorf("expected diagnosis to be preserved, got %s", resp.Diagnosis)
	}
	if actions := h.audit.actions(); len(actions) != 1 || actions[0] != "MEDICAL_RECORD_CREATED" {
		t.Fatalf("expected a MEDICAL_RECORD_CREATED audit entry, got %v", actions)
	}
}

func TestCreateMedicalRecordRejectsUnknownPatient(t *testing.T) {
	h := newHarness()
	doctor := seedDoctor(h)

	req := dto.CreateMedicalRecordRequest{PatientID: "000000000000000000000000", DoctorID: doctor.ID.Hex(), Diagnosis: "x"}
	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown patient")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreateMedicalRecordRejectsUnknownDoctor(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)

	req := dto.CreateMedicalRecordRequest{PatientID: patient.ID.Hex(), DoctorID: "000000000000000000000000", Diagnosis: "x"}
	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown doctor")
	}
}

func TestUpdateMedicalRecordAppliesPartialChangesAndAudits(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)

	created, err := h.svc.Create(context.Background(), validCreateRequest(patient, doctor), audit.Actor{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	newDiagnosis := "Confirmed angina"
	updated, err := h.svc.Update(context.Background(), created.ID, dto.UpdateMedicalRecordRequest{Diagnosis: &newDiagnosis}, audit.Actor{})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Diagnosis != newDiagnosis {
		t.Errorf("expected diagnosis to be updated, got %s", updated.Diagnosis)
	}
	if updated.Complaint != created.Complaint {
		t.Errorf("expected complaint to remain unchanged")
	}

	if actions := h.audit.actions(); len(actions) != 2 || actions[1] != "MEDICAL_RECORD_AMENDED" {
		t.Fatalf("expected a MEDICAL_RECORD_AMENDED audit entry, got %v", actions)
	}
}

func TestUpdateMedicalRecordRejectsEmptyBody(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	created, _ := h.svc.Create(context.Background(), validCreateRequest(patient, doctor), audit.Actor{})

	_, err := h.svc.Update(context.Background(), created.ID, dto.UpdateMedicalRecordRequest{}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an update with no fields")
	}
}

func TestUpdateMedicalRecordReturnsNotFoundForUnknownID(t *testing.T) {
	h := newHarness()
	diagnosis := "x"

	_, err := h.svc.Update(context.Background(), "000000000000000000000000", dto.UpdateMedicalRecordRequest{Diagnosis: &diagnosis}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown record id")
	}
	if code := appErrCode(t, err); code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %s", code)
	}
}

func TestSearchFiltersByPatientForConsultationHistory(t *testing.T) {
	h := newHarness()
	patientA := seedPatient(h)
	patientB := &patientsDomain.Patient{Name: "Neha Singh", PatientCode: "PAT-10002", Status: patientsDomain.StatusActive}
	h.patients.seed(patientB)
	doctor := seedDoctor(h)

	if _, err := h.svc.Create(context.Background(), validCreateRequest(patientA, doctor), audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := h.svc.Create(context.Background(), validCreateRequest(patientA, doctor), audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := h.svc.Create(context.Background(), validCreateRequest(patientB, doctor), audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	result, err := h.svc.Search(context.Background(), dto.SearchMedicalRecordsQuery{PatientID: patientA.ID.Hex()})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected 2 records for patientA's consultation history, got %d", result.Total)
	}
}

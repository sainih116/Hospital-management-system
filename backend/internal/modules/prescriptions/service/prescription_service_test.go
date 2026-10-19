package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	doctorsDomain "hms-backend/internal/modules/doctors/domain"
	medicalRecordsDomain "hms-backend/internal/modules/medicalrecords/domain"
	patientsDomain "hms-backend/internal/modules/patients/domain"

	"hms-backend/internal/modules/prescriptions/dto"
)

type harness struct {
	svc           PrescriptionService
	prescriptions *fakePrescriptionRepository
	patients      *fakePatientRepository
	doctors       *fakeDoctorRepository
	records       *fakeMedicalRecordRepository
	audit         *fakeAuditRecorder
}

func newHarness() *harness {
	prescriptions := newFakePrescriptionRepository()
	patients := newFakePatientRepository()
	doctors := newFakeDoctorRepository()
	records := newFakeMedicalRecordRepository()
	auditRecorder := &fakeAuditRecorder{}

	svc := NewPrescriptionService(Deps{
		Prescriptions: prescriptions, Patients: patients, Doctors: doctors,
		MedicalRecord: records, Audit: auditRecorder, Logger: zap.NewNop(),
	})

	return &harness{svc: svc, prescriptions: prescriptions, patients: patients, doctors: doctors, records: records, audit: auditRecorder}
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

func validCreateRequest(patient *patientsDomain.Patient, doctor *doctorsDomain.Doctor) dto.CreatePrescriptionRequest {
	return dto.CreatePrescriptionRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(),
		Items: []dto.ItemRequest{
			{MedicineName: "Paracetamol 500mg", Dose: "1 tablet", Frequency: "Twice daily", Duration: "5 days"},
		},
	}
}

func TestCreatePrescriptionSucceeds(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)

	resp, err := h.svc.Create(context.Background(), validCreateRequest(patient, doctor), audit.Actor{UserID: "u1", Role: "DOCTOR"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if resp.Status != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %s", resp.Status)
	}
	if len(resp.Items) != 1 || resp.Items[0].MedicineName != "Paracetamol 500mg" {
		t.Errorf("expected the medicine item to be preserved, got %+v", resp.Items)
	}
	if resp.PatientName != patient.Name || resp.DoctorName != doctor.Name {
		t.Errorf("expected snapshotted names, got patient=%s doctor=%s", resp.PatientName, resp.DoctorName)
	}
	if actions := h.audit.actions(); len(actions) != 1 || actions[0] != "PRESCRIPTION_CREATED" {
		t.Fatalf("expected a PRESCRIPTION_CREATED audit entry, got %v", actions)
	}
}

func TestCreatePrescriptionLinksToMedicalRecord(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	record := &medicalRecordsDomain.MedicalRecord{PatientID: patient.ID, DoctorID: doctor.ID, Diagnosis: "Flu"}
	h.records.seed(record)

	req := validCreateRequest(patient, doctor)
	req.MedicalRecordID = record.ID.Hex()

	resp, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if resp.MedicalRecordID != record.ID.Hex() {
		t.Errorf("expected medicalRecordId to be linked, got %s", resp.MedicalRecordID)
	}
}

func TestCreatePrescriptionRejectsUnknownMedicalRecord(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)

	req := validCreateRequest(patient, doctor)
	req.MedicalRecordID = "000000000000000000000000"

	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown medical record link")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreatePrescriptionRejectsUnknownPatient(t *testing.T) {
	h := newHarness()
	doctor := seedDoctor(h)

	req := dto.CreatePrescriptionRequest{
		PatientID: "000000000000000000000000", DoctorID: doctor.ID.Hex(),
		Items: []dto.ItemRequest{{MedicineName: "X", Dose: "1", Frequency: "1", Duration: "1"}},
	}
	_, err := h.svc.Create(context.Background(), req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown patient")
	}
}

func TestChangeStatusCancelsActivePrescription(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	created, err := h.svc.Create(context.Background(), validCreateRequest(patient, doctor), audit.Actor{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	cancelled, err := h.svc.ChangeStatus(context.Background(), created.ID, dto.ChangeStatusRequest{Status: "CANCELLED"}, audit.Actor{})
	if err != nil {
		t.Fatalf("ChangeStatus returned error: %v", err)
	}
	if cancelled.Status != "CANCELLED" {
		t.Fatalf("expected status CANCELLED, got %s", cancelled.Status)
	}

	actions := h.audit.actions()
	if len(actions) != 2 || actions[1] != "PRESCRIPTION_CANCELLED" {
		t.Fatalf("expected a PRESCRIPTION_CANCELLED audit entry, got %v", actions)
	}
}

func TestChangeStatusRejectsCancellingAlreadyCancelledPrescription(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	created, _ := h.svc.Create(context.Background(), validCreateRequest(patient, doctor), audit.Actor{})

	if _, err := h.svc.ChangeStatus(context.Background(), created.ID, dto.ChangeStatusRequest{Status: "CANCELLED"}, audit.Actor{}); err != nil {
		t.Fatalf("first ChangeStatus returned error: %v", err)
	}

	_, err := h.svc.ChangeStatus(context.Background(), created.ID, dto.ChangeStatusRequest{Status: "CANCELLED"}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error cancelling an already-cancelled prescription")
	}
	if code := appErrCode(t, err); code != "INVALID_STATUS_TRANSITION" {
		t.Fatalf("expected INVALID_STATUS_TRANSITION, got %s", code)
	}
}

func TestSearchFiltersByStatus(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)

	first, err := h.svc.Create(context.Background(), validCreateRequest(patient, doctor), audit.Actor{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := h.svc.Create(context.Background(), validCreateRequest(patient, doctor), audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := h.svc.ChangeStatus(context.Background(), first.ID, dto.ChangeStatusRequest{Status: "CANCELLED"}, audit.Actor{}); err != nil {
		t.Fatalf("ChangeStatus returned error: %v", err)
	}

	result, err := h.svc.Search(context.Background(), dto.SearchPrescriptionsQuery{Status: "ACTIVE"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected 1 active prescription, got %d", result.Total)
	}
}

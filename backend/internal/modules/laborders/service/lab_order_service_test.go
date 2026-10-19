package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	authDomain "hms-backend/internal/modules/auth/domain"
	doctorsDomain "hms-backend/internal/modules/doctors/domain"
	labTestsDomain "hms-backend/internal/modules/labtests/domain"
	patientsDomain "hms-backend/internal/modules/patients/domain"

	"hms-backend/internal/modules/laborders/dto"
)

type harness struct {
	svc      LabOrderService
	orders   *fakeOrderRepository
	results  *fakeResultRepository
	tests    *fakeLabTestRepository
	patients *fakePatientRepository
	doctors  *fakeDoctorRepository
	users    *fakeUserRepository
	audit    *fakeAuditRecorder
}

func newHarness() *harness {
	orders := newFakeOrderRepository()
	results := newFakeResultRepository()
	tests := newFakeLabTestRepository()
	patients := newFakePatientRepository()
	doctors := newFakeDoctorRepository()
	users := newFakeUserRepository()
	auditRecorder := &fakeAuditRecorder{}

	svc := NewLabOrderService(Deps{
		Orders: orders, Results: results, Tests: tests, Patients: patients, Doctors: doctors,
		Users: users, Sequences: newFakeSequenceGenerator(), Audit: auditRecorder, Logger: zap.NewNop(),
	})

	return &harness{svc: svc, orders: orders, results: results, tests: tests, patients: patients, doctors: doctors, users: users, audit: auditRecorder}
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

func seedLabTest(h *harness) *labTestsDomain.LabTest {
	t := &labTestsDomain.LabTest{Code: "CBC", Name: "Complete Blood Count", Unit: "cells/mcL", NormalRange: "4500-11000", Cost: 250, Status: labTestsDomain.StatusActive}
	h.tests.seed(t)
	return t
}

func seedLabTechUser(h *harness) *authDomain.User {
	u := &authDomain.User{Name: "Suresh Kumar", Email: "suresh@hms.local", RoleCode: authDomain.RoleLabTech, Status: authDomain.UserStatusActive}
	h.users.seed(u)
	return u
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected an *apperror.AppError, got %T: %v", err, err)
	}
	return appErr.Code
}

func createOrder(t *testing.T, h *harness, patient *patientsDomain.Patient, doctor *doctorsDomain.Doctor, test *labTestsDomain.LabTest) *dto.LabOrderResponse {
	t.Helper()
	order, err := h.svc.CreateOrder(context.Background(), dto.CreateLabOrderRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), TestID: test.ID.Hex(),
	}, audit.Actor{UserID: "doc1", Role: authDomain.RoleDoctor})
	if err != nil {
		t.Fatalf("CreateOrder returned error: %v", err)
	}
	return order
}

func TestCreateOrderGeneratesSampleCodeAndSnapshots(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	test := seedLabTest(h)

	order := createOrder(t, h, patient, doctor, test)

	if order.SampleCode != "SMP-00001" {
		t.Errorf("expected sample code SMP-00001, got %s", order.SampleCode)
	}
	if order.PatientName != patient.Name || order.DoctorName != doctor.Name || order.TestName != test.Name {
		t.Errorf("expected snapshotted names, got %+v", order)
	}
	if order.Status != "ORDERED" {
		t.Errorf("expected status ORDERED, got %s", order.Status)
	}
	if actions := h.audit.actions(); len(actions) != 1 || actions[0] != "LAB_ORDER_CREATED" {
		t.Fatalf("expected a LAB_ORDER_CREATED audit entry, got %v", actions)
	}
}

func TestCreateOrderRejectsUnknownTest(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)

	_, err := h.svc.CreateOrder(context.Background(), dto.CreateLabOrderRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), TestID: "000000000000000000000000",
	}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown test")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestEnterResultMovesOrderToAwaitingVerification(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	test := seedLabTest(h)
	labTech := seedLabTechUser(h)
	order := createOrder(t, h, patient, doctor, test)

	updated, err := h.svc.EnterResult(context.Background(), order.ID, dto.EnterResultRequest{
		ResultValue: "7200 cells/mcL", Flag: "NORMAL",
	}, audit.Actor{UserID: labTech.ID.Hex(), Role: authDomain.RoleLabTech})
	if err != nil {
		t.Fatalf("EnterResult returned error: %v", err)
	}

	if updated.Status != "AWAITING_VERIFICATION" {
		t.Errorf("expected status AWAITING_VERIFICATION, got %s", updated.Status)
	}
	if updated.Result == nil || updated.Result.ResultValue != "7200 cells/mcL" {
		t.Fatalf("expected the result to be attached to the response, got %+v", updated.Result)
	}
	if updated.Result.EnteredByName != labTech.Name {
		t.Errorf("expected enteredByName to resolve to %s, got %s", labTech.Name, updated.Result.EnteredByName)
	}
}

func TestEnterResultRejectsDuplicateEntry(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	test := seedLabTest(h)
	order := createOrder(t, h, patient, doctor, test)

	req := dto.EnterResultRequest{ResultValue: "x", Flag: "NORMAL"}
	if _, err := h.svc.EnterResult(context.Background(), order.ID, req, audit.Actor{}); err != nil {
		t.Fatalf("first EnterResult returned error: %v", err)
	}

	_, err := h.svc.EnterResult(context.Background(), order.ID, req, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error entering a second result for the same order")
	}
	if code := appErrCode(t, err); code != "INVALID_STATUS_TRANSITION" {
		t.Fatalf("expected INVALID_STATUS_TRANSITION, got %s", code)
	}
}

func TestVerifyResultCompletesOrder(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	test := seedLabTest(h)
	labTech := seedLabTechUser(h)
	order := createOrder(t, h, patient, doctor, test)

	entered, err := h.svc.EnterResult(context.Background(), order.ID, dto.EnterResultRequest{ResultValue: "x", Flag: "NORMAL"}, audit.Actor{})
	if err != nil {
		t.Fatalf("EnterResult returned error: %v", err)
	}

	verified, err := h.svc.VerifyResult(context.Background(), entered.Result.ID, audit.Actor{UserID: labTech.ID.Hex(), Role: authDomain.RoleLabTech})
	if err != nil {
		t.Fatalf("VerifyResult returned error: %v", err)
	}
	if verified.Status != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got %s", verified.Status)
	}
	if verified.Result.VerifiedByName != labTech.Name {
		t.Errorf("expected verifiedByName to resolve to %s, got %s", labTech.Name, verified.Result.VerifiedByName)
	}
	if verified.Result.VerifiedAt == nil {
		t.Fatalf("expected verifiedAt to be set")
	}
}

func TestVerifyResultRejectsDoubleVerification(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	test := seedLabTest(h)
	order := createOrder(t, h, patient, doctor, test)

	entered, err := h.svc.EnterResult(context.Background(), order.ID, dto.EnterResultRequest{ResultValue: "x", Flag: "NORMAL"}, audit.Actor{})
	if err != nil {
		t.Fatalf("EnterResult returned error: %v", err)
	}
	if _, err := h.svc.VerifyResult(context.Background(), entered.Result.ID, audit.Actor{}); err != nil {
		t.Fatalf("first VerifyResult returned error: %v", err)
	}

	_, err = h.svc.VerifyResult(context.Background(), entered.Result.ID, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error re-verifying an already-verified result")
	}
	if code := appErrCode(t, err); code != "ALREADY_VERIFIED" {
		t.Fatalf("expected ALREADY_VERIFIED, got %s", code)
	}
}

func TestCancelOrderOnlyAllowedBeforeResultEntry(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	test := seedLabTest(h)
	order := createOrder(t, h, patient, doctor, test)

	if _, err := h.svc.EnterResult(context.Background(), order.ID, dto.EnterResultRequest{ResultValue: "x", Flag: "NORMAL"}, audit.Actor{}); err != nil {
		t.Fatalf("EnterResult returned error: %v", err)
	}

	_, err := h.svc.CancelOrder(context.Background(), order.ID, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error cancelling an order that already has a result")
	}
	if code := appErrCode(t, err); code != "INVALID_STATUS_TRANSITION" {
		t.Fatalf("expected INVALID_STATUS_TRANSITION, got %s", code)
	}
}

func TestGetReportRequiresCompletion(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h)
	test := seedLabTest(h)
	order := createOrder(t, h, patient, doctor, test)

	_, err := h.svc.GetReport(context.Background(), order.ID)
	if err == nil {
		t.Fatalf("expected an error requesting a report for an unverified order")
	}
	if code := appErrCode(t, err); code != "REPORT_NOT_READY" {
		t.Fatalf("expected REPORT_NOT_READY, got %s", code)
	}

	entered, err := h.svc.EnterResult(context.Background(), order.ID, dto.EnterResultRequest{ResultValue: "x", Flag: "NORMAL"}, audit.Actor{})
	if err != nil {
		t.Fatalf("EnterResult returned error: %v", err)
	}
	if _, err := h.svc.VerifyResult(context.Background(), entered.Result.ID, audit.Actor{}); err != nil {
		t.Fatalf("VerifyResult returned error: %v", err)
	}

	report, err := h.svc.GetReport(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("expected the report to be available after verification, got error: %v", err)
	}
	if report.Order.Status != "COMPLETED" {
		t.Errorf("expected report order status COMPLETED, got %s", report.Order.Status)
	}
}

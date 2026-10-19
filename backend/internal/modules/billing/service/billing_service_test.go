package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	patientsDomain "hms-backend/internal/modules/patients/domain"

	"hms-backend/internal/modules/billing/dto"
)

type testFixture struct {
	bills     *fakeBillRepository
	payments  *fakePaymentRepository
	patients  *fakePatientRepository
	users     *fakeUserRepository
	sequences *fakeSequenceGenerator
	audit     *fakeAuditRecorder
	svc       BillingService
}

func newTestFixture() *testFixture {
	f := &testFixture{
		bills:     newFakeBillRepository(),
		payments:  newFakePaymentRepository(),
		patients:  newFakePatientRepository(),
		users:     newFakeUserRepository(),
		sequences: newFakeSequenceGenerator(),
		audit:     &fakeAuditRecorder{},
	}
	f.svc = NewBillingService(Deps{
		Bills: f.bills, Payments: f.payments, Patients: f.patients,
		Users: f.users, Sequences: f.sequences, Audit: f.audit, Logger: zap.NewNop(),
	})
	return f
}

func testActor() audit.Actor {
	return audit.Actor{UserID: "", Email: "admin@hms.local", Role: "ADMIN"}
}

func newBillRequest(patientID string) dto.CreateBillRequest {
	return dto.CreateBillRequest{
		PatientID: patientID,
		Items: []dto.CreateBillItemRequest{
			{Description: "Consultation", Category: "CONSULTATION", Quantity: 1, UnitPrice: 500},
			{Description: "CBC Test", Category: "LAB", Quantity: 1, UnitPrice: 250},
		},
		Discount: 50,
		Tax:      0,
	}
}

func TestCreateBillComputesTotals(t *testing.T) {
	f := newTestFixture()
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Jane Doe"})

	bill, err := f.svc.CreateBill(context.Background(), newBillRequest(patient.ID.Hex()), testActor())
	if err != nil {
		t.Fatalf("CreateBill returned an unexpected error: %v", err)
	}

	if bill.BillNo != "INV-00001" {
		t.Errorf("expected bill number INV-00001, got %s", bill.BillNo)
	}
	if bill.Subtotal != 750 {
		t.Errorf("expected subtotal 750, got %v", bill.Subtotal)
	}
	if bill.TotalAmount != 700 {
		t.Errorf("expected totalAmount 700 (750 subtotal - 50 discount), got %v", bill.TotalAmount)
	}
	if bill.Balance != 700 {
		t.Errorf("expected balance 700 before any payment, got %v", bill.Balance)
	}
	if bill.Status != "UNPAID" {
		t.Errorf("expected status UNPAID, got %s", bill.Status)
	}

	actions := f.audit.actions()
	if len(actions) != 1 || actions[0] != "BILL_CREATED" {
		t.Errorf("expected a single BILL_CREATED audit entry, got %+v", actions)
	}
}

func TestRecordPaymentPartialThenFull(t *testing.T) {
	f := newTestFixture()
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Jane Doe"})
	bill, err := f.svc.CreateBill(context.Background(), newBillRequest(patient.ID.Hex()), testActor())
	if err != nil {
		t.Fatalf("CreateBill returned an unexpected error: %v", err)
	}

	partial, err := f.svc.RecordPayment(context.Background(), bill.ID, dto.RecordPaymentRequest{Amount: 300, Method: "CASH"}, testActor())
	if err != nil {
		t.Fatalf("RecordPayment (partial) returned an unexpected error: %v", err)
	}
	if partial.Status != "PARTIALLY_PAID" {
		t.Errorf("expected status PARTIALLY_PAID after a 300/700 payment, got %s", partial.Status)
	}
	if partial.Balance != 400 {
		t.Errorf("expected balance 400 after a 300 payment, got %v", partial.Balance)
	}
	if len(partial.Payments) != 1 {
		t.Fatalf("expected 1 payment recorded, got %d", len(partial.Payments))
	}

	full, err := f.svc.RecordPayment(context.Background(), bill.ID, dto.RecordPaymentRequest{Amount: 400, Method: "CARD"}, testActor())
	if err != nil {
		t.Fatalf("RecordPayment (final) returned an unexpected error: %v", err)
	}
	if full.Status != "PAID" {
		t.Errorf("expected status PAID after the balance is fully settled, got %s", full.Status)
	}
	if full.Balance != 0 {
		t.Errorf("expected balance 0 once fully paid, got %v", full.Balance)
	}
	if len(full.Payments) != 2 {
		t.Fatalf("expected 2 payments recorded across both calls, got %d", len(full.Payments))
	}
}

func TestRecordPaymentRejectsOverpayment(t *testing.T) {
	f := newTestFixture()
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Jane Doe"})
	bill, err := f.svc.CreateBill(context.Background(), newBillRequest(patient.ID.Hex()), testActor())
	if err != nil {
		t.Fatalf("CreateBill returned an unexpected error: %v", err)
	}

	_, err = f.svc.RecordPayment(context.Background(), bill.ID, dto.RecordPaymentRequest{Amount: 1000, Method: "CASH"}, testActor())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "OVERPAYMENT" {
		t.Fatalf("expected an OVERPAYMENT error, got: %v", err)
	}

	reloaded, err := f.svc.GetByID(context.Background(), bill.ID)
	if err != nil {
		t.Fatalf("GetByID returned an unexpected error: %v", err)
	}
	if reloaded.AmountPaid != 0 {
		t.Errorf("expected amountPaid to remain 0 after a rejected overpayment, got %v", reloaded.AmountPaid)
	}
}

func TestRecordPaymentRejectsOnAlreadyPaidBill(t *testing.T) {
	f := newTestFixture()
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Jane Doe"})
	bill, err := f.svc.CreateBill(context.Background(), newBillRequest(patient.ID.Hex()), testActor())
	if err != nil {
		t.Fatalf("CreateBill returned an unexpected error: %v", err)
	}
	if _, err := f.svc.RecordPayment(context.Background(), bill.ID, dto.RecordPaymentRequest{Amount: 700, Method: "CASH"}, testActor()); err != nil {
		t.Fatalf("RecordPayment (full) returned an unexpected error: %v", err)
	}

	_, err = f.svc.RecordPayment(context.Background(), bill.ID, dto.RecordPaymentRequest{Amount: 10, Method: "CASH"}, testActor())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "BILL_ALREADY_PAID" {
		t.Fatalf("expected a BILL_ALREADY_PAID error, got: %v", err)
	}
}

func TestCancelBillRejectsWhenPaymentsExist(t *testing.T) {
	f := newTestFixture()
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Jane Doe"})
	bill, err := f.svc.CreateBill(context.Background(), newBillRequest(patient.ID.Hex()), testActor())
	if err != nil {
		t.Fatalf("CreateBill returned an unexpected error: %v", err)
	}
	if _, err := f.svc.RecordPayment(context.Background(), bill.ID, dto.RecordPaymentRequest{Amount: 100, Method: "CASH"}, testActor()); err != nil {
		t.Fatalf("RecordPayment returned an unexpected error: %v", err)
	}

	_, err = f.svc.CancelBill(context.Background(), bill.ID, testActor())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "BILL_HAS_PAYMENTS" {
		t.Fatalf("expected a BILL_HAS_PAYMENTS error, got: %v", err)
	}
}

func TestCancelBillSucceedsWithNoPayments(t *testing.T) {
	f := newTestFixture()
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Jane Doe"})
	bill, err := f.svc.CreateBill(context.Background(), newBillRequest(patient.ID.Hex()), testActor())
	if err != nil {
		t.Fatalf("CreateBill returned an unexpected error: %v", err)
	}

	cancelled, err := f.svc.CancelBill(context.Background(), bill.ID, testActor())
	if err != nil {
		t.Fatalf("CancelBill returned an unexpected error: %v", err)
	}
	if cancelled.Status != "CANCELLED" {
		t.Errorf("expected status CANCELLED, got %s", cancelled.Status)
	}
}

func TestSearchOutstandingOnlyExcludesPaidAndCancelled(t *testing.T) {
	f := newTestFixture()
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Jane Doe"})

	unpaid, err := f.svc.CreateBill(context.Background(), newBillRequest(patient.ID.Hex()), testActor())
	if err != nil {
		t.Fatalf("CreateBill returned an unexpected error: %v", err)
	}
	paid, err := f.svc.CreateBill(context.Background(), newBillRequest(patient.ID.Hex()), testActor())
	if err != nil {
		t.Fatalf("CreateBill returned an unexpected error: %v", err)
	}
	if _, err := f.svc.RecordPayment(context.Background(), paid.ID, dto.RecordPaymentRequest{Amount: 700, Method: "CASH"}, testActor()); err != nil {
		t.Fatalf("RecordPayment returned an unexpected error: %v", err)
	}

	result, err := f.svc.Search(context.Background(), dto.SearchBillsQuery{OutstandingOnly: true, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search returned an unexpected error: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != unpaid.ID {
		t.Fatalf("expected only the unpaid bill to match outstandingOnly, got %+v", result.Items)
	}
}

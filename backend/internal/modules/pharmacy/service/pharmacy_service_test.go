package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	prescriptionsDomain "hms-backend/internal/modules/prescriptions/domain"

	"hms-backend/internal/modules/pharmacy/domain"
	"hms-backend/internal/modules/pharmacy/dto"
)

type testFixture struct {
	medicines     *fakeMedicineRepository
	batches       *fakeBatchRepository
	movements     *fakeStockMovementRepository
	prescriptions *fakePrescriptionRepository
	users         *fakeUserRepository
	audit         *fakeAuditRecorder
	svc           PharmacyService
}

func newTestFixture() *testFixture {
	f := &testFixture{
		medicines:     newFakeMedicineRepository(),
		batches:       newFakeBatchRepository(),
		movements:     newFakeStockMovementRepository(),
		prescriptions: newFakePrescriptionRepository(),
		users:         newFakeUserRepository(),
		audit:         &fakeAuditRecorder{},
	}
	f.svc = NewPharmacyService(Deps{
		Medicines:     f.medicines,
		Batches:       f.batches,
		Movements:     f.movements,
		Prescriptions: f.prescriptions,
		Users:         f.users,
		Audit:         f.audit,
		Logger:        zap.NewNop(),
	})
	return f
}

func testActor() audit.Actor {
	return audit.Actor{UserID: "", Email: "pharmacist@hms.local", Role: "PHARMACIST"}
}

func days(n int) time.Time {
	return time.Now().UTC().AddDate(0, 0, n)
}

func TestDispenseSingleBatchSufficientStock(t *testing.T) {
	f := newTestFixture()

	medicine := f.medicines.seed(&domain.Medicine{Name: "Paracetamol 500mg", ReorderLevel: 10, CurrentStock: 50})
	batch := f.batches.seed(&domain.Batch{MedicineID: medicine.ID, MedicineName: medicine.Name, BatchNo: "B1", ExpiryDate: days(90), Quantity: 50})
	prescription := f.prescriptions.seed(&prescriptionsDomain.Prescription{Status: prescriptionsDomain.StatusActive})

	resp, err := f.svc.Dispense(context.Background(), dto.DispenseRequest{
		PrescriptionID: prescription.ID.Hex(),
		Items:          []dto.DispenseItemRequest{{MedicineID: medicine.ID.Hex(), Quantity: 20}},
	}, testActor())
	if err != nil {
		t.Fatalf("Dispense returned an unexpected error: %v", err)
	}

	if len(resp.Items) != 1 || resp.Items[0].Quantity != 20 {
		t.Fatalf("unexpected response items: %+v", resp.Items)
	}
	if len(resp.Items[0].Batches) != 1 || resp.Items[0].Batches[0].BatchID != batch.ID.Hex() {
		t.Fatalf("expected the single batch to be fully attributed, got: %+v", resp.Items[0].Batches)
	}

	updatedBatch, _ := f.batches.FindByID(context.Background(), batch.ID.Hex())
	if updatedBatch.Quantity != 30 {
		t.Fatalf("expected batch quantity 30 after dispensing 20 of 50, got %d", updatedBatch.Quantity)
	}

	updatedMedicine, _ := f.medicines.FindByID(context.Background(), medicine.ID.Hex())
	if updatedMedicine.CurrentStock != 30 {
		t.Fatalf("expected medicine currentStock 30 after dispensing 20 of 50, got %d", updatedMedicine.CurrentStock)
	}

	if f.prescriptions.statusOf(prescription.ID) != prescriptionsDomain.StatusDispensed {
		t.Fatalf("expected prescription to be marked DISPENSED")
	}

	if f.movements.count() != 1 {
		t.Fatalf("expected exactly one stock movement to be recorded, got %d", f.movements.count())
	}
}

func TestDispenseFEFOSpansMultipleBatchesInExpiryOrder(t *testing.T) {
	f := newTestFixture()

	medicine := f.medicines.seed(&domain.Medicine{Name: "Amoxicillin 500mg", ReorderLevel: 10, CurrentStock: 30})
	// Seed out of expiry order to prove FEFO sorts rather than relying on
	// insertion order.
	farBatch := f.batches.seed(&domain.Batch{MedicineID: medicine.ID, MedicineName: medicine.Name, BatchNo: "FAR", ExpiryDate: days(180), Quantity: 20})
	nearBatch := f.batches.seed(&domain.Batch{MedicineID: medicine.ID, MedicineName: medicine.Name, BatchNo: "NEAR", ExpiryDate: days(10), Quantity: 10})
	prescription := f.prescriptions.seed(&prescriptionsDomain.Prescription{Status: prescriptionsDomain.StatusActive})

	resp, err := f.svc.Dispense(context.Background(), dto.DispenseRequest{
		PrescriptionID: prescription.ID.Hex(),
		Items:          []dto.DispenseItemRequest{{MedicineID: medicine.ID.Hex(), Quantity: 15}},
	}, testActor())
	if err != nil {
		t.Fatalf("Dispense returned an unexpected error: %v", err)
	}

	allocations := resp.Items[0].Batches
	if len(allocations) != 2 {
		t.Fatalf("expected the request to span both batches, got %+v", allocations)
	}
	if allocations[0].BatchID != nearBatch.ID.Hex() || allocations[0].Quantity != 10 {
		t.Fatalf("expected the near-expiry batch to be fully consumed first, got %+v", allocations[0])
	}
	if allocations[1].BatchID != farBatch.ID.Hex() || allocations[1].Quantity != 5 {
		t.Fatalf("expected the remaining 5 units to come from the far-expiry batch, got %+v", allocations[1])
	}

	updatedNear, _ := f.batches.FindByID(context.Background(), nearBatch.ID.Hex())
	updatedFar, _ := f.batches.FindByID(context.Background(), farBatch.ID.Hex())
	if updatedNear.Quantity != 0 {
		t.Fatalf("expected near-expiry batch to be fully depleted, got %d", updatedNear.Quantity)
	}
	if updatedFar.Quantity != 15 {
		t.Fatalf("expected far-expiry batch to have 15 left (20-5), got %d", updatedFar.Quantity)
	}
}

func TestDispenseExcludesExpiredBatches(t *testing.T) {
	f := newTestFixture()

	medicine := f.medicines.seed(&domain.Medicine{Name: "Pantoprazole 40mg", ReorderLevel: 10, CurrentStock: 20})
	f.batches.seed(&domain.Batch{MedicineID: medicine.ID, MedicineName: medicine.Name, BatchNo: "EXPIRED", ExpiryDate: days(-5), Quantity: 20})
	prescription := f.prescriptions.seed(&prescriptionsDomain.Prescription{Status: prescriptionsDomain.StatusActive})

	_, err := f.svc.Dispense(context.Background(), dto.DispenseRequest{
		PrescriptionID: prescription.ID.Hex(),
		Items:          []dto.DispenseItemRequest{{MedicineID: medicine.ID.Hex(), Quantity: 5}},
	}, testActor())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "INSUFFICIENT_STOCK" {
		t.Fatalf("expected an INSUFFICIENT_STOCK error since the only batch is expired, got: %v", err)
	}

	if f.prescriptions.statusOf(prescription.ID) != prescriptionsDomain.StatusActive {
		t.Fatalf("expected the prescription to remain ACTIVE after a rejected dispense")
	}
}

func TestDispenseRejectsWhenRequestedQuantityExceedsStock(t *testing.T) {
	f := newTestFixture()

	medicine := f.medicines.seed(&domain.Medicine{Name: "ORS", ReorderLevel: 5, CurrentStock: 10})
	f.batches.seed(&domain.Batch{MedicineID: medicine.ID, MedicineName: medicine.Name, BatchNo: "B1", ExpiryDate: days(60), Quantity: 10})
	prescription := f.prescriptions.seed(&prescriptionsDomain.Prescription{Status: prescriptionsDomain.StatusActive})

	_, err := f.svc.Dispense(context.Background(), dto.DispenseRequest{
		PrescriptionID: prescription.ID.Hex(),
		Items:          []dto.DispenseItemRequest{{MedicineID: medicine.ID.Hex(), Quantity: 25}},
	}, testActor())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "INSUFFICIENT_STOCK" {
		t.Fatalf("expected an INSUFFICIENT_STOCK error, got: %v", err)
	}

	updatedMedicine, _ := f.medicines.FindByID(context.Background(), medicine.ID.Hex())
	if updatedMedicine.CurrentStock != 10 {
		t.Fatalf("expected stock to be untouched after a rejected dispense, got %d", updatedMedicine.CurrentStock)
	}
	if f.prescriptions.statusOf(prescription.ID) != prescriptionsDomain.StatusActive {
		t.Fatalf("expected the prescription to remain ACTIVE after a rejected dispense")
	}
}

func TestDispenseRejectsNonActivePrescription(t *testing.T) {
	f := newTestFixture()

	medicine := f.medicines.seed(&domain.Medicine{Name: "Paracetamol 500mg", ReorderLevel: 10, CurrentStock: 50})
	f.batches.seed(&domain.Batch{MedicineID: medicine.ID, MedicineName: medicine.Name, BatchNo: "B1", ExpiryDate: days(90), Quantity: 50})
	prescription := f.prescriptions.seed(&prescriptionsDomain.Prescription{Status: prescriptionsDomain.StatusDispensed})

	_, err := f.svc.Dispense(context.Background(), dto.DispenseRequest{
		PrescriptionID: prescription.ID.Hex(),
		Items:          []dto.DispenseItemRequest{{MedicineID: medicine.ID.Hex(), Quantity: 5}},
	}, testActor())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "PRESCRIPTION_NOT_DISPENSABLE" {
		t.Fatalf("expected a PRESCRIPTION_NOT_DISPENSABLE error, got: %v", err)
	}
}

func TestReceiveBatchIncreasesMedicineStockAndRecordsMovement(t *testing.T) {
	f := newTestFixture()

	medicine := f.medicines.seed(&domain.Medicine{Name: "Paracetamol 500mg", ReorderLevel: 10, CurrentStock: 0})

	batchResp, err := f.svc.ReceiveBatch(context.Background(), medicine.ID.Hex(), dto.ReceiveBatchRequest{
		BatchNo: "B1", ExpiryDate: days(90), Quantity: 100, PurchasePrice: 1.5, SellingPrice: 2.5,
	}, testActor())
	if err != nil {
		t.Fatalf("ReceiveBatch returned an unexpected error: %v", err)
	}
	if batchResp.Quantity != 100 {
		t.Fatalf("expected batch quantity 100, got %d", batchResp.Quantity)
	}

	updatedMedicine, _ := f.medicines.FindByID(context.Background(), medicine.ID.Hex())
	if updatedMedicine.CurrentStock != 100 {
		t.Fatalf("expected medicine currentStock 100 after receiving, got %d", updatedMedicine.CurrentStock)
	}
	if f.movements.count() != 1 {
		t.Fatalf("expected exactly one stock movement to be recorded, got %d", f.movements.count())
	}

	actions := f.audit.actions()
	if len(actions) == 0 || actions[len(actions)-1] != "BATCH_RECEIVED" {
		t.Fatalf("expected a BATCH_RECEIVED audit entry, got %+v", actions)
	}
}

func TestReceiveBatchRejectsPastExpiryDate(t *testing.T) {
	f := newTestFixture()

	medicine := f.medicines.seed(&domain.Medicine{Name: "Paracetamol 500mg", ReorderLevel: 10, CurrentStock: 0})

	_, err := f.svc.ReceiveBatch(context.Background(), medicine.ID.Hex(), dto.ReceiveBatchRequest{
		BatchNo: "B1", ExpiryDate: days(-1), Quantity: 100,
	}, testActor())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrValidation.Code {
		t.Fatalf("expected a validation error for a past expiry date, got: %v", err)
	}
}

func TestSearchMedicinesLowStockFilter(t *testing.T) {
	f := newTestFixture()

	f.medicines.seed(&domain.Medicine{Name: "Low", ReorderLevel: 10, CurrentStock: 5})
	f.medicines.seed(&domain.Medicine{Name: "Plenty", ReorderLevel: 10, CurrentStock: 100})

	result, err := f.svc.SearchMedicines(context.Background(), dto.SearchMedicinesQuery{LowStock: true, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("SearchMedicines returned an unexpected error: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Name != "Low" {
		t.Fatalf("expected only the low-stock medicine to match, got %+v", result.Items)
	}
}

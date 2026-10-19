package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	prescriptionsDomain "hms-backend/internal/modules/prescriptions/domain"
	prescriptionsRepo "hms-backend/internal/modules/prescriptions/repository"

	"hms-backend/internal/modules/pharmacy/domain"
	"hms-backend/internal/modules/pharmacy/dto"
	"hms-backend/internal/modules/pharmacy/repository"
	"hms-backend/internal/pkg/pagination"
)

func insufficientStockError() *apperror.AppError {
	return apperror.New("INSUFFICIENT_STOCK", "Not enough non-expired stock is available to fulfill this quantity.", 409)
}

// Dispense fulfills every requested item against inventory using FEFO
// (first-expiry-first-out) allocation, then marks the prescription
// DISPENSED. This deployment runs MongoDB standalone (see root
// docker-compose.yml, same constraint documented in the Appointments and
// Lab Orders modules), so there is no multi-document transaction to wrap
// this in. Two things stand in for one:
//
//  1. The prescription is marked DISPENSED *before* any stock is touched
//     (fail fast if it isn't ACTIVE), and every batch decrement is atomic
//     and guarded against going negative (repository.ErrInsufficientBatchStock),
//     so concurrent dispenses can never oversell a batch.
//  2. If any step after that fails, every batch decrement already applied
//     *by this call* is explicitly reverted (compensating action), and the
//     prescription is reverted back to ACTIVE — the same pattern used by
//     Appointments' Reschedule.
//
// The one residual gap: there is a small window between checking the
// prescription's status and marking it DISPENSED where a second,
// concurrent dispense of the *same* prescription could both pass the
// check. This is accepted here as a documented tradeoff — the same kind
// of gap Appointments documents for overlapping-but-not-identical booking
// races — rather than changing the Prescriptions module's repository
// interface to add a conditional update method for a scenario (two
// pharmacists dispensing the literal same script at the same instant)
// that is far rarer than the appointment-double-booking case the spec
// explicitly calls out as a required test.
func (s *pharmacyService) Dispense(ctx context.Context, req dto.DispenseRequest, actor audit.Actor) (*dto.DispenseResponse, error) {
	prescriptionOID, err := primitive.ObjectIDFromHex(req.PrescriptionID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"prescriptionId": "must be a valid id"})
	}

	prescription, err := s.prescriptions.FindByID(ctx, req.PrescriptionID)
	if err != nil {
		if errors.Is(err, prescriptionsRepo.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"prescriptionId": "prescription not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if prescription.Status != prescriptionsDomain.StatusActive {
		return nil, apperror.New("PRESCRIPTION_NOT_DISPENSABLE", "Only an active prescription can be dispensed.", 409)
	}

	// Resolve and validate every requested medicine before touching any
	// stock, so an unknown/inactive medicine never leaves a partial
	// dispense behind.
	type resolvedItem struct {
		medicine *domain.Medicine
		quantity int
	}
	resolved := make([]resolvedItem, 0, len(req.Items))
	for _, item := range req.Items {
		medicine, err := s.medicines.FindByID(ctx, item.MedicineID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, apperror.ErrValidation.WithDetails(map[string]string{"medicineId": "medicine not found: " + item.MedicineID})
			}
			return nil, apperror.ErrInternal.WithCause(err)
		}
		if medicine.Status != domain.StatusActive {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"medicineId": "medicine is not active: " + item.MedicineID})
		}
		resolved = append(resolved, resolvedItem{medicine: medicine, quantity: item.Quantity})
	}

	// Pre-check total non-expired stock per medicine. This narrows the
	// race window (see doc comment) without needing a transaction.
	now := nowFunc()
	for _, item := range resolved {
		batches, err := s.batches.FindAvailableForMedicine(ctx, item.medicine.ID, now)
		if err != nil {
			return nil, apperror.ErrInternal.WithCause(err)
		}
		var available int
		for _, b := range batches {
			available += b.Quantity
		}
		if available < item.quantity {
			return nil, insufficientStockError()
		}
	}

	// Fail fast, before any stock is touched, if the prescription is no
	// longer dispensable (see doc comment for the residual race this
	// still leaves).
	updatedPrescription, err := s.prescriptions.UpdateFields(ctx, prescriptionOID, bson.M{"status": string(prescriptionsDomain.StatusDispensed)})
	if err != nil {
		if errors.Is(err, prescriptionsRepo.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	type appliedDecrement struct {
		batchID primitive.ObjectID
		amount  int
	}
	var applied []appliedDecrement

	rollback := func() {
		for _, a := range applied {
			if err := s.batches.IncrementQuantity(context.Background(), a.batchID, a.amount); err != nil {
				s.log.Error("failed to revert a batch decrement after a failed dispense; inventory may be understated",
					zap.String("batchId", a.batchID.Hex()), zap.Int("amount", a.amount), zap.Error(err))
			}
		}
		if _, err := s.prescriptions.UpdateFields(context.Background(), prescriptionOID, bson.M{"status": string(prescriptionsDomain.StatusActive)}); err != nil {
			s.log.Error("failed to revert prescription status after a failed dispense",
				zap.String("prescriptionId", prescriptionOID.Hex()), zap.Error(err))
		}
	}

	performerName := s.resolveUserName(ctx, actor.UserID, actor.Email)
	var performedBy primitive.ObjectID
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			performedBy = oid
		}
	}

	responseItems := make([]dto.DispensedItemResponse, 0, len(resolved))

	for _, item := range resolved {
		batches, err := s.batches.FindAvailableForMedicine(ctx, item.medicine.ID, now)
		if err != nil {
			rollback()
			return nil, apperror.ErrInternal.WithCause(err)
		}

		remaining := item.quantity
		allocations := make([]dto.BatchAllocation, 0, 1)

		for _, batch := range batches {
			if remaining <= 0 {
				break
			}
			take := remaining
			if batch.Quantity < take {
				take = batch.Quantity
			}

			if _, err := s.batches.DecrementQuantity(ctx, batch.ID, take); err != nil {
				rollback()
				if errors.Is(err, repository.ErrInsufficientBatchStock) {
					return nil, insufficientStockError()
				}
				return nil, apperror.ErrInternal.WithCause(err)
			}
			applied = append(applied, appliedDecrement{batchID: batch.ID, amount: take})
			allocations = append(allocations, dto.BatchAllocation{BatchID: batch.ID.Hex(), BatchNo: batch.BatchNo, Quantity: take})
			remaining -= take

			s.recordMovement(ctx, domain.StockMovement{
				MedicineID: item.medicine.ID, MedicineName: item.medicine.Name,
				BatchID: batch.ID, BatchNo: batch.BatchNo, Type: domain.MovementOut, Quantity: take,
				Reason: "Dispensed for prescription " + req.PrescriptionID, PrescriptionID: &prescriptionOID,
				PerformedBy: performedBy, PerformedByName: performerName,
			})
		}

		if remaining > 0 {
			// The pre-check passed but a concurrent dispense consumed the
			// stock in between — revert everything from this call.
			rollback()
			return nil, insufficientStockError()
		}

		if _, err := s.medicines.AdjustStock(ctx, item.medicine.ID, -item.quantity); err != nil {
			rollback()
			return nil, apperror.ErrInternal.WithCause(err)
		}

		responseItems = append(responseItems, dto.DispensedItemResponse{
			MedicineID: item.medicine.ID.Hex(), MedicineName: item.medicine.Name, Quantity: item.quantity, Batches: allocations,
		})
	}

	s.recordAudit(ctx, actor, "MEDICINE_DISPENSED", updatedPrescription.ID.Hex(), map[string]any{"itemCount": len(resolved)})

	return &dto.DispenseResponse{
		PrescriptionID: updatedPrescription.ID.Hex(),
		DispensedAt:    nowFunc(),
		Items:          responseItems,
	}, nil
}

func (s *pharmacyService) SearchStockMovements(ctx context.Context, query dto.SearchStockMovementsQuery) (pagination.Result[dto.StockMovementResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.StockMovementFilter{MedicineID: query.MedicineID, Type: query.Type}

	movements, total, err := s.movements.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.StockMovementResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.StockMovementResponse, 0, len(movements))
	for _, m := range movements {
		items = append(items, toStockMovementResponse(&m))
	}

	return pagination.NewResult(items, page, total), nil
}

func toStockMovementResponse(m *domain.StockMovement) dto.StockMovementResponse {
	resp := dto.StockMovementResponse{
		ID: m.ID.Hex(), MedicineID: m.MedicineID.Hex(), MedicineName: m.MedicineName,
		BatchID: m.BatchID.Hex(), BatchNo: m.BatchNo, Type: string(m.Type), Quantity: m.Quantity,
		Reason: m.Reason, PerformedByName: m.PerformedByName, CreatedAt: m.CreatedAt,
	}
	if m.PrescriptionID != nil {
		resp.PrescriptionID = m.PrescriptionID.Hex()
	}
	return resp
}

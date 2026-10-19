// Package service implements the pharmacy module's business logic:
// medicine catalog management, batch receiving, and FEFO-based dispensing
// against prescriptions issued by the EMR module.
package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/modules/pharmacy/domain"
	"hms-backend/internal/modules/pharmacy/dto"
	"hms-backend/internal/modules/pharmacy/repository"
	"hms-backend/internal/pkg/pagination"
)

func (s *pharmacyService) CreateMedicine(ctx context.Context, req dto.CreateMedicineRequest, actor audit.Actor) (*dto.MedicineResponse, error) {
	medicine := &domain.Medicine{
		Name: req.Name, GenericName: req.GenericName, Category: req.Category,
		Manufacturer: req.Manufacturer, Unit: req.Unit, ReorderLevel: req.ReorderLevel,
		CurrentStock: 0, Status: domain.StatusActive,
	}

	if err := s.medicines.Create(ctx, medicine); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "MEDICINE_CREATED", medicine.ID.Hex(), nil)

	response := toMedicineResponse(medicine)
	return &response, nil
}

func (s *pharmacyService) UpdateMedicine(ctx context.Context, id string, req dto.UpdateMedicineRequest, actor audit.Actor) (*dto.MedicineResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	set := bson.M{}
	if req.Name != nil {
		set["name"] = *req.Name
	}
	if req.GenericName != nil {
		set["genericName"] = *req.GenericName
	}
	if req.Category != nil {
		set["category"] = *req.Category
	}
	if req.Manufacturer != nil {
		set["manufacturer"] = *req.Manufacturer
	}
	if req.Unit != nil {
		set["unit"] = *req.Unit
	}
	if req.ReorderLevel != nil {
		set["reorderLevel"] = *req.ReorderLevel
	}
	if req.Status != nil {
		set["status"] = *req.Status
	}
	if len(set) == 0 {
		return nil, apperror.ErrValidation.WithDetails("no fields to update")
	}

	medicine, err := s.medicines.Update(ctx, oid, set)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "MEDICINE_UPDATED", medicine.ID.Hex(), nil)

	response := toMedicineResponse(medicine)
	return &response, nil
}

func (s *pharmacyService) GetMedicineByID(ctx context.Context, id string) (*dto.MedicineResponse, error) {
	medicine, err := s.medicines.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toMedicineResponse(medicine)
	return &response, nil
}

func (s *pharmacyService) SearchMedicines(ctx context.Context, query dto.SearchMedicinesQuery) (pagination.Result[dto.MedicineResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.MedicineSearchFilter{Query: query.Query, LowStock: query.LowStock}

	medicines, total, err := s.medicines.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.MedicineResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.MedicineResponse, 0, len(medicines))
	for _, m := range medicines {
		items = append(items, toMedicineResponse(&m))
	}

	return pagination.NewResult(items, page, total), nil
}

func (s *pharmacyService) ReceiveBatch(ctx context.Context, medicineID string, req dto.ReceiveBatchRequest, actor audit.Actor) (*dto.BatchResponse, error) {
	medicine, err := s.medicines.FindByID(ctx, medicineID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"medicineId": "medicine not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if req.ExpiryDate.Before(nowFunc()) {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"expiryDate": "cannot be in the past"})
	}

	batch := &domain.Batch{
		MedicineID: medicine.ID, MedicineName: medicine.Name,
		BatchNo: req.BatchNo, ExpiryDate: req.ExpiryDate, Quantity: req.Quantity,
		PurchasePrice: req.PurchasePrice, SellingPrice: req.SellingPrice,
	}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			batch.ReceivedBy = oid
		}
	}

	if err := s.batches.Create(ctx, batch); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	if _, err := s.medicines.AdjustStock(ctx, medicine.ID, req.Quantity); err != nil {
		s.log.Error("failed to adjust medicine stock after receiving a batch; currentStock is now out of sync with inventory",
			zap.String("medicineId", medicineID), zap.Error(err))
	}

	performerName := s.resolveUserName(ctx, actor.UserID, actor.Email)
	s.recordMovement(ctx, domain.StockMovement{
		MedicineID: medicine.ID, MedicineName: medicine.Name, BatchID: batch.ID, BatchNo: batch.BatchNo,
		Type: domain.MovementIn, Quantity: req.Quantity, Reason: "Batch received",
		PerformedBy: batch.ReceivedBy, PerformedByName: performerName,
	})

	s.recordAudit(ctx, actor, "BATCH_RECEIVED", batch.ID.Hex(), map[string]any{"medicineId": medicineID, "quantity": req.Quantity})

	response := toBatchResponse(batch)
	return &response, nil
}

func (s *pharmacyService) ListBatchesForMedicine(ctx context.Context, medicineID string) ([]dto.BatchResponse, error) {
	oid, err := primitive.ObjectIDFromHex(medicineID)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	batches, err := s.batches.FindByMedicineID(ctx, oid)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.BatchResponse, 0, len(batches))
	for _, b := range batches {
		items = append(items, toBatchResponse(&b))
	}
	return items, nil
}

func (s *pharmacyService) ListExpiringBatches(ctx context.Context, withinDays int) ([]dto.BatchResponse, error) {
	cutoff := nowFunc().AddDate(0, 0, withinDays)

	batches, err := s.batches.FindExpiringBefore(ctx, cutoff)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.BatchResponse, 0, len(batches))
	for _, b := range batches {
		items = append(items, toBatchResponse(&b))
	}
	return items, nil
}

func toMedicineResponse(m *domain.Medicine) dto.MedicineResponse {
	return dto.MedicineResponse{
		ID: m.ID.Hex(), Name: m.Name, GenericName: m.GenericName, Category: m.Category,
		Manufacturer: m.Manufacturer, Unit: m.Unit, ReorderLevel: m.ReorderLevel,
		CurrentStock: m.CurrentStock, IsLowStock: m.IsLowStock(), Status: string(m.Status),
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func toBatchResponse(b *domain.Batch) dto.BatchResponse {
	return dto.BatchResponse{
		ID: b.ID.Hex(), MedicineID: b.MedicineID.Hex(), MedicineName: b.MedicineName,
		BatchNo: b.BatchNo, ExpiryDate: b.ExpiryDate, Quantity: b.Quantity, IsExpired: b.IsExpired(nowFunc()),
		PurchasePrice: b.PurchasePrice, SellingPrice: b.SellingPrice, CreatedAt: b.CreatedAt,
	}
}

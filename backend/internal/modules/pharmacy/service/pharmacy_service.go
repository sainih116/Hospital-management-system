package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"hms-backend/internal/audit"
	authRepo "hms-backend/internal/modules/auth/repository"
	prescriptionsRepo "hms-backend/internal/modules/prescriptions/repository"

	"hms-backend/internal/modules/pharmacy/domain"
	"hms-backend/internal/modules/pharmacy/dto"
	"hms-backend/internal/modules/pharmacy/repository"
	"hms-backend/internal/pkg/pagination"
)

// nowFunc is a seam for tests to control "the current time" without
// sleeping/mocking the clock globally.
var nowFunc = func() time.Time { return time.Now().UTC() }

// PharmacyService is the pharmacy module's public business-logic contract.
type PharmacyService interface {
	CreateMedicine(ctx context.Context, req dto.CreateMedicineRequest, actor audit.Actor) (*dto.MedicineResponse, error)
	UpdateMedicine(ctx context.Context, id string, req dto.UpdateMedicineRequest, actor audit.Actor) (*dto.MedicineResponse, error)
	GetMedicineByID(ctx context.Context, id string) (*dto.MedicineResponse, error)
	SearchMedicines(ctx context.Context, query dto.SearchMedicinesQuery) (pagination.Result[dto.MedicineResponse], error)

	ReceiveBatch(ctx context.Context, medicineID string, req dto.ReceiveBatchRequest, actor audit.Actor) (*dto.BatchResponse, error)
	ListBatchesForMedicine(ctx context.Context, medicineID string) ([]dto.BatchResponse, error)
	ListExpiringBatches(ctx context.Context, withinDays int) ([]dto.BatchResponse, error)

	Dispense(ctx context.Context, req dto.DispenseRequest, actor audit.Actor) (*dto.DispenseResponse, error)
	SearchStockMovements(ctx context.Context, query dto.SearchStockMovementsQuery) (pagination.Result[dto.StockMovementResponse], error)
}

type pharmacyService struct {
	medicines     repository.MedicineRepository
	batches       repository.BatchRepository
	movements     repository.StockMovementRepository
	prescriptions prescriptionsRepo.PrescriptionRepository
	users         authRepo.UserRepository
	audit         audit.Recorder
	log           *zap.Logger
}

// Deps bundles every collaborator pharmacyService needs.
type Deps struct {
	Medicines     repository.MedicineRepository
	Batches       repository.BatchRepository
	Movements     repository.StockMovementRepository
	Prescriptions prescriptionsRepo.PrescriptionRepository
	Users         authRepo.UserRepository
	Audit         audit.Recorder
	Logger        *zap.Logger
}

// NewPharmacyService builds the pharmacy module's service implementation.
func NewPharmacyService(d Deps) PharmacyService {
	return &pharmacyService{
		medicines: d.Medicines, batches: d.Batches, movements: d.Movements,
		prescriptions: d.Prescriptions, users: d.Users, audit: d.Audit, log: d.Logger,
	}
}

// resolveUserName looks up a display name for the acting user, falling
// back to their email (always available from the JWT claims) if the
// lookup fails — the same pattern used by the lab orders module for its
// "entered/verified by" fields.
func (s *pharmacyService) resolveUserName(ctx context.Context, userID, fallbackEmail string) string {
	if userID == "" {
		return fallbackEmail
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fallbackEmail
	}
	return user.Name
}

func (s *pharmacyService) recordMovement(ctx context.Context, movement domain.StockMovement) {
	if err := s.movements.Create(ctx, &movement); err != nil {
		s.log.Warn("failed to record stock movement", zap.String("medicineId", movement.MedicineID.Hex()), zap.Error(err))
	}
}

func (s *pharmacyService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string, metadata map[string]any) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID: actor.UserID, ActorEmail: actor.Email, ActorRole: actor.Role,
		Action: action, EntityType: "pharmacy", EntityID: entityID, Metadata: metadata,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry", zap.String("action", action), zap.String("entityId", entityID), zap.Error(err))
	}
}

// Package service implements the lab tests module's business logic.
package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/modules/labtests/domain"
	"hms-backend/internal/modules/labtests/dto"
	"hms-backend/internal/modules/labtests/repository"
)

// LabTestService is the lab tests module's public business-logic contract.
type LabTestService interface {
	Create(ctx context.Context, req dto.CreateLabTestRequest, actor audit.Actor) (*dto.LabTestResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateLabTestRequest, actor audit.Actor) (*dto.LabTestResponse, error)
	GetByID(ctx context.Context, id string) (*dto.LabTestResponse, error)
	List(ctx context.Context) ([]dto.LabTestResponse, error)
}

type labTestService struct {
	tests repository.LabTestRepository
	audit audit.Recorder
	log   *zap.Logger
}

// Deps bundles every collaborator labTestService needs.
type Deps struct {
	Tests  repository.LabTestRepository
	Audit  audit.Recorder
	Logger *zap.Logger
}

// NewLabTestService builds the lab tests module's service implementation.
func NewLabTestService(d Deps) LabTestService {
	return &labTestService{tests: d.Tests, audit: d.Audit, log: d.Logger}
}

func (s *labTestService) Create(ctx context.Context, req dto.CreateLabTestRequest, actor audit.Actor) (*dto.LabTestResponse, error) {
	if _, err := s.tests.FindByCode(ctx, req.Code); err == nil {
		return nil, apperror.New("LAB_TEST_CODE_TAKEN", "A lab test with this code already exists.", 409)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	test := &domain.LabTest{
		Code: req.Code, Name: req.Name, Category: req.Category, SampleType: req.SampleType,
		Unit: req.Unit, NormalRange: req.NormalRange, Cost: req.Cost, Status: domain.StatusActive,
	}

	if err := s.tests.Create(ctx, test); err != nil {
		if errors.Is(err, repository.ErrCodeTaken) {
			return nil, apperror.New("LAB_TEST_CODE_TAKEN", "A lab test with this code already exists.", 409)
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "LAB_TEST_CREATED", test.ID.Hex())

	response := toLabTestResponse(test)
	return &response, nil
}

func (s *labTestService) Update(ctx context.Context, id string, req dto.UpdateLabTestRequest, actor audit.Actor) (*dto.LabTestResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	set := bson.M{}
	if req.Name != nil {
		set["name"] = *req.Name
	}
	if req.Category != nil {
		set["category"] = *req.Category
	}
	if req.SampleType != nil {
		set["sampleType"] = *req.SampleType
	}
	if req.Unit != nil {
		set["unit"] = *req.Unit
	}
	if req.NormalRange != nil {
		set["normalRange"] = *req.NormalRange
	}
	if req.Cost != nil {
		set["cost"] = *req.Cost
	}
	if req.Status != nil {
		set["status"] = *req.Status
	}
	if len(set) == 0 {
		return nil, apperror.ErrValidation.WithDetails("no fields to update")
	}

	test, err := s.tests.Update(ctx, oid, set)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "LAB_TEST_UPDATED", test.ID.Hex())

	response := toLabTestResponse(test)
	return &response, nil
}

func (s *labTestService) GetByID(ctx context.Context, id string) (*dto.LabTestResponse, error) {
	test, err := s.tests.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toLabTestResponse(test)
	return &response, nil
}

func (s *labTestService) List(ctx context.Context) ([]dto.LabTestResponse, error) {
	tests, err := s.tests.List(ctx)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.LabTestResponse, 0, len(tests))
	for _, t := range tests {
		items = append(items, toLabTestResponse(&t))
	}
	return items, nil
}

func (s *labTestService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID: actor.UserID, ActorEmail: actor.Email, ActorRole: actor.Role,
		Action: action, EntityType: "lab_test", EntityID: entityID,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry", zap.String("action", action), zap.Error(err))
	}
}

func toLabTestResponse(t *domain.LabTest) dto.LabTestResponse {
	return dto.LabTestResponse{
		ID: t.ID.Hex(), Code: t.Code, Name: t.Name, Category: t.Category, SampleType: t.SampleType,
		Unit: t.Unit, NormalRange: t.NormalRange, Cost: t.Cost, Status: string(t.Status),
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

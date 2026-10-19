// Package service implements the departments module's business logic.
package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/modules/departments/domain"
	"hms-backend/internal/modules/departments/dto"
	"hms-backend/internal/modules/departments/repository"
)

// DepartmentService is the departments module's public business-logic
// contract.
type DepartmentService interface {
	Create(ctx context.Context, req dto.CreateDepartmentRequest, actor audit.Actor) (*dto.DepartmentResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateDepartmentRequest, actor audit.Actor) (*dto.DepartmentResponse, error)
	GetByID(ctx context.Context, id string) (*dto.DepartmentResponse, error)
	List(ctx context.Context) ([]dto.DepartmentResponse, error)
}

type departmentService struct {
	departments repository.DepartmentRepository
	audit       audit.Recorder
	log         *zap.Logger
}

// Deps bundles every collaborator departmentService needs.
type Deps struct {
	Departments repository.DepartmentRepository
	Audit       audit.Recorder
	Logger      *zap.Logger
}

// NewDepartmentService builds the departments module's service
// implementation.
func NewDepartmentService(d Deps) DepartmentService {
	return &departmentService{departments: d.Departments, audit: d.Audit, log: d.Logger}
}

func (s *departmentService) Create(ctx context.Context, req dto.CreateDepartmentRequest, actor audit.Actor) (*dto.DepartmentResponse, error) {
	if _, err := s.departments.FindByCode(ctx, req.Code); err == nil {
		return nil, apperror.New("DEPARTMENT_CODE_TAKEN", "A department with this code already exists.", 409)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	department := &domain.Department{Name: req.Name, Code: req.Code, Status: domain.StatusActive}

	if err := s.departments.Create(ctx, department); err != nil {
		if errors.Is(err, repository.ErrCodeTaken) {
			return nil, apperror.New("DEPARTMENT_CODE_TAKEN", "A department with this code already exists.", 409)
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "DEPARTMENT_CREATED", department.ID.Hex())

	response := toDepartmentResponse(department)
	return &response, nil
}

func (s *departmentService) Update(ctx context.Context, id string, req dto.UpdateDepartmentRequest, actor audit.Actor) (*dto.DepartmentResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	set := bson.M{}
	if req.Name != nil {
		set["name"] = *req.Name
	}
	if req.Status != nil {
		set["status"] = *req.Status
	}
	if len(set) == 0 {
		return nil, apperror.ErrValidation.WithDetails("no fields to update")
	}

	department, err := s.departments.Update(ctx, oid, set)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "DEPARTMENT_UPDATED", department.ID.Hex())

	response := toDepartmentResponse(department)
	return &response, nil
}

func (s *departmentService) GetByID(ctx context.Context, id string) (*dto.DepartmentResponse, error) {
	department, err := s.departments.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toDepartmentResponse(department)
	return &response, nil
}

func (s *departmentService) List(ctx context.Context) ([]dto.DepartmentResponse, error) {
	departments, err := s.departments.List(ctx)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.DepartmentResponse, 0, len(departments))
	for _, d := range departments {
		items = append(items, toDepartmentResponse(&d))
	}
	return items, nil
}

func (s *departmentService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID: actor.UserID, ActorEmail: actor.Email, ActorRole: actor.Role,
		Action: action, EntityType: "department", EntityID: entityID,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry", zap.String("action", action), zap.Error(err))
	}
}

func toDepartmentResponse(d *domain.Department) dto.DepartmentResponse {
	return dto.DepartmentResponse{
		ID: d.ID.Hex(), Name: d.Name, Code: d.Code, Status: string(d.Status),
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

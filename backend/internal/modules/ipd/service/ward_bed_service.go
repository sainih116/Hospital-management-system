package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"

	"hms-backend/internal/modules/ipd/domain"
	"hms-backend/internal/modules/ipd/dto"
	"hms-backend/internal/modules/ipd/repository"
)

func (s *ipdService) CreateWard(ctx context.Context, req dto.CreateWardRequest, actor audit.Actor) (*dto.WardResponse, error) {
	ward := &domain.Ward{Name: req.Name, Code: req.Code, Status: domain.StatusActive}

	if err := s.wards.Create(ctx, ward); err != nil {
		if errors.Is(err, repository.ErrCodeTaken) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"code": "already in use"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "WARD_CREATED", ward.ID.Hex(), nil)

	response := toWardResponse(ward)
	return &response, nil
}

func (s *ipdService) UpdateWard(ctx context.Context, id string, req dto.UpdateWardRequest, actor audit.Actor) (*dto.WardResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	set := bson.M{}
	if req.Name != nil {
		set["name"] = *req.Name
	}
	if req.Code != nil {
		set["code"] = *req.Code
	}
	if req.Status != nil {
		set["status"] = *req.Status
	}
	if len(set) == 0 {
		return nil, apperror.ErrValidation.WithDetails("no fields to update")
	}

	ward, err := s.wards.Update(ctx, oid, set)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "WARD_UPDATED", ward.ID.Hex(), nil)

	response := toWardResponse(ward)
	return &response, nil
}

func (s *ipdService) ListWards(ctx context.Context) ([]dto.WardResponse, error) {
	wards, err := s.wards.List(ctx)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.WardResponse, 0, len(wards))
	for _, w := range wards {
		items = append(items, toWardResponse(&w))
	}
	return items, nil
}

func (s *ipdService) CreateBed(ctx context.Context, req dto.CreateBedRequest, actor audit.Actor) (*dto.BedResponse, error) {
	wardOID, err := primitive.ObjectIDFromHex(req.WardID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"wardId": "must be a valid id"})
	}

	ward, err := s.wards.FindByID(ctx, req.WardID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"wardId": "ward not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	bed := &domain.Bed{WardID: wardOID, WardName: ward.Name, BedNumber: req.BedNumber, Status: domain.BedAvailable}

	if err := s.beds.Create(ctx, bed); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "BED_CREATED", bed.ID.Hex(), map[string]any{"wardId": req.WardID})

	response := toBedResponse(bed)
	return &response, nil
}

func (s *ipdService) UpdateBedStatus(ctx context.Context, id string, req dto.UpdateBedRequest, actor audit.Actor) (*dto.BedResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	existing, err := s.beds.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if existing.Status == domain.BedOccupied {
		return nil, apperror.New("BED_OCCUPIED", "An occupied bed's status can only change via discharge.", 409)
	}

	bed, err := s.beds.SetStatus(ctx, oid, domain.BedStatus(req.Status))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "BED_STATUS_CHANGED", bed.ID.Hex(), map[string]any{"status": req.Status})

	response := toBedResponse(bed)
	return &response, nil
}

func (s *ipdService) ListBeds(ctx context.Context, wardID, status string) ([]dto.BedResponse, error) {
	beds, err := s.beds.List(ctx, repository.BedFilter{WardID: wardID, Status: status})
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.BedResponse, 0, len(beds))
	for _, b := range beds {
		items = append(items, toBedResponse(&b))
	}
	return items, nil
}

func toWardResponse(w *domain.Ward) dto.WardResponse {
	return dto.WardResponse{
		ID: w.ID.Hex(), Name: w.Name, Code: w.Code, Status: string(w.Status),
		CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	}
}

func toBedResponse(b *domain.Bed) dto.BedResponse {
	return dto.BedResponse{
		ID: b.ID.Hex(), WardID: b.WardID.Hex(), WardName: b.WardName, BedNumber: b.BedNumber,
		Status: string(b.Status), CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
	}
}

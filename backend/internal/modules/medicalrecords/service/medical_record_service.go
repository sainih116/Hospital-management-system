// Package service implements the medical records module's business logic:
// patient/doctor existence validation, name snapshotting, and audit
// logging of every create/amend.
package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/medicalrecords/domain"
	"hms-backend/internal/modules/medicalrecords/dto"
	"hms-backend/internal/modules/medicalrecords/repository"
	"hms-backend/internal/pkg/pagination"
)

// MedicalRecordService is the medical records module's public
// business-logic contract.
type MedicalRecordService interface {
	Create(ctx context.Context, req dto.CreateMedicalRecordRequest, actor audit.Actor) (*dto.MedicalRecordResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateMedicalRecordRequest, actor audit.Actor) (*dto.MedicalRecordResponse, error)
	GetByID(ctx context.Context, id string) (*dto.MedicalRecordResponse, error)
	Search(ctx context.Context, query dto.SearchMedicalRecordsQuery) (pagination.Result[dto.MedicalRecordResponse], error)
}

type medicalRecordService struct {
	records  repository.MedicalRecordRepository
	patients patientsRepo.PatientRepository
	doctors  doctorsRepo.DoctorRepository
	audit    audit.Recorder
	log      *zap.Logger
}

// Deps bundles every collaborator medicalRecordService needs.
type Deps struct {
	Records  repository.MedicalRecordRepository
	Patients patientsRepo.PatientRepository
	Doctors  doctorsRepo.DoctorRepository
	Audit    audit.Recorder
	Logger   *zap.Logger
}

// NewMedicalRecordService builds the medical records module's service
// implementation.
func NewMedicalRecordService(d Deps) MedicalRecordService {
	return &medicalRecordService{records: d.Records, patients: d.Patients, doctors: d.Doctors, audit: d.Audit, log: d.Logger}
}

func (s *medicalRecordService) Create(ctx context.Context, req dto.CreateMedicalRecordRequest, actor audit.Actor) (*dto.MedicalRecordResponse, error) {
	patientOID, err := primitive.ObjectIDFromHex(req.PatientID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"patientId": "must be a valid id"})
	}
	doctorOID, err := primitive.ObjectIDFromHex(req.DoctorID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"doctorId": "must be a valid id"})
	}

	patient, err := s.patients.FindByID(ctx, req.PatientID)
	if err != nil {
		if errors.Is(err, patientsRepo.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"patientId": "patient not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	doctor, err := s.doctors.FindByID(ctx, req.DoctorID)
	if err != nil {
		if errors.Is(err, doctorsRepo.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"doctorId": "doctor not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	record := &domain.MedicalRecord{
		PatientID:   patientOID,
		PatientName: patient.Name,
		DoctorID:    doctorOID,
		DoctorName:  doctor.Name,
		Complaint:   req.Complaint,
		HPI:         req.HPI,
		History:     req.History,
		Examination: req.Examination,
		Diagnosis:   req.Diagnosis,
		Treatment:   req.Treatment,
		FollowUp:    req.FollowUp,
	}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			record.CreatedBy = oid
		}
	}

	if err := s.records.Create(ctx, record); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "MEDICAL_RECORD_CREATED", record.ID.Hex(), nil)

	response := toMedicalRecordResponse(record)
	return &response, nil
}

func (s *medicalRecordService) Update(ctx context.Context, id string, req dto.UpdateMedicalRecordRequest, actor audit.Actor) (*dto.MedicalRecordResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	set := bson.M{}
	var changedFields []string

	if req.Complaint != nil {
		set["complaint"] = *req.Complaint
		changedFields = append(changedFields, "complaint")
	}
	if req.HPI != nil {
		set["hpi"] = *req.HPI
		changedFields = append(changedFields, "hpi")
	}
	if req.History != nil {
		set["history"] = *req.History
		changedFields = append(changedFields, "history")
	}
	if req.Examination != nil {
		set["examination"] = *req.Examination
		changedFields = append(changedFields, "examination")
	}
	if req.Diagnosis != nil {
		set["diagnosis"] = *req.Diagnosis
		changedFields = append(changedFields, "diagnosis")
	}
	if req.Treatment != nil {
		set["treatment"] = *req.Treatment
		changedFields = append(changedFields, "treatment")
	}
	if req.FollowUp != nil {
		set["followUp"] = *req.FollowUp
		changedFields = append(changedFields, "followUp")
	}

	if len(set) == 0 {
		return nil, apperror.ErrValidation.WithDetails("no fields to update")
	}

	record, err := s.records.Update(ctx, oid, set)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "MEDICAL_RECORD_AMENDED", record.ID.Hex(), map[string]any{"fields": changedFields})

	response := toMedicalRecordResponse(record)
	return &response, nil
}

func (s *medicalRecordService) GetByID(ctx context.Context, id string) (*dto.MedicalRecordResponse, error) {
	record, err := s.records.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toMedicalRecordResponse(record)
	return &response, nil
}

func (s *medicalRecordService) Search(ctx context.Context, query dto.SearchMedicalRecordsQuery) (pagination.Result[dto.MedicalRecordResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.SearchFilter{PatientID: query.PatientID, DoctorID: query.DoctorID}

	records, total, err := s.records.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.MedicalRecordResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.MedicalRecordResponse, 0, len(records))
	for _, r := range records {
		items = append(items, toMedicalRecordResponse(&r))
	}

	return pagination.NewResult(items, page, total), nil
}

func (s *medicalRecordService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string, metadata map[string]any) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID: actor.UserID, ActorEmail: actor.Email, ActorRole: actor.Role,
		Action: action, EntityType: "medical_record", EntityID: entityID, Metadata: metadata,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry", zap.String("action", action), zap.String("entityId", entityID), zap.Error(err))
	}
}

func toMedicalRecordResponse(r *domain.MedicalRecord) dto.MedicalRecordResponse {
	return dto.MedicalRecordResponse{
		ID:          r.ID.Hex(),
		PatientID:   r.PatientID.Hex(),
		PatientName: r.PatientName,
		DoctorID:    r.DoctorID.Hex(),
		DoctorName:  r.DoctorName,
		Complaint:   r.Complaint,
		HPI:         r.HPI,
		History:     r.History,
		Examination: r.Examination,
		Diagnosis:   r.Diagnosis,
		Treatment:   r.Treatment,
		FollowUp:    r.FollowUp,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

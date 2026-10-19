// Package service implements the prescriptions module's business logic:
// patient/doctor/medical-record existence validation, name snapshotting,
// and audit logging of every issue/cancel.
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
	medicalRecordsRepo "hms-backend/internal/modules/medicalrecords/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/prescriptions/domain"
	"hms-backend/internal/modules/prescriptions/dto"
	"hms-backend/internal/modules/prescriptions/repository"
	"hms-backend/internal/pkg/pagination"
)

// PrescriptionService is the prescriptions module's public business-logic
// contract.
type PrescriptionService interface {
	Create(ctx context.Context, req dto.CreatePrescriptionRequest, actor audit.Actor) (*dto.PrescriptionResponse, error)
	ChangeStatus(ctx context.Context, id string, req dto.ChangeStatusRequest, actor audit.Actor) (*dto.PrescriptionResponse, error)
	GetByID(ctx context.Context, id string) (*dto.PrescriptionResponse, error)
	Search(ctx context.Context, query dto.SearchPrescriptionsQuery) (pagination.Result[dto.PrescriptionResponse], error)
}

type prescriptionService struct {
	prescriptions repository.PrescriptionRepository
	patients      patientsRepo.PatientRepository
	doctors       doctorsRepo.DoctorRepository
	medicalRecord medicalRecordsRepo.MedicalRecordRepository
	audit         audit.Recorder
	log           *zap.Logger
}

// Deps bundles every collaborator prescriptionService needs.
type Deps struct {
	Prescriptions repository.PrescriptionRepository
	Patients      patientsRepo.PatientRepository
	Doctors       doctorsRepo.DoctorRepository
	MedicalRecord medicalRecordsRepo.MedicalRecordRepository
	Audit         audit.Recorder
	Logger        *zap.Logger
}

// NewPrescriptionService builds the prescriptions module's service
// implementation.
func NewPrescriptionService(d Deps) PrescriptionService {
	return &prescriptionService{
		prescriptions: d.Prescriptions, patients: d.Patients, doctors: d.Doctors,
		medicalRecord: d.MedicalRecord, audit: d.Audit, log: d.Logger,
	}
}

func (s *prescriptionService) Create(ctx context.Context, req dto.CreatePrescriptionRequest, actor audit.Actor) (*dto.PrescriptionResponse, error) {
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

	var medicalRecordOID *primitive.ObjectID
	if req.MedicalRecordID != "" {
		oid, err := primitive.ObjectIDFromHex(req.MedicalRecordID)
		if err != nil {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"medicalRecordId": "must be a valid id"})
		}
		if _, err := s.medicalRecord.FindByID(ctx, req.MedicalRecordID); err != nil {
			if errors.Is(err, medicalRecordsRepo.ErrNotFound) {
				return nil, apperror.ErrValidation.WithDetails(map[string]string{"medicalRecordId": "medical record not found"})
			}
			return nil, apperror.ErrInternal.WithCause(err)
		}
		medicalRecordOID = &oid
	}

	items := make([]domain.Item, 0, len(req.Items))
	for _, i := range req.Items {
		items = append(items, domain.Item{
			MedicineName: i.MedicineName, Dose: i.Dose, Frequency: i.Frequency,
			Duration: i.Duration, Instructions: i.Instructions,
		})
	}

	prescription := &domain.Prescription{
		PatientID:       patientOID,
		PatientName:     patient.Name,
		DoctorID:        doctorOID,
		DoctorName:      doctor.Name,
		MedicalRecordID: medicalRecordOID,
		Items:           items,
		Status:          domain.StatusActive,
	}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			prescription.CreatedBy = oid
		}
	}

	if err := s.prescriptions.Create(ctx, prescription); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "PRESCRIPTION_CREATED", prescription.ID.Hex(), nil)

	response := toPrescriptionResponse(prescription)
	return &response, nil
}

func (s *prescriptionService) ChangeStatus(ctx context.Context, id string, req dto.ChangeStatusRequest, actor audit.Actor) (*dto.PrescriptionResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	existing, err := s.prescriptions.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	if existing.Status != domain.StatusActive {
		return nil, apperror.New("INVALID_STATUS_TRANSITION", "Only an active prescription can be cancelled.", 409)
	}

	updated, err := s.prescriptions.UpdateFields(ctx, oid, bson.M{"status": req.Status})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "PRESCRIPTION_CANCELLED", updated.ID.Hex(), nil)

	response := toPrescriptionResponse(updated)
	return &response, nil
}

func (s *prescriptionService) GetByID(ctx context.Context, id string) (*dto.PrescriptionResponse, error) {
	p, err := s.prescriptions.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toPrescriptionResponse(p)
	return &response, nil
}

func (s *prescriptionService) Search(ctx context.Context, query dto.SearchPrescriptionsQuery) (pagination.Result[dto.PrescriptionResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.SearchFilter{PatientID: query.PatientID, DoctorID: query.DoctorID, Status: query.Status}

	prescriptions, total, err := s.prescriptions.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.PrescriptionResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.PrescriptionResponse, 0, len(prescriptions))
	for _, p := range prescriptions {
		items = append(items, toPrescriptionResponse(&p))
	}

	return pagination.NewResult(items, page, total), nil
}

func (s *prescriptionService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string, metadata map[string]any) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID: actor.UserID, ActorEmail: actor.Email, ActorRole: actor.Role,
		Action: action, EntityType: "prescription", EntityID: entityID, Metadata: metadata,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry", zap.String("action", action), zap.String("entityId", entityID), zap.Error(err))
	}
}

func toPrescriptionResponse(p *domain.Prescription) dto.PrescriptionResponse {
	items := make([]dto.ItemResponse, 0, len(p.Items))
	for _, i := range p.Items {
		items = append(items, dto.ItemResponse{
			MedicineName: i.MedicineName, Dose: i.Dose, Frequency: i.Frequency,
			Duration: i.Duration, Instructions: i.Instructions,
		})
	}

	resp := dto.PrescriptionResponse{
		ID:          p.ID.Hex(),
		PatientID:   p.PatientID.Hex(),
		PatientName: p.PatientName,
		DoctorID:    p.DoctorID.Hex(),
		DoctorName:  p.DoctorName,
		Items:       items,
		Status:      string(p.Status),
		IssuedAt:    p.IssuedAt,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
	if p.MedicalRecordID != nil {
		resp.MedicalRecordID = p.MedicalRecordID.Hex()
	}
	return resp
}

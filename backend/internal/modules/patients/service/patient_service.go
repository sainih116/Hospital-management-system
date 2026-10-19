// Package service implements the patients module's business logic:
// validation beyond what request binding can express, patient code
// generation, and audit logging of every create/update/delete.
package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/modules/patients/domain"
	"hms-backend/internal/modules/patients/dto"
	"hms-backend/internal/modules/patients/repository"
	"hms-backend/internal/pkg/pagination"
)

const dobLayout = "2006-01-02"

var phonePattern = regexp.MustCompile(`^[0-9]{7,15}$`)

// SequenceGenerator generates the next value for a named counter. It is
// satisfied by *sequence.Store; defining the interface here (rather than
// depending on that concrete type) lets tests supply an in-memory fake.
type SequenceGenerator interface {
	Next(ctx context.Context, key string) (int64, error)
}

// PatientService is the patients module's public business-logic contract.
type PatientService interface {
	Create(ctx context.Context, req dto.CreatePatientRequest, actor audit.Actor) (*dto.PatientResponse, error)
	Update(ctx context.Context, id string, req dto.UpdatePatientRequest, actor audit.Actor) (*dto.PatientResponse, error)
	Delete(ctx context.Context, id string, actor audit.Actor) error
	GetByID(ctx context.Context, id string) (*dto.PatientResponse, error)
	Search(ctx context.Context, query dto.SearchPatientsQuery) (pagination.Result[dto.PatientResponse], error)
}

type patientService struct {
	patients  repository.PatientRepository
	sequences SequenceGenerator
	audit     audit.Recorder
	log       *zap.Logger
}

// Deps bundles every collaborator patientService needs.
type Deps struct {
	Patients  repository.PatientRepository
	Sequences SequenceGenerator
	Audit     audit.Recorder
	Logger    *zap.Logger
}

// NewPatientService builds the patients module's service implementation.
func NewPatientService(d Deps) PatientService {
	return &patientService{patients: d.Patients, sequences: d.Sequences, audit: d.Audit, log: d.Logger}
}

func (s *patientService) Create(ctx context.Context, req dto.CreatePatientRequest, actor audit.Actor) (*dto.PatientResponse, error) {
	dob, err := parseDOB(req.DOB)
	if err != nil {
		return nil, err
	}
	if !isValidPhone(req.Phone) {
		return nil, invalidPhoneError()
	}
	if err := validateEmergencyContact(req.EmergencyContact); err != nil {
		return nil, err
	}

	seq, err := s.sequences.Next(ctx, "patients")
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	patient := &domain.Patient{
		PatientCode:      formatPatientCode(seq),
		Name:             req.Name,
		DOB:              dob,
		Gender:           domain.Gender(req.Gender),
		Phone:            req.Phone,
		Email:            req.Email,
		Address:          req.Address,
		BloodGroup:       req.BloodGroup,
		Allergies:        req.Allergies,
		EmergencyContact: toDomainEmergencyContact(req.EmergencyContact),
		Insurance:        toDomainInsurance(req.Insurance),
		Type:             domain.PatientType(req.Type),
		Status:           domain.StatusActive,
	}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			patient.CreatedBy = oid
		}
	}

	if err := s.patients.Create(ctx, patient); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "PATIENT_CREATED", patient.ID.Hex(), nil)

	response := toPatientResponse(patient)
	return &response, nil
}

func (s *patientService) Update(ctx context.Context, id string, req dto.UpdatePatientRequest, actor audit.Actor) (*dto.PatientResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	set := bson.M{}
	var changedFields []string

	if req.Name != nil {
		set["name"] = *req.Name
		changedFields = append(changedFields, "name")
	}
	if req.DOB != nil {
		dob, err := parseDOB(*req.DOB)
		if err != nil {
			return nil, err
		}
		set["dob"] = dob
		changedFields = append(changedFields, "dob")
	}
	if req.Gender != nil {
		set["gender"] = *req.Gender
		changedFields = append(changedFields, "gender")
	}
	if req.Phone != nil {
		if !isValidPhone(*req.Phone) {
			return nil, invalidPhoneError()
		}
		set["phone"] = *req.Phone
		changedFields = append(changedFields, "phone")
	}
	if req.Email != nil {
		set["email"] = *req.Email
		changedFields = append(changedFields, "email")
	}
	if req.Address != nil {
		set["address"] = *req.Address
		changedFields = append(changedFields, "address")
	}
	if req.BloodGroup != nil {
		set["bloodGroup"] = *req.BloodGroup
		changedFields = append(changedFields, "bloodGroup")
	}
	if req.Allergies != nil {
		set["allergies"] = *req.Allergies
		changedFields = append(changedFields, "allergies")
	}
	if req.EmergencyContact != nil {
		if err := validateEmergencyContact(req.EmergencyContact); err != nil {
			return nil, err
		}
		set["emergencyContact"] = toDomainEmergencyContact(req.EmergencyContact)
		changedFields = append(changedFields, "emergencyContact")
	}
	if req.Insurance != nil {
		set["insurance"] = toDomainInsurance(req.Insurance)
		changedFields = append(changedFields, "insurance")
	}
	if req.Type != nil {
		set["type"] = *req.Type
		changedFields = append(changedFields, "type")
	}

	if len(set) == 0 {
		return nil, apperror.ErrValidation.WithDetails("no fields to update")
	}

	patient, err := s.patients.Update(ctx, oid, set)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "PATIENT_UPDATED", patient.ID.Hex(), map[string]any{"fields": changedFields})

	response := toPatientResponse(patient)
	return &response, nil
}

func (s *patientService) Delete(ctx context.Context, id string, actor audit.Actor) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrNotFound
	}

	if err := s.patients.SoftDelete(ctx, oid); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.ErrNotFound
		}
		return apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "PATIENT_DELETED", id, nil)
	return nil
}

func (s *patientService) GetByID(ctx context.Context, id string) (*dto.PatientResponse, error) {
	patient, err := s.patients.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toPatientResponse(patient)
	return &response, nil
}

func (s *patientService) Search(ctx context.Context, query dto.SearchPatientsQuery) (pagination.Result[dto.PatientResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.SearchFilter{
		Query:      query.Query,
		Gender:     query.Gender,
		BloodGroup: query.BloodGroup,
		Type:       query.Type,
	}

	patients, total, err := s.patients.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.PatientResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.PatientResponse, 0, len(patients))
	for _, p := range patients {
		items = append(items, toPatientResponse(&p))
	}

	return pagination.NewResult(items, page, total), nil
}

func (s *patientService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string, metadata map[string]any) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID:    actor.UserID,
		ActorEmail: actor.Email,
		ActorRole:  actor.Role,
		Action:     action,
		EntityType: "patient",
		EntityID:   entityID,
		Metadata:   metadata,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry",
			zap.String("action", action), zap.String("entityId", entityID), zap.Error(err))
	}
}

func parseDOB(raw string) (time.Time, error) {
	dob, err := time.Parse(dobLayout, raw)
	if err != nil {
		return time.Time{}, apperror.ErrValidation.WithDetails(map[string]string{"dob": "must be in YYYY-MM-DD format"})
	}

	now := time.Now().UTC()
	if dob.After(now) {
		return time.Time{}, apperror.ErrValidation.WithDetails(map[string]string{"dob": "cannot be in the future"})
	}
	if now.Sub(dob).Hours() > 150*365*24 {
		return time.Time{}, apperror.ErrValidation.WithDetails(map[string]string{"dob": "is not a plausible date of birth"})
	}

	return dob, nil
}

func isValidPhone(phone string) bool {
	return phonePattern.MatchString(phone)
}

func invalidPhoneError() *apperror.AppError {
	return apperror.ErrValidation.WithDetails(map[string]string{"phone": "must be 7-15 digits"})
}

// validateEmergencyContact reuses the same phone format check applied to a
// patient's own primary phone (isValidPhone) — the DTO's binding tags can
// only enforce "required", not the digit-count pattern, so this mirrors
// the pattern already used for the top-level Phone field. A nil request
// (emergency contact omitted) is valid.
func validateEmergencyContact(req *dto.EmergencyContactRequest) error {
	if req == nil {
		return nil
	}
	if !isValidPhone(req.Phone) {
		return apperror.ErrValidation.WithDetails(map[string]string{"emergencyContact.phone": "must be 7-15 digits"})
	}
	return nil
}

func formatPatientCode(seq int64) string {
	return fmt.Sprintf("PAT-%05d", 10000+seq)
}

func toDomainEmergencyContact(req *dto.EmergencyContactRequest) *domain.EmergencyContact {
	if req == nil {
		return nil
	}
	return &domain.EmergencyContact{Name: req.Name, Relationship: req.Relationship, Phone: req.Phone}
}

func toDomainInsurance(req *dto.InsuranceRequest) *domain.Insurance {
	if req == nil {
		return nil
	}
	return &domain.Insurance{Provider: req.Provider, PolicyNumber: req.PolicyNumber}
}

func toPatientResponse(p *domain.Patient) dto.PatientResponse {
	response := dto.PatientResponse{
		ID:          p.ID.Hex(),
		PatientCode: p.PatientCode,
		Name:        p.Name,
		DOB:         p.DOB.Format(dobLayout),
		Gender:      string(p.Gender),
		Phone:       p.Phone,
		Email:       p.Email,
		Address:     p.Address,
		BloodGroup:  p.BloodGroup,
		Allergies:   p.Allergies,
		Type:        string(p.Type),
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
	if p.EmergencyContact != nil {
		response.EmergencyContact = &dto.EmergencyContact{
			Name: p.EmergencyContact.Name, Relationship: p.EmergencyContact.Relationship, Phone: p.EmergencyContact.Phone,
		}
	}
	if p.Insurance != nil {
		response.Insurance = &dto.Insurance{Provider: p.Insurance.Provider, PolicyNumber: p.Insurance.PolicyNumber}
	}
	return response
}

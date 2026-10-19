package service

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/ipd/domain"
	"hms-backend/internal/modules/ipd/dto"
	"hms-backend/internal/modules/ipd/repository"
	"hms-backend/internal/pkg/pagination"
)

// AdmitPatient admits a patient into a specific bed. This deployment runs
// MongoDB standalone (see root docker-compose.yml, the same constraint
// documented in Appointments/Pharmacy), so there is no multi-document
// transaction wrapping "occupy the bed" and "create the admission
// record" together. Instead: the bed is occupied first via an atomic
// guard-filter update (repository.BedRepository.Occupy — closes the
// blueprint's "two admissions, one bed" race on its own), and if
// creating the admission record fails for any reason afterward, the bed
// is explicitly released as a compensating action — the same pattern
// Appointments' Reschedule and Pharmacy's Dispense already use.
func (s *ipdService) AdmitPatient(ctx context.Context, req dto.AdmitPatientRequest, actor audit.Actor) (*dto.AdmissionResponse, error) {
	patientOID, err := primitive.ObjectIDFromHex(req.PatientID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"patientId": "must be a valid id"})
	}
	doctorOID, err := primitive.ObjectIDFromHex(req.DoctorID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"doctorId": "must be a valid id"})
	}
	bedOID, err := primitive.ObjectIDFromHex(req.BedID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"bedId": "must be a valid id"})
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

	bed, err := s.beds.FindByID(ctx, req.BedID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"bedId": "bed not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	if _, err := s.beds.Occupy(ctx, bedOID); err != nil {
		if errors.Is(err, repository.ErrBedNotAvailable) {
			return nil, apperror.New("BED_NOT_AVAILABLE", "This bed is no longer available.", 409)
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	admission := &domain.Admission{
		PatientID: patientOID, PatientName: patient.Name,
		DoctorID: doctorOID, DoctorName: doctor.Name,
		BedID: bedOID, BedNumber: bed.BedNumber, WardName: bed.WardName,
		Reason: req.Reason, Status: domain.AdmissionActive, AdmittedAt: time.Now().UTC(),
	}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			admission.CreatedBy = oid
		}
	}

	if err := s.admissions.Create(ctx, admission); err != nil {
		if releaseErr := s.beds.Release(context.Background(), bedOID); releaseErr != nil {
			s.log.Error("failed to release a bed after a failed admission; bed may be stuck OCCUPIED",
				zap.String("bedId", bedOID.Hex()), zap.Error(releaseErr))
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "PATIENT_ADMITTED", admission.ID.Hex(), map[string]any{"bedId": req.BedID})

	response := toAdmissionResponse(admission)
	return &response, nil
}

func (s *ipdService) DischargePatient(ctx context.Context, id string, req dto.DischargeRequest, actor audit.Actor) (*dto.AdmissionResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	existing, err := s.admissions.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if existing.Status != domain.AdmissionActive {
		return nil, apperror.New("ADMISSION_NOT_ACTIVE", "Only an admitted patient can be discharged.", 409)
	}

	now := time.Now().UTC()
	updated, err := s.admissions.UpdateFields(ctx, oid, bson.M{
		"status": string(domain.AdmissionDischarged), "dischargedAt": now, "dischargeNote": req.Note,
	})
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	if err := s.beds.Release(ctx, existing.BedID); err != nil {
		s.log.Error("failed to release bed after discharge; bed may be stuck OCCUPIED",
			zap.String("bedId", existing.BedID.Hex()), zap.Error(err))
	}

	s.recordAudit(ctx, actor, "PATIENT_DISCHARGED", updated.ID.Hex(), nil)

	response := toAdmissionResponse(updated)
	return &response, nil
}

func (s *ipdService) GetAdmissionByID(ctx context.Context, id string) (*dto.AdmissionResponse, error) {
	admission, err := s.admissions.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toAdmissionResponse(admission)
	return &response, nil
}

func (s *ipdService) SearchAdmissions(ctx context.Context, query dto.SearchAdmissionsQuery) (pagination.Result[dto.AdmissionResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.AdmissionFilter{PatientID: query.PatientID, DoctorID: query.DoctorID, Status: query.Status}

	admissions, total, err := s.admissions.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.AdmissionResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.AdmissionResponse, 0, len(admissions))
	for _, a := range admissions {
		items = append(items, toAdmissionResponse(&a))
	}

	return pagination.NewResult(items, page, total), nil
}

func toAdmissionResponse(a *domain.Admission) dto.AdmissionResponse {
	return dto.AdmissionResponse{
		ID: a.ID.Hex(), PatientID: a.PatientID.Hex(), PatientName: a.PatientName,
		DoctorID: a.DoctorID.Hex(), DoctorName: a.DoctorName,
		BedID: a.BedID.Hex(), BedNumber: a.BedNumber, WardName: a.WardName,
		Reason: a.Reason, Status: string(a.Status), AdmittedAt: a.AdmittedAt,
		DischargedAt: a.DischargedAt, DischargeNote: a.DischargeNote,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

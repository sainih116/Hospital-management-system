// Package service implements the IPD module's business logic: ward/bed
// catalog management, and the admit/discharge workflow that atomically
// occupies/releases a bed as it runs.
package service

import (
	"context"

	"go.uber.org/zap"

	"hms-backend/internal/audit"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/ipd/dto"
	"hms-backend/internal/modules/ipd/repository"
	"hms-backend/internal/pkg/pagination"
)

// IPDService is the IPD module's public business-logic contract.
type IPDService interface {
	CreateWard(ctx context.Context, req dto.CreateWardRequest, actor audit.Actor) (*dto.WardResponse, error)
	UpdateWard(ctx context.Context, id string, req dto.UpdateWardRequest, actor audit.Actor) (*dto.WardResponse, error)
	ListWards(ctx context.Context) ([]dto.WardResponse, error)

	CreateBed(ctx context.Context, req dto.CreateBedRequest, actor audit.Actor) (*dto.BedResponse, error)
	UpdateBedStatus(ctx context.Context, id string, req dto.UpdateBedRequest, actor audit.Actor) (*dto.BedResponse, error)
	ListBeds(ctx context.Context, wardID, status string) ([]dto.BedResponse, error)

	AdmitPatient(ctx context.Context, req dto.AdmitPatientRequest, actor audit.Actor) (*dto.AdmissionResponse, error)
	DischargePatient(ctx context.Context, id string, req dto.DischargeRequest, actor audit.Actor) (*dto.AdmissionResponse, error)
	GetAdmissionByID(ctx context.Context, id string) (*dto.AdmissionResponse, error)
	SearchAdmissions(ctx context.Context, query dto.SearchAdmissionsQuery) (pagination.Result[dto.AdmissionResponse], error)
}

type ipdService struct {
	wards      repository.WardRepository
	beds       repository.BedRepository
	admissions repository.AdmissionRepository
	patients   patientsRepo.PatientRepository
	doctors    doctorsRepo.DoctorRepository
	audit      audit.Recorder
	log        *zap.Logger
}

// Deps bundles every collaborator ipdService needs.
type Deps struct {
	Wards      repository.WardRepository
	Beds       repository.BedRepository
	Admissions repository.AdmissionRepository
	Patients   patientsRepo.PatientRepository
	Doctors    doctorsRepo.DoctorRepository
	Audit      audit.Recorder
	Logger     *zap.Logger
}

// NewIPDService builds the IPD module's service implementation.
func NewIPDService(d Deps) IPDService {
	return &ipdService{
		wards: d.Wards, beds: d.Beds, admissions: d.Admissions,
		patients: d.Patients, doctors: d.Doctors, audit: d.Audit, log: d.Logger,
	}
}

func (s *ipdService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string, metadata map[string]any) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID: actor.UserID, ActorEmail: actor.Email, ActorRole: actor.Role,
		Action: action, EntityType: "ipd", EntityID: entityID, Metadata: metadata,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry", zap.String("action", action), zap.String("entityId", entityID), zap.Error(err))
	}
}

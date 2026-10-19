// Package service implements the doctors module's business logic:
// cross-referencing the linked user account and department, time-slot
// validation, and audit logging.
package service

import (
	"context"
	"errors"
	"regexp"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	authDomain "hms-backend/internal/modules/auth/domain"
	authRepo "hms-backend/internal/modules/auth/repository"
	deptDomain "hms-backend/internal/modules/departments/domain"
	deptRepo "hms-backend/internal/modules/departments/repository"
	"hms-backend/internal/modules/doctors/domain"
	"hms-backend/internal/modules/doctors/dto"
	"hms-backend/internal/modules/doctors/repository"
	"hms-backend/internal/pkg/pagination"
)

var timePattern = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

// DoctorService is the doctors module's public business-logic contract.
type DoctorService interface {
	Create(ctx context.Context, req dto.CreateDoctorRequest, actor audit.Actor) (*dto.DoctorResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateDoctorRequest, actor audit.Actor) (*dto.DoctorResponse, error)
	Delete(ctx context.Context, id string, actor audit.Actor) error
	GetByID(ctx context.Context, id string) (*dto.DoctorResponse, error)
	Schedule(ctx context.Context, id string) (*dto.ScheduleResponse, error)
	Search(ctx context.Context, query dto.SearchDoctorsQuery) (pagination.Result[dto.DoctorResponse], error)
}

type doctorService struct {
	doctors     repository.DoctorRepository
	users       authRepo.UserRepository
	departments deptRepo.DepartmentRepository
	audit       audit.Recorder
	log         *zap.Logger
}

// Deps bundles every collaborator doctorService needs.
type Deps struct {
	Doctors     repository.DoctorRepository
	Users       authRepo.UserRepository
	Departments deptRepo.DepartmentRepository
	Audit       audit.Recorder
	Logger      *zap.Logger
}

// NewDoctorService builds the doctors module's service implementation.
func NewDoctorService(d Deps) DoctorService {
	return &doctorService{doctors: d.Doctors, users: d.Users, departments: d.Departments, audit: d.Audit, log: d.Logger}
}

func (s *doctorService) Create(ctx context.Context, req dto.CreateDoctorRequest, actor audit.Actor) (*dto.DoctorResponse, error) {
	user, err := s.loadDoctorUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	department, err := s.loadActiveDepartment(ctx, req.DepartmentID)
	if err != nil {
		return nil, err
	}

	slots, err := toDomainSlots(req.Availability)
	if err != nil {
		return nil, err
	}

	userOID, _ := primitive.ObjectIDFromHex(req.UserID)
	deptOID, _ := primitive.ObjectIDFromHex(req.DepartmentID)

	doctor := &domain.Doctor{
		UserID:         userOID,
		Name:           user.Name,
		Email:          user.Email,
		DepartmentID:   deptOID,
		DepartmentName: department.Name,
		LicenseNo:      req.LicenseNo,
		Specialization: req.Specialization,
		Fee:            req.Fee,
		Bio:            req.Bio,
		Availability:   slots,
		Status:         domain.StatusActive,
	}

	if err := s.doctors.Create(ctx, doctor); err != nil {
		switch {
		case errors.Is(err, repository.ErrUserAlreadyLinked):
			return nil, apperror.New("DOCTOR_PROFILE_EXISTS", "This user already has a doctor profile.", 409)
		case errors.Is(err, repository.ErrLicenseTaken):
			return nil, apperror.New("LICENSE_TAKEN", "A doctor with this license number already exists.", 409)
		default:
			return nil, apperror.ErrInternal.WithCause(err)
		}
	}

	s.recordAudit(ctx, actor, "DOCTOR_CREATED", doctor.ID.Hex(), nil)

	response := toDoctorResponse(doctor)
	return &response, nil
}

func (s *doctorService) Update(ctx context.Context, id string, req dto.UpdateDoctorRequest, actor audit.Actor) (*dto.DoctorResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	set := bson.M{}
	var changedFields []string

	if req.DepartmentID != nil {
		department, err := s.loadActiveDepartment(ctx, *req.DepartmentID)
		if err != nil {
			return nil, err
		}
		deptOID, _ := primitive.ObjectIDFromHex(*req.DepartmentID)
		set["departmentId"] = deptOID
		set["departmentName"] = department.Name
		changedFields = append(changedFields, "department")
	}
	if req.LicenseNo != nil {
		set["licenseNo"] = *req.LicenseNo
		changedFields = append(changedFields, "licenseNo")
	}
	if req.Specialization != nil {
		set["specialization"] = *req.Specialization
		changedFields = append(changedFields, "specialization")
	}
	if req.Fee != nil {
		set["fee"] = *req.Fee
		changedFields = append(changedFields, "fee")
	}
	if req.Bio != nil {
		set["bio"] = *req.Bio
		changedFields = append(changedFields, "bio")
	}
	if req.Availability != nil {
		slots, err := toDomainSlots(*req.Availability)
		if err != nil {
			return nil, err
		}
		set["availability"] = slots
		changedFields = append(changedFields, "availability")
	}

	if len(set) == 0 {
		return nil, apperror.ErrValidation.WithDetails("no fields to update")
	}

	doctor, err := s.doctors.Update(ctx, oid, set)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return nil, apperror.ErrNotFound
		case errors.Is(err, repository.ErrLicenseTaken):
			return nil, apperror.New("LICENSE_TAKEN", "A doctor with this license number already exists.", 409)
		default:
			return nil, apperror.ErrInternal.WithCause(err)
		}
	}

	s.recordAudit(ctx, actor, "DOCTOR_UPDATED", doctor.ID.Hex(), map[string]any{"fields": changedFields})

	response := toDoctorResponse(doctor)
	return &response, nil
}

func (s *doctorService) Delete(ctx context.Context, id string, actor audit.Actor) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrNotFound
	}

	if err := s.doctors.SoftDelete(ctx, oid); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.ErrNotFound
		}
		return apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "DOCTOR_DEACTIVATED", id, nil)
	return nil
}

func (s *doctorService) GetByID(ctx context.Context, id string) (*dto.DoctorResponse, error) {
	doctor, err := s.doctors.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toDoctorResponse(doctor)
	return &response, nil
}

func (s *doctorService) Schedule(ctx context.Context, id string) (*dto.ScheduleResponse, error) {
	doctor, err := s.doctors.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	return &dto.ScheduleResponse{DoctorID: doctor.ID.Hex(), Availability: toResponseSlots(doctor.Availability)}, nil
}

func (s *doctorService) Search(ctx context.Context, query dto.SearchDoctorsQuery) (pagination.Result[dto.DoctorResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.SearchFilter{
		Query:          query.Query,
		DepartmentID:   query.DepartmentID,
		Specialization: query.Specialization,
	}

	doctors, total, err := s.doctors.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.DoctorResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.DoctorResponse, 0, len(doctors))
	for _, d := range doctors {
		items = append(items, toDoctorResponse(&d))
	}

	return pagination.NewResult(items, page, total), nil
}

// loadDoctorUser validates that userID refers to an existing, active user
// with the DOCTOR role — the precondition for attaching a clinical profile.
func (s *doctorService) loadDoctorUser(ctx context.Context, userID string) (*authDomain.User, error) {
	if _, err := primitive.ObjectIDFromHex(userID); err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"userId": "must be a valid id"})
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, authRepo.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"userId": "user not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	if user.RoleCode != authDomain.RoleDoctor {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"userId": "user does not have the DOCTOR role"})
	}
	if !user.IsActive() {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"userId": "user account is not active"})
	}

	return user, nil
}

// loadActiveDepartment validates that departmentID refers to an existing,
// active department.
func (s *doctorService) loadActiveDepartment(ctx context.Context, departmentID string) (*deptDomain.Department, error) {
	department, err := s.departments.FindByID(ctx, departmentID)
	if err != nil {
		if errors.Is(err, deptRepo.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"departmentId": "department not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if department.Status != deptDomain.StatusActive {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"departmentId": "department is not active"})
	}
	return department, nil
}

func (s *doctorService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string, metadata map[string]any) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID: actor.UserID, ActorEmail: actor.Email, ActorRole: actor.Role,
		Action: action, EntityType: "doctor", EntityID: entityID, Metadata: metadata,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry", zap.String("action", action), zap.String("entityId", entityID), zap.Error(err))
	}
}

func toDomainSlots(reqSlots []dto.AvailabilitySlotRequest) ([]domain.AvailabilitySlot, error) {
	slots := make([]domain.AvailabilitySlot, 0, len(reqSlots))
	for _, s := range reqSlots {
		if s.DayOfWeek < 0 || s.DayOfWeek > 6 {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"availability": "dayOfWeek must be 0-6"})
		}
		if !timePattern.MatchString(s.StartTime) || !timePattern.MatchString(s.EndTime) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"availability": "times must be in HH:MM 24h format"})
		}
		if s.StartTime >= s.EndTime {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"availability": "startTime must be before endTime"})
		}
		slots = append(slots, domain.AvailabilitySlot{DayOfWeek: s.DayOfWeek, StartTime: s.StartTime, EndTime: s.EndTime})
	}
	return slots, nil
}

func toResponseSlots(slots []domain.AvailabilitySlot) []dto.AvailabilitySlotResponse {
	out := make([]dto.AvailabilitySlotResponse, 0, len(slots))
	for _, s := range slots {
		out = append(out, dto.AvailabilitySlotResponse{DayOfWeek: s.DayOfWeek, StartTime: s.StartTime, EndTime: s.EndTime})
	}
	return out
}

func toDoctorResponse(d *domain.Doctor) dto.DoctorResponse {
	return dto.DoctorResponse{
		ID:             d.ID.Hex(),
		UserID:         d.UserID.Hex(),
		Name:           d.Name,
		Email:          d.Email,
		DepartmentID:   d.DepartmentID.Hex(),
		DepartmentName: d.DepartmentName,
		LicenseNo:      d.LicenseNo,
		Specialization: d.Specialization,
		Fee:            d.Fee,
		Bio:            d.Bio,
		Availability:   toResponseSlots(d.Availability),
		Status:         string(d.Status),
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

// Package service implements the appointments module's business logic:
// booking validation (patient/doctor existence, availability-window and
// conflict checks), the appointment state machine, and reschedule-as-a-
// new-record semantics.
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
	authDomain "hms-backend/internal/modules/auth/domain"
	doctorsDomain "hms-backend/internal/modules/doctors/domain"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/appointments/domain"
	"hms-backend/internal/modules/appointments/dto"
	"hms-backend/internal/modules/appointments/repository"
	"hms-backend/internal/pkg/pagination"
)

// AppointmentService is the appointments module's public business-logic
// contract.
type AppointmentService interface {
	Book(ctx context.Context, req dto.BookAppointmentRequest, actor audit.Actor) (*dto.AppointmentResponse, error)
	ChangeStatus(ctx context.Context, id string, req dto.ChangeStatusRequest, actor audit.Actor) (*dto.AppointmentResponse, error)
	Reschedule(ctx context.Context, id string, req dto.RescheduleRequest, actor audit.Actor) (*dto.AppointmentResponse, error)
	GetByID(ctx context.Context, id string) (*dto.AppointmentResponse, error)
	Search(ctx context.Context, query dto.SearchAppointmentsQuery) (pagination.Result[dto.AppointmentResponse], error)
}

type appointmentService struct {
	appointments repository.AppointmentRepository
	patients     patientsRepo.PatientRepository
	doctors      doctorsRepo.DoctorRepository
	audit        audit.Recorder
	log          *zap.Logger
}

// Deps bundles every collaborator appointmentService needs.
type Deps struct {
	Appointments repository.AppointmentRepository
	Patients     patientsRepo.PatientRepository
	Doctors      doctorsRepo.DoctorRepository
	Audit        audit.Recorder
	Logger       *zap.Logger
}

// NewAppointmentService builds the appointments module's service
// implementation.
func NewAppointmentService(d Deps) AppointmentService {
	return &appointmentService{appointments: d.Appointments, patients: d.Patients, doctors: d.Doctors, audit: d.Audit, log: d.Logger}
}

func appointmentConflictError() *apperror.AppError {
	return apperror.New("APPOINTMENT_CONFLICT", "Doctor already has an appointment in this time range", 409)
}

func (s *appointmentService) Book(ctx context.Context, req dto.BookAppointmentRequest, actor audit.Actor) (*dto.AppointmentResponse, error) {
	if err := validateWindow(req.StartAt, req.EndAt); err != nil {
		return nil, err
	}

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

	if !isWithinAvailability(doctor, req.StartAt, req.EndAt) {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"startAt": "requested time is outside the doctor's available hours"})
	}

	hasConflict, err := s.appointments.HasConflict(ctx, doctorOID, req.StartAt, req.EndAt, nil)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if hasConflict {
		return nil, appointmentConflictError()
	}

	appt := &domain.Appointment{
		PatientID:   patientOID,
		PatientName: patient.Name,
		DoctorID:    doctorOID,
		DoctorName:  doctor.Name,
		StartAt:     req.StartAt,
		EndAt:       req.EndAt,
		Status:      domain.StatusScheduled,
		Reason:      req.Reason,
	}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			appt.CreatedBy = oid
		}
	}

	if err := s.appointments.Create(ctx, appt); err != nil {
		if errors.Is(err, repository.ErrSlotTaken) {
			return nil, appointmentConflictError()
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "APPOINTMENT_BOOKED", appt.ID.Hex(), nil)

	response := toAppointmentResponse(appt)
	return &response, nil
}

func (s *appointmentService) ChangeStatus(ctx context.Context, id string, req dto.ChangeStatusRequest, actor audit.Actor) (*dto.AppointmentResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	appt, err := s.appointments.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	targetStatus := domain.Status(req.Status)

	if err := s.authorizeStatusChange(ctx, actor, appt, targetStatus); err != nil {
		return nil, err
	}

	if !appt.Status.CanTransitionTo(targetStatus) {
		return nil, apperror.New("INVALID_STATUS_TRANSITION",
			"Cannot move an appointment from "+string(appt.Status)+" to "+string(targetStatus)+".", 409)
	}

	set := bson.M{"status": string(targetStatus)}
	if (targetStatus == domain.StatusCancelled || targetStatus == domain.StatusNoShow) && req.Reason != "" {
		set["cancelReason"] = req.Reason
	}

	updated, err := s.appointments.UpdateFields(ctx, oid, set)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "APPOINTMENT_STATUS_CHANGED", updated.ID.Hex(), map[string]any{
		"from": string(appt.Status), "to": string(targetStatus),
	})

	response := toAppointmentResponse(updated)
	return &response, nil
}

// authorizeStatusChange implements the fine-grained rule the blueprint's
// two spec sections disagree on at a glance: the RBAC matrix lists Doctor
// as "View" only for Appointments, while the REST table lists Doctor among
// the roles allowed to PATCH an appointment's status. The reconciliation
// used here: Admin/Receptionist (appointment.manage) may make any
// transition; a Doctor may only progress their *own* appointments through
// the clinical states (in-progress/completed/no-show) — never cancel,
// confirm, or touch another doctor's appointment.
func (s *appointmentService) authorizeStatusChange(ctx context.Context, actor audit.Actor, appt *domain.Appointment, target domain.Status) error {
	if actor.Role == authDomain.RoleAdmin || actor.Role == authDomain.RoleReceptionist {
		return nil
	}
	if actor.Role != authDomain.RoleDoctor {
		return apperror.ErrForbidden
	}

	actorOID, err := primitive.ObjectIDFromHex(actor.UserID)
	if err != nil {
		return apperror.ErrForbidden
	}

	doctorProfile, err := s.doctors.FindByUserID(ctx, actorOID)
	if err != nil || doctorProfile.ID != appt.DoctorID {
		return apperror.ErrForbidden
	}

	allowedForDoctor := map[domain.Status]bool{
		domain.StatusInProgress: true,
		domain.StatusCompleted:  true,
		domain.StatusNoShow:     true,
	}
	if !allowedForDoctor[target] {
		return apperror.ErrForbidden
	}

	return nil
}

func (s *appointmentService) Reschedule(ctx context.Context, id string, req dto.RescheduleRequest, actor audit.Actor) (*dto.AppointmentResponse, error) {
	if err := validateWindow(req.StartAt, req.EndAt); err != nil {
		return nil, err
	}

	oldAppt, err := s.appointments.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	if oldAppt.Status.IsTerminal() {
		return nil, apperror.New("APPOINTMENT_TERMINAL", "This appointment can no longer be rescheduled.", 409)
	}

	doctor, err := s.doctors.FindByID(ctx, oldAppt.DoctorID.Hex())
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if !isWithinAvailability(doctor, req.StartAt, req.EndAt) {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"startAt": "requested time is outside the doctor's available hours"})
	}

	hasConflict, err := s.appointments.HasConflict(ctx, oldAppt.DoctorID, req.StartAt, req.EndAt, &oldAppt.ID)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if hasConflict {
		return nil, appointmentConflictError()
	}

	// Mark the old appointment RESCHEDULED *before* creating the
	// replacement. This is what lets rescheduling reuse the exact same
	// slot (RESCHEDULED is outside the partial unique index's coverage —
	// see the repository package doc — so it stops holding the calendar
	// slot immediately). If creating the replacement then fails for any
	// reason, the old appointment is reverted to its previous status
	// rather than leaving the appointment silently lost — no transactions
	// are available on this standalone MongoDB deployment, so this
	// explicit compensating action is the safety net.
	previousStatus := oldAppt.Status
	if _, err := s.appointments.UpdateFields(ctx, oldAppt.ID, bson.M{"status": string(domain.StatusRescheduled)}); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	newAppt := &domain.Appointment{
		PatientID:         oldAppt.PatientID,
		PatientName:       oldAppt.PatientName,
		DoctorID:          oldAppt.DoctorID,
		DoctorName:        oldAppt.DoctorName,
		StartAt:           req.StartAt,
		EndAt:             req.EndAt,
		Status:            domain.StatusScheduled,
		Reason:            oldAppt.Reason,
		RescheduledFromID: &oldAppt.ID,
	}
	if actor.UserID != "" {
		if actorOID, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			newAppt.CreatedBy = actorOID
		}
	}

	if err := s.appointments.Create(ctx, newAppt); err != nil {
		if _, revertErr := s.appointments.UpdateFields(ctx, oldAppt.ID, bson.M{"status": string(previousStatus)}); revertErr != nil {
			s.log.Error("failed to revert appointment status after a failed reschedule; appointment may be incorrectly left as RESCHEDULED with no replacement",
				zap.String("appointmentId", oldAppt.ID.Hex()), zap.Error(revertErr))
		}
		if errors.Is(err, repository.ErrSlotTaken) {
			return nil, appointmentConflictError()
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	if _, err := s.appointments.UpdateFields(ctx, oldAppt.ID, bson.M{"rescheduledToId": newAppt.ID}); err != nil {
		s.log.Error("failed to link the old appointment to its replacement",
			zap.String("oldAppointmentId", oldAppt.ID.Hex()), zap.String("newAppointmentId", newAppt.ID.Hex()), zap.Error(err))
	}

	s.recordAudit(ctx, actor, "APPOINTMENT_RESCHEDULED", newAppt.ID.Hex(), map[string]any{
		"fromAppointmentId": oldAppt.ID.Hex(),
	})

	response := toAppointmentResponse(newAppt)
	return &response, nil
}

func (s *appointmentService) GetByID(ctx context.Context, id string) (*dto.AppointmentResponse, error) {
	appt, err := s.appointments.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toAppointmentResponse(appt)
	return &response, nil
}

func (s *appointmentService) Search(ctx context.Context, query dto.SearchAppointmentsQuery) (pagination.Result[dto.AppointmentResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.SearchFilter{
		PatientID: query.PatientID,
		DoctorID:  query.DoctorID,
		Status:    query.Status,
		From:      query.From,
		To:        query.To,
	}

	appointments, total, err := s.appointments.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.AppointmentResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.AppointmentResponse, 0, len(appointments))
	for _, a := range appointments {
		items = append(items, toAppointmentResponse(&a))
	}

	return pagination.NewResult(items, page, total), nil
}

func (s *appointmentService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string, metadata map[string]any) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID: actor.UserID, ActorEmail: actor.Email, ActorRole: actor.Role,
		Action: action, EntityType: "appointment", EntityID: entityID, Metadata: metadata,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry", zap.String("action", action), zap.String("entityId", entityID), zap.Error(err))
	}
}

func validateWindow(start, end time.Time) error {
	if !start.Before(end) {
		return apperror.ErrValidation.WithDetails(map[string]string{"endAt": "must be after startAt"})
	}
	if !start.After(time.Now().UTC()) {
		return apperror.ErrValidation.WithDetails(map[string]string{"startAt": "must be in the future"})
	}
	return nil
}

// isWithinAvailability reports whether [start, end) falls inside one of
// the doctor's recurring weekly availability slots. A doctor with no
// availability configured is treated as unrestricted (see
// doctors/domain.Doctor's Availability field doc).
func isWithinAvailability(doctor *doctorsDomain.Doctor, start, end time.Time) bool {
	if len(doctor.Availability) == 0 {
		return true
	}

	weekday := int(start.Weekday())
	startHHMM := start.Format("15:04")
	endHHMM := end.Format("15:04")

	for _, slot := range doctor.Availability {
		if slot.DayOfWeek == weekday && startHHMM >= slot.StartTime && endHHMM <= slot.EndTime {
			return true
		}
	}
	return false
}

func toAppointmentResponse(a *domain.Appointment) dto.AppointmentResponse {
	resp := dto.AppointmentResponse{
		ID:           a.ID.Hex(),
		PatientID:    a.PatientID.Hex(),
		PatientName:  a.PatientName,
		DoctorID:     a.DoctorID.Hex(),
		DoctorName:   a.DoctorName,
		StartAt:      a.StartAt,
		EndAt:        a.EndAt,
		Status:       string(a.Status),
		Reason:       a.Reason,
		Notes:        a.Notes,
		CancelReason: a.CancelReason,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
	if a.RescheduledFromID != nil {
		resp.RescheduledFromID = a.RescheduledFromID.Hex()
	}
	if a.RescheduledToID != nil {
		resp.RescheduledToID = a.RescheduledToID.Hex()
	}
	return resp
}

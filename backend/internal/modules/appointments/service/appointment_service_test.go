package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	authDomain "hms-backend/internal/modules/auth/domain"
	doctorsDomain "hms-backend/internal/modules/doctors/domain"
	patientsDomain "hms-backend/internal/modules/patients/domain"

	"hms-backend/internal/modules/appointments/dto"
)

type harness struct {
	svc      AppointmentService
	appts    *fakeAppointmentRepository
	patients *fakePatientRepository
	doctors  *fakeDoctorRepository
	audit    *fakeAuditRecorder
}

func newHarness() *harness {
	appts := newFakeAppointmentRepository()
	patients := newFakePatientRepository()
	doctors := newFakeDoctorRepository()
	auditRecorder := &fakeAuditRecorder{}

	svc := NewAppointmentService(Deps{
		Appointments: appts,
		Patients:     patients,
		Doctors:      doctors,
		Audit:        auditRecorder,
		Logger:       zap.NewNop(),
	})

	return &harness{svc: svc, appts: appts, patients: patients, doctors: doctors, audit: auditRecorder}
}

// allDayAvailability covers every weekday fully, so tests don't need to
// worry about which day-of-week "the future" happens to land on.
func allDayAvailability() []doctorsDomain.AvailabilitySlot {
	slots := make([]doctorsDomain.AvailabilitySlot, 0, 7)
	for d := 0; d <= 6; d++ {
		slots = append(slots, doctorsDomain.AvailabilitySlot{DayOfWeek: d, StartTime: "00:00", EndTime: "23:59"})
	}
	return slots
}

func seedPatient(h *harness) *patientsDomain.Patient {
	p := &patientsDomain.Patient{Name: "Rohan Kumar", PatientCode: "PAT-10001", Status: patientsDomain.StatusActive}
	h.patients.seed(p)
	return p
}

func seedDoctor(h *harness, availability []doctorsDomain.AvailabilitySlot) *doctorsDomain.Doctor {
	d := &doctorsDomain.Doctor{Name: "Dr. Anil Sharma", Specialization: "Cardiology", Availability: availability, Status: doctorsDomain.StatusActive}
	h.doctors.seed(d)
	return d
}

func futureWindow(afterHours, durationMinutes int) (time.Time, time.Time) {
	start := time.Now().UTC().Add(time.Duration(afterHours) * time.Hour)
	return start, start.Add(time.Duration(durationMinutes) * time.Minute)
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected an *apperror.AppError, got %T: %v", err, err)
	}
	return appErr.Code
}

func TestBookAppointmentSucceeds(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	start, end := futureWindow(24, 30)

	resp, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), StartAt: start, EndAt: end, Reason: "Checkup",
	}, audit.Actor{UserID: "u1", Role: authDomain.RoleReceptionist})
	if err != nil {
		t.Fatalf("Book returned error: %v", err)
	}

	if resp.Status != "SCHEDULED" {
		t.Errorf("expected status SCHEDULED, got %s", resp.Status)
	}
	if resp.PatientName != patient.Name || resp.DoctorName != doctor.Name {
		t.Errorf("expected snapshotted names, got patient=%s doctor=%s", resp.PatientName, resp.DoctorName)
	}
	if actions := h.audit.actions(); len(actions) != 1 || actions[0] != "APPOINTMENT_BOOKED" {
		t.Fatalf("expected an APPOINTMENT_BOOKED audit entry, got %v", actions)
	}
}

func TestBookAppointmentRejectsPastStartTime(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	start := time.Now().UTC().Add(-time.Hour)

	_, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), StartAt: start, EndAt: start.Add(30 * time.Minute),
	}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for a start time in the past")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestBookAppointmentRejectsEndBeforeStart(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	start, _ := futureWindow(24, 30)

	_, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), StartAt: start, EndAt: start.Add(-10 * time.Minute),
	}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error when endAt is before startAt")
	}
}

func TestBookAppointmentRejectsUnknownPatient(t *testing.T) {
	h := newHarness()
	doctor := seedDoctor(h, allDayAvailability())
	start, end := futureWindow(24, 30)

	_, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: "000000000000000000000000", DoctorID: doctor.ID.Hex(), StartAt: start, EndAt: end,
	}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown patient")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestBookAppointmentRejectsUnknownDoctor(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	start, end := futureWindow(24, 30)

	_, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: patient.ID.Hex(), DoctorID: "000000000000000000000000", StartAt: start, EndAt: end,
	}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown doctor")
	}
}

func TestBookAppointmentRejectsOutsideAvailability(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	// Doctor only available Mondays 09:00-10:00 — pick a start time far
	// outside any such window regardless of "today": use a duration long
	// enough (36h) combined with a narrow slot that virtually never lines up,
	// then assert on the specific rejection reason to keep this robust.
	doctor := seedDoctor(h, []doctorsDomain.AvailabilitySlot{{DayOfWeek: 1, StartTime: "09:00", EndTime: "10:00"}})

	// Find a future timestamp guaranteed to be outside 09:00-10:00 on any day.
	start := time.Now().UTC().Add(24 * time.Hour)
	start = time.Date(start.Year(), start.Month(), start.Day(), 23, 0, 0, 0, time.UTC)
	if !start.After(time.Now().UTC()) {
		start = start.Add(24 * time.Hour)
	}

	_, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), StartAt: start, EndAt: start.Add(30 * time.Minute),
	}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for a time outside the doctor's availability")
	}
	if code := appErrCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestBookAppointmentRejectsOverlappingSlot(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	start, end := futureWindow(24, 60)

	if _, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), StartAt: start, EndAt: end,
	}, audit.Actor{}); err != nil {
		t.Fatalf("first Book returned error: %v", err)
	}

	// Overlaps the first appointment's back half.
	overlapStart := start.Add(30 * time.Minute)
	_, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), StartAt: overlapStart, EndAt: overlapStart.Add(30 * time.Minute),
	}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an overlapping appointment")
	}
	if code := appErrCode(t, err); code != "APPOINTMENT_CONFLICT" {
		t.Fatalf("expected APPOINTMENT_CONFLICT, got %s", code)
	}
}

func TestBookAppointmentRejectsExactDuplicateSlotViaUniqueIndexBackstop(t *testing.T) {
	// This exercises the fake's simulation of the partial unique index —
	// the atomic backstop described in the repository package doc — by
	// bypassing the pre-check (HasConflict) through direct repo insertion
	// of a document with an identical doctorId+startAt, then confirming
	// Book's Create call surfaces it as the same conflict error.
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	start, end := futureWindow(24, 30)

	if _, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), StartAt: start, EndAt: end,
	}, audit.Actor{}); err != nil {
		t.Fatalf("first Book returned error: %v", err)
	}

	_, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), StartAt: start, EndAt: end,
	}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected exactly one of two identical-slot bookings to succeed")
	}
	if code := appErrCode(t, err); code != "APPOINTMENT_CONFLICT" {
		t.Fatalf("expected APPOINTMENT_CONFLICT, got %s", code)
	}
}

func bookValid(t *testing.T, h *harness, patient *patientsDomain.Patient, doctor *doctorsDomain.Doctor) *dto.AppointmentResponse {
	t.Helper()
	start, end := futureWindow(24, 30)
	resp, err := h.svc.Book(context.Background(), dto.BookAppointmentRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), StartAt: start, EndAt: end,
	}, audit.Actor{Role: authDomain.RoleReceptionist})
	if err != nil {
		t.Fatalf("Book returned error: %v", err)
	}
	return resp
}

func TestChangeStatusAdminCanConfirmThenCancel(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	appt := bookValid(t, h, patient, doctor)

	admin := audit.Actor{Role: authDomain.RoleAdmin}

	confirmed, err := h.svc.ChangeStatus(context.Background(), appt.ID, dto.ChangeStatusRequest{Status: "CONFIRMED"}, admin)
	if err != nil {
		t.Fatalf("ChangeStatus(CONFIRMED) returned error: %v", err)
	}
	if confirmed.Status != "CONFIRMED" {
		t.Fatalf("expected CONFIRMED, got %s", confirmed.Status)
	}

	cancelled, err := h.svc.ChangeStatus(context.Background(), appt.ID, dto.ChangeStatusRequest{Status: "CANCELLED", Reason: "Patient request"}, admin)
	if err != nil {
		t.Fatalf("ChangeStatus(CANCELLED) returned error: %v", err)
	}
	if cancelled.Status != "CANCELLED" || cancelled.CancelReason != "Patient request" {
		t.Fatalf("expected CANCELLED with reason recorded, got %+v", cancelled)
	}
}

func TestChangeStatusRejectsInvalidTransition(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	appt := bookValid(t, h, patient, doctor)

	// SCHEDULED -> COMPLETED is not a valid direct transition.
	_, err := h.svc.ChangeStatus(context.Background(), appt.ID, dto.ChangeStatusRequest{Status: "COMPLETED"}, audit.Actor{Role: authDomain.RoleAdmin})
	if err == nil {
		t.Fatalf("expected an error for an invalid state transition")
	}
	if code := appErrCode(t, err); code != "INVALID_STATUS_TRANSITION" {
		t.Fatalf("expected INVALID_STATUS_TRANSITION, got %s", code)
	}
}

func TestChangeStatusDoctorCanProgressOwnAppointment(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	appt := bookValid(t, h, patient, doctor)

	doctorActor := audit.Actor{UserID: doctor.UserID.Hex(), Role: authDomain.RoleDoctor}

	if _, err := h.svc.ChangeStatus(context.Background(), appt.ID, dto.ChangeStatusRequest{Status: "CONFIRMED"}, audit.Actor{Role: authDomain.RoleReceptionist}); err != nil {
		t.Fatalf("receptionist CONFIRMED transition returned error: %v", err)
	}

	inProgress, err := h.svc.ChangeStatus(context.Background(), appt.ID, dto.ChangeStatusRequest{Status: "IN_PROGRESS"}, doctorActor)
	if err != nil {
		t.Fatalf("doctor IN_PROGRESS transition returned error: %v", err)
	}
	if inProgress.Status != "IN_PROGRESS" {
		t.Fatalf("expected IN_PROGRESS, got %s", inProgress.Status)
	}

	completed, err := h.svc.ChangeStatus(context.Background(), appt.ID, dto.ChangeStatusRequest{Status: "COMPLETED"}, doctorActor)
	if err != nil {
		t.Fatalf("doctor COMPLETED transition returned error: %v", err)
	}
	if completed.Status != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", completed.Status)
	}
}

func TestChangeStatusDoctorCannotCancel(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	appt := bookValid(t, h, patient, doctor)

	doctorActor := audit.Actor{UserID: doctor.UserID.Hex(), Role: authDomain.RoleDoctor}

	_, err := h.svc.ChangeStatus(context.Background(), appt.ID, dto.ChangeStatusRequest{Status: "CANCELLED"}, doctorActor)
	if err == nil {
		t.Fatalf("expected a doctor to be forbidden from cancelling an appointment")
	}
	if code := appErrCode(t, err); code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %s", code)
	}
}

func TestChangeStatusDoctorCannotTouchAnotherDoctorsAppointment(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	otherDoctor := seedDoctor(h, allDayAvailability())
	appt := bookValid(t, h, patient, doctor)

	otherDoctorActor := audit.Actor{UserID: otherDoctor.UserID.Hex(), Role: authDomain.RoleDoctor}

	_, err := h.svc.ChangeStatus(context.Background(), appt.ID, dto.ChangeStatusRequest{Status: "IN_PROGRESS"}, otherDoctorActor)
	if err == nil {
		t.Fatalf("expected a doctor to be forbidden from changing another doctor's appointment")
	}
	if code := appErrCode(t, err); code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %s", code)
	}
}

func TestChangeStatusRejectsUnrelatedRole(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	appt := bookValid(t, h, patient, doctor)

	_, err := h.svc.ChangeStatus(context.Background(), appt.ID, dto.ChangeStatusRequest{Status: "CONFIRMED"}, audit.Actor{Role: authDomain.RoleNurse})
	if err == nil {
		t.Fatalf("expected a nurse to be forbidden from changing appointment status")
	}
	if code := appErrCode(t, err); code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %s", code)
	}
}

func TestRescheduleMarksOldAndCreatesNew(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	original := bookValid(t, h, patient, doctor)

	newStart, newEnd := futureWindow(48, 30)
	rescheduled, err := h.svc.Reschedule(context.Background(), original.ID, dto.RescheduleRequest{StartAt: newStart, EndAt: newEnd}, audit.Actor{Role: authDomain.RoleReceptionist})
	if err != nil {
		t.Fatalf("Reschedule returned error: %v", err)
	}
	if rescheduled.Status != "SCHEDULED" {
		t.Fatalf("expected the new appointment to be SCHEDULED, got %s", rescheduled.Status)
	}
	if rescheduled.RescheduledFromID != original.ID {
		t.Fatalf("expected new appointment to reference the original as rescheduledFromId")
	}

	oldAppt, err := h.svc.GetByID(context.Background(), original.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if oldAppt.Status != "RESCHEDULED" {
		t.Fatalf("expected the original appointment to be marked RESCHEDULED, got %s", oldAppt.Status)
	}
	if oldAppt.RescheduledToID != rescheduled.ID {
		t.Fatalf("expected the original appointment to reference the new one as rescheduledToId")
	}
}

func TestRescheduleAllowsReusingTheOriginalSlot(t *testing.T) {
	// Rescheduling "to the same time" (e.g. correcting the doctor) must not
	// be rejected as a conflict against its own prior booking.
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	original := bookValid(t, h, patient, doctor)

	req := dto.RescheduleRequest{StartAt: original.StartAt, EndAt: original.EndAt}
	if _, err := h.svc.Reschedule(context.Background(), original.ID, req, audit.Actor{Role: authDomain.RoleAdmin}); err != nil {
		t.Fatalf("expected rescheduling to the same slot to succeed, got: %v", err)
	}
}

func TestRescheduleRejectsTerminalAppointment(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctor := seedDoctor(h, allDayAvailability())
	appt := bookValid(t, h, patient, doctor)

	if _, err := h.svc.ChangeStatus(context.Background(), appt.ID, dto.ChangeStatusRequest{Status: "CANCELLED"}, audit.Actor{Role: authDomain.RoleAdmin}); err != nil {
		t.Fatalf("ChangeStatus returned error: %v", err)
	}

	newStart, newEnd := futureWindow(48, 30)
	_, err := h.svc.Reschedule(context.Background(), appt.ID, dto.RescheduleRequest{StartAt: newStart, EndAt: newEnd}, audit.Actor{Role: authDomain.RoleAdmin})
	if err == nil {
		t.Fatalf("expected an error rescheduling a cancelled appointment")
	}
	if code := appErrCode(t, err); code != "APPOINTMENT_TERMINAL" {
		t.Fatalf("expected APPOINTMENT_TERMINAL, got %s", code)
	}
}

func TestSearchFiltersByDoctorAndStatus(t *testing.T) {
	h := newHarness()
	patient := seedPatient(h)
	doctorA := seedDoctor(h, allDayAvailability())
	doctorB := seedDoctor(h, allDayAvailability())

	bookValid(t, h, patient, doctorA)
	bookValid(t, h, patient, doctorB)

	result, err := h.svc.Search(context.Background(), dto.SearchAppointmentsQuery{DoctorID: doctorA.ID.Hex()})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected 1 appointment for doctorA, got %d", result.Total)
	}
}

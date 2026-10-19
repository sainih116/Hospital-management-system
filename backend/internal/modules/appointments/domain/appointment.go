// Package domain holds the appointments module's core entity and state
// machine.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status models the appointment lifecycle from the blueprint: a happy path
// of scheduled -> confirmed -> in-progress -> completed, with cancelled,
// no-show and rescheduled as terminal alternate paths.
type Status string

const (
	StatusScheduled   Status = "SCHEDULED"
	StatusConfirmed   Status = "CONFIRMED"
	StatusInProgress  Status = "IN_PROGRESS"
	StatusCompleted   Status = "COMPLETED"
	StatusCancelled   Status = "CANCELLED"
	StatusNoShow      Status = "NO_SHOW"
	StatusRescheduled Status = "RESCHEDULED"
)

// terminalStatuses no longer occupy the doctor's calendar — a new
// appointment can reuse the same slot.
var terminalStatuses = map[Status]bool{
	StatusCancelled:   true,
	StatusNoShow:      true,
	StatusRescheduled: true,
	StatusCompleted:   true,
}

// IsTerminal reports whether an appointment in this status can still
// transition to another status.
func (s Status) IsTerminal() bool {
	return terminalStatuses[s]
}

// HoldsCalendarSlot reports whether an appointment in this status still
// blocks its doctor/time slot from being double-booked. Completed
// appointments are terminal (no further transitions) but did legitimately
// happen, so — unlike cancelled/no-show/rescheduled — they still count as
// having occupied the slot.
func (s Status) HoldsCalendarSlot() bool {
	return s != StatusCancelled && s != StatusNoShow && s != StatusRescheduled
}

// allowedTransitions is the appointment state machine. Reschedule is
// handled as its own operation (it creates a new appointment document) —
// see service.Reschedule — rather than as a same-record transition.
var allowedTransitions = map[Status]map[Status]bool{
	StatusScheduled:  {StatusConfirmed: true, StatusCancelled: true, StatusNoShow: true},
	StatusConfirmed:  {StatusInProgress: true, StatusCancelled: true, StatusNoShow: true},
	StatusInProgress: {StatusCompleted: true, StatusCancelled: true},
}

// CanTransitionTo reports whether moving from s to next is a valid state
// machine transition.
func (s Status) CanTransitionTo(next Status) bool {
	return allowedTransitions[s][next]
}

// Appointment is a scheduled consultation between a patient and a doctor.
// PatientName/DoctorName are snapshotted at booking time (same
// join-free-reads tradeoff used by the doctors module) since neither
// patient nor doctor names can currently be edited after creation.
type Appointment struct {
	ID                primitive.ObjectID  `bson:"_id,omitempty"`
	PatientID         primitive.ObjectID  `bson:"patientId"`
	PatientName       string              `bson:"patientName"`
	DoctorID          primitive.ObjectID  `bson:"doctorId"`
	DoctorName        string              `bson:"doctorName"`
	StartAt           time.Time           `bson:"startAt"`
	EndAt             time.Time           `bson:"endAt"`
	Status            Status              `bson:"status"`
	Reason            string              `bson:"reason,omitempty"`
	Notes             string              `bson:"notes,omitempty"`
	CancelReason      string              `bson:"cancelReason,omitempty"`
	RescheduledFromID *primitive.ObjectID `bson:"rescheduledFromId,omitempty"`
	RescheduledToID   *primitive.ObjectID `bson:"rescheduledToId,omitempty"`
	CreatedBy         primitive.ObjectID  `bson:"createdBy,omitempty"`
	CreatedAt         time.Time           `bson:"createdAt"`
	UpdatedAt         time.Time           `bson:"updatedAt"`
}

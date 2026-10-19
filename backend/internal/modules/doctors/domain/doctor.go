// Package domain holds the doctors module's core entity. A Doctor is a
// clinical profile layered on top of a User account (created in Phase 1
// with role DOCTOR) — the profile carries department/specialization/fee/
// availability data that authentication has no business knowing about.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status tracks whether a doctor profile is currently active.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

// AvailabilitySlot is one recurring weekly availability window, e.g.
// "Monday 09:00-17:00". Dated exceptions (leave, on-call overrides) are
// intentionally out of scope here — Phase 4 (Appointments) revisits this
// if conflict-checking needs finer granularity than a weekly recurrence.
type AvailabilitySlot struct {
	// DayOfWeek follows time.Weekday: 0=Sunday .. 6=Saturday.
	DayOfWeek int    `bson:"dayOfWeek"`
	StartTime string `bson:"startTime"` // 24h "HH:MM"
	EndTime   string `bson:"endTime"`   // 24h "HH:MM"
}

// Doctor is a clinical staff profile. Name/Email are snapshotted from the
// linked User at creation time (the system has no user-profile-edit
// feature yet, so staleness risk is currently nil) to keep search/list
// reads join-free; DepartmentName is refreshed whenever the doctor's
// department assignment changes.
type Doctor struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         primitive.ObjectID `bson:"userId"`
	Name           string             `bson:"name"`
	Email          string             `bson:"email"`
	DepartmentID   primitive.ObjectID `bson:"departmentId"`
	DepartmentName string             `bson:"departmentName"`
	LicenseNo      string             `bson:"licenseNo"`
	Specialization string             `bson:"specialization"`
	Fee            float64            `bson:"fee"`
	Bio            string             `bson:"bio,omitempty"`
	Availability   []AvailabilitySlot `bson:"availability,omitempty"`
	Status         Status             `bson:"status"`
	CreatedAt      time.Time          `bson:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt"`
	DeletedAt      *time.Time         `bson:"deletedAt,omitempty"`
}

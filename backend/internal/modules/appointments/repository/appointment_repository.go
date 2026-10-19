// Package repository implements MongoDB persistence for the appointments
// module.
//
// Double-booking prevention: this deployment runs MongoDB as a standalone
// instance (see root docker-compose.yml), not a replica set, so
// multi-document ACID transactions (session.WithTransaction) are not
// available — they require a replica set or mongos. Two defenses are used
// instead, deliberately layered:
//
//  1. A pre-insert overlap query (HasConflict) checks for any other
//     calendar-holding appointment for the same doctor whose [start, end)
//     range intersects the requested one. This catches the general
//     overlapping-appointments case and is what Create calls first.
//  2. A partial unique index on (doctorId, startAt), scoped to
//     calendar-holding statuses, is the atomic backstop for the literal
//     race described in the blueprint's test case ("two users book the
//     same doctor/time slot simultaneously") — the exact scenario a
//     calendar UI offering discrete slots produces. Unique index
//     enforcement is atomic per document in MongoDB even without
//     transactions, so when two requests for the identical slot reach the
//     database at the same time, exactly one insert succeeds and the
//     other fails with a duplicate-key error, which Create translates to
//     ErrSlotTaken.
//
// A genuine race between two *overlapping-but-not-identical* start times
// (e.g. 10:00-10:45 vs 10:15-10:30 booked in the same instant) is not
// fully closed by this combination — only a replica-set transaction or an
// external lock service closes that window completely. That gap is
// accepted here as the pragmatic tradeoff for a single-node deployment;
// revisit if this ever runs against a replica set.
package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/appointments/domain"
	"hms-backend/internal/pkg/pagination"
)

// ErrNotFound is returned when a lookup finds no matching document.
var ErrNotFound = errors.New("repository: not found")

// ErrSlotTaken is returned when the doctor already has a calendar-holding
// appointment starting at the exact same time (see package doc).
var ErrSlotTaken = errors.New("repository: doctor already booked for this exact time slot")

// calendarHoldingStatuses lists the statuses that occupy a doctor's
// calendar — kept as a bson.A of strings (rather than reusing
// domain.Status) so it plugs directly into $nin/$in query clauses.
var calendarHoldingStatuses = bson.A{
	string(domain.StatusScheduled),
	string(domain.StatusConfirmed),
	string(domain.StatusInProgress),
	string(domain.StatusCompleted),
}

var nonCalendarHoldingStatuses = bson.A{
	string(domain.StatusCancelled),
	string(domain.StatusNoShow),
	string(domain.StatusRescheduled),
}

// SearchFilter narrows an appointment search. Empty/nil fields are not
// filtered on.
type SearchFilter struct {
	PatientID string
	DoctorID  string
	Status    string
	From      *time.Time
	To        *time.Time
}

// AppointmentRepository persists and retrieves appointments.
type AppointmentRepository interface {
	Create(ctx context.Context, appointment *domain.Appointment) error
	FindByID(ctx context.Context, id string) (*domain.Appointment, error)
	HasConflict(ctx context.Context, doctorID primitive.ObjectID, start, end time.Time, excludeID *primitive.ObjectID) (bool, error)
	UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Appointment, error)
	Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.Appointment, int64, error)
}

type appointmentRepository struct {
	collection *mongo.Collection
}

// NewAppointmentRepository builds a MongoDB-backed AppointmentRepository.
func NewAppointmentRepository(db *mongo.Database) AppointmentRepository {
	return &appointmentRepository{collection: db.Collection("appointments")}
}

// EnsureIndexes creates the indexes the appointments collection relies on.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("appointments").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// The double-booking backstop described in the package doc.
			Keys: bson.D{{Key: "doctorId", Value: 1}, {Key: "startAt", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("uniq_doctor_slot_active").
				SetPartialFilterExpression(bson.M{"status": bson.M{"$in": calendarHoldingStatuses}}),
		},
		{
			Keys:    bson.D{{Key: "doctorId", Value: 1}, {Key: "startAt", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_doctor_startAt_status"),
		},
		{
			Keys:    bson.D{{Key: "patientId", Value: 1}, {Key: "startAt", Value: 1}},
			Options: options.Index().SetName("idx_patient_startAt"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
	})
	return err
}

func (r *appointmentRepository) Create(ctx context.Context, appointment *domain.Appointment) error {
	now := time.Now().UTC()
	appointment.CreatedAt = now
	appointment.UpdatedAt = now
	if appointment.Status == "" {
		appointment.Status = domain.StatusScheduled
	}

	result, err := r.collection.InsertOne(ctx, appointment)
	if mongo.IsDuplicateKeyError(err) {
		return ErrSlotTaken
	}
	if err != nil {
		return err
	}
	appointment.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *appointmentRepository) FindByID(ctx context.Context, id string) (*domain.Appointment, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var appt domain.Appointment
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&appt)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &appt, nil
}

// HasConflict reports whether the doctor has any other calendar-holding
// appointment whose [start, end) range intersects [start, end). Uses the
// standard interval-overlap condition: existing.start < end AND
// existing.end > start.
func (r *appointmentRepository) HasConflict(ctx context.Context, doctorID primitive.ObjectID, start, end time.Time, excludeID *primitive.ObjectID) (bool, error) {
	filter := bson.M{
		"doctorId": doctorID,
		"status":   bson.M{"$nin": nonCalendarHoldingStatuses},
		"startAt":  bson.M{"$lt": end},
		"endAt":    bson.M{"$gt": start},
	}
	if excludeID != nil {
		filter["_id"] = bson.M{"$ne": *excludeID}
	}

	count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *appointmentRepository) UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Appointment, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var appt domain.Appointment
	if err := result.Decode(&appt); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &appt, nil
}

func (r *appointmentRepository) Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.Appointment, int64, error) {
	query := bson.M{}

	if filter.PatientID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.PatientID); err == nil {
			query["patientId"] = oid
		}
	}
	if filter.DoctorID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.DoctorID); err == nil {
			query["doctorId"] = oid
		}
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.From != nil || filter.To != nil {
		startAt := bson.M{}
		if filter.From != nil {
			startAt["$gte"] = *filter.From
		}
		if filter.To != nil {
			startAt["$lte"] = *filter.To
		}
		query["startAt"] = startAt
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "startAt", Value: 1}}).
		SetSkip(page.Skip()).
		SetLimit(page.Limit())

	cursor, err := r.collection.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var appointments []domain.Appointment
	if err := cursor.All(ctx, &appointments); err != nil {
		return nil, 0, err
	}

	return appointments, total, nil
}

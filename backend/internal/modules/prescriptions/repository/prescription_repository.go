// Package repository implements MongoDB persistence for the prescriptions
// module.
package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/prescriptions/domain"
	"hms-backend/internal/pkg/pagination"
)

// ErrNotFound is returned when a lookup finds no matching document.
var ErrNotFound = errors.New("repository: not found")

// SearchFilter narrows a prescription search. Empty fields are not
// filtered on.
type SearchFilter struct {
	PatientID string
	DoctorID  string
	Status    string
}

// PrescriptionRepository persists and retrieves prescriptions.
type PrescriptionRepository interface {
	Create(ctx context.Context, prescription *domain.Prescription) error
	FindByID(ctx context.Context, id string) (*domain.Prescription, error)
	UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Prescription, error)
	Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.Prescription, int64, error)
}

type prescriptionRepository struct {
	collection *mongo.Collection
}

// NewPrescriptionRepository builds a MongoDB-backed PrescriptionRepository.
func NewPrescriptionRepository(db *mongo.Database) PrescriptionRepository {
	return &prescriptionRepository{collection: db.Collection("prescriptions")}
}

// EnsureIndexes creates the indexes the prescriptions collection relies on.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("prescriptions").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "patientId", Value: 1}, {Key: "issuedAt", Value: -1}},
			Options: options.Index().SetName("idx_patient_issuedAt"),
		},
		{
			Keys:    bson.D{{Key: "doctorId", Value: 1}, {Key: "issuedAt", Value: -1}},
			Options: options.Index().SetName("idx_doctor_issuedAt"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
	})
	return err
}

func (r *prescriptionRepository) Create(ctx context.Context, prescription *domain.Prescription) error {
	now := time.Now().UTC()
	prescription.CreatedAt = now
	prescription.UpdatedAt = now
	if prescription.Status == "" {
		prescription.Status = domain.StatusActive
	}
	if prescription.IssuedAt.IsZero() {
		prescription.IssuedAt = now
	}

	result, err := r.collection.InsertOne(ctx, prescription)
	if err != nil {
		return err
	}
	prescription.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *prescriptionRepository) FindByID(ctx context.Context, id string) (*domain.Prescription, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var p domain.Prescription
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *prescriptionRepository) UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Prescription, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var p domain.Prescription
	if err := result.Decode(&p); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *prescriptionRepository) Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.Prescription, int64, error) {
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

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "issuedAt", Value: -1}}).
		SetSkip(page.Skip()).
		SetLimit(page.Limit())

	cursor, err := r.collection.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var prescriptions []domain.Prescription
	if err := cursor.All(ctx, &prescriptions); err != nil {
		return nil, 0, err
	}

	return prescriptions, total, nil
}

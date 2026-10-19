// Package repository implements MongoDB persistence for the medical
// records module.
package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/medicalrecords/domain"
	"hms-backend/internal/pkg/pagination"
)

// ErrNotFound is returned when a lookup finds no matching document.
var ErrNotFound = errors.New("repository: not found")

// SearchFilter narrows a medical record search. Empty fields are not
// filtered on.
type SearchFilter struct {
	PatientID string
	DoctorID  string
}

// MedicalRecordRepository persists and retrieves medical records.
type MedicalRecordRepository interface {
	Create(ctx context.Context, record *domain.MedicalRecord) error
	FindByID(ctx context.Context, id string) (*domain.MedicalRecord, error)
	Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.MedicalRecord, error)
	Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.MedicalRecord, int64, error)
}

type medicalRecordRepository struct {
	collection *mongo.Collection
}

// NewMedicalRecordRepository builds a MongoDB-backed MedicalRecordRepository.
func NewMedicalRecordRepository(db *mongo.Database) MedicalRecordRepository {
	return &medicalRecordRepository{collection: db.Collection("medical_records")}
}

// EnsureIndexes creates the indexes the medical_records collection relies on.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("medical_records").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "patientId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_patient_createdAt"),
		},
		{
			Keys:    bson.D{{Key: "doctorId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_doctor_createdAt"),
		},
	})
	return err
}

func (r *medicalRecordRepository) Create(ctx context.Context, record *domain.MedicalRecord) error {
	now := time.Now().UTC()
	record.CreatedAt = now
	record.UpdatedAt = now

	result, err := r.collection.InsertOne(ctx, record)
	if err != nil {
		return err
	}
	record.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *medicalRecordRepository) FindByID(ctx context.Context, id string) (*domain.MedicalRecord, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var record domain.MedicalRecord
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&record)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *medicalRecordRepository) Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.MedicalRecord, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var record domain.MedicalRecord
	if err := result.Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (r *medicalRecordRepository) Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.MedicalRecord, int64, error) {
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

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(page.Skip()).
		SetLimit(page.Limit())

	cursor, err := r.collection.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var records []domain.MedicalRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

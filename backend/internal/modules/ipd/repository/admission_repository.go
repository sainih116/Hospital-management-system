package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/ipd/domain"
	"hms-backend/internal/pkg/pagination"
)

// AdmissionFilter narrows an admission search. Empty fields are not
// filtered on.
type AdmissionFilter struct {
	PatientID string
	DoctorID  string
	Status    string
}

// AdmissionRepository persists and retrieves admissions.
type AdmissionRepository interface {
	Create(ctx context.Context, admission *domain.Admission) error
	FindByID(ctx context.Context, id string) (*domain.Admission, error)
	UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Admission, error)
	Search(ctx context.Context, filter AdmissionFilter, page pagination.Request) ([]domain.Admission, int64, error)
}

type admissionRepository struct {
	collection *mongo.Collection
}

// NewAdmissionRepository builds a MongoDB-backed AdmissionRepository.
func NewAdmissionRepository(db *mongo.Database) AdmissionRepository {
	return &admissionRepository{collection: db.Collection("admissions")}
}

// EnsureAdmissionIndexes creates the indexes the admissions collection
// relies on.
func EnsureAdmissionIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("admissions").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "patientId", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_patient_status"),
		},
		{
			Keys:    bson.D{{Key: "bedId", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_bed_status"),
		},
		{
			Keys:    bson.D{{Key: "doctorId", Value: 1}, {Key: "admittedAt", Value: -1}},
			Options: options.Index().SetName("idx_doctor_admittedAt"),
		},
	})
	return err
}

func (r *admissionRepository) Create(ctx context.Context, admission *domain.Admission) error {
	now := time.Now().UTC()
	admission.CreatedAt = now
	admission.UpdatedAt = now
	if admission.Status == "" {
		admission.Status = domain.AdmissionActive
	}
	if admission.AdmittedAt.IsZero() {
		admission.AdmittedAt = now
	}

	result, err := r.collection.InsertOne(ctx, admission)
	if err != nil {
		return err
	}
	admission.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *admissionRepository) FindByID(ctx context.Context, id string) (*domain.Admission, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var admission domain.Admission
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&admission)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &admission, nil
}

func (r *admissionRepository) UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Admission, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var admission domain.Admission
	if err := result.Decode(&admission); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &admission, nil
}

func (r *admissionRepository) Search(ctx context.Context, filter AdmissionFilter, page pagination.Request) ([]domain.Admission, int64, error) {
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
		SetSort(bson.D{{Key: "admittedAt", Value: -1}}).
		SetSkip(page.Skip()).
		SetLimit(page.Limit())

	cursor, err := r.collection.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var admissions []domain.Admission
	if err := cursor.All(ctx, &admissions); err != nil {
		return nil, 0, err
	}

	return admissions, total, nil
}

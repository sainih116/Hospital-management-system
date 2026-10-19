// Package repository implements MongoDB persistence for the patients
// module.
package repository

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/patients/domain"
	"hms-backend/internal/pkg/pagination"
)

// ErrNotFound is returned when a lookup finds no (non-deleted) document.
var ErrNotFound = errors.New("repository: not found")

// SearchFilter narrows a patient search. Empty fields are not filtered on.
type SearchFilter struct {
	Query      string // matched against name, patientCode and phone
	Gender     string
	BloodGroup string
	Type       string
}

// PatientRepository persists and retrieves patient records.
type PatientRepository interface {
	Create(ctx context.Context, patient *domain.Patient) error
	FindByID(ctx context.Context, id string) (*domain.Patient, error)
	FindByPatientCode(ctx context.Context, code string) (*domain.Patient, error)
	Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Patient, error)
	SoftDelete(ctx context.Context, id primitive.ObjectID) error
	Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.Patient, int64, error)
}

type patientRepository struct {
	collection *mongo.Collection
}

// NewPatientRepository builds a MongoDB-backed PatientRepository.
func NewPatientRepository(db *mongo.Database) PatientRepository {
	return &patientRepository{collection: db.Collection("patients")}
}

// EnsureIndexes creates the indexes the patients collection relies on.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("patients").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "patientCode", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_patientCode"),
		},
		{
			Keys:    bson.D{{Key: "phone", Value: 1}},
			Options: options.Index().SetName("idx_phone"),
		},
		{
			Keys:    bson.D{{Key: "name", Value: "text"}},
			Options: options.Index().SetName("txt_name"),
		},
		{
			Keys:    bson.D{{Key: "deletedAt", Value: 1}},
			Options: options.Index().SetName("idx_deletedAt"),
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_createdAt"),
		},
	})
	return err
}

func notDeleted() bson.M {
	return bson.M{"deletedAt": bson.M{"$exists": false}}
}

func (r *patientRepository) Create(ctx context.Context, patient *domain.Patient) error {
	now := time.Now().UTC()
	patient.CreatedAt = now
	patient.UpdatedAt = now
	if patient.Status == "" {
		patient.Status = domain.StatusActive
	}

	result, err := r.collection.InsertOne(ctx, patient)
	if err != nil {
		return err
	}
	patient.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *patientRepository) FindByID(ctx context.Context, id string) (*domain.Patient, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	filter := notDeleted()
	filter["_id"] = oid

	var patient domain.Patient
	err = r.collection.FindOne(ctx, filter).Decode(&patient)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &patient, nil
}

func (r *patientRepository) FindByPatientCode(ctx context.Context, code string) (*domain.Patient, error) {
	filter := notDeleted()
	filter["patientCode"] = code

	var patient domain.Patient
	err := r.collection.FindOne(ctx, filter).Decode(&patient)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &patient, nil
}

func (r *patientRepository) Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Patient, error) {
	set["updatedAt"] = time.Now().UTC()

	filter := notDeleted()
	filter["_id"] = id

	result := r.collection.FindOneAndUpdate(ctx, filter,
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var patient domain.Patient
	if err := result.Decode(&patient); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &patient, nil
}

func (r *patientRepository) SoftDelete(ctx context.Context, id primitive.ObjectID) error {
	filter := notDeleted()
	filter["_id"] = id

	now := time.Now().UTC()
	result, err := r.collection.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{"deletedAt": now, "status": domain.StatusInactive, "updatedAt": now},
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *patientRepository) Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.Patient, int64, error) {
	query := notDeleted()

	if filter.Gender != "" {
		query["gender"] = filter.Gender
	}
	if filter.BloodGroup != "" {
		query["bloodGroup"] = filter.BloodGroup
	}
	if filter.Type != "" {
		query["type"] = filter.Type
	}
	if filter.Query != "" {
		pattern := primitive.Regex{Pattern: regexp.QuoteMeta(filter.Query), Options: "i"}
		query["$or"] = bson.A{
			bson.M{"name": pattern},
			bson.M{"patientCode": pattern},
			bson.M{"phone": pattern},
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

	var patients []domain.Patient
	if err := cursor.All(ctx, &patients); err != nil {
		return nil, 0, err
	}

	return patients, total, nil
}

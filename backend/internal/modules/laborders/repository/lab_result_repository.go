package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/laborders/domain"
)

// ErrAlreadyExists is returned when a lab order already has a result
// (results are one-to-one with orders — a correction re-enters via the
// same result record, there is no "second result" concept).
var ErrAlreadyExists = errors.New("repository: lab order already has a result")

// LabResultRepository persists and retrieves lab results.
type LabResultRepository interface {
	Create(ctx context.Context, result *domain.LabResult) error
	FindByID(ctx context.Context, id string) (*domain.LabResult, error)
	FindByLabOrderID(ctx context.Context, labOrderID primitive.ObjectID) (*domain.LabResult, error)
	UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.LabResult, error)
}

type labResultRepository struct {
	collection *mongo.Collection
}

// NewLabResultRepository builds a MongoDB-backed LabResultRepository.
func NewLabResultRepository(db *mongo.Database) LabResultRepository {
	return &labResultRepository{collection: db.Collection("lab_results")}
}

func (r *labResultRepository) Create(ctx context.Context, result *domain.LabResult) error {
	now := time.Now().UTC()
	result.CreatedAt = now
	result.UpdatedAt = now
	if result.EnteredAt.IsZero() {
		result.EnteredAt = now
	}

	insertResult, err := r.collection.InsertOne(ctx, result)
	if mongo.IsDuplicateKeyError(err) {
		return ErrAlreadyExists
	}
	if err != nil {
		return err
	}
	result.ID = insertResult.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *labResultRepository) FindByID(ctx context.Context, id string) (*domain.LabResult, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var result domain.LabResult
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *labResultRepository) FindByLabOrderID(ctx context.Context, labOrderID primitive.ObjectID) (*domain.LabResult, error) {
	var result domain.LabResult
	err := r.collection.FindOne(ctx, bson.M{"labOrderId": labOrderID}).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *labResultRepository) UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.LabResult, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var labResult domain.LabResult
	if err := result.Decode(&labResult); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &labResult, nil
}

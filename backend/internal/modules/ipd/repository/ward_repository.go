// Package repository implements MongoDB persistence for the IPD module:
// wards, beds, and admissions.
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
)

// ErrNotFound is returned when a lookup finds no matching document.
var ErrNotFound = errors.New("repository: not found")

// ErrCodeTaken is returned when a ward code is already in use.
var ErrCodeTaken = errors.New("repository: ward code already exists")

// WardRepository persists and retrieves wards.
type WardRepository interface {
	Create(ctx context.Context, ward *domain.Ward) error
	FindByID(ctx context.Context, id string) (*domain.Ward, error)
	List(ctx context.Context) ([]domain.Ward, error)
	Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Ward, error)
}

type wardRepository struct {
	collection *mongo.Collection
}

// NewWardRepository builds a MongoDB-backed WardRepository.
func NewWardRepository(db *mongo.Database) WardRepository {
	return &wardRepository{collection: db.Collection("wards")}
}

// EnsureWardIndexes creates the indexes the wards collection relies on.
func EnsureWardIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("wards").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "code", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_code"),
	})
	return err
}

func (r *wardRepository) Create(ctx context.Context, ward *domain.Ward) error {
	now := time.Now().UTC()
	ward.CreatedAt = now
	ward.UpdatedAt = now
	if ward.Status == "" {
		ward.Status = domain.StatusActive
	}

	result, err := r.collection.InsertOne(ctx, ward)
	if mongo.IsDuplicateKeyError(err) {
		return ErrCodeTaken
	}
	if err != nil {
		return err
	}
	ward.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *wardRepository) FindByID(ctx context.Context, id string) (*domain.Ward, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var ward domain.Ward
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&ward)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ward, nil
}

func (r *wardRepository) List(ctx context.Context) ([]domain.Ward, error) {
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var wards []domain.Ward
	if err := cursor.All(ctx, &wards); err != nil {
		return nil, err
	}
	return wards, nil
}

func (r *wardRepository) Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Ward, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var ward domain.Ward
	if err := result.Decode(&ward); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ward, nil
}

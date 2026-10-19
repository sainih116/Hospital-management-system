// Package repository implements MongoDB persistence for the lab tests
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

	"hms-backend/internal/modules/labtests/domain"
)

// ErrNotFound is returned when a lookup finds no matching document.
var ErrNotFound = errors.New("repository: not found")

// ErrCodeTaken is returned when a test code is already in use.
var ErrCodeTaken = errors.New("repository: lab test code already exists")

// LabTestRepository persists and retrieves lab test catalog entries.
type LabTestRepository interface {
	Create(ctx context.Context, test *domain.LabTest) error
	FindByID(ctx context.Context, id string) (*domain.LabTest, error)
	FindByCode(ctx context.Context, code string) (*domain.LabTest, error)
	List(ctx context.Context) ([]domain.LabTest, error)
	Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.LabTest, error)
}

type labTestRepository struct {
	collection *mongo.Collection
}

// NewLabTestRepository builds a MongoDB-backed LabTestRepository.
func NewLabTestRepository(db *mongo.Database) LabTestRepository {
	return &labTestRepository{collection: db.Collection("lab_tests")}
}

// EnsureIndexes creates the indexes the lab_tests collection relies on.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("lab_tests").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "code", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_code"),
	})
	return err
}

func (r *labTestRepository) Create(ctx context.Context, test *domain.LabTest) error {
	now := time.Now().UTC()
	test.CreatedAt = now
	test.UpdatedAt = now
	if test.Status == "" {
		test.Status = domain.StatusActive
	}

	result, err := r.collection.InsertOne(ctx, test)
	if mongo.IsDuplicateKeyError(err) {
		return ErrCodeTaken
	}
	if err != nil {
		return err
	}
	test.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *labTestRepository) FindByID(ctx context.Context, id string) (*domain.LabTest, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var test domain.LabTest
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&test)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &test, nil
}

func (r *labTestRepository) FindByCode(ctx context.Context, code string) (*domain.LabTest, error) {
	var test domain.LabTest
	err := r.collection.FindOne(ctx, bson.M{"code": code}).Decode(&test)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &test, nil
}

func (r *labTestRepository) List(ctx context.Context) ([]domain.LabTest, error) {
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tests []domain.LabTest
	if err := cursor.All(ctx, &tests); err != nil {
		return nil, err
	}
	return tests, nil
}

func (r *labTestRepository) Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.LabTest, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var test domain.LabTest
	if err := result.Decode(&test); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &test, nil
}

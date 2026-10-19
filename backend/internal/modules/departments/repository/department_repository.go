// Package repository implements MongoDB persistence for the departments
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

	"hms-backend/internal/modules/departments/domain"
)

// ErrNotFound is returned when a lookup finds no matching document.
var ErrNotFound = errors.New("repository: not found")

// ErrCodeTaken is returned when a department code is already in use.
var ErrCodeTaken = errors.New("repository: department code already exists")

// DepartmentRepository persists and retrieves departments.
type DepartmentRepository interface {
	Create(ctx context.Context, department *domain.Department) error
	FindByID(ctx context.Context, id string) (*domain.Department, error)
	FindByCode(ctx context.Context, code string) (*domain.Department, error)
	List(ctx context.Context) ([]domain.Department, error)
	Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Department, error)
}

type departmentRepository struct {
	collection *mongo.Collection
}

// NewDepartmentRepository builds a MongoDB-backed DepartmentRepository.
func NewDepartmentRepository(db *mongo.Database) DepartmentRepository {
	return &departmentRepository{collection: db.Collection("departments")}
}

// EnsureIndexes creates the indexes the departments collection relies on.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("departments").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "code", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_code"),
	})
	return err
}

func (r *departmentRepository) Create(ctx context.Context, department *domain.Department) error {
	now := time.Now().UTC()
	department.CreatedAt = now
	department.UpdatedAt = now
	if department.Status == "" {
		department.Status = domain.StatusActive
	}

	result, err := r.collection.InsertOne(ctx, department)
	if mongo.IsDuplicateKeyError(err) {
		return ErrCodeTaken
	}
	if err != nil {
		return err
	}
	department.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *departmentRepository) FindByID(ctx context.Context, id string) (*domain.Department, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var department domain.Department
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&department)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &department, nil
}

func (r *departmentRepository) FindByCode(ctx context.Context, code string) (*domain.Department, error) {
	var department domain.Department
	err := r.collection.FindOne(ctx, bson.M{"code": code}).Decode(&department)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &department, nil
}

func (r *departmentRepository) List(ctx context.Context) ([]domain.Department, error) {
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var departments []domain.Department
	if err := cursor.All(ctx, &departments); err != nil {
		return nil, err
	}
	return departments, nil
}

func (r *departmentRepository) Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Department, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var department domain.Department
	if err := result.Decode(&department); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &department, nil
}

// Package repository implements MongoDB persistence for the auth module,
// exposed through interfaces so the service layer never depends on the
// driver directly.
package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/modules/auth/domain"
	"hms-backend/internal/pkg/pagination"
)

// ErrNotFound is returned by repository lookups that find no document.
var ErrNotFound = errors.New("repository: not found")

// UserRepository persists and retrieves staff user accounts.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	IsActiveUser(ctx context.Context, id string) (bool, error)
	UpdatePasswordHash(ctx context.Context, id primitive.ObjectID, passwordHash string) error
	UpdateLastLoginAt(ctx context.Context, id primitive.ObjectID, at time.Time) error
	List(ctx context.Context, roleCode string, page pagination.Request) ([]domain.User, int64, error)
}

type userRepository struct {
	collection *mongo.Collection
}

// NewUserRepository builds a MongoDB-backed UserRepository.
func NewUserRepository(db *mongo.Database) UserRepository {
	return &userRepository{collection: db.Collection("users")}
}

// EnsureUserIndexes creates the indexes the users collection relies on.
// Safe to call on every startup — index creation is idempotent.
func EnsureUserIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("users").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_email"),
		},
		{
			Keys:    bson.D{{Key: "roleCode", Value: 1}},
			Options: options.Index().SetName("idx_roleCode"),
		},
	})
	return err
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	result, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return err
	}
	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var user domain.User
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) IsActiveUser(ctx context.Context, id string) (bool, error) {
	user, err := r.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return user.IsActive(), nil
}

func (r *userRepository) UpdatePasswordHash(ctx context.Context, id primitive.ObjectID, passwordHash string) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"passwordHash": passwordHash, "updatedAt": time.Now().UTC()}},
	)
	return err
}

func (r *userRepository) UpdateLastLoginAt(ctx context.Context, id primitive.ObjectID, at time.Time) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"lastLoginAt": at, "updatedAt": time.Now().UTC()}},
	)
	return err
}

// List returns a page of users, optionally filtered by role. Used by
// Admin-only tooling (e.g. picking a DOCTOR-role account to attach a
// clinical profile to) — never exposed to non-Admin roles.
func (r *userRepository) List(ctx context.Context, roleCode string, page pagination.Request) ([]domain.User, int64, error) {
	filter := bson.M{}
	if roleCode != "" {
		filter["roleCode"] = roleCode
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "name", Value: 1}}).
		SetSkip(page.Skip()).
		SetLimit(page.Limit())

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []domain.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

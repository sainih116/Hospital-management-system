package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/auth/domain"
)

// RoleRepository persists and retrieves RBAC roles.
type RoleRepository interface {
	FindByCode(ctx context.Context, code string) (*domain.Role, error)
	List(ctx context.Context) ([]domain.Role, error)
	Upsert(ctx context.Context, role *domain.Role) error
}

type roleRepository struct {
	collection *mongo.Collection
}

// NewRoleRepository builds a MongoDB-backed RoleRepository.
func NewRoleRepository(db *mongo.Database) RoleRepository {
	return &roleRepository{collection: db.Collection("roles")}
}

// EnsureRoleIndexes creates the indexes the roles collection relies on.
func EnsureRoleIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("roles").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "code", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_code"),
	})
	return err
}

func (r *roleRepository) FindByCode(ctx context.Context, code string) (*domain.Role, error) {
	var role domain.Role
	err := r.collection.FindOne(ctx, bson.M{"code": code}).Decode(&role)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) List(ctx context.Context) ([]domain.Role, error) {
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "code", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var roles []domain.Role
	if err := cursor.All(ctx, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// Upsert creates or updates a role by code. Used by the seed command to
// keep the RBAC matrix in the database in sync with domain.DefaultRolePermissions.
func (r *roleRepository) Upsert(ctx context.Context, role *domain.Role) error {
	now := time.Now().UTC()
	role.UpdatedAt = now

	_, err := r.collection.UpdateOne(ctx,
		bson.M{"code": role.Code},
		bson.M{
			"$set": bson.M{
				"name":        role.Name,
				"permissions": role.Permissions,
				"updatedAt":   role.UpdatedAt,
			},
			"$setOnInsert": bson.M{"createdAt": now},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

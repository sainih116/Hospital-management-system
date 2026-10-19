package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/auth/domain"
)

// PasswordResetRepository persists single-use password reset tokens.
type PasswordResetRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	FindByHash(ctx context.Context, hash string) (*domain.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id primitive.ObjectID) error
}

type passwordResetRepository struct {
	collection *mongo.Collection
}

// NewPasswordResetRepository builds a MongoDB-backed PasswordResetRepository.
func NewPasswordResetRepository(db *mongo.Database) PasswordResetRepository {
	return &passwordResetRepository{collection: db.Collection("password_reset_tokens")}
}

// EnsurePasswordResetIndexes creates the indexes the password_reset_tokens
// collection relies on, including a TTL index for automatic cleanup.
func EnsurePasswordResetIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("password_reset_tokens").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "tokenHash", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_tokenHash"),
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("ttl_expiresAt"),
		},
	})
	return err
}

func (r *passwordResetRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	token.CreatedAt = time.Now().UTC()

	result, err := r.collection.InsertOne(ctx, token)
	if err != nil {
		return err
	}
	token.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *passwordResetRepository) FindByHash(ctx context.Context, hash string) (*domain.PasswordResetToken, error) {
	var token domain.PasswordResetToken
	err := r.collection.FindOne(ctx, bson.M{"tokenHash": hash}).Decode(&token)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *passwordResetRepository) MarkUsed(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now().UTC()
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"usedAt": now}},
	)
	return err
}

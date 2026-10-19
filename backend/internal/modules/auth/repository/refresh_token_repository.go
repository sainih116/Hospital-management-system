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

// RefreshTokenRepository persists hashed refresh tokens for session
// rotation and revocation.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, id primitive.ObjectID, replacedByHash string) error
	RevokeAllForUser(ctx context.Context, userID primitive.ObjectID) error
}

type refreshTokenRepository struct {
	collection *mongo.Collection
}

// NewRefreshTokenRepository builds a MongoDB-backed RefreshTokenRepository.
func NewRefreshTokenRepository(db *mongo.Database) RefreshTokenRepository {
	return &refreshTokenRepository{collection: db.Collection("refresh_tokens")}
}

// EnsureRefreshTokenIndexes creates the indexes the refresh_tokens
// collection relies on, including a TTL index that lets MongoDB
// automatically purge expired sessions.
func EnsureRefreshTokenIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("refresh_tokens").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "tokenHash", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_tokenHash"),
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("idx_userId"),
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("ttl_expiresAt"),
		},
	})
	return err
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	token.CreatedAt = time.Now().UTC()

	result, err := r.collection.InsertOne(ctx, token)
	if err != nil {
		return err
	}
	token.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *refreshTokenRepository) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	err := r.collection.FindOne(ctx, bson.M{"tokenHash": hash}).Decode(&token)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id primitive.ObjectID, replacedByHash string) error {
	now := time.Now().UTC()
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"revokedAt": now, "replacedByHash": replacedByHash}},
	)
	return err
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID primitive.ObjectID) error {
	now := time.Now().UTC()
	_, err := r.collection.UpdateMany(ctx,
		bson.M{"userId": userID, "revokedAt": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"revokedAt": now}},
	)
	return err
}

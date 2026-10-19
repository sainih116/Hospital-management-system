package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// EnsureIndexes creates every index the auth module's collections rely on.
// Index creation is idempotent, so this is safe to call on every startup.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if err := EnsureUserIndexes(ctx, db); err != nil {
		return fmt.Errorf("auth: ensure user indexes: %w", err)
	}
	if err := EnsureRoleIndexes(ctx, db); err != nil {
		return fmt.Errorf("auth: ensure role indexes: %w", err)
	}
	if err := EnsureRefreshTokenIndexes(ctx, db); err != nil {
		return fmt.Errorf("auth: ensure refresh token indexes: %w", err)
	}
	if err := EnsurePasswordResetIndexes(ctx, db); err != nil {
		return fmt.Errorf("auth: ensure password reset indexes: %w", err)
	}
	return nil
}

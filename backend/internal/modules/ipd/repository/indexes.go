package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// EnsureIndexes creates every index the IPD module's collections rely on.
// Idempotent — safe to call on every startup.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if err := EnsureWardIndexes(ctx, db); err != nil {
		return fmt.Errorf("ipd: ensure ward indexes: %w", err)
	}
	if err := EnsureBedIndexes(ctx, db); err != nil {
		return fmt.Errorf("ipd: ensure bed indexes: %w", err)
	}
	if err := EnsureAdmissionIndexes(ctx, db); err != nil {
		return fmt.Errorf("ipd: ensure admission indexes: %w", err)
	}
	return nil
}

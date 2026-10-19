package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// EnsureIndexes creates every index the pharmacy module's collections
// rely on. Idempotent — safe to call on every startup.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if err := EnsureMedicineIndexes(ctx, db); err != nil {
		return fmt.Errorf("pharmacy: ensure medicine indexes: %w", err)
	}
	if err := EnsureBatchIndexes(ctx, db); err != nil {
		return fmt.Errorf("pharmacy: ensure batch indexes: %w", err)
	}
	if err := EnsureStockMovementIndexes(ctx, db); err != nil {
		return fmt.Errorf("pharmacy: ensure stock movement indexes: %w", err)
	}
	return nil
}

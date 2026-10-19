package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// EnsureIndexes creates every index the billing module's collections rely
// on. Idempotent — safe to call on every startup.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if err := EnsureBillIndexes(ctx, db); err != nil {
		return fmt.Errorf("billing: ensure bill indexes: %w", err)
	}
	if err := EnsurePaymentIndexes(ctx, db); err != nil {
		return fmt.Errorf("billing: ensure payment indexes: %w", err)
	}
	return nil
}

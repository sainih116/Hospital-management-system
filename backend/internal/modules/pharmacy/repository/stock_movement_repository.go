package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/pharmacy/domain"
	"hms-backend/internal/pkg/pagination"
)

// StockMovementFilter narrows a stock movement search. Empty fields are
// not filtered on.
type StockMovementFilter struct {
	MedicineID string
	Type       string
}

// StockMovementRepository persists and retrieves the stock ledger.
type StockMovementRepository interface {
	Create(ctx context.Context, movement *domain.StockMovement) error
	Search(ctx context.Context, filter StockMovementFilter, page pagination.Request) ([]domain.StockMovement, int64, error)
}

type stockMovementRepository struct {
	collection *mongo.Collection
}

// NewStockMovementRepository builds a MongoDB-backed StockMovementRepository.
func NewStockMovementRepository(db *mongo.Database) StockMovementRepository {
	return &stockMovementRepository{collection: db.Collection("stock_movements")}
}

// EnsureStockMovementIndexes creates the indexes the stock_movements
// collection relies on.
func EnsureStockMovementIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("stock_movements").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "medicineId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_medicine_createdAt"),
		},
		{
			Keys:    bson.D{{Key: "prescriptionId", Value: 1}},
			Options: options.Index().SetName("idx_prescriptionId"),
		},
	})
	return err
}

func (r *stockMovementRepository) Create(ctx context.Context, movement *domain.StockMovement) error {
	if movement.CreatedAt.IsZero() {
		movement.CreatedAt = time.Now().UTC()
	}

	result, err := r.collection.InsertOne(ctx, movement)
	if err != nil {
		return err
	}
	movement.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *stockMovementRepository) Search(ctx context.Context, filter StockMovementFilter, page pagination.Request) ([]domain.StockMovement, int64, error) {
	query := bson.M{}

	if filter.MedicineID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.MedicineID); err == nil {
			query["medicineId"] = oid
		}
	}
	if filter.Type != "" {
		query["type"] = filter.Type
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(page.Skip()).
		SetLimit(page.Limit())

	cursor, err := r.collection.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var movements []domain.StockMovement
	if err := cursor.All(ctx, &movements); err != nil {
		return nil, 0, err
	}

	return movements, total, nil
}

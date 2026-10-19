package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/pharmacy/domain"
)

// ErrInsufficientBatchStock is returned when a DecrementQuantity call
// would take a batch's quantity below zero — the same atomic-guard
// pattern as MedicineRepository.AdjustStock, applied per-batch so FEFO
// dispensing can never oversell an individual batch even under
// concurrent requests.
var ErrInsufficientBatchStock = errors.New("repository: insufficient batch stock")

// BatchRepository persists and retrieves inventory batches. The
// collection is named "inventory", matching the phase's own collection
// list (the blueprint's DB design calls the equivalent table
// "medicine_batches" — same concept, this phase's naming wins since it's
// what's actually being built against).
type BatchRepository interface {
	Create(ctx context.Context, batch *domain.Batch) error
	FindByID(ctx context.Context, id string) (*domain.Batch, error)
	// FindAvailableForMedicine returns non-expired batches with positive
	// quantity, sorted oldest-expiry-first (FEFO order).
	FindAvailableForMedicine(ctx context.Context, medicineID primitive.ObjectID, now time.Time) ([]domain.Batch, error)
	FindByMedicineID(ctx context.Context, medicineID primitive.ObjectID) ([]domain.Batch, error)
	FindExpiringBefore(ctx context.Context, cutoff time.Time) ([]domain.Batch, error)
	// DecrementQuantity only applies if the batch has at least `amount`
	// remaining; otherwise it returns ErrInsufficientBatchStock.
	DecrementQuantity(ctx context.Context, id primitive.ObjectID, amount int) (*domain.Batch, error)
	IncrementQuantity(ctx context.Context, id primitive.ObjectID, amount int) error
}

type batchRepository struct {
	collection *mongo.Collection
}

// NewBatchRepository builds a MongoDB-backed BatchRepository.
func NewBatchRepository(db *mongo.Database) BatchRepository {
	return &batchRepository{collection: db.Collection("inventory")}
}

// EnsureBatchIndexes creates the indexes the inventory collection relies on.
func EnsureBatchIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("inventory").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "medicineId", Value: 1}, {Key: "expiryDate", Value: 1}},
			Options: options.Index().SetName("idx_medicine_expiry"),
		},
		{
			Keys:    bson.D{{Key: "batchNo", Value: 1}},
			Options: options.Index().SetName("idx_batchNo"),
		},
	})
	return err
}

func (r *batchRepository) Create(ctx context.Context, batch *domain.Batch) error {
	now := time.Now().UTC()
	batch.CreatedAt = now
	batch.UpdatedAt = now

	result, err := r.collection.InsertOne(ctx, batch)
	if err != nil {
		return err
	}
	batch.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *batchRepository) FindByID(ctx context.Context, id string) (*domain.Batch, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var batch domain.Batch
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&batch)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (r *batchRepository) FindAvailableForMedicine(ctx context.Context, medicineID primitive.ObjectID, now time.Time) ([]domain.Batch, error) {
	filter := bson.M{
		"medicineId": medicineID,
		"quantity":   bson.M{"$gt": 0},
		"expiryDate": bson.M{"$gt": now},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "expiryDate", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var batches []domain.Batch
	if err := cursor.All(ctx, &batches); err != nil {
		return nil, err
	}
	return batches, nil
}

func (r *batchRepository) FindByMedicineID(ctx context.Context, medicineID primitive.ObjectID) ([]domain.Batch, error) {
	cursor, err := r.collection.Find(ctx,
		bson.M{"medicineId": medicineID},
		options.Find().SetSort(bson.D{{Key: "expiryDate", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var batches []domain.Batch
	if err := cursor.All(ctx, &batches); err != nil {
		return nil, err
	}
	return batches, nil
}

func (r *batchRepository) FindExpiringBefore(ctx context.Context, cutoff time.Time) ([]domain.Batch, error) {
	filter := bson.M{
		"quantity":   bson.M{"$gt": 0},
		"expiryDate": bson.M{"$lte": cutoff},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "expiryDate", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var batches []domain.Batch
	if err := cursor.All(ctx, &batches); err != nil {
		return nil, err
	}
	return batches, nil
}

func (r *batchRepository) DecrementQuantity(ctx context.Context, id primitive.ObjectID, amount int) (*domain.Batch, error) {
	filter := bson.M{"_id": id, "quantity": bson.M{"$gte": amount}}

	result := r.collection.FindOneAndUpdate(ctx, filter,
		bson.M{"$inc": bson.M{"quantity": -amount}, "$set": bson.M{"updatedAt": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var batch domain.Batch
	err := result.Decode(&batch)
	if errors.Is(err, mongo.ErrNoDocuments) {
		if _, findErr := r.FindByID(ctx, id.Hex()); findErr == nil {
			return nil, ErrInsufficientBatchStock
		}
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (r *batchRepository) IncrementQuantity(ctx context.Context, id primitive.ObjectID, amount int) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$inc": bson.M{"quantity": amount}, "$set": bson.M{"updatedAt": time.Now().UTC()}},
	)
	return err
}

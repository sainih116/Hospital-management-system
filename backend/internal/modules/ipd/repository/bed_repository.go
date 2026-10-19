package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/ipd/domain"
)

// ErrBedNotAvailable is returned when Occupy is attempted on a bed that
// isn't currently AVAILABLE — the atomic guard against double-assigning a
// bed to two active admissions, the same $gte-guard-filter pattern
// Pharmacy's AdjustStock uses to prevent overselling.
var ErrBedNotAvailable = errors.New("repository: bed is not available")

// BedFilter narrows a bed listing. Empty fields are not filtered on.
type BedFilter struct {
	WardID string
	Status string
}

// BedRepository persists and retrieves beds.
type BedRepository interface {
	Create(ctx context.Context, bed *domain.Bed) error
	FindByID(ctx context.Context, id string) (*domain.Bed, error)
	List(ctx context.Context, filter BedFilter) ([]domain.Bed, error)
	// Occupy atomically transitions a bed from AVAILABLE to OCCUPIED. It
	// only succeeds if the bed is still AVAILABLE at the moment of the
	// update — concurrent admission attempts for the same bed can never
	// both succeed, closing the exact race the blueprint's critical
	// constraint calls out.
	Occupy(ctx context.Context, id primitive.ObjectID) (*domain.Bed, error)
	// Release transitions a bed back to AVAILABLE — called when an
	// admission is discharged or cancelled.
	Release(ctx context.Context, id primitive.ObjectID) error
	SetStatus(ctx context.Context, id primitive.ObjectID, status domain.BedStatus) (*domain.Bed, error)
}

type bedRepository struct {
	collection *mongo.Collection
}

// NewBedRepository builds a MongoDB-backed BedRepository.
func NewBedRepository(db *mongo.Database) BedRepository {
	return &bedRepository{collection: db.Collection("beds")}
}

// EnsureBedIndexes creates the indexes the beds collection relies on.
func EnsureBedIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("beds").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "wardId", Value: 1}, {Key: "bedNumber", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_ward_bedNumber"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
	})
	return err
}

func (r *bedRepository) Create(ctx context.Context, bed *domain.Bed) error {
	now := time.Now().UTC()
	bed.CreatedAt = now
	bed.UpdatedAt = now
	if bed.Status == "" {
		bed.Status = domain.BedAvailable
	}

	result, err := r.collection.InsertOne(ctx, bed)
	if err != nil {
		return err
	}
	bed.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *bedRepository) FindByID(ctx context.Context, id string) (*domain.Bed, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var bed domain.Bed
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&bed)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &bed, nil
}

func (r *bedRepository) List(ctx context.Context, filter BedFilter) ([]domain.Bed, error) {
	query := bson.M{}
	if filter.WardID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.WardID); err == nil {
			query["wardId"] = oid
		}
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}

	cursor, err := r.collection.Find(ctx, query, options.Find().SetSort(bson.D{{Key: "wardName", Value: 1}, {Key: "bedNumber", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var beds []domain.Bed
	if err := cursor.All(ctx, &beds); err != nil {
		return nil, err
	}
	return beds, nil
}

func (r *bedRepository) Occupy(ctx context.Context, id primitive.ObjectID) (*domain.Bed, error) {
	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id, "status": string(domain.BedAvailable)},
		bson.M{"$set": bson.M{"status": string(domain.BedOccupied), "updatedAt": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var bed domain.Bed
	err := result.Decode(&bed)
	if errors.Is(err, mongo.ErrNoDocuments) {
		if _, findErr := r.FindByID(ctx, id.Hex()); findErr == nil {
			return nil, ErrBedNotAvailable
		}
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &bed, nil
}

func (r *bedRepository) Release(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"status": string(domain.BedAvailable), "updatedAt": time.Now().UTC()}},
	)
	return err
}

func (r *bedRepository) SetStatus(ctx context.Context, id primitive.ObjectID, status domain.BedStatus) (*domain.Bed, error) {
	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"status": string(status), "updatedAt": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var bed domain.Bed
	if err := result.Decode(&bed); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &bed, nil
}

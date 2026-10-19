// Package repository implements MongoDB persistence for the pharmacy
// module: medicines, inventory (batches), and the stock-movement ledger.
package repository

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/pharmacy/domain"
	"hms-backend/internal/pkg/pagination"
)

// ErrNotFound is returned when a lookup finds no matching document.
var ErrNotFound = errors.New("repository: not found")

// ErrInsufficientStock is returned when an AdjustStock call would take a
// medicine's current stock below zero — the atomic guard against
// overselling described in the domain package doc.
var ErrInsufficientStock = errors.New("repository: insufficient stock")

// MedicineSearchFilter narrows a medicine search. Empty/zero fields are
// not filtered on.
type MedicineSearchFilter struct {
	Query    string
	LowStock bool
}

// MedicineRepository persists and retrieves the medicine catalog.
type MedicineRepository interface {
	Create(ctx context.Context, medicine *domain.Medicine) error
	FindByID(ctx context.Context, id string) (*domain.Medicine, error)
	FindByName(ctx context.Context, name string) (*domain.Medicine, error)
	Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Medicine, error)
	Search(ctx context.Context, filter MedicineSearchFilter, page pagination.Request) ([]domain.Medicine, int64, error)
	// AdjustStock atomically changes CurrentStock by delta (positive for a
	// received batch, negative for a dispense) and, for negative deltas,
	// only applies if the result would stay >= 0 — otherwise it returns
	// ErrInsufficientStock without modifying anything.
	AdjustStock(ctx context.Context, id primitive.ObjectID, delta int) (*domain.Medicine, error)
}

type medicineRepository struct {
	collection *mongo.Collection
}

// NewMedicineRepository builds a MongoDB-backed MedicineRepository.
func NewMedicineRepository(db *mongo.Database) MedicineRepository {
	return &medicineRepository{collection: db.Collection("medicines")}
}

// EnsureMedicineIndexes creates the indexes the medicines collection
// relies on.
func EnsureMedicineIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("medicines").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: "text"}, {Key: "genericName", Value: "text"}},
			Options: options.Index().SetName("txt_name_genericName"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
	})
	return err
}

func (r *medicineRepository) Create(ctx context.Context, medicine *domain.Medicine) error {
	now := time.Now().UTC()
	medicine.CreatedAt = now
	medicine.UpdatedAt = now
	if medicine.Status == "" {
		medicine.Status = domain.StatusActive
	}

	result, err := r.collection.InsertOne(ctx, medicine)
	if err != nil {
		return err
	}
	medicine.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *medicineRepository) FindByID(ctx context.Context, id string) (*domain.Medicine, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var medicine domain.Medicine
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&medicine)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &medicine, nil
}

func (r *medicineRepository) FindByName(ctx context.Context, name string) (*domain.Medicine, error) {
	var medicine domain.Medicine
	err := r.collection.FindOne(ctx, bson.M{"name": name}).Decode(&medicine)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &medicine, nil
}

func (r *medicineRepository) Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Medicine, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var medicine domain.Medicine
	if err := result.Decode(&medicine); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &medicine, nil
}

func (r *medicineRepository) Search(ctx context.Context, filter MedicineSearchFilter, page pagination.Request) ([]domain.Medicine, int64, error) {
	query := bson.M{}

	if filter.Query != "" {
		pattern := primitive.Regex{Pattern: regexp.QuoteMeta(filter.Query), Options: "i"}
		query["$or"] = bson.A{
			bson.M{"name": pattern},
			bson.M{"genericName": pattern},
		}
	}
	if filter.LowStock {
		query["$expr"] = bson.M{"$lte": bson.A{"$currentStock", "$reorderLevel"}}
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "name", Value: 1}}).
		SetSkip(page.Skip()).
		SetLimit(page.Limit())

	cursor, err := r.collection.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var medicines []domain.Medicine
	if err := cursor.All(ctx, &medicines); err != nil {
		return nil, 0, err
	}

	return medicines, total, nil
}

func (r *medicineRepository) AdjustStock(ctx context.Context, id primitive.ObjectID, delta int) (*domain.Medicine, error) {
	filter := bson.M{"_id": id}
	if delta < 0 {
		filter["currentStock"] = bson.M{"$gte": -delta}
	}

	result := r.collection.FindOneAndUpdate(ctx, filter,
		bson.M{"$inc": bson.M{"currentStock": delta}, "$set": bson.M{"updatedAt": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var medicine domain.Medicine
	err := result.Decode(&medicine)
	if errors.Is(err, mongo.ErrNoDocuments) {
		if delta < 0 {
			// Distinguish "medicine doesn't exist" from "not enough stock"
			// by re-checking existence without the stock guard.
			if _, findErr := r.FindByID(ctx, id.Hex()); findErr == nil {
				return nil, ErrInsufficientStock
			}
		}
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &medicine, nil
}

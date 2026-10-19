// Package repository implements MongoDB persistence for the lab orders
// module. Orders and results are separate collections (matching the
// blueprint's DB design) even though the module treats them as one
// workflow at the domain/service layer.
package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/laborders/domain"
	"hms-backend/internal/pkg/pagination"
)

// ErrNotFound is returned when a lookup finds no matching document.
var ErrNotFound = errors.New("repository: not found")

// SearchFilter narrows a lab order search. Empty fields are not filtered
// on.
type SearchFilter struct {
	PatientID string
	DoctorID  string
	Status    string
}

// LabOrderRepository persists and retrieves lab orders.
type LabOrderRepository interface {
	Create(ctx context.Context, order *domain.LabOrder) error
	FindByID(ctx context.Context, id string) (*domain.LabOrder, error)
	UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.LabOrder, error)
	Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.LabOrder, int64, error)
}

type labOrderRepository struct {
	collection *mongo.Collection
}

// NewLabOrderRepository builds a MongoDB-backed LabOrderRepository.
func NewLabOrderRepository(db *mongo.Database) LabOrderRepository {
	return &labOrderRepository{collection: db.Collection("lab_orders")}
}

// EnsureIndexes creates the indexes the lab_orders and lab_results
// collections rely on.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if _, err := db.Collection("lab_orders").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "patientId", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_patient_status"),
		},
		{
			Keys:    bson.D{{Key: "doctorId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_doctor_createdAt"),
		},
		{
			Keys:    bson.D{{Key: "sampleCode", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_sampleCode"),
		},
	}); err != nil {
		return err
	}

	_, err := db.Collection("lab_results").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "labOrderId", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_labOrderId"),
	})
	return err
}

func (r *labOrderRepository) Create(ctx context.Context, order *domain.LabOrder) error {
	now := time.Now().UTC()
	order.CreatedAt = now
	order.UpdatedAt = now
	if order.Status == "" {
		order.Status = domain.StatusOrdered
	}

	result, err := r.collection.InsertOne(ctx, order)
	if err != nil {
		return err
	}
	order.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *labOrderRepository) FindByID(ctx context.Context, id string) (*domain.LabOrder, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var order domain.LabOrder
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&order)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *labOrderRepository) UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.LabOrder, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var order domain.LabOrder
	if err := result.Decode(&order); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (r *labOrderRepository) Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.LabOrder, int64, error) {
	query := bson.M{}

	if filter.PatientID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.PatientID); err == nil {
			query["patientId"] = oid
		}
	}
	if filter.DoctorID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.DoctorID); err == nil {
			query["doctorId"] = oid
		}
	}
	if filter.Status != "" {
		query["status"] = filter.Status
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

	var orders []domain.LabOrder
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// Package repository implements MongoDB persistence for the billing
// module. Bills and payments are separate collections (matching the
// pattern already used for Lab Orders/Lab Results) even though the module
// treats them as one workflow at the domain/service layer.
package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/billing/domain"
	"hms-backend/internal/pkg/pagination"
)

// ErrNotFound is returned when a lookup finds no matching document.
var ErrNotFound = errors.New("repository: not found")

// ErrOverpayment is returned when RecordPayment would take a bill's
// AmountPaid above its TotalAmount.
var ErrOverpayment = errors.New("repository: payment exceeds outstanding balance")

// SearchFilter narrows a bill search. Empty fields are not filtered on.
type SearchFilter struct {
	PatientID       string
	Status          string
	OutstandingOnly bool
}

// BillRepository persists and retrieves bills.
type BillRepository interface {
	Create(ctx context.Context, bill *domain.Bill) error
	FindByID(ctx context.Context, id string) (*domain.Bill, error)
	UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Bill, error)
	Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.Bill, int64, error)
	// RecordPayment atomically increments AmountPaid by amount and, only if
	// the result would stay <= TotalAmount, applies it — otherwise it
	// returns ErrOverpayment without modifying anything. The same
	// $gte/$lte-guard-filter pattern Pharmacy's AdjustStock uses to prevent
	// overselling.
	RecordPayment(ctx context.Context, id primitive.ObjectID, amount float64) (*domain.Bill, error)
}

type billRepository struct {
	collection *mongo.Collection
}

// NewBillRepository builds a MongoDB-backed BillRepository.
func NewBillRepository(db *mongo.Database) BillRepository {
	return &billRepository{collection: db.Collection("bills")}
}

// EnsureBillIndexes creates the indexes the bills collection relies on.
func EnsureBillIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("bills").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "patientId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_patient_createdAt"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			Keys:    bson.D{{Key: "billNo", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_billNo"),
		},
	})
	return err
}

func (r *billRepository) Create(ctx context.Context, bill *domain.Bill) error {
	now := time.Now().UTC()
	bill.CreatedAt = now
	bill.UpdatedAt = now
	if bill.Status == "" {
		bill.Status = domain.StatusUnpaid
	}

	result, err := r.collection.InsertOne(ctx, bill)
	if err != nil {
		return err
	}
	bill.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *billRepository) FindByID(ctx context.Context, id string) (*domain.Bill, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var bill domain.Bill
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&bill)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &bill, nil
}

func (r *billRepository) UpdateFields(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Bill, error) {
	set["updatedAt"] = time.Now().UTC()

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var bill domain.Bill
	if err := result.Decode(&bill); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &bill, nil
}

func (r *billRepository) Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.Bill, int64, error) {
	query := bson.M{}

	if filter.PatientID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.PatientID); err == nil {
			query["patientId"] = oid
		}
	}
	if filter.OutstandingOnly {
		query["status"] = bson.M{"$in": bson.A{string(domain.StatusUnpaid), string(domain.StatusPartiallyPaid)}}
	} else if filter.Status != "" {
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

	var bills []domain.Bill
	if err := cursor.All(ctx, &bills); err != nil {
		return nil, 0, err
	}

	return bills, total, nil
}

func (r *billRepository) RecordPayment(ctx context.Context, id primitive.ObjectID, amount float64) (*domain.Bill, error) {
	filter := bson.M{
		"_id": id,
		"$expr": bson.M{
			"$lte": bson.A{bson.M{"$add": bson.A{"$amountPaid", amount}}, "$totalAmount"},
		},
	}

	result := r.collection.FindOneAndUpdate(ctx, filter,
		bson.M{"$inc": bson.M{"amountPaid": amount}, "$set": bson.M{"updatedAt": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var bill domain.Bill
	err := result.Decode(&bill)
	if errors.Is(err, mongo.ErrNoDocuments) {
		if _, findErr := r.FindByID(ctx, id.Hex()); findErr == nil {
			return nil, ErrOverpayment
		}
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &bill, nil
}

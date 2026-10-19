package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/billing/domain"
)

// PaymentRepository persists and retrieves payments.
type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	FindByBillID(ctx context.Context, billID primitive.ObjectID) ([]domain.Payment, error)
}

type paymentRepository struct {
	collection *mongo.Collection
}

// NewPaymentRepository builds a MongoDB-backed PaymentRepository.
func NewPaymentRepository(db *mongo.Database) PaymentRepository {
	return &paymentRepository{collection: db.Collection("payments")}
}

// EnsurePaymentIndexes creates the indexes the payments collection relies
// on.
func EnsurePaymentIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("payments").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "billId", Value: 1}, {Key: "paidAt", Value: -1}},
		Options: options.Index().SetName("idx_bill_paidAt"),
	})
	return err
}

func (r *paymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	now := time.Now().UTC()
	payment.CreatedAt = now
	if payment.PaidAt.IsZero() {
		payment.PaidAt = now
	}

	result, err := r.collection.InsertOne(ctx, payment)
	if err != nil {
		return err
	}
	payment.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *paymentRepository) FindByBillID(ctx context.Context, billID primitive.ObjectID) ([]domain.Payment, error) {
	findOpts := options.Find().SetSort(bson.D{{Key: "paidAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{"billId": billID}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var payments []domain.Payment
	if err := cursor.All(ctx, &payments); err != nil {
		return nil, err
	}
	return payments, nil
}

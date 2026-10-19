package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Method is how a payment was received.
type Method string

const (
	MethodCash         Method = "CASH"
	MethodCard         Method = "CARD"
	MethodUPI          Method = "UPI"
	MethodBankTransfer Method = "BANK_TRANSFER"
	MethodInsurance    Method = "INSURANCE"
)

// Payment is one amount received against a Bill. A bill can have several
// payments (partial payments over time), so this is a separate collection
// rather than an embedded array — the same "tightly-coupled workflow,
// separate collections" split Lab Orders uses for its Results.
type Payment struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	BillID         primitive.ObjectID `bson:"billId"`
	Amount         float64            `bson:"amount"`
	Method         Method             `bson:"method"`
	ReferenceNo    string             `bson:"referenceNo,omitempty"`
	ReceivedBy     primitive.ObjectID `bson:"receivedBy,omitempty"`
	ReceivedByName string             `bson:"receivedByName,omitempty"`
	PaidAt         time.Time          `bson:"paidAt"`
	CreatedAt      time.Time          `bson:"createdAt"`
}

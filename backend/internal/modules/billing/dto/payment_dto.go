package dto

import "time"

// RecordPaymentRequest records a payment against a bill.
type RecordPaymentRequest struct {
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Method      string  `json:"method" binding:"required,oneof=CASH CARD UPI BANK_TRANSFER INSURANCE"`
	ReferenceNo string  `json:"referenceNo" binding:"max=100"`
}

// PaymentResponse is the public representation of one recorded payment.
type PaymentResponse struct {
	ID             string    `json:"id"`
	BillID         string    `json:"billId"`
	Amount         float64   `json:"amount"`
	Method         string    `json:"method"`
	ReferenceNo    string    `json:"referenceNo,omitempty"`
	ReceivedByName string    `json:"receivedByName,omitempty"`
	PaidAt         time.Time `json:"paidAt"`
}

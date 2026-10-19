// Package service implements the billing module's business logic: bill
// creation, payment recording (updating each bill's status as amounts
// come in), cancellation, and audit logging of every step.
package service

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	authRepo "hms-backend/internal/modules/auth/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/billing/domain"
	"hms-backend/internal/modules/billing/dto"
	"hms-backend/internal/modules/billing/repository"
	"hms-backend/internal/pkg/pagination"
)

// SequenceGenerator generates the next value for a named counter (see
// internal/pkg/sequence.Store, which satisfies this).
type SequenceGenerator interface {
	Next(ctx context.Context, key string) (int64, error)
}

// BillingService is the billing module's public business-logic contract.
type BillingService interface {
	CreateBill(ctx context.Context, req dto.CreateBillRequest, actor audit.Actor) (*dto.BillResponse, error)
	GetByID(ctx context.Context, id string) (*dto.BillResponse, error)
	Search(ctx context.Context, query dto.SearchBillsQuery) (pagination.Result[dto.BillResponse], error)
	RecordPayment(ctx context.Context, billID string, req dto.RecordPaymentRequest, actor audit.Actor) (*dto.BillResponse, error)
	CancelBill(ctx context.Context, id string, actor audit.Actor) (*dto.BillResponse, error)
}

type billingService struct {
	bills     repository.BillRepository
	payments  repository.PaymentRepository
	patients  patientsRepo.PatientRepository
	users     authRepo.UserRepository
	sequences SequenceGenerator
	audit     audit.Recorder
	log       *zap.Logger
}

// Deps bundles every collaborator billingService needs.
type Deps struct {
	Bills     repository.BillRepository
	Payments  repository.PaymentRepository
	Patients  patientsRepo.PatientRepository
	Users     authRepo.UserRepository
	Sequences SequenceGenerator
	Audit     audit.Recorder
	Logger    *zap.Logger
}

// NewBillingService builds the billing module's service implementation.
func NewBillingService(d Deps) BillingService {
	return &billingService{
		bills: d.Bills, payments: d.Payments, patients: d.Patients,
		users: d.Users, sequences: d.Sequences, audit: d.Audit, log: d.Logger,
	}
}

func (s *billingService) CreateBill(ctx context.Context, req dto.CreateBillRequest, actor audit.Actor) (*dto.BillResponse, error) {
	patientOID, err := primitive.ObjectIDFromHex(req.PatientID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"patientId": "must be a valid id"})
	}

	patient, err := s.patients.FindByID(ctx, req.PatientID)
	if err != nil {
		if errors.Is(err, patientsRepo.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"patientId": "patient not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	items := make([]domain.Item, 0, len(req.Items))
	var subtotal float64
	for _, item := range req.Items {
		amount := float64(item.Quantity) * item.UnitPrice
		items = append(items, domain.Item{
			Description: item.Description, Category: item.Category,
			Quantity: item.Quantity, UnitPrice: item.UnitPrice, Amount: amount,
		})
		subtotal += amount
	}

	totalAmount := subtotal - req.Discount + req.Tax
	if totalAmount < 0 {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"discount": "cannot exceed subtotal plus tax"})
	}

	seq, err := s.sequences.Next(ctx, "bills")
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	bill := &domain.Bill{
		BillNo:    fmt.Sprintf("INV-%05d", seq),
		PatientID: patientOID, PatientName: patient.Name,
		Items: items, Subtotal: subtotal, Discount: req.Discount, Tax: req.Tax,
		TotalAmount: totalAmount, AmountPaid: 0, Status: domain.StatusUnpaid,
	}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			bill.CreatedBy = oid
		}
	}

	if err := s.bills.Create(ctx, bill); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "BILL_CREATED", bill.ID.Hex(), map[string]any{"totalAmount": totalAmount})

	response := toBillResponse(bill, nil)
	return &response, nil
}

func (s *billingService) GetByID(ctx context.Context, id string) (*dto.BillResponse, error) {
	bill, err := s.bills.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	payments, err := s.payments.FindByBillID(ctx, bill.ID)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toBillResponse(bill, payments)
	return &response, nil
}

func (s *billingService) Search(ctx context.Context, query dto.SearchBillsQuery) (pagination.Result[dto.BillResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.SearchFilter{PatientID: query.PatientID, Status: query.Status, OutstandingOnly: query.OutstandingOnly}

	bills, total, err := s.bills.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.BillResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.BillResponse, 0, len(bills))
	for _, b := range bills {
		items = append(items, toBillResponse(&b, nil))
	}

	return pagination.NewResult(items, page, total), nil
}

func (s *billingService) RecordPayment(ctx context.Context, billID string, req dto.RecordPaymentRequest, actor audit.Actor) (*dto.BillResponse, error) {
	billOID, err := primitive.ObjectIDFromHex(billID)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	bill, err := s.bills.FindByID(ctx, billID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if bill.Status == domain.StatusCancelled {
		return nil, apperror.New("BILL_CANCELLED", "A cancelled bill cannot receive payments.", 409)
	}
	if bill.Status == domain.StatusPaid {
		return nil, apperror.New("BILL_ALREADY_PAID", "This bill has already been paid in full.", 409)
	}

	updatedBill, err := s.bills.RecordPayment(ctx, billOID, req.Amount)
	if err != nil {
		if errors.Is(err, repository.ErrOverpayment) {
			return nil, apperror.New("OVERPAYMENT", "This payment would exceed the bill's outstanding balance.", 409)
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	newStatus := domain.StatusFor(updatedBill.TotalAmount, updatedBill.AmountPaid)
	updatedBill, err = s.bills.UpdateFields(ctx, billOID, bson.M{"status": string(newStatus)})
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	payment := &domain.Payment{
		BillID: billOID, Amount: req.Amount, Method: domain.Method(req.Method), ReferenceNo: req.ReferenceNo,
	}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			payment.ReceivedBy = oid
		}
	}
	payment.ReceivedByName = s.resolveUserName(ctx, actor.UserID, actor.Email)

	if err := s.payments.Create(ctx, payment); err != nil {
		s.log.Error("failed to record payment after updating bill amountPaid; the ledger and the bill total are now out of sync",
			zap.String("billId", billID), zap.Error(err))
	}

	s.recordAudit(ctx, actor, "PAYMENT_RECORDED", billID, map[string]any{"amount": req.Amount, "method": req.Method})

	payments, err := s.payments.FindByBillID(ctx, billOID)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	response := toBillResponse(updatedBill, payments)
	return &response, nil
}

func (s *billingService) CancelBill(ctx context.Context, id string, actor audit.Actor) (*dto.BillResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	bill, err := s.bills.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if bill.Status == domain.StatusCancelled {
		return nil, apperror.New("ALREADY_CANCELLED", "This bill has already been cancelled.", 409)
	}
	if bill.AmountPaid > 0 {
		return nil, apperror.New("BILL_HAS_PAYMENTS", "A bill with recorded payments cannot be cancelled.", 409)
	}

	updated, err := s.bills.UpdateFields(ctx, oid, bson.M{"status": string(domain.StatusCancelled)})
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "BILL_CANCELLED", updated.ID.Hex(), nil)

	response := toBillResponse(updated, nil)
	return &response, nil
}

// resolveUserName looks up a display name for the acting user, falling
// back to their email (always available from the JWT claims) if the
// lookup fails — the same pattern used by Lab Orders and Pharmacy for
// their "entered/received by" fields.
func (s *billingService) resolveUserName(ctx context.Context, userID, fallbackEmail string) string {
	if userID == "" {
		return fallbackEmail
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fallbackEmail
	}
	return user.Name
}

func (s *billingService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string, metadata map[string]any) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID: actor.UserID, ActorEmail: actor.Email, ActorRole: actor.Role,
		Action: action, EntityType: "billing", EntityID: entityID, Metadata: metadata,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry", zap.String("action", action), zap.String("entityId", entityID), zap.Error(err))
	}
}

func toBillResponse(b *domain.Bill, payments []domain.Payment) dto.BillResponse {
	items := make([]dto.BillItemResponse, 0, len(b.Items))
	for _, item := range b.Items {
		items = append(items, dto.BillItemResponse{
			Description: item.Description, Category: item.Category,
			Quantity: item.Quantity, UnitPrice: item.UnitPrice, Amount: item.Amount,
		})
	}

	resp := dto.BillResponse{
		ID: b.ID.Hex(), BillNo: b.BillNo, PatientID: b.PatientID.Hex(), PatientName: b.PatientName,
		Items: items, Subtotal: b.Subtotal, Discount: b.Discount, Tax: b.Tax,
		TotalAmount: b.TotalAmount, AmountPaid: b.AmountPaid, Balance: b.Balance(),
		Status: string(b.Status), CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
	}

	if payments != nil {
		resp.Payments = make([]dto.PaymentResponse, 0, len(payments))
		for _, p := range payments {
			resp.Payments = append(resp.Payments, dto.PaymentResponse{
				ID: p.ID.Hex(), BillID: p.BillID.Hex(), Amount: p.Amount, Method: string(p.Method),
				ReferenceNo: p.ReferenceNo, ReceivedByName: p.ReceivedByName, PaidAt: p.PaidAt,
			})
		}
	}

	return resp
}

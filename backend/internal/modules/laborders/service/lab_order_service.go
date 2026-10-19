// Package service implements the lab orders module's business logic:
// patient/doctor/test validation and snapshotting, sample code generation,
// the order/result workflow (order -> result entry -> verification ->
// report), and audit logging of every step.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	authRepo "hms-backend/internal/modules/auth/repository"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	labTestsRepo "hms-backend/internal/modules/labtests/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/laborders/domain"
	"hms-backend/internal/modules/laborders/dto"
	"hms-backend/internal/modules/laborders/repository"
	"hms-backend/internal/pkg/pagination"
)

// SequenceGenerator generates the next value for a named counter (see
// internal/pkg/sequence.Store, which satisfies this).
type SequenceGenerator interface {
	Next(ctx context.Context, key string) (int64, error)
}

// LabOrderService is the lab orders module's public business-logic
// contract.
type LabOrderService interface {
	CreateOrder(ctx context.Context, req dto.CreateLabOrderRequest, actor audit.Actor) (*dto.LabOrderResponse, error)
	CancelOrder(ctx context.Context, id string, actor audit.Actor) (*dto.LabOrderResponse, error)
	EnterResult(ctx context.Context, orderID string, req dto.EnterResultRequest, actor audit.Actor) (*dto.LabOrderResponse, error)
	VerifyResult(ctx context.Context, resultID string, actor audit.Actor) (*dto.LabOrderResponse, error)
	GetByID(ctx context.Context, id string) (*dto.LabOrderResponse, error)
	GetReport(ctx context.Context, id string) (*dto.LabReportResponse, error)
	Search(ctx context.Context, query dto.SearchLabOrdersQuery) (pagination.Result[dto.LabOrderResponse], error)
}

type labOrderService struct {
	orders    repository.LabOrderRepository
	results   repository.LabResultRepository
	tests     labTestsRepo.LabTestRepository
	patients  patientsRepo.PatientRepository
	doctors   doctorsRepo.DoctorRepository
	users     authRepo.UserRepository
	sequences SequenceGenerator
	audit     audit.Recorder
	log       *zap.Logger
}

// Deps bundles every collaborator labOrderService needs.
type Deps struct {
	Orders    repository.LabOrderRepository
	Results   repository.LabResultRepository
	Tests     labTestsRepo.LabTestRepository
	Patients  patientsRepo.PatientRepository
	Doctors   doctorsRepo.DoctorRepository
	Users     authRepo.UserRepository
	Sequences SequenceGenerator
	Audit     audit.Recorder
	Logger    *zap.Logger
}

// NewLabOrderService builds the lab orders module's service implementation.
func NewLabOrderService(d Deps) LabOrderService {
	return &labOrderService{
		orders: d.Orders, results: d.Results, tests: d.Tests, patients: d.Patients,
		doctors: d.Doctors, users: d.Users, sequences: d.Sequences, audit: d.Audit, log: d.Logger,
	}
}

func (s *labOrderService) CreateOrder(ctx context.Context, req dto.CreateLabOrderRequest, actor audit.Actor) (*dto.LabOrderResponse, error) {
	patientOID, err := primitive.ObjectIDFromHex(req.PatientID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"patientId": "must be a valid id"})
	}
	doctorOID, err := primitive.ObjectIDFromHex(req.DoctorID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"doctorId": "must be a valid id"})
	}
	testOID, err := primitive.ObjectIDFromHex(req.TestID)
	if err != nil {
		return nil, apperror.ErrValidation.WithDetails(map[string]string{"testId": "must be a valid id"})
	}

	patient, err := s.patients.FindByID(ctx, req.PatientID)
	if err != nil {
		if errors.Is(err, patientsRepo.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"patientId": "patient not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	doctor, err := s.doctors.FindByID(ctx, req.DoctorID)
	if err != nil {
		if errors.Is(err, doctorsRepo.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"doctorId": "doctor not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	test, err := s.tests.FindByID(ctx, req.TestID)
	if err != nil {
		if errors.Is(err, labTestsRepo.ErrNotFound) {
			return nil, apperror.ErrValidation.WithDetails(map[string]string{"testId": "lab test not found"})
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	seq, err := s.sequences.Next(ctx, "lab_orders")
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	order := &domain.LabOrder{
		PatientID: patientOID, PatientName: patient.Name,
		DoctorID: doctorOID, DoctorName: doctor.Name,
		TestID: testOID, TestCode: test.Code, TestName: test.Name,
		Unit: test.Unit, NormalRange: test.NormalRange,
		SampleCode: fmt.Sprintf("SMP-%05d", seq),
		Status:     domain.StatusOrdered,
	}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			order.CreatedBy = oid
		}
	}

	if err := s.orders.Create(ctx, order); err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "LAB_ORDER_CREATED", order.ID.Hex(), nil)

	response := toLabOrderResponse(order, nil)
	return &response, nil
}

func (s *labOrderService) CancelOrder(ctx context.Context, id string, actor audit.Actor) (*dto.LabOrderResponse, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	order, err := s.orders.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if order.Status != domain.StatusOrdered {
		return nil, apperror.New("INVALID_STATUS_TRANSITION", "Only a newly-ordered test (before a result is entered) can be cancelled.", 409)
	}

	updated, err := s.orders.UpdateFields(ctx, oid, bson.M{"status": string(domain.StatusCancelled)})
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "LAB_ORDER_CANCELLED", updated.ID.Hex(), nil)

	response := toLabOrderResponse(updated, nil)
	return &response, nil
}

func (s *labOrderService) EnterResult(ctx context.Context, orderID string, req dto.EnterResultRequest, actor audit.Actor) (*dto.LabOrderResponse, error) {
	orderOID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if order.Status != domain.StatusOrdered {
		return nil, apperror.New("INVALID_STATUS_TRANSITION", "A result can only be entered for a newly-ordered test.", 409)
	}

	result := &domain.LabResult{
		LabOrderID:  orderOID,
		ResultValue: req.ResultValue,
		Flag:        domain.Flag(req.Flag),
		Notes:       req.Notes,
	}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			result.EnteredBy = oid
			result.EnteredByName = s.resolveUserName(ctx, actor.UserID, actor.Email)
		}
	}

	if err := s.results.Create(ctx, result); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil, apperror.New("RESULT_ALREADY_ENTERED", "A result has already been entered for this order.", 409)
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	updatedOrder, err := s.orders.UpdateFields(ctx, orderOID, bson.M{"status": string(domain.StatusAwaitingVerification)})
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "LAB_RESULT_ENTERED", result.ID.Hex(), map[string]any{"labOrderId": orderID, "flag": req.Flag})

	response := toLabOrderResponse(updatedOrder, result)
	return &response, nil
}

func (s *labOrderService) VerifyResult(ctx context.Context, resultID string, actor audit.Actor) (*dto.LabOrderResponse, error) {
	resultOID, err := primitive.ObjectIDFromHex(resultID)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	result, err := s.results.FindByID(ctx, resultID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if result.IsVerified() {
		return nil, apperror.New("ALREADY_VERIFIED", "This result has already been verified.", 409)
	}

	verifiedByName := s.resolveUserName(ctx, actor.UserID, actor.Email)

	set := bson.M{"verifiedByName": verifiedByName, "verifiedAt": time.Now().UTC()}
	if actor.UserID != "" {
		if oid, err := primitive.ObjectIDFromHex(actor.UserID); err == nil {
			set["verifiedBy"] = oid
		}
	}

	updatedResult, err := s.results.UpdateFields(ctx, resultOID, set)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	updatedOrder, err := s.orders.UpdateFields(ctx, result.LabOrderID, bson.M{"status": string(domain.StatusCompleted)})
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	s.recordAudit(ctx, actor, "LAB_RESULT_VERIFIED", updatedResult.ID.Hex(), map[string]any{"labOrderId": result.LabOrderID.Hex()})

	response := toLabOrderResponse(updatedOrder, updatedResult)
	return &response, nil
}

func (s *labOrderService) GetByID(ctx context.Context, id string) (*dto.LabOrderResponse, error) {
	order, err := s.orders.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal.WithCause(err)
	}

	result, err := s.results.FindByLabOrderID(ctx, order.ID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.ErrInternal.WithCause(err)
	}
	if errors.Is(err, repository.ErrNotFound) {
		result = nil
	}

	response := toLabOrderResponse(order, result)
	return &response, nil
}

func (s *labOrderService) GetReport(ctx context.Context, id string) (*dto.LabReportResponse, error) {
	order, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order.Status != string(domain.StatusCompleted) {
		return nil, apperror.New("REPORT_NOT_READY", "The report is only available once the result has been verified.", 409)
	}
	return &dto.LabReportResponse{Order: *order}, nil
}

func (s *labOrderService) Search(ctx context.Context, query dto.SearchLabOrdersQuery) (pagination.Result[dto.LabOrderResponse], error) {
	page := pagination.New(query.Page, query.PageSize)

	filter := repository.SearchFilter{PatientID: query.PatientID, DoctorID: query.DoctorID, Status: query.Status}

	orders, total, err := s.orders.Search(ctx, filter, page)
	if err != nil {
		return pagination.Result[dto.LabOrderResponse]{}, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.LabOrderResponse, 0, len(orders))
	for _, o := range orders {
		var result *domain.LabResult
		if r, err := s.results.FindByLabOrderID(ctx, o.ID); err == nil {
			result = r
		}
		items = append(items, toLabOrderResponse(&o, result))
	}

	return pagination.NewResult(items, page, total), nil
}

// resolveUserName looks up a display name for the acting user, falling
// back to their email (always available from the JWT claims) if the
// lookup fails for any reason — a printable report should never end up
// with a blank "entered/verified by" field.
func (s *labOrderService) resolveUserName(ctx context.Context, userID, fallbackEmail string) string {
	if userID == "" {
		return fallbackEmail
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fallbackEmail
	}
	return user.Name
}

func (s *labOrderService) recordAudit(ctx context.Context, actor audit.Actor, action, entityID string, metadata map[string]any) {
	err := s.audit.Record(ctx, audit.Entry{
		ActorID: actor.UserID, ActorEmail: actor.Email, ActorRole: actor.Role,
		Action: action, EntityType: "lab_order", EntityID: entityID, Metadata: metadata,
	})
	if err != nil {
		s.log.Warn("failed to record audit entry", zap.String("action", action), zap.String("entityId", entityID), zap.Error(err))
	}
}

func toLabOrderResponse(o *domain.LabOrder, result *domain.LabResult) dto.LabOrderResponse {
	resp := dto.LabOrderResponse{
		ID: o.ID.Hex(), PatientID: o.PatientID.Hex(), PatientName: o.PatientName,
		DoctorID: o.DoctorID.Hex(), DoctorName: o.DoctorName,
		TestID: o.TestID.Hex(), TestCode: o.TestCode, TestName: o.TestName,
		Unit: o.Unit, NormalRange: o.NormalRange, SampleCode: o.SampleCode,
		Status: string(o.Status), CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
	if result != nil {
		resp.Result = &dto.LabResultResponse{
			ID: result.ID.Hex(), ResultValue: result.ResultValue, Flag: string(result.Flag),
			Notes: result.Notes, EnteredByName: result.EnteredByName, EnteredAt: result.EnteredAt,
			VerifiedByName: result.VerifiedByName, VerifiedAt: result.VerifiedAt,
		}
	}
	return resp
}

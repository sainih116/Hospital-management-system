// Package handler adapts HTTP requests to the billing service layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/billing/dto"
	"hms-backend/internal/modules/billing/service"
	"hms-backend/internal/response"
)

// Handler serves the billing module's HTTP endpoints.
type Handler struct {
	service service.BillingService
}

// NewHandler builds a billing Handler.
func NewHandler(svc service.BillingService) *Handler {
	return &Handler{service: svc}
}

// Create godoc
// @Summary      Raise a new bill
// @Tags         billing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateBillRequest  true  "New bill"
// @Success      201  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /billing/bills [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateBillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	bill, err := h.service.CreateBill(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, bill)
}

// Search godoc
// @Summary      Search bills
// @Tags         billing
// @Produce      json
// @Security     BearerAuth
// @Param        patientId        query  string  false  "Filter by patient"
// @Param        status           query  string  false  "Filter by status"
// @Param        outstandingOnly  query  bool    false  "Only UNPAID/PARTIALLY_PAID bills"
// @Param        page             query  int     false  "Page number (default 1)"
// @Param        pageSize         query  int     false  "Items per page (default 20, max 100)"
// @Success      200  {object}  response.Envelope
// @Router       /billing/bills [get]
func (h *Handler) Search(c *gin.Context) {
	var query dto.SearchBillsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.Search(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

// GetByID godoc
// @Summary      Bill details
// @Description  Includes the bill's full payment history.
// @Tags         billing
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Bill ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /billing/bills/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	bill, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, bill)
}

// RecordPayment godoc
// @Summary      Record a payment against a bill
// @Tags         billing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                       true  "Bill ID"
// @Param        request  body  dto.RecordPaymentRequest  true  "Payment details"
// @Success      200  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /billing/bills/{id}/payments [post]
func (h *Handler) RecordPayment(c *gin.Context) {
	var req dto.RecordPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	bill, err := h.service.RecordPayment(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, bill)
}

// Cancel godoc
// @Summary      Cancel a bill
// @Description  Only a bill with no recorded payments can be cancelled.
// @Tags         billing
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Bill ID"
// @Success      200  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /billing/bills/{id}/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	bill, err := h.service.CancelBill(c.Request.Context(), c.Param("id"), actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, bill)
}

func actorFromContext(c *gin.Context) audit.Actor {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		return audit.Actor{}
	}
	return audit.Actor{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}
}

// Package handler adapts HTTP requests to the lab orders service layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/laborders/dto"
	"hms-backend/internal/modules/laborders/service"
	"hms-backend/internal/response"
)

// Handler serves the lab orders module's HTTP endpoints.
type Handler struct {
	service service.LabOrderService
}

// NewHandler builds a lab orders Handler.
func NewHandler(svc service.LabOrderService) *Handler {
	return &Handler{service: svc}
}

// Create godoc
// @Summary      Create a lab order
// @Tags         lab-orders
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateLabOrderRequest  true  "New lab order"
// @Success      201  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /lab-orders [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateLabOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	order, err := h.service.CreateOrder(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, order)
}

// Search godoc
// @Summary      Search lab orders
// @Tags         lab-orders
// @Produce      json
// @Security     BearerAuth
// @Param        patientId  query  string  false  "Filter by patient"
// @Param        doctorId   query  string  false  "Filter by doctor"
// @Param        status     query  string  false  "Filter by status"
// @Param        page       query  int     false  "Page number (default 1)"
// @Param        pageSize   query  int     false  "Items per page (default 20, max 100)"
// @Success      200  {object}  response.Envelope
// @Router       /lab-orders [get]
func (h *Handler) Search(c *gin.Context) {
	var query dto.SearchLabOrdersQuery
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
// @Summary      Lab order details
// @Tags         lab-orders
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lab order ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /lab-orders/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	order, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, order)
}

// Cancel godoc
// @Summary      Cancel a lab order
// @Description  Only a newly-ordered test (before a result is entered) can be cancelled.
// @Tags         lab-orders
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lab order ID"
// @Success      200  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /lab-orders/{id}/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	order, err := h.service.CancelOrder(c.Request.Context(), c.Param("id"), actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, order)
}

// EnterResult godoc
// @Summary      Enter a lab result
// @Tags         lab-orders
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                  true  "Lab order ID"
// @Param        request  body  dto.EnterResultRequest  true  "Measured result"
// @Success      200  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /lab-orders/{id}/result [post]
func (h *Handler) EnterResult(c *gin.Context) {
	var req dto.EnterResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	order, err := h.service.EnterResult(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, order)
}

// VerifyResult godoc
// @Summary      Verify a lab result
// @Description  Admin/Lab Technician only — never the ordering Doctor.
// @Tags         lab-orders
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lab result ID"
// @Success      200  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /lab-results/{id}/verify [patch]
func (h *Handler) VerifyResult(c *gin.Context) {
	order, err := h.service.VerifyResult(c.Request.Context(), c.Param("id"), actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, order)
}

// Report godoc
// @Summary      Printable lab report
// @Description  Returns the consolidated order+result data for a completed, verified order. The frontend renders this as a printable page (browser print-to-PDF).
// @Tags         lab-orders
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lab order ID"
// @Success      200  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /lab-orders/{id}/report [get]
func (h *Handler) Report(c *gin.Context) {
	report, err := h.service.GetReport(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, report)
}

func actorFromContext(c *gin.Context) audit.Actor {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		return audit.Actor{}
	}
	return audit.Actor{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}
}

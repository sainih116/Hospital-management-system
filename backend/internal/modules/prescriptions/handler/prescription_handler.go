// Package handler adapts HTTP requests to the prescriptions service layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/prescriptions/dto"
	"hms-backend/internal/modules/prescriptions/service"
	"hms-backend/internal/response"
)

// Handler serves the prescriptions module's HTTP endpoints.
type Handler struct {
	service service.PrescriptionService
}

// NewHandler builds a prescriptions Handler.
func NewHandler(svc service.PrescriptionService) *Handler {
	return &Handler{service: svc}
}

// Create godoc
// @Summary      Issue a prescription
// @Tags         prescriptions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreatePrescriptionRequest  true  "New prescription"
// @Success      201  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /prescriptions [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreatePrescriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	prescription, err := h.service.Create(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, prescription)
}

// Search godoc
// @Summary      Search prescriptions
// @Tags         prescriptions
// @Produce      json
// @Security     BearerAuth
// @Param        patientId  query  string  false  "Filter by patient"
// @Param        doctorId   query  string  false  "Filter by doctor"
// @Param        status     query  string  false  "Filter by status (ACTIVE, CANCELLED)"
// @Param        page       query  int     false  "Page number (default 1)"
// @Param        pageSize   query  int     false  "Items per page (default 20, max 100)"
// @Success      200  {object}  response.Envelope
// @Router       /prescriptions [get]
func (h *Handler) Search(c *gin.Context) {
	var query dto.SearchPrescriptionsQuery
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
// @Summary      Prescription details
// @Tags         prescriptions
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Prescription ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /prescriptions/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	prescription, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, prescription)
}

// ChangeStatus godoc
// @Summary      Cancel a prescription
// @Description  The only transition this endpoint allows is ACTIVE -> CANCELLED. A DISPENSED transition is set by the Pharmacy module (Phase 7).
// @Tags         prescriptions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                  true  "Prescription ID"
// @Param        request  body  dto.ChangeStatusRequest  true  "New status"
// @Success      200  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /prescriptions/{id}/status [patch]
func (h *Handler) ChangeStatus(c *gin.Context) {
	var req dto.ChangeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	prescription, err := h.service.ChangeStatus(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, prescription)
}

func actorFromContext(c *gin.Context) audit.Actor {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		return audit.Actor{}
	}
	return audit.Actor{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}
}

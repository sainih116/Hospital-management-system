// Package handler adapts HTTP requests to the medical records service
// layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/medicalrecords/dto"
	"hms-backend/internal/modules/medicalrecords/service"
	"hms-backend/internal/response"
)

// Handler serves the medical records module's HTTP endpoints.
type Handler struct {
	service service.MedicalRecordService
}

// NewHandler builds a medical records Handler.
func NewHandler(svc service.MedicalRecordService) *Handler {
	return &Handler{service: svc}
}

// Create godoc
// @Summary      Create a clinical note
// @Description  Documents a clinical encounter: complaint, HPI, history, examination, diagnosis, treatment and follow-up.
// @Tags         medical-records
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateMedicalRecordRequest  true  "New clinical note"
// @Success      201  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /medical-records [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateMedicalRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	record, err := h.service.Create(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, record)
}

// Search godoc
// @Summary      Search medical records
// @Description  Paginated medical record search — also serves as a patient's consultation history via ?patientId=.
// @Tags         medical-records
// @Produce      json
// @Security     BearerAuth
// @Param        patientId  query  string  false  "Filter by patient"
// @Param        doctorId   query  string  false  "Filter by doctor"
// @Param        page       query  int     false  "Page number (default 1)"
// @Param        pageSize   query  int     false  "Items per page (default 20, max 100)"
// @Success      200  {object}  response.Envelope
// @Router       /medical-records [get]
func (h *Handler) Search(c *gin.Context) {
	var query dto.SearchMedicalRecordsQuery
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
// @Summary      Medical record details
// @Tags         medical-records
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Medical record ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /medical-records/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	record, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, record)
}

// Update godoc
// @Summary      Amend a clinical note
// @Tags         medical-records
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                          true  "Medical record ID"
// @Param        request  body  dto.UpdateMedicalRecordRequest  true  "Fields to update"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /medical-records/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	var req dto.UpdateMedicalRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	record, err := h.service.Update(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, record)
}

func actorFromContext(c *gin.Context) audit.Actor {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		return audit.Actor{}
	}
	return audit.Actor{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}
}

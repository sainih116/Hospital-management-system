// Package handler adapts HTTP requests to the patients service layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/patients/dto"
	"hms-backend/internal/modules/patients/service"
	"hms-backend/internal/response"
)

// Handler serves the patients module's HTTP endpoints.
type Handler struct {
	service service.PatientService
}

// NewHandler builds a patients Handler.
func NewHandler(svc service.PatientService) *Handler {
	return &Handler{service: svc}
}

// Create godoc
// @Summary      Register a patient
// @Description  Creates a patient record and assigns a sequential patient code (PAT-NNNNN).
// @Tags         patients
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreatePatientRequest  true  "New patient"
// @Success      201      {object}  response.Envelope
// @Failure      400      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Router       /patients [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	patient, err := h.service.Create(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, patient)
}

// Search godoc
// @Summary      Search patients
// @Description  Paginated, filterable patient search (name, patient code, phone, gender, blood group, type).
// @Tags         patients
// @Produce      json
// @Security     BearerAuth
// @Param        query       query     string  false  "Free-text match against name, patient code and phone"
// @Param        gender      query     string  false  "M, F or O"
// @Param        bloodGroup  query     string  false  "A+, A-, B+, B-, AB+, AB-, O+ or O-"
// @Param        type        query     string  false  "OPD or IPD"
// @Param        page        query     int     false  "Page number (default 1)"
// @Param        pageSize    query     int     false  "Items per page (default 20, max 100)"
// @Success      200  {object}  response.Envelope
// @Router       /patients [get]
func (h *Handler) Search(c *gin.Context) {
	var query dto.SearchPatientsQuery
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
// @Summary      Patient profile
// @Description  Returns a single patient's profile by ID.
// @Tags         patients
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Patient ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /patients/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	patient, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, patient)
}

// Update godoc
// @Summary      Update a patient
// @Description  Partially updates a patient's demographic/contact information.
// @Tags         patients
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                    true  "Patient ID"
// @Param        request  body  dto.UpdatePatientRequest  true  "Fields to update"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /patients/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	var req dto.UpdatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	patient, err := h.service.Update(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, patient)
}

// Delete godoc
// @Summary      Deactivate a patient
// @Description  Soft-deletes a patient record (retained for compliance, hidden from normal search/list results).
// @Tags         patients
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Patient ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /patients/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id"), actorFromContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"deleted": true})
}

func actorFromContext(c *gin.Context) audit.Actor {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		return audit.Actor{}
	}
	return audit.Actor{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}
}

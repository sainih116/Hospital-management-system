// Package handler adapts HTTP requests to the doctors service layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/doctors/dto"
	"hms-backend/internal/modules/doctors/service"
	"hms-backend/internal/response"
)

// Handler serves the doctors module's HTTP endpoints.
type Handler struct {
	service service.DoctorService
}

// NewHandler builds a doctors Handler.
func NewHandler(svc service.DoctorService) *Handler {
	return &Handler{service: svc}
}

// Create godoc
// @Summary      Create a doctor profile
// @Description  Admin-only. Attaches a clinical profile to an existing user account with the DOCTOR role.
// @Tags         doctors
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateDoctorRequest  true  "New doctor profile"
// @Success      201  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /doctors [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	doctor, err := h.service.Create(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, doctor)
}

// Search godoc
// @Summary      Search doctors
// @Description  Paginated, filterable doctor directory (name, license, specialization, department).
// @Tags         doctors
// @Produce      json
// @Security     BearerAuth
// @Param        query           query  string  false  "Free-text match against name, license number and specialization"
// @Param        departmentId    query  string  false  "Filter by department"
// @Param        specialization  query  string  false  "Filter by specialization"
// @Param        page            query  int     false  "Page number (default 1)"
// @Param        pageSize        query  int     false  "Items per page (default 20, max 100)"
// @Success      200  {object}  response.Envelope
// @Router       /doctors [get]
func (h *Handler) Search(c *gin.Context) {
	var query dto.SearchDoctorsQuery
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
// @Summary      Doctor profile
// @Tags         doctors
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Doctor ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /doctors/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	doctor, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, doctor)
}

// Schedule godoc
// @Summary      Doctor availability
// @Description  Returns a doctor's recurring weekly availability windows.
// @Tags         doctors
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Doctor ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /doctors/{id}/schedule [get]
func (h *Handler) Schedule(c *gin.Context) {
	schedule, err := h.service.Schedule(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, schedule)
}

// Update godoc
// @Summary      Update a doctor profile
// @Description  Admin-only. Partially updates department, license, specialization, fee, bio or availability.
// @Tags         doctors
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                  true  "Doctor ID"
// @Param        request  body  dto.UpdateDoctorRequest  true  "Fields to update"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /doctors/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	var req dto.UpdateDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	doctor, err := h.service.Update(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, doctor)
}

// Delete godoc
// @Summary      Deactivate a doctor profile
// @Description  Admin-only. Soft-deletes a doctor profile (retained for history, hidden from search).
// @Tags         doctors
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Doctor ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /doctors/{id} [delete]
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

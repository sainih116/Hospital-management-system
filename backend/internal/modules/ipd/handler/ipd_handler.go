// Package handler adapts HTTP requests to the IPD service layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/ipd/dto"
	"hms-backend/internal/modules/ipd/service"
	"hms-backend/internal/response"
)

// Handler serves the IPD module's HTTP endpoints.
type Handler struct {
	service service.IPDService
}

// NewHandler builds an IPD Handler.
func NewHandler(svc service.IPDService) *Handler {
	return &Handler{service: svc}
}

// CreateWard godoc
// @Summary      Add a ward
// @Tags         ipd
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateWardRequest  true  "New ward"
// @Success      201  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /ipd/wards [post]
func (h *Handler) CreateWard(c *gin.Context) {
	var req dto.CreateWardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	ward, err := h.service.CreateWard(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, ward)
}

// UpdateWard godoc
// @Summary      Update a ward
// @Tags         ipd
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                 true  "Ward ID"
// @Param        request  body  dto.UpdateWardRequest  true  "Fields to update"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /ipd/wards/{id} [patch]
func (h *Handler) UpdateWard(c *gin.Context) {
	var req dto.UpdateWardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	ward, err := h.service.UpdateWard(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, ward)
}

// ListWards godoc
// @Summary      List wards
// @Tags         ipd
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope
// @Router       /ipd/wards [get]
func (h *Handler) ListWards(c *gin.Context) {
	wards, err := h.service.ListWards(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"wards": wards})
}

// CreateBed godoc
// @Summary      Add a bed to a ward
// @Tags         ipd
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateBedRequest  true  "New bed"
// @Success      201  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /ipd/beds [post]
func (h *Handler) CreateBed(c *gin.Context) {
	var req dto.CreateBedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	bed, err := h.service.CreateBed(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, bed)
}

// UpdateBedStatus godoc
// @Summary      Change a bed's status (AVAILABLE/MAINTENANCE)
// @Tags         ipd
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                true  "Bed ID"
// @Param        request  body  dto.UpdateBedRequest  true  "New status"
// @Success      200  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /ipd/beds/{id} [patch]
func (h *Handler) UpdateBedStatus(c *gin.Context) {
	var req dto.UpdateBedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	bed, err := h.service.UpdateBedStatus(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, bed)
}

// ListBeds godoc
// @Summary      List beds (the bed board)
// @Tags         ipd
// @Produce      json
// @Security     BearerAuth
// @Param        wardId  query  string  false  "Filter by ward"
// @Param        status  query  string  false  "Filter by status"
// @Success      200  {object}  response.Envelope
// @Router       /ipd/beds [get]
func (h *Handler) ListBeds(c *gin.Context) {
	beds, err := h.service.ListBeds(c.Request.Context(), c.Query("wardId"), c.Query("status"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"beds": beds})
}

// AdmitPatient godoc
// @Summary      Admit a patient into a bed
// @Tags         ipd
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  dto.AdmitPatientRequest  true  "Admission details"
// @Success      201  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /ipd/admissions [post]
func (h *Handler) AdmitPatient(c *gin.Context) {
	var req dto.AdmitPatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	admission, err := h.service.AdmitPatient(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, admission)
}

// DischargePatient godoc
// @Summary      Discharge an admitted patient and release their bed
// @Tags         ipd
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                 true  "Admission ID"
// @Param        request  body  dto.DischargeRequest  true  "Discharge note"
// @Success      200  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /ipd/admissions/{id}/discharge [post]
func (h *Handler) DischargePatient(c *gin.Context) {
	var req dto.DischargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	admission, err := h.service.DischargePatient(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, admission)
}

// GetAdmission godoc
// @Summary      Admission details
// @Tags         ipd
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Admission ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /ipd/admissions/{id} [get]
func (h *Handler) GetAdmission(c *gin.Context) {
	admission, err := h.service.GetAdmissionByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, admission)
}

// SearchAdmissions godoc
// @Summary      Search admissions
// @Tags         ipd
// @Produce      json
// @Security     BearerAuth
// @Param        patientId  query  string  false  "Filter by patient"
// @Param        doctorId   query  string  false  "Filter by doctor"
// @Param        status     query  string  false  "Filter by status"
// @Param        page       query  int     false  "Page number (default 1)"
// @Param        pageSize   query  int     false  "Items per page (default 20, max 100)"
// @Success      200  {object}  response.Envelope
// @Router       /ipd/admissions [get]
func (h *Handler) SearchAdmissions(c *gin.Context) {
	var query dto.SearchAdmissionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.SearchAdmissions(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

func actorFromContext(c *gin.Context) audit.Actor {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		return audit.Actor{}
	}
	return audit.Actor{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}
}

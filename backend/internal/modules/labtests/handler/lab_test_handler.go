// Package handler adapts HTTP requests to the lab tests service layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/labtests/dto"
	"hms-backend/internal/modules/labtests/service"
	"hms-backend/internal/response"
)

// Handler serves the lab tests module's HTTP endpoints.
type Handler struct {
	service service.LabTestService
}

// NewHandler builds a lab tests Handler.
func NewHandler(svc service.LabTestService) *Handler {
	return &Handler{service: svc}
}

// Create godoc
// @Summary      Add a lab test
// @Description  Admin-only. Adds a test to the orderable catalog.
// @Tags         lab-tests
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateLabTestRequest  true  "New lab test"
// @Success      201  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /lab-tests [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateLabTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	test, err := h.service.Create(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, test)
}

// List godoc
// @Summary      List lab tests
// @Tags         lab-tests
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope
// @Router       /lab-tests [get]
func (h *Handler) List(c *gin.Context) {
	tests, err := h.service.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"labTests": tests})
}

// GetByID godoc
// @Summary      Lab test details
// @Tags         lab-tests
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lab test ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /lab-tests/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	test, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, test)
}

// Update godoc
// @Summary      Update a lab test
// @Description  Admin-only.
// @Tags         lab-tests
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                    true  "Lab test ID"
// @Param        request  body  dto.UpdateLabTestRequest  true  "Fields to update"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /lab-tests/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	var req dto.UpdateLabTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	test, err := h.service.Update(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, test)
}

func actorFromContext(c *gin.Context) audit.Actor {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		return audit.Actor{}
	}
	return audit.Actor{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}
}

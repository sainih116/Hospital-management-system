// Package handler adapts HTTP requests to the departments service layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/departments/dto"
	"hms-backend/internal/modules/departments/service"
	"hms-backend/internal/response"
)

// Handler serves the departments module's HTTP endpoints.
type Handler struct {
	service service.DepartmentService
}

// NewHandler builds a departments Handler.
func NewHandler(svc service.DepartmentService) *Handler {
	return &Handler{service: svc}
}

// Create godoc
// @Summary      Create a department
// @Description  Admin-only. Creates a hospital department (e.g. Cardiology, Pediatrics).
// @Tags         departments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateDepartmentRequest  true  "New department"
// @Success      201  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /departments [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	department, err := h.service.Create(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, department)
}

// List godoc
// @Summary      List departments
// @Description  Returns every department (active and inactive).
// @Tags         departments
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope
// @Router       /departments [get]
func (h *Handler) List(c *gin.Context) {
	departments, err := h.service.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"departments": departments})
}

// GetByID godoc
// @Summary      Department details
// @Tags         departments
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Department ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /departments/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	department, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, department)
}

// Update godoc
// @Summary      Update a department
// @Description  Admin-only. Updates a department's name or active/inactive status.
// @Tags         departments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                       true  "Department ID"
// @Param        request  body  dto.UpdateDepartmentRequest  true  "Fields to update"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /departments/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	var req dto.UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	department, err := h.service.Update(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, department)
}

func actorFromContext(c *gin.Context) audit.Actor {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		return audit.Actor{}
	}
	return audit.Actor{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}
}

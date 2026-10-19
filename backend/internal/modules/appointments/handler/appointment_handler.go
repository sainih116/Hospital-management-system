// Package handler adapts HTTP requests to the appointments service layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/appointments/dto"
	"hms-backend/internal/modules/appointments/service"
	"hms-backend/internal/response"
)

// Handler serves the appointments module's HTTP endpoints.
type Handler struct {
	service service.AppointmentService
}

// NewHandler builds an appointments Handler.
func NewHandler(svc service.AppointmentService) *Handler {
	return &Handler{service: svc}
}

// Book godoc
// @Summary      Book an appointment
// @Description  Creates a SCHEDULED appointment. Rejects overlapping/duplicate-slot bookings for the same doctor.
// @Tags         appointments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.BookAppointmentRequest  true  "New appointment"
// @Success      201  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /appointments [post]
func (h *Handler) Book(c *gin.Context) {
	var req dto.BookAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	appt, err := h.service.Book(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, appt)
}

// Search godoc
// @Summary      Search appointments
// @Description  Paginated, filterable appointment search — also serves as the doctor schedule view (filter by doctorId + from/to).
// @Tags         appointments
// @Produce      json
// @Security     BearerAuth
// @Param        patientId  query  string  false  "Filter by patient"
// @Param        doctorId   query  string  false  "Filter by doctor"
// @Param        status     query  string  false  "Filter by status"
// @Param        from       query  string  false  "Range start (RFC3339)"
// @Param        to         query  string  false  "Range end (RFC3339)"
// @Param        page       query  int     false  "Page number (default 1)"
// @Param        pageSize   query  int     false  "Items per page (default 20, max 100)"
// @Success      200  {object}  response.Envelope
// @Router       /appointments [get]
func (h *Handler) Search(c *gin.Context) {
	var query dto.SearchAppointmentsQuery
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
// @Summary      Appointment details
// @Tags         appointments
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Appointment ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /appointments/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	appt, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, appt)
}

// ChangeStatus godoc
// @Summary      Change an appointment's status
// @Description  Admin/Receptionist may make any valid transition. A Doctor may only progress their own appointments through IN_PROGRESS/COMPLETED/NO_SHOW.
// @Tags         appointments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                     true  "Appointment ID"
// @Param        request  body  dto.ChangeStatusRequest  true  "New status"
// @Success      200  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /appointments/{id}/status [patch]
func (h *Handler) ChangeStatus(c *gin.Context) {
	var req dto.ChangeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	appt, err := h.service.ChangeStatus(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, appt)
}

// Reschedule godoc
// @Summary      Reschedule an appointment
// @Description  Admin/Receptionist only. Marks the current appointment RESCHEDULED and creates a new SCHEDULED one at the requested time.
// @Tags         appointments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                    true  "Appointment ID"
// @Param        request  body  dto.RescheduleRequest  true  "New time slot"
// @Success      201  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /appointments/{id}/reschedule [post]
func (h *Handler) Reschedule(c *gin.Context) {
	var req dto.RescheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	appt, err := h.service.Reschedule(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, appt)
}

func actorFromContext(c *gin.Context) audit.Actor {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		return audit.Actor{}
	}
	return audit.Actor{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}
}

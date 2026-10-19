// Package handler adapts HTTP requests to the reporting service layer.
package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/modules/reports/dto"
	"hms-backend/internal/modules/reports/service"
	"hms-backend/internal/response"
)

// Handler serves the reporting module's HTTP endpoints.
type Handler struct {
	service service.ReportsService
}

// NewHandler builds a reporting Handler.
func NewHandler(svc service.ReportsService) *Handler {
	return &Handler{service: svc}
}

// Summary godoc
// @Summary      Dashboard summary
// @Description  Admin-only. Cheap top-level counts/sums for the dashboard's summary cards.
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope
// @Router       /reports/summary [get]
func (h *Handler) Summary(c *gin.Context) {
	result, err := h.service.Summary(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, result)
}

// AppointmentsByStatus godoc
// @Summary      Appointments by status
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Param        from  query  string  false  "Start date (YYYY-MM-DD), default 30 days ago"
// @Param        to    query  string  false  "End date (YYYY-MM-DD), default today"
// @Success      200  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /reports/appointments-by-status [get]
func (h *Handler) AppointmentsByStatus(c *gin.Context) {
	var query dto.DateRangeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.AppointmentsByStatus(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"items": result})
}

// RevenueTrend godoc
// @Summary      Revenue collected per day
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Param        from  query  string  false  "Start date (YYYY-MM-DD), default 30 days ago"
// @Param        to    query  string  false  "End date (YYYY-MM-DD), default today"
// @Success      200  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /reports/revenue-trend [get]
func (h *Handler) RevenueTrend(c *gin.Context) {
	var query dto.DateRangeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.RevenueTrend(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"items": result})
}

// PatientRegistrationTrend godoc
// @Summary      New patient registrations per day
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Param        from  query  string  false  "Start date (YYYY-MM-DD), default 30 days ago"
// @Param        to    query  string  false  "End date (YYYY-MM-DD), default today"
// @Success      200  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /reports/patient-registrations-trend [get]
func (h *Handler) PatientRegistrationTrend(c *gin.Context) {
	var query dto.DateRangeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.PatientRegistrationTrend(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"items": result})
}

// DoctorWorkload godoc
// @Summary      Top doctors by appointment count
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Param        from      query  string  false  "Start date (YYYY-MM-DD), default 30 days ago"
// @Param        to        query  string  false  "End date (YYYY-MM-DD), default today"
// @Param        limit     query  int     false  "Max doctors returned (default 10, max 50)"
// @Success      200  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /reports/doctor-workload [get]
func (h *Handler) DoctorWorkload(c *gin.Context) {
	var query dto.DoctorWorkloadQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.DoctorWorkload(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"items": result})
}

// LabTestVolume godoc
// @Summary      Top ordered lab tests by volume
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Param        from      query  string  false  "Start date (YYYY-MM-DD), default 30 days ago"
// @Param        to        query  string  false  "End date (YYYY-MM-DD), default today"
// @Param        limit     query  int     false  "Max tests returned (default 10, max 50)"
// @Success      200  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /reports/lab-test-volume [get]
func (h *Handler) LabTestVolume(c *gin.Context) {
	var query dto.DoctorWorkloadQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.LabTestVolume(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"items": result})
}

// Export godoc
// @Summary      Export the full report set as an Excel workbook
// @Description  Admin-only. One sheet per report (Summary, Appointments by Status, Revenue Trend, Patient Registrations, Doctor Workload, Lab Test Volume) for the given date range.
// @Tags         reports
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security     BearerAuth
// @Param        from  query  string  false  "Start date (YYYY-MM-DD), default 30 days ago"
// @Param        to    query  string  false  "End date (YYYY-MM-DD), default today"
// @Success      200
// @Failure      422  {object}  response.Envelope
// @Router       /reports/export [get]
func (h *Handler) Export(c *gin.Context) {
	var query dto.DateRangeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	file, err := h.service.Export(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}

	filename := fmt.Sprintf("hms-report-%s.xlsx", time.Now().UTC().Format("2006-01-02"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", file)
}

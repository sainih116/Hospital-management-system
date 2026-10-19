// Package handler adapts HTTP requests to the audit logging service layer.
package handler

import (
	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/modules/auditlogs/dto"
	"hms-backend/internal/modules/auditlogs/service"
	"hms-backend/internal/response"
)

// Handler serves the audit logging module's HTTP endpoints.
type Handler struct {
	service service.AuditLogService
}

// NewHandler builds an audit logging Handler.
func NewHandler(svc service.AuditLogService) *Handler {
	return &Handler{service: svc}
}

// Search godoc
// @Summary      Search audit logs
// @Description  Admin-only. Every mutating action across every module is recorded here as it happens (internal/audit.Recorder) — this endpoint only queries the resulting history.
// @Tags         audit-logs
// @Produce      json
// @Security     BearerAuth
// @Param        actorId     query  string  false  "Filter by the acting user's ID"
// @Param        entityType  query  string  false  "Filter by entity type, e.g. patient, pharmacy, billing"
// @Param        entityId    query  string  false  "Filter by a specific entity's ID"
// @Param        action      query  string  false  "Filter by action code, e.g. BILL_CREATED"
// @Param        from        query  string  false  "RFC3339 lower bound on createdAt"
// @Param        to          query  string  false  "RFC3339 upper bound on createdAt"
// @Param        page        query  int     false  "Page number (default 1)"
// @Param        pageSize    query  int     false  "Items per page (default 20, max 100)"
// @Success      200  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /audit-logs [get]
func (h *Handler) Search(c *gin.Context) {
	var query dto.SearchAuditLogsQuery
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

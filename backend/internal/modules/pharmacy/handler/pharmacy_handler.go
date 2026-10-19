// Package handler adapts HTTP requests to the pharmacy service layer.
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/pharmacy/dto"
	"hms-backend/internal/modules/pharmacy/service"
	"hms-backend/internal/response"
)

// Handler serves the pharmacy module's HTTP endpoints.
type Handler struct {
	service service.PharmacyService
}

// NewHandler builds a pharmacy Handler.
func NewHandler(svc service.PharmacyService) *Handler {
	return &Handler{service: svc}
}

// CreateMedicine godoc
// @Summary      Add a medicine to the catalog
// @Tags         pharmacy
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateMedicineRequest  true  "New medicine"
// @Success      201  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /pharmacy/medicines [post]
func (h *Handler) CreateMedicine(c *gin.Context) {
	var req dto.CreateMedicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	medicine, err := h.service.CreateMedicine(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, medicine)
}

// UpdateMedicine godoc
// @Summary      Update a medicine
// @Tags         pharmacy
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                     true  "Medicine ID"
// @Param        request  body  dto.UpdateMedicineRequest  true  "Fields to update"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /pharmacy/medicines/{id} [patch]
func (h *Handler) UpdateMedicine(c *gin.Context) {
	var req dto.UpdateMedicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	medicine, err := h.service.UpdateMedicine(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, medicine)
}

// GetMedicine godoc
// @Summary      Medicine details
// @Tags         pharmacy
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Medicine ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /pharmacy/medicines/{id} [get]
func (h *Handler) GetMedicine(c *gin.Context) {
	medicine, err := h.service.GetMedicineByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, medicine)
}

// SearchMedicines godoc
// @Summary      Search the medicine catalog
// @Tags         pharmacy
// @Produce      json
// @Security     BearerAuth
// @Param        query     query  string  false  "Name/generic name search"
// @Param        lowStock  query  bool    false  "Only medicines at or below reorder level"
// @Param        page      query  int     false  "Page number"
// @Param        pageSize  query  int     false  "Items per page"
// @Success      200  {object}  response.Envelope
// @Router       /pharmacy/medicines [get]
func (h *Handler) SearchMedicines(c *gin.Context) {
	var query dto.SearchMedicinesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.SearchMedicines(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

// ReceiveBatch godoc
// @Summary      Receive a batch of stock for a medicine
// @Tags         pharmacy
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                   true  "Medicine ID"
// @Param        request  body  dto.ReceiveBatchRequest  true  "Batch details"
// @Success      201  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /pharmacy/medicines/{id}/batches [post]
func (h *Handler) ReceiveBatch(c *gin.Context) {
	var req dto.ReceiveBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	batch, err := h.service.ReceiveBatch(c.Request.Context(), c.Param("id"), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, batch)
}

// ListBatches godoc
// @Summary      List batches for a medicine
// @Tags         pharmacy
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Medicine ID"
// @Success      200  {object}  response.Envelope
// @Router       /pharmacy/medicines/{id}/batches [get]
func (h *Handler) ListBatches(c *gin.Context) {
	batches, err := h.service.ListBatchesForMedicine(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"batches": batches})
}

// ListExpiringBatches godoc
// @Summary      List batches expiring soon
// @Tags         pharmacy
// @Produce      json
// @Security     BearerAuth
// @Param        withinDays  query  int  false  "Expiry window in days (default 30)"
// @Success      200  {object}  response.Envelope
// @Router       /pharmacy/batches/expiring [get]
func (h *Handler) ListExpiringBatches(c *gin.Context) {
	withinDays := 30
	if raw := c.Query("withinDays"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			withinDays = parsed
		}
	}

	batches, err := h.service.ListExpiringBatches(c.Request.Context(), withinDays)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"batches": batches})
}

// Dispense godoc
// @Summary      Dispense a prescription
// @Description  Fulfills every item on an active prescription using FEFO batch allocation and marks it DISPENSED.
// @Tags         pharmacy
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  dto.DispenseRequest  true  "Prescription and items to dispense"
// @Success      200  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /pharmacy/dispense [post]
func (h *Handler) Dispense(c *gin.Context) {
	var req dto.DispenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.Dispense(c.Request.Context(), req, actorFromContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

// SearchStockMovements godoc
// @Summary      Search the stock movement ledger
// @Tags         pharmacy
// @Produce      json
// @Security     BearerAuth
// @Param        medicineId  query  string  false  "Filter by medicine"
// @Param        type        query  string  false  "IN or OUT"
// @Param        page        query  int     false  "Page number"
// @Param        pageSize    query  int     false  "Items per page"
// @Success      200  {object}  response.Envelope
// @Router       /pharmacy/stock-movements [get]
func (h *Handler) SearchStockMovements(c *gin.Context) {
	var query dto.SearchStockMovementsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.ErrValidation.WithCause(err))
		return
	}

	result, err := h.service.SearchStockMovements(c.Request.Context(), query)
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

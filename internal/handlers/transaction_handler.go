package handlers

import (
	"net/http"
	"strconv"
	"time"

	"transaction-api/internal/middleware"
	"transaction-api/internal/models"
	"transaction-api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

type TransactionHandler struct {
	service   *services.TransactionService
	validator *validator.Validate
}

func NewTransactionHandler(service *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		service:   service,
		validator: validator.New(),
	}
}

// CreateTransaction creates a new transaction
// @Summary Create transaction
// @Description Create a new transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Param transaction body models.TransactionRequest true "Transaction data"
// @Success 201 {object} models.Transaction
// @Failure 400 {object} middleware.ErrorResponse
// @Failure 500 {object} middleware.ErrorResponse
// @Router /transactions [post]
func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	var req models.TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.SendValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		middleware.SendValidationError(c, err.Error())
		return
	}

	transaction, err := h.service.CreateTransaction(&req)
	if err != nil {
		middleware.SendError(c, http.StatusInternalServerError, "internal_server_error", err.Error())
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

// GetTransactionByID retrieves a transaction by ID
// @Summary Get transaction by ID
// @Description Get a specific transaction by ID
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 200 {object} models.Transaction
// @Failure 400 {object} middleware.ErrorResponse
// @Failure 404 {object} middleware.ErrorResponse
// @Failure 500 {object} middleware.ErrorResponse
// @Router /transactions/{id} [get]
func (h *TransactionHandler) GetTransactionByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		middleware.SendError(c, http.StatusBadRequest, "invalid_id", "Invalid transaction ID")
		return
	}

	transaction, err := h.service.GetTransactionByID(uint(id))
	if err != nil {
		if err.Error() == "transaction not found" {
			middleware.SendError(c, http.StatusNotFound, "not_found", "Transaction not found")
			return
		}
		middleware.SendError(c, http.StatusInternalServerError, "internal_server_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// GetTransactions retrieves transactions with filtering and pagination
// @Summary Get transactions
// @Description Get transactions with optional filtering and pagination
// @Tags transactions
// @Accept json
// @Produce json
// @Param user_id query int false "Filter by User ID"
// @Param status query string false "Filter by Status" Enums(pending, success, failed)
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} models.TransactionResponse
// @Failure 400 {object} middleware.ErrorResponse
// @Failure 500 {object} middleware.ErrorResponse
// @Router /transactions [get]
func (h *TransactionHandler) GetTransactions(c *gin.Context) {
	var query models.TransactionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		middleware.SendValidationError(c, err.Error())
		return
	}

	// Validate status if provided
	if query.Status != "" {
		if query.Status != models.StatusPending && query.Status != models.StatusSuccess && query.Status != models.StatusFailed {
			middleware.SendError(c, http.StatusBadRequest, "invalid_status", "Status must be one of: pending, success, failed")
			return
		}
	}

	response, err := h.service.GetTransactions(&query)
	if err != nil {
		middleware.SendError(c, http.StatusInternalServerError, "internal_server_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateTransaction updates a transaction status
// @Summary Update transaction
// @Description Update transaction status
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Param transaction body models.TransactionUpdateRequest true "Transaction update data"
// @Success 200 {object} models.Transaction
// @Failure 400 {object} middleware.ErrorResponse
// @Failure 404 {object} middleware.ErrorResponse
// @Failure 500 {object} middleware.ErrorResponse
// @Router /transactions/{id} [put]
func (h *TransactionHandler) UpdateTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		middleware.SendError(c, http.StatusBadRequest, "invalid_id", "Invalid transaction ID")
		return
	}

	var req models.TransactionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.SendValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		middleware.SendValidationError(c, err.Error())
		return
	}

	transaction, err := h.service.UpdateTransaction(uint(id), &req)
	if err != nil {
		if err.Error() == "transaction not found" {
			middleware.SendError(c, http.StatusNotFound, "not_found", "Transaction not found")
			return
		}
		middleware.SendError(c, http.StatusInternalServerError, "internal_server_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// DeleteTransaction deletes a transaction
// @Summary Delete transaction
// @Description Delete a transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 204
// @Failure 400 {object} middleware.ErrorResponse
// @Failure 404 {object} middleware.ErrorResponse
// @Failure 500 {object} middleware.ErrorResponse
// @Router /transactions/{id} [delete]
func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		middleware.SendError(c, http.StatusBadRequest, "invalid_id", "Invalid transaction ID")
		return
	}

	err = h.service.DeleteTransaction(uint(id))
	if err != nil {
		if err.Error() == "transaction not found" {
			middleware.SendError(c, http.StatusNotFound, "not_found", "Transaction not found")
			return
		}
		middleware.SendError(c, http.StatusInternalServerError, "internal_server_error", err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// GetDashboardSummary retrieves dashboard summary data
// @Summary Get dashboard summary
// @Description Get dashboard summary with transaction statistics
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} models.DashboardSummary
// @Failure 500 {object} middleware.ErrorResponse
// @Router /dashboard/summary [get]
func (h *TransactionHandler) GetDashboardSummary(c *gin.Context) {
	summary, err := h.service.GetDashboardSummary()
	if err != nil {
		logrus.WithError(err).Error("Failed to get dashboard summary")
		middleware.SendError(c, http.StatusInternalServerError, "internal_server_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, summary)
}

// HealthCheck provides a health check endpoint
// @Summary Health check
// @Description Check if the service is healthy
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *TransactionHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "transaction-api",
		"timestamp": time.Now().UTC(),
	})
}
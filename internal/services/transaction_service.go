package services

import (
	"fmt"
	"math"
	"time"

	"transaction-api/internal/models"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type TransactionService struct {
	db *gorm.DB
}

func NewTransactionService(db *gorm.DB) *TransactionService {
	return &TransactionService{db: db}
}

// CreateTransaction creates a new transaction
func (s *TransactionService) CreateTransaction(req *models.TransactionRequest) (*models.Transaction, error) {
	transaction := &models.Transaction{
		UserID: req.UserID,
		Amount: req.Amount,
		Status: models.StatusPending,
	}

	if err := s.db.Create(transaction).Error; err != nil {
		logrus.WithError(err).Error("Failed to create transaction")
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"transaction_id": transaction.ID,
		"user_id":        transaction.UserID,
		"amount":         transaction.Amount,
	}).Info("Transaction created successfully")

	return transaction, nil
}

// GetTransactionByID retrieves a transaction by ID
func (s *TransactionService) GetTransactionByID(id uint) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := s.db.First(&transaction, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("transaction not found")
		}
		logrus.WithError(err).Error("Failed to get transaction")
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &transaction, nil
}

// GetTransactions retrieves transactions with filtering and pagination
func (s *TransactionService) GetTransactions(query *models.TransactionQuery) (*models.TransactionResponse, error) {
	var transactions []models.Transaction
	var total int64

	// Build query
	db := s.db.Model(&models.Transaction{})

	// Apply filters
	if query.UserID != 0 {
		db = db.Where("user_id = ?", query.UserID)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	// Count total records
	if err := db.Count(&total).Error; err != nil {
		logrus.WithError(err).Error("Failed to count transactions")
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Set default pagination
	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Page <= 0 {
		query.Page = 1
	}

	offset := (query.Page - 1) * query.Limit

	// Get transactions with pagination
	if err := db.Offset(offset).Limit(query.Limit).Order("created_at DESC").Find(&transactions).Error; err != nil {
		logrus.WithError(err).Error("Failed to get transactions")
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(query.Limit)))

	response := &models.TransactionResponse{
		Data:       transactions,
		Total:      total,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}

	return response, nil
}

// UpdateTransaction updates a transaction status
func (s *TransactionService) UpdateTransaction(id uint, req *models.TransactionUpdateRequest) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := s.db.First(&transaction, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	transaction.Status = req.Status
	if err := s.db.Save(&transaction).Error; err != nil {
		logrus.WithError(err).Error("Failed to update transaction")
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"transaction_id": transaction.ID,
		"new_status":     transaction.Status,
	}).Info("Transaction updated successfully")

	return &transaction, nil
}

// DeleteTransaction soft deletes a transaction
func (s *TransactionService) DeleteTransaction(id uint) error {
	var transaction models.Transaction
	if err := s.db.First(&transaction, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("transaction not found")
		}
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	if err := s.db.Delete(&transaction).Error; err != nil {
		logrus.WithError(err).Error("Failed to delete transaction")
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	logrus.WithField("transaction_id", id).Info("Transaction deleted successfully")
	return nil
}

// GetDashboardSummary retrieves dashboard summary data
func (s *TransactionService) GetDashboardSummary() (*models.DashboardSummary, error) {
	var summary models.DashboardSummary

	// Get today's date range
	today := time.Now().UTC().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	// Total successful transactions today
	var totalSuccessToday int64
	if err := s.db.Model(&models.Transaction{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", models.StatusSuccess, today, tomorrow).
		Count(&totalSuccessToday).Error; err != nil {
		return nil, fmt.Errorf("failed to count today's successful transactions: %w", err)
	}
	summary.TotalSuccessToday = totalSuccessToday

	// Total transactions
	var totalTransactions int64
	if err := s.db.Model(&models.Transaction{}).Count(&totalTransactions).Error; err != nil {
		return nil, fmt.Errorf("failed to count total transactions: %w", err)
	}
	summary.TotalTransactions = totalTransactions

	// Average amount per user
	var avgResult struct {
		AvgAmount float64
	}
	if err := s.db.Model(&models.Transaction{}).
		Select("AVG(amount) as avg_amount").
		Where("status = ?", models.StatusSuccess).
		Scan(&avgResult).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate average amount: %w", err)
	}
	summary.AverageAmountPerUser = avgResult.AvgAmount

	// Total amount (all successful transactions)
	var totalAmountResult struct {
		TotalAmount float64
	}
	if err := s.db.Model(&models.Transaction{}).
		Select("SUM(amount) as total_amount").
		Where("status = ?", models.StatusSuccess).
		Scan(&totalAmountResult).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate total amount: %w", err)
	}
	summary.TotalAmount = totalAmountResult.TotalAmount

	// Total amount today
	var totalAmountTodayResult struct {
		TotalAmount float64
	}
	if err := s.db.Model(&models.Transaction{}).
		Select("SUM(amount) as total_amount").
		Where("status = ? AND created_at >= ? AND created_at < ?", models.StatusSuccess, today, tomorrow).
		Scan(&totalAmountTodayResult).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate today's total amount: %w", err)
	}
	summary.TotalAmountToday = totalAmountTodayResult.TotalAmount

	// Status distribution
	var statusResults []struct {
		Status string
		Count  int64
	}
	if err := s.db.Model(&models.Transaction{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusResults).Error; err != nil {
		return nil, fmt.Errorf("failed to get status distribution: %w", err)
	}

	summary.StatusDistribution = make(map[string]int64)
	for _, result := range statusResults {
		summary.StatusDistribution[result.Status] = result.Count
	}

	// Recent transactions (latest 10)
	var recentTransactions []models.Transaction
	if err := s.db.Order("created_at DESC").Limit(10).Find(&recentTransactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent transactions: %w", err)
	}
	summary.RecentTransactions = recentTransactions

	return &summary, nil
}
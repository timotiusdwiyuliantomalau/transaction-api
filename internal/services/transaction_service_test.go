package services

import (
	"testing"
	"time"
	"transaction-api/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type TransactionServiceTestSuite struct {
	suite.Suite
	db      *gorm.DB
	service *TransactionService
}

func (suite *TransactionServiceTestSuite) SetupTest() {
	// Use SQLite in-memory database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// Auto migrate the schema
	err = db.AutoMigrate(&models.Transaction{})
	suite.Require().NoError(err)

	suite.db = db
	suite.service = NewTransactionService(db)
}

func (suite *TransactionServiceTestSuite) TearDownTest() {
	sqlDB, err := suite.db.DB()
	suite.Require().NoError(err)
	sqlDB.Close()
}

func (suite *TransactionServiceTestSuite) TestCreateTransaction() {
	req := &models.TransactionRequest{
		UserID: 1,
		Amount: 100.50,
	}

	transaction, err := suite.service.CreateTransaction(req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), transaction)
	assert.Equal(suite.T(), uint(1), transaction.UserID)
	assert.Equal(suite.T(), 100.50, transaction.Amount)
	assert.Equal(suite.T(), models.StatusPending, transaction.Status)
	assert.NotZero(suite.T(), transaction.ID)
}

func (suite *TransactionServiceTestSuite) TestGetTransactionByID() {
	// Create a test transaction
	transaction := &models.Transaction{
		UserID: 1,
		Amount: 100.50,
		Status: models.StatusSuccess,
	}
	err := suite.db.Create(transaction).Error
	suite.Require().NoError(err)

	// Test getting existing transaction
	result, err := suite.service.GetTransactionByID(transaction.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), transaction.ID, result.ID)
	assert.Equal(suite.T(), transaction.UserID, result.UserID)
	assert.Equal(suite.T(), transaction.Amount, result.Amount)

	// Test getting non-existing transaction
	result, err = suite.service.GetTransactionByID(999)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "transaction not found")
}

func (suite *TransactionServiceTestSuite) TestGetTransactions() {
	// Create test transactions
	transactions := []models.Transaction{
		{UserID: 1, Amount: 100.0, Status: models.StatusSuccess},
		{UserID: 1, Amount: 200.0, Status: models.StatusPending},
		{UserID: 2, Amount: 300.0, Status: models.StatusSuccess},
		{UserID: 2, Amount: 400.0, Status: models.StatusFailed},
	}

	for i := range transactions {
		err := suite.db.Create(&transactions[i]).Error
		suite.Require().NoError(err)
	}

	// Test getting all transactions
	query := &models.TransactionQuery{
		Page:  1,
		Limit: 10,
	}
	response, err := suite.service.GetTransactions(query)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), int64(4), response.Total)
	assert.Equal(suite.T(), 4, len(response.Data))

	// Test filtering by UserID
	query = &models.TransactionQuery{
		UserID: 1,
		Page:   1,
		Limit:  10,
	}
	response, err = suite.service.GetTransactions(query)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2), response.Total)
	assert.Equal(suite.T(), 2, len(response.Data))

	// Test filtering by Status
	query = &models.TransactionQuery{
		Status: models.StatusSuccess,
		Page:   1,
		Limit:  10,
	}
	response, err = suite.service.GetTransactions(query)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2), response.Total)
	assert.Equal(suite.T(), 2, len(response.Data))

	// Test pagination
	query = &models.TransactionQuery{
		Page:  1,
		Limit: 2,
	}
	response, err = suite.service.GetTransactions(query)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(4), response.Total)
	assert.Equal(suite.T(), 2, len(response.Data))
	assert.Equal(suite.T(), 2, response.TotalPages)
}

func (suite *TransactionServiceTestSuite) TestUpdateTransaction() {
	// Create a test transaction
	transaction := &models.Transaction{
		UserID: 1,
		Amount: 100.50,
		Status: models.StatusPending,
	}
	err := suite.db.Create(transaction).Error
	suite.Require().NoError(err)

	// Test updating existing transaction
	updateReq := &models.TransactionUpdateRequest{
		Status: models.StatusSuccess,
	}
	result, err := suite.service.UpdateTransaction(transaction.ID, updateReq)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), models.StatusSuccess, result.Status)

	// Test updating non-existing transaction
	result, err = suite.service.UpdateTransaction(999, updateReq)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "transaction not found")
}

func (suite *TransactionServiceTestSuite) TestDeleteTransaction() {
	// Create a test transaction
	transaction := &models.Transaction{
		UserID: 1,
		Amount: 100.50,
		Status: models.StatusPending,
	}
	err := suite.db.Create(transaction).Error
	suite.Require().NoError(err)

	// Test deleting existing transaction
	err = suite.service.DeleteTransaction(transaction.ID)
	assert.NoError(suite.T(), err)

	// Verify transaction is soft deleted
	var deletedTransaction models.Transaction
	err = suite.db.Unscoped().First(&deletedTransaction, transaction.ID).Error
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), deletedTransaction.DeletedAt)

	// Test deleting non-existing transaction
	err = suite.service.DeleteTransaction(999)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "transaction not found")
}

func (suite *TransactionServiceTestSuite) TestGetDashboardSummary() {
	// Create test transactions with different statuses and dates
	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	transactions := []models.Transaction{
		{UserID: 1, Amount: 100.0, Status: models.StatusSuccess, CreatedAt: today.Add(time.Hour)},
		{UserID: 1, Amount: 200.0, Status: models.StatusSuccess, CreatedAt: today.Add(2 * time.Hour)},
		{UserID: 2, Amount: 300.0, Status: models.StatusSuccess, CreatedAt: yesterday},
		{UserID: 2, Amount: 400.0, Status: models.StatusPending, CreatedAt: today.Add(3 * time.Hour)},
		{UserID: 3, Amount: 500.0, Status: models.StatusFailed, CreatedAt: today.Add(4 * time.Hour)},
	}

	for i := range transactions {
		err := suite.db.Create(&transactions[i]).Error
		suite.Require().NoError(err)
	}

	summary, err := suite.service.GetDashboardSummary()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), summary)

	// Check total transactions
	assert.Equal(suite.T(), int64(5), summary.TotalTransactions)

	// Check total successful transactions today (should be 2)
	assert.Equal(suite.T(), int64(2), summary.TotalSuccessToday)

	// Check average amount per user (should be average of successful transactions: (100+200+300)/3 = 200)
	assert.Equal(suite.T(), 200.0, summary.AverageAmountPerUser)

	// Check total amount (successful transactions: 100+200+300 = 600)
	assert.Equal(suite.T(), 600.0, summary.TotalAmount)

	// Check total amount today (successful transactions today: 100+200 = 300)
	assert.Equal(suite.T(), 300.0, summary.TotalAmountToday)

	// Check status distribution
	assert.Equal(suite.T(), int64(3), summary.StatusDistribution["success"])
	assert.Equal(suite.T(), int64(1), summary.StatusDistribution["pending"])
	assert.Equal(suite.T(), int64(1), summary.StatusDistribution["failed"])

	// Check recent transactions (should have 5 transactions)
	assert.Equal(suite.T(), 5, len(summary.RecentTransactions))
}

func TestTransactionServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionServiceTestSuite))
}
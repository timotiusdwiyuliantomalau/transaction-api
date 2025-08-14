package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"transaction-api/internal/models"
	"transaction-api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"github.com/glebarez/sqlite"
)

type TransactionHandlerTestSuite struct {
	suite.Suite
	db      *gorm.DB
	service *services.TransactionService
	handler *TransactionHandler
	router  *gin.Engine
}

func (suite *TransactionHandlerTestSuite) SetupTest() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Use SQLite in-memory database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// Auto migrate the schema
	err = db.AutoMigrate(&models.Transaction{})
	suite.Require().NoError(err)

	suite.db = db
	suite.service = services.NewTransactionService(db)
	suite.handler = NewTransactionHandler(suite.service)

	// Setup router
	router := gin.New()
	router.POST("/transactions", suite.handler.CreateTransaction)
	router.GET("/transactions", suite.handler.GetTransactions)
	router.GET("/transactions/:id", suite.handler.GetTransactionByID)
	router.PUT("/transactions/:id", suite.handler.UpdateTransaction)
	router.DELETE("/transactions/:id", suite.handler.DeleteTransaction)
	router.GET("/dashboard/summary", suite.handler.GetDashboardSummary)
	router.GET("/health", suite.handler.HealthCheck)

	suite.router = router
}

func (suite *TransactionHandlerTestSuite) TearDownTest() {
	sqlDB, err := suite.db.DB()
	suite.Require().NoError(err)
	sqlDB.Close()
}

func (suite *TransactionHandlerTestSuite) TestCreateTransaction() {
	// Test valid request
	reqBody := models.TransactionRequest{
		UserID: 1,
		Amount: 100.50,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/transactions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response models.Transaction
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), uint(1), response.UserID)
	assert.Equal(suite.T(), 100.50, response.Amount)
	assert.Equal(suite.T(), models.StatusPending, response.Status)

	// Test invalid request (missing required fields)
	invalidReq := map[string]interface{}{
		"amount": 100.50,
		// missing user_id
	}
	jsonBody, _ = json.Marshal(invalidReq)

	req, _ = http.NewRequest("POST", "/transactions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *TransactionHandlerTestSuite) TestGetTransactionByID() {
	// Create a test transaction
	transaction := &models.Transaction{
		UserID: 1,
		Amount: 100.50,
		Status: models.StatusSuccess,
	}
	err := suite.db.Create(transaction).Error
	suite.Require().NoError(err)

	// Test getting existing transaction
	req, _ := http.NewRequest("GET", fmt.Sprintf("/transactions/%d", transaction.ID), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.Transaction
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), transaction.ID, response.ID)

	// Test getting non-existing transaction
	req, _ = http.NewRequest("GET", "/transactions/999", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	// Test invalid ID
	req, _ = http.NewRequest("GET", "/transactions/invalid", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *TransactionHandlerTestSuite) TestGetTransactions() {
	// Create test transactions
	transactions := []models.Transaction{
		{UserID: 1, Amount: 100.0, Status: models.StatusSuccess},
		{UserID: 1, Amount: 200.0, Status: models.StatusPending},
		{UserID: 2, Amount: 300.0, Status: models.StatusSuccess},
	}

	for i := range transactions {
		err := suite.db.Create(&transactions[i]).Error
		suite.Require().NoError(err)
	}

	// Test getting all transactions
	req, _ := http.NewRequest("GET", "/transactions", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.TransactionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), response.Total)
	assert.Equal(suite.T(), 3, len(response.Data))

	// Test filtering by user_id
	req, _ = http.NewRequest("GET", "/transactions?user_id=1", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2), response.Total)

	// Test invalid status filter
	req, _ = http.NewRequest("GET", "/transactions?status=invalid", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *TransactionHandlerTestSuite) TestUpdateTransaction() {
	// Create a test transaction
	transaction := &models.Transaction{
		UserID: 1,
		Amount: 100.50,
		Status: models.StatusPending,
	}
	err := suite.db.Create(transaction).Error
	suite.Require().NoError(err)

	// Test updating existing transaction
	updateReq := models.TransactionUpdateRequest{
		Status: models.StatusSuccess,
	}
	jsonBody, _ := json.Marshal(updateReq)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("/transactions/%d", transaction.ID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.Transaction
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.StatusSuccess, response.Status)

	// Test updating non-existing transaction
	req, _ = http.NewRequest("PUT", "/transactions/999", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	// Test invalid status
	invalidUpdateReq := models.TransactionUpdateRequest{
		Status: "invalid",
	}
	jsonBody, _ = json.Marshal(invalidUpdateReq)

	req, _ = http.NewRequest("PUT", fmt.Sprintf("/transactions/%d", transaction.ID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *TransactionHandlerTestSuite) TestDeleteTransaction() {
	// Create a test transaction
	transaction := &models.Transaction{
		UserID: 1,
		Amount: 100.50,
		Status: models.StatusPending,
	}
	err := suite.db.Create(transaction).Error
	suite.Require().NoError(err)

	// Test deleting existing transaction
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/transactions/%d", transaction.ID), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)

	// Test deleting non-existing transaction
	req, _ = http.NewRequest("DELETE", "/transactions/999", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	// Test invalid ID
	req, _ = http.NewRequest("DELETE", "/transactions/invalid", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *TransactionHandlerTestSuite) TestGetDashboardSummary() {
	// Create test transactions
	transactions := []models.Transaction{
		{UserID: 1, Amount: 100.0, Status: models.StatusSuccess},
		{UserID: 2, Amount: 200.0, Status: models.StatusPending},
		{UserID: 3, Amount: 300.0, Status: models.StatusFailed},
	}

	for i := range transactions {
		err := suite.db.Create(&transactions[i]).Error
		suite.Require().NoError(err)
	}

	req, _ := http.NewRequest("GET", "/dashboard/summary", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.DashboardSummary
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), response.TotalTransactions)
	assert.NotNil(suite.T(), response.StatusDistribution)
	assert.Equal(suite.T(), 3, len(response.RecentTransactions))
}

func (suite *TransactionHandlerTestSuite) TestHealthCheck() {
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
	assert.Equal(suite.T(), "transaction-api", response["service"])
}

func TestTransactionHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionHandlerTestSuite))
}
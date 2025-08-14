package models

import (
	"time"

	"gorm.io/gorm"
)

type TransactionStatus string

const (
	StatusPending TransactionStatus = "pending"
	StatusSuccess TransactionStatus = "success"
	StatusFailed  TransactionStatus = "failed"
)

type Transaction struct {
	ID        uint              `json:"id" gorm:"primaryKey"`
	UserID    uint              `json:"user_id" gorm:"not null;index" validate:"required"`
	Amount    float64           `json:"amount" gorm:"not null" validate:"required,gt=0"`
	Status    TransactionStatus `json:"status" gorm:"not null;default:'pending'" validate:"required,oneof=pending success failed"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	DeletedAt gorm.DeletedAt    `json:"-" gorm:"index"`
}

// TransactionRequest represents the request payload for creating transactions
type TransactionRequest struct {
	UserID uint    `json:"user_id" validate:"required"`
	Amount float64 `json:"amount" validate:"required,gt=0"`
}

// TransactionUpdateRequest represents the request payload for updating transactions
type TransactionUpdateRequest struct {
	Status TransactionStatus `json:"status" validate:"required,oneof=pending success failed"`
}

// TransactionQuery represents query parameters for filtering transactions
type TransactionQuery struct {
	UserID uint              `form:"user_id"`
	Status TransactionStatus `form:"status"`
	Limit  int               `form:"limit"`
	Offset int               `form:"offset"`
	Page   int               `form:"page"`
}

// TransactionResponse represents the response structure for transactions
type TransactionResponse struct {
	Data       []Transaction `json:"data"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int           `json:"total_pages"`
}

// DashboardSummary represents the dashboard summary data
type DashboardSummary struct {
	TotalSuccessToday     int64   `json:"total_success_today"`
	AverageAmountPerUser  float64 `json:"average_amount_per_user"`
	TotalTransactions     int64   `json:"total_transactions"`
	RecentTransactions    []Transaction `json:"recent_transactions"`
	TotalAmount           float64 `json:"total_amount"`
	TotalAmountToday      float64 `json:"total_amount_today"`
	StatusDistribution    map[string]int64 `json:"status_distribution"`
}
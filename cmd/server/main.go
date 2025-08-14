package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"transaction-api/internal/config"
	"transaction-api/internal/database"
	"transaction-api/internal/handlers"
	"transaction-api/internal/middleware"
	"transaction-api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}

	// Setup logger
	middleware.SetupLogger(cfg.Log.Level)
	logrus.Info("Starting Transaction API server...")

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Initialize database
	db, err := database.NewDatabase(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize database")
	}
	defer func() {
		if err := db.Close(); err != nil {
			logrus.WithError(err).Error("Failed to close database connection")
		}
	}()

	// Initialize services
	transactionService := services.NewTransactionService(db.DB)

	// Initialize handlers
	transactionHandler := handlers.NewTransactionHandler(transactionService)

	// Setup routes
	router := setupRoutes(transactionHandler)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logrus.WithField("port", cfg.Server.Port).Info("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("Server forced to shutdown")
	} else {
		logrus.Info("Server shutdown complete")
	}
}

func setupRoutes(transactionHandler *handlers.TransactionHandler) *gin.Engine {
	router := gin.New()

	// Add middleware
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.ErrorHandler())
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Health check endpoint
	router.GET("/health", transactionHandler.HealthCheck)

	// API version 1 routes
	v1 := router.Group("/api/v1")
	{
		// Transaction routes
		transactions := v1.Group("/transactions")
		{
			transactions.POST("", transactionHandler.CreateTransaction)
			transactions.GET("", transactionHandler.GetTransactions)
			transactions.GET("/:id", transactionHandler.GetTransactionByID)
			transactions.PUT("/:id", transactionHandler.UpdateTransaction)
			transactions.DELETE("/:id", transactionHandler.DeleteTransaction)
		}

		// Dashboard routes
		dashboard := v1.Group("/dashboard")
		{
			dashboard.GET("/summary", transactionHandler.GetDashboardSummary)
		}
	}

	// Legacy routes (without versioning) for backward compatibility
	router.POST("/transactions", transactionHandler.CreateTransaction)
	router.GET("/transactions", transactionHandler.GetTransactions)
	router.GET("/transactions/:id", transactionHandler.GetTransactionByID)
	router.PUT("/transactions/:id", transactionHandler.UpdateTransaction)
	router.DELETE("/transactions/:id", transactionHandler.DeleteTransaction)
	router.GET("/dashboard/summary", transactionHandler.GetDashboardSummary)

	return router
}
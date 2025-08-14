package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggerMiddleware creates a gin middleware for logging requests
func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logrus.WithFields(logrus.Fields{
			"status_code":  param.StatusCode,
			"latency":      param.Latency,
			"client_ip":    param.ClientIP,
			"method":       param.Method,
			"path":         param.Path,
			"user_agent":   param.Request.UserAgent(),
			"error":        param.ErrorMessage,
		}).Info("HTTP Request")

		return ""
	})
}

// SetupLogger configures the global logger
func SetupLogger(level string) {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}

	logrus.SetLevel(logLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
}
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// ErrorHandler is a middleware that handles panics and errors
func ErrorHandler() gin.HandlerFunc {
	return gin.Recovery()
}

// SendError sends a standardized error response
func SendError(c *gin.Context, statusCode int, err string, message ...string) {
	response := ErrorResponse{
		Error: err,
	}

	if len(message) > 0 {
		response.Message = message[0]
	}

	logrus.WithFields(logrus.Fields{
		"status_code": statusCode,
		"error":       err,
		"path":        c.Request.URL.Path,
		"method":      c.Request.Method,
	}).Error("HTTP Error Response")

	c.JSON(statusCode, response)
}

// SendValidationError sends validation error response
func SendValidationError(c *gin.Context, details interface{}) {
	response := ErrorResponse{
		Error:   "validation_error",
		Message: "Request validation failed",
		Details: details,
	}

	logrus.WithFields(logrus.Fields{
		"error":   "validation_error",
		"details": details,
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
	}).Warn("Validation Error")

	c.JSON(http.StatusBadRequest, response)
}
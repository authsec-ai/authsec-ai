package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HealthController handles SPIRE health check endpoints
type HealthController struct {
	logger *logrus.Entry
}

// NewHealthController creates a new health controller
func NewHealthController(logger *logrus.Entry) *HealthController {
	return &HealthController{
		logger: logger,
	}
}

// Health handles GET /spire/health
func (h *HealthController) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "spire"})
}

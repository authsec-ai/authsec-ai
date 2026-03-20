package sdkmgr

import (
	"net/http"
	"strings"

	sdkmgrSvc "github.com/authsec-ai/authsec/services/sdkmgr"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DashboardController handles the 5 dashboard endpoints.
type DashboardController struct {
	Service *sdkmgrSvc.DashboardService
}

// NewDashboardController creates a new controller wired to the service.
func NewDashboardController(svc *sdkmgrSvc.DashboardService) *DashboardController {
	return &DashboardController{Service: svc}
}

// ---------- Request types ----------

type getStatisticsRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
}

type getAdminUsersRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
}

// ---------- Handlers ----------

// Health handles GET /health
func (c *DashboardController) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.Service.HealthCheck())
}

// Sessions handles POST /sessions (disabled — returns 0)
func (c *DashboardController) Sessions(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, 0)
}

// Statistics handles POST /statistics (requires auth)
func (c *DashboardController) Statistics(ctx *gin.Context) {
	var req getStatisticsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logrus.WithField("tenant_id", req.TenantID).Info("fetching session statistics")
	result := c.Service.GetSessionStatistics(req.TenantID)

	if success, ok := result["success"].(bool); ok && !success {
		errMsg, _ := result["error"].(string)
		if errMsg != "" && strings.Contains(strings.ToLower(errMsg), "not found") {
			ctx.JSON(http.StatusNotFound, result)
			return
		}
		ctx.JSON(http.StatusInternalServerError, result)
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// Users handles POST /users (disabled — returns 0)
func (c *DashboardController) Users(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, 0)
}

// AdminUsers handles POST /admin-users (requires auth)
func (c *DashboardController) AdminUsers(ctx *gin.Context) {
	var req getAdminUsersRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logrus.WithField("tenant_id", req.TenantID).Info("fetching admin users")
	result := c.Service.GetAdminUsers(req.TenantID)

	if success, ok := result["success"].(bool); ok && !success {
		ctx.JSON(http.StatusInternalServerError, result)
		return
	}

	ctx.JSON(http.StatusOK, result)
}


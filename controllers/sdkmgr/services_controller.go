package sdkmgr

import (
	"net/http"

	sdkmgrSvc "github.com/authsec-ai/authsec/services/sdkmgr"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ServicesController handles the 3 Services endpoints.
type ServicesController struct {
	Service *sdkmgrSvc.ServicesService
}

// NewServicesController creates a new controller wired to the service.
func NewServicesController(svc *sdkmgrSvc.ServicesService) *ServicesController {
	return &ServicesController{Service: svc}
}

// ---------- Request types ----------

type serviceCredentialsRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	ServiceName string `json:"service_name" binding:"required"`
}

type userDetailsRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	ServiceName string `json:"service_name" binding:"required"`
}

// ---------- Handlers ----------

// Health handles GET /health
func (c *ServicesController) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.Service.HealthCheck())
}

// GetCredentials handles POST /credentials
func (c *ServicesController) GetCredentials(ctx *gin.Context) {
	var req serviceCredentialsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.Service.GetServiceCredentials(req.SessionID, req.ServiceName)
	if err != nil {
		logrus.WithError(err).WithField("service_name", req.ServiceName).Error("get credentials failed")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// GetUserDetails handles POST /user-details
func (c *ServicesController) GetUserDetails(ctx *gin.Context) {
	var req userDetailsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := c.Service.GetServiceUserDetails(req.SessionID)
	ctx.JSON(http.StatusOK, result)
}

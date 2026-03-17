package platform

import (
	"log"
	"net/http"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
)

// HubSpotController handles HubSpot integration endpoints
type HubSpotController struct {
	hubspotService *services.HubSpotService
}

// NewHubSpotController creates a new HubSpot controller
func NewHubSpotController() *HubSpotController {
	cfg := config.GetConfig()
	return &HubSpotController{
		hubspotService: services.NewHubSpotService(cfg.HubSpotAccessToken),
	}
}

// SyncContactRequest represents the request body for contact sync
type SyncContactRequest struct {
	Email        string `json:"email" binding:"required,email"`
	TenantDomain string `json:"tenant_domain" binding:"required"`
	TenantID     string `json:"tenant_id" binding:"required"`
}

// SyncContact syncs a contact to HubSpot CRM
// @Summary Sync contact to HubSpot
// @Description Creates or updates a contact in HubSpot CRM. Non-blocking: returns success even if HubSpot sync fails.
// @Tags HubSpot
// @Accept json
// @Produce json
// @Param input body SyncContactRequest true "Contact data to sync"
// @Success 200 {object} map[string]interface{} "Contact synced successfully"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid input"
// @Failure 500 {object} map[string]interface{} "HubSpot sync failed"
// @Router /uflow/hubspot/contacts/sync [post]
func (hc *HubSpotController) SyncContact(c *gin.Context) {
	var req SyncContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	contactID, err := hc.hubspotService.SyncContact(req.Email, req.TenantDomain, req.TenantID)
	if err != nil {
		log.Printf("[HubSpot] Failed to sync contact %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to sync contact to HubSpot",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"hubspot_contact_id": contactID,
		"message":            "Contact synced successfully",
	})
}

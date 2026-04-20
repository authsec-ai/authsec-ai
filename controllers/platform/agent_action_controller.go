package platform

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AgentActionController handles HTTP endpoints for the Agent Action Guard
type AgentActionController struct {
	actionService *services.AgentActionService
}

// NewAgentActionController creates a new agent action controller
func NewAgentActionController() (*AgentActionController, error) {
	db := config.GetDatabase()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	pushService, err := services.NewPushNotificationService()
	if err != nil {
		fmt.Printf("Warning: Push notification service not initialized: %v\n", err)
		pushService = nil
	}

	return &AgentActionController{
		actionService: services.NewAgentActionService(db, pushService),
	}, nil
}

// ========================================
// Agent-facing endpoints (JWT auth)
// ========================================

// EvaluateAction evaluates an agent action for risk and returns approval status
// @Summary Evaluate agent action
// @Description Any AI agent calls this to request approval for a risky action. Returns auto_approved for low risk, or pending with polling info for higher risk.
// @Tags Agent Action Guard
// @Accept json
// @Produce json
// @Param request body models.AgentActionEvaluateRequest true "Action evaluation request"
// @Success 200 {object} models.AgentActionEvaluateResponse "Evaluation result"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/agent/actions/evaluate [post]
func (ctrl *AgentActionController) EvaluateAction(c *gin.Context) {
	var req models.AgentActionEvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	userIDStr, _ := middlewares.ResolveUserID(c)
	userEmail := ""
	if email, exists := c.Get("email"); exists {
		userEmail = email.(string)
	}
	tenantIDStr := ""
	if tid, exists := c.Get("tenant_id"); exists {
		tenantIDStr = tid.(string)
	}

	if req.UserEmail == "" {
		req.UserEmail = userEmail
	}

	resp, err := ctrl.actionService.EvaluateAction(&req, userIDStr, userEmail, tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to evaluate action", "details": err.Error()})
		return
	}

	if resp.Error != "" && resp.Error == models.AgentErrorUserNotFound {
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	middlewares.Audit(c, "agent_action", resp.ActionReqID, "evaluate", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"agent_id":   req.AgentID,
			"action":     req.Action,
			"resource":   req.Resource,
			"risk_score": resp.RiskScore,
			"risk_level": resp.RiskLevel,
			"status":     resp.Status,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// PollActionStatus polls for the status of an action request
// @Summary Poll action status
// @Description Agent polls this endpoint to check if the action has been approved or denied
// @Tags Agent Action Guard
// @Accept json
// @Produce json
// @Param action_req_id query string true "Action request ID"
// @Success 200 {object} models.AgentActionStatusResponse "Action status"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/agent/actions/status [get]
func (ctrl *AgentActionController) PollActionStatus(c *gin.Context) {
	actionReqID := c.Query("action_req_id")
	if actionReqID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action_req_id is required"})
		return
	}

	resp, err := ctrl.actionService.PollActionStatus(actionReqID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to poll action status", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ========================================
// Human-facing endpoints (JWT auth)
// ========================================

// RespondToAction allows a human to approve or deny an agent action
// @Summary Respond to action request
// @Description Human approves or denies an agent action via mobile app or web
// @Tags Agent Action Guard
// @Accept json
// @Produce json
// @Param request body models.AgentActionRespondRequest true "Response"
// @Success 200 {object} models.AgentActionRespondResponse "Response result"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/agent/actions/respond [post]
func (ctrl *AgentActionController) RespondToAction(c *gin.Context) {
	var req models.AgentActionRespondRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, _ := uuid.Parse(userIDStr)

	approverEmail := ""
	if email, exists := c.Get("email"); exists {
		approverEmail = email.(string)
	}

	resp, err := ctrl.actionService.RespondToAction(
		req.ActionReqID,
		userID,
		approverEmail,
		req.Approved,
		req.Reason,
		req.BiometricVerified,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to respond to action", "details": err.Error()})
		return
	}

	decision := "denied"
	if req.Approved {
		decision = "approved"
	}

	middlewares.Audit(c, "agent_action", req.ActionReqID, "respond", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"decision":  decision,
			"reason":    req.Reason,
			"biometric": req.BiometricVerified,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// GetPendingActions returns pending (non-expired) action requests for the authenticated user.
// Filters by both tenant_id and user_id — only the user whose account triggered the action sees it.
func (ctrl *AgentActionController) GetPendingActions(c *gin.Context) {
	tenantID := ctrl.getTenantID(c)
	if tenantID == (uuid.UUID{}) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant not found"})
		return
	}

	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	actions, err := ctrl.actionService.GetPendingActions(tenantID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending actions"})
		return
	}

	if actions == nil {
		actions = []models.AgentActionRequest{}
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(actions),
		"actions": actions,
	})
}

// ========================================
// Admin endpoints (JWT auth + admin role)
// ========================================

// ListRiskPolicies lists all risk policies for the tenant
// @Summary List risk policies
// @Description Retrieves all risk policies for the authenticated tenant
// @Tags Agent Action Guard - Admin
// @Produce json
// @Success 200 {object} models.RiskPolicyListResponse "Risk policies"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/admin/risk-policies [get]
func (ctrl *AgentActionController) ListRiskPolicies(c *gin.Context) {
	tenantID := ctrl.getTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	policies, err := ctrl.actionService.GetRiskPolicies(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list risk policies", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.RiskPolicyListResponse{
		Success:  true,
		Policies: policies,
		Total:    len(policies),
	})
}

// CreateRiskPolicy creates a new risk policy
// @Summary Create risk policy
// @Description Creates a new risk policy for the tenant
// @Tags Agent Action Guard - Admin
// @Accept json
// @Produce json
// @Param request body models.RiskPolicyCreateRequest true "Risk policy"
// @Success 201 {object} models.RiskPolicyResponse "Created policy"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/admin/risk-policies [post]
func (ctrl *AgentActionController) CreateRiskPolicy(c *gin.Context) {
	tenantID := ctrl.getTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	var req models.RiskPolicyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	policy, err := ctrl.actionService.CreateRiskPolicy(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create risk policy", "details": err.Error()})
		return
	}

	middlewares.Audit(c, "risk_policy", policy.ID.String(), "create", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"name":           policy.Name,
			"action_pattern": policy.ActionPattern,
			"base_score":     policy.BaseScore,
		},
	})

	c.JSON(http.StatusCreated, models.RiskPolicyResponse{
		Success: true,
		Policy:  policy,
		Message: "Risk policy created",
	})
}

// UpdateRiskPolicy updates an existing risk policy
// @Summary Update risk policy
// @Description Updates an existing risk policy
// @Tags Agent Action Guard - Admin
// @Accept json
// @Produce json
// @Param id path string true "Policy ID"
// @Param request body models.RiskPolicyUpdateRequest true "Updates"
// @Success 200 {object} models.RiskPolicyResponse "Updated policy"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/admin/risk-policies/{id} [put]
func (ctrl *AgentActionController) UpdateRiskPolicy(c *gin.Context) {
	tenantID := ctrl.getTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	policyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	var req models.RiskPolicyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	policy, err := ctrl.actionService.UpdateRiskPolicy(policyID, tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update risk policy", "details": err.Error()})
		return
	}

	middlewares.Audit(c, "risk_policy", policy.ID.String(), "update", nil)

	c.JSON(http.StatusOK, models.RiskPolicyResponse{
		Success: true,
		Policy:  policy,
		Message: "Risk policy updated",
	})
}

// DeleteRiskPolicy soft-deletes a risk policy
// @Summary Delete risk policy
// @Description Deactivates a risk policy
// @Tags Agent Action Guard - Admin
// @Produce json
// @Param id path string true "Policy ID"
// @Success 200 {object} map[string]string "Deleted"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/admin/risk-policies/{id} [delete]
func (ctrl *AgentActionController) DeleteRiskPolicy(c *gin.Context) {
	tenantID := ctrl.getTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	policyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	if err := ctrl.actionService.DeleteRiskPolicy(policyID, tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete risk policy", "details": err.Error()})
		return
	}

	middlewares.Audit(c, "risk_policy", policyID.String(), "delete", nil)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Risk policy deleted"})
}

// GetSettings retrieves tenant agent guard settings
// @Summary Get agent guard settings
// @Tags Agent Action Guard - Admin
// @Produce json
// @Success 200 {object} models.AgentGuardSettingsResponse "Settings"
// @Router /uflow/admin/agent-guard/settings [get]
func (ctrl *AgentActionController) GetSettings(c *gin.Context) {
	tenantID := ctrl.getTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	settings, err := ctrl.actionService.GetSettings(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.AgentGuardSettingsResponse{
		Success:  true,
		Settings: settings,
	})
}

// UpdateSettings updates tenant agent guard settings
// @Summary Update agent guard settings
// @Tags Agent Action Guard - Admin
// @Accept json
// @Produce json
// @Param request body models.AgentGuardSettingsRequest true "Settings update"
// @Success 200 {object} models.AgentGuardSettingsResponse "Updated settings"
// @Router /uflow/admin/agent-guard/settings [put]
func (ctrl *AgentActionController) UpdateSettings(c *gin.Context) {
	tenantID := ctrl.getTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	var req models.AgentGuardSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	settings, err := ctrl.actionService.UpdateSettings(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings", "details": err.Error()})
		return
	}

	middlewares.Audit(c, "agent_guard_settings", tenantID.String(), "update", nil)

	c.JSON(http.StatusOK, models.AgentGuardSettingsResponse{
		Success:  true,
		Settings: settings,
		Message:  "Settings updated",
	})
}

// GetAuditLog retrieves the agent action audit log
// @Summary Get agent audit log
// @Tags Agent Action Guard - Admin
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} models.AgentAuditListResponse "Audit log"
// @Router /uflow/admin/agent-audit [get]
func (ctrl *AgentActionController) GetAuditLog(c *gin.Context) {
	tenantID := ctrl.getTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	entries, total, err := ctrl.actionService.GetAuditLog(tenantID, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit log", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.AgentAuditListResponse{
		Success: true,
		Entries: entries,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

// ========================================
// Helpers
// ========================================

func (ctrl *AgentActionController) getTenantID(c *gin.Context) uuid.UUID {
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return uuid.Nil
	}
	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id"})
		return uuid.Nil
	}
	return tenantID
}

package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// AgentActionService orchestrates the human-in-the-loop approval flow for agent actions.
// It ties together risk evaluation, push notifications (via existing CIBA infra), and audit logging.
type AgentActionService struct {
	actionRepo  *database.AgentActionRepository
	riskEngine  *RiskEngineService
	pushService *PushNotificationService
}

// NewAgentActionService creates a new agent action service
func NewAgentActionService(
	db *database.DBConnection,
	pushService *PushNotificationService,
) *AgentActionService {
	actionRepo := database.NewAgentActionRepository(db)
	return &AgentActionService{
		actionRepo:  actionRepo,
		riskEngine:  NewRiskEngineService(actionRepo),
		pushService: pushService,
	}
}

// EvaluateAction is the main entry point — any agent calls this.
// userID, userEmail, tenantIDStr come from the JWT (set by AuthMiddleware).
func (s *AgentActionService) EvaluateAction(req *models.AgentActionEvaluateRequest, jwtUserID, jwtUserEmail, jwtTenantID string) (*models.AgentActionEvaluateResponse, error) {

	// Step 1: Use identity from JWT — already authenticated by middleware
	userID, err := uuid.Parse(jwtUserID)
	if err != nil {
		return &models.AgentActionEvaluateResponse{
			Error:            models.AgentErrorUserNotFound,
			ErrorDescription: "Invalid user ID from token",
		}, nil
	}

	// Resolve tenant: prefer JWT tenant, fallback to client_id lookup
	var tenantID uuid.UUID
	if jwtTenantID != "" {
		tenantID, err = uuid.Parse(jwtTenantID)
		if err != nil {
			return &models.AgentActionEvaluateResponse{
				Error:            models.AgentErrorInvalidAction,
				ErrorDescription: "Invalid tenant ID from token",
			}, nil
		}
	} else if req.ClientID != "" {
		clientID, err := uuid.Parse(req.ClientID)
		if err != nil {
			return &models.AgentActionEvaluateResponse{
				Error:            models.AgentErrorInvalidAction,
				ErrorDescription: "Invalid client_id format",
			}, nil
		}
		tenantID, err = s.resolveTenantFromClientID(clientID)
		if err != nil {
			return &models.AgentActionEvaluateResponse{
				Error:            models.AgentErrorInvalidAction,
				ErrorDescription: fmt.Sprintf("Could not resolve tenant: %v", err),
			}, nil
		}
	} else {
		return &models.AgentActionEvaluateResponse{
			Error:            models.AgentErrorInvalidAction,
			ErrorDescription: "No tenant_id in token and no client_id in request",
		}, nil
	}

	userEmail := jwtUserEmail
	if userEmail == "" {
		userEmail = req.UserEmail
	}

	// Step 2: Get tenant settings (or create defaults)
	settings, err := s.actionRepo.GetOrCreateSettings(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent guard settings: %w", err)
	}

	// Step 3: Evaluate risk
	evaluation, err := s.riskEngine.Evaluate(
		tenantID, req.AgentID,
		req.Action, req.Resource,
		req.Metadata, settings,
	)
	if err != nil {
		return nil, fmt.Errorf("risk evaluation failed: %w", err)
	}

	// Step 4: Generate action request ID
	actionReqID, err := s.generateActionReqID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate action request ID: %w", err)
	}

	// Step 5: Build risk factors as JSONBArray
	var riskFactorsArray models.JSONBArray
	for _, f := range evaluation.Factors {
		riskFactorsArray = append(riskFactorsArray, map[string]interface{}{
			"factor": f.Factor,
			"score":  f.Score,
			"reason": f.Reason,
		})
	}

	// Step 6: Create the action request record
	expiresAt := time.Now().Add(time.Duration(settings.ApprovalTimeoutSeconds) * time.Second).Unix()

	actionRequest := &models.AgentActionRequest{
		ID:          uuid.New(),
		ActionReqID: actionReqID,
		TenantID:    tenantID,
		UserID:      userID,
		UserEmail:   userEmail,

		AgentID:        req.AgentID,
		AgentName:      req.AgentName,
		AgentFramework: req.AgentFramework,
		SessionID:      req.SessionID,

		Action:   req.Action,
		Resource: req.Resource,
		Detail:   req.Detail,
		Metadata: models.JSONBMap(req.Metadata),

		RiskScore:       evaluation.Score,
		RiskLevel:       evaluation.Level,
		RiskFactors:     riskFactorsArray,
		MatchedPolicyID: evaluation.MatchedPolicyID,

		ApprovalType:      evaluation.ApprovalType,
		RequiredApprovals: evaluation.RequiredApprovals,
		ReceivedApprovals: 0,

		ExpiresAt: expiresAt,
	}

	// Step 7: Auto-approve if risk is low enough
	if evaluation.ApprovalType == "auto" {
		actionRequest.Status = models.AgentActionAutoApproved
		now := time.Now().Unix()
		actionRequest.DecidedAt = &now

		if err := s.actionRepo.CreateActionRequest(actionRequest); err != nil {
			return nil, fmt.Errorf("failed to create action request: %w", err)
		}

		s.writeAuditLog(actionRequest, models.AgentActionAutoApproved, nil)

		return &models.AgentActionEvaluateResponse{
			ActionReqID:  actionReqID,
			Status:       models.AgentActionAutoApproved,
			RiskScore:    evaluation.Score,
			RiskLevel:    evaluation.Level,
			ApprovalType: "auto",
			Message:      "Action auto-approved based on risk assessment",
		}, nil
	}

	// Step 8: Requires human approval — send push notification
	actionRequest.Status = "pending"

	tenantIDStr := tenantID.String()
	tenantDB, tenantDBErr := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)

	var deviceToken string
	var deviceTokenID *uuid.UUID

	if tenantDBErr == nil {
		tenantRepo := database.NewTenantDeviceRepository(tenantDB)
		devices, devErr := tenantRepo.GetTenantDeviceTokensByUserID(userID, tenantID)
		if devErr == nil && len(devices) > 0 {
			deviceToken = devices[0].DeviceToken
			deviceTokenID = &devices[0].ID
		}
	}

	if deviceToken == "" {
		if err := s.actionRepo.CreateActionRequest(actionRequest); err != nil {
			return nil, fmt.Errorf("failed to create action request: %w", err)
		}

		return &models.AgentActionEvaluateResponse{
			ActionReqID:  actionReqID,
			Status:       "pending",
			RiskScore:    evaluation.Score,
			RiskLevel:    evaluation.Level,
			ApprovalType: evaluation.ApprovalType,
			ExpiresIn:    settings.ApprovalTimeoutSeconds,
			Interval:     settings.PollingIntervalSeconds,
			Message:      "Action requires approval. No push device registered — approve via web or API.",
		}, nil
	}

	actionRequest.DeviceTokenID = deviceTokenID

	if err := s.actionRepo.CreateActionRequest(actionRequest); err != nil {
		return nil, fmt.Errorf("failed to create action request: %w", err)
	}

	if s.pushService != nil {
		bindingMessage := fmt.Sprintf("[%s] %s wants to %s on %s",
			evaluation.Level, req.AgentName, req.Action, req.Resource)
		if req.Detail != "" {
			bindingMessage = fmt.Sprintf("[%s] %s: %s", evaluation.Level, req.AgentName, req.Detail)
		}

		err = s.pushService.SendActionApproval(
			deviceToken,
			actionReqID,
			bindingMessage,
			userEmail,
			req.Action,
			req.Resource,
			evaluation.Score,
			evaluation.Level,
		)
		if err != nil {
			fmt.Printf("Warning: failed to send action approval push: %v\n", err)
		}
	}

	return &models.AgentActionEvaluateResponse{
		ActionReqID:  actionReqID,
		Status:       "pending",
		RiskScore:    evaluation.Score,
		RiskLevel:    evaluation.Level,
		ApprovalType: evaluation.ApprovalType,
		ExpiresIn:    settings.ApprovalTimeoutSeconds,
		Interval:     settings.PollingIntervalSeconds,
		Message:      "Action requires human approval. Push notification sent.",
	}, nil
}

// PollActionStatus checks the current status of an action request (agent polls this)
func (s *AgentActionService) PollActionStatus(actionReqID string) (*models.AgentActionStatusResponse, error) {

	req, err := s.actionRepo.GetActionRequestByID(actionReqID)
	if err != nil {
		return &models.AgentActionStatusResponse{
			Error:            "not_found",
			ErrorDescription: "Action request not found",
		}, nil
	}

	s.actionRepo.UpdateLastPolled(actionReqID)

	if req.Status == "pending" && req.IsExpired() {
		s.actionRepo.UpdateActionRequestStatus(actionReqID, models.AgentActionExpired)
		s.writeAuditLog(req, models.AgentActionExpired, nil)

		return &models.AgentActionStatusResponse{
			ActionReqID:      actionReqID,
			Status:           models.AgentActionExpired,
			RiskScore:        req.RiskScore,
			RiskLevel:        req.RiskLevel,
			Error:            "expired_token",
			ErrorDescription: "Approval request expired without response",
		}, nil
	}

	switch req.Status {
	case "pending":
		return &models.AgentActionStatusResponse{
			ActionReqID:      actionReqID,
			Status:           "pending",
			RiskScore:        req.RiskScore,
			RiskLevel:        req.RiskLevel,
			Error:            models.AgentActionPending,
			ErrorDescription: "Waiting for human approval",
		}, nil

	case models.AgentActionApproved, models.AgentActionAutoApproved:
		return &models.AgentActionStatusResponse{
			ActionReqID: actionReqID,
			Status:      req.Status,
			RiskScore:   req.RiskScore,
			RiskLevel:   req.RiskLevel,
			Decision:    "approved",
		}, nil

	case models.AgentActionDenied:
		return &models.AgentActionStatusResponse{
			ActionReqID:      actionReqID,
			Status:           models.AgentActionDenied,
			RiskScore:        req.RiskScore,
			RiskLevel:        req.RiskLevel,
			Decision:         "denied",
			Error:            "access_denied",
			ErrorDescription: "Human denied this action",
		}, nil

	default:
		return &models.AgentActionStatusResponse{
			ActionReqID:      actionReqID,
			Status:           req.Status,
			RiskScore:        req.RiskScore,
			RiskLevel:        req.RiskLevel,
			Error:            "expired_token",
			ErrorDescription: "Request is no longer active",
		}, nil
	}
}

// RespondToAction processes a human's approval or denial
func (s *AgentActionService) RespondToAction(
	actionReqID string,
	approverUserID uuid.UUID,
	approverEmail string,
	approved bool,
	reason string,
	biometricVerified bool,
) (*models.AgentActionRespondResponse, error) {

	req, err := s.actionRepo.GetActionRequestByID(actionReqID)
	if err != nil {
		return &models.AgentActionRespondResponse{
			Success: false,
			Message: "Action request not found",
		}, nil
	}

	if !req.IsPending() {
		return &models.AgentActionRespondResponse{
			Success: false,
			Message: fmt.Sprintf("Request is no longer pending (status: %s)", req.Status),
		}, nil
	}

	decision := &models.AgentActionDecision{
		ID:                uuid.New(),
		ActionRequestID:   req.ID,
		ApproverUserID:    approverUserID,
		ApproverEmail:     approverEmail,
		BiometricVerified: biometricVerified,
	}

	if approved {
		decision.Decision = "approved"
	} else {
		decision.Decision = "denied"
		decision.Reason = reason
	}

	if err := s.actionRepo.CreateDecision(decision); err != nil {
		return nil, fmt.Errorf("failed to record decision: %w", err)
	}

	if !approved {
		s.actionRepo.UpdateActionRequestStatus(actionReqID, models.AgentActionDenied)
		s.writeAuditLog(req, models.AgentActionDenied, []map[string]interface{}{
			{"user_id": approverUserID.String(), "email": approverEmail, "decision": "denied", "reason": reason},
		})

		return &models.AgentActionRespondResponse{
			Success: true,
			Message: "Action denied",
		}, nil
	}

	received, required, err := s.actionRepo.IncrementApprovalCount(actionReqID)
	if err != nil {
		return nil, fmt.Errorf("failed to update approval count: %w", err)
	}

	if received >= required {
		s.actionRepo.UpdateActionRequestStatus(actionReqID, models.AgentActionApproved)
		s.writeAuditLog(req, models.AgentActionApproved, []map[string]interface{}{
			{"user_id": approverUserID.String(), "email": approverEmail, "decision": "approved"},
		})

		return &models.AgentActionRespondResponse{
			Success: true,
			Message: "Action approved",
		}, nil
	}

	return &models.AgentActionRespondResponse{
		Success: true,
		Message: fmt.Sprintf("Approval recorded (%d/%d required)", received, required),
	}, nil
}

// GetPendingActions returns all pending (non-expired) action requests for a specific user in a tenant.
func (s *AgentActionService) GetPendingActions(tenantID uuid.UUID, userID uuid.UUID) ([]models.AgentActionRequest, error) {
	return s.actionRepo.GetPendingActionsByUser(tenantID, userID)
}

// GetRiskPolicies retrieves all risk policies for a tenant
func (s *AgentActionService) GetRiskPolicies(tenantID uuid.UUID) ([]models.RiskPolicy, error) {
	return s.actionRepo.GetRiskPoliciesByTenant(tenantID)
}

// CreateRiskPolicy creates a new risk policy
func (s *AgentActionService) CreateRiskPolicy(tenantID uuid.UUID, req *models.RiskPolicyCreateRequest) (*models.RiskPolicy, error) {
	policy := &models.RiskPolicy{
		ID:                        uuid.New(),
		TenantID:                  tenantID,
		Name:                      req.Name,
		Description:               req.Description,
		ActionPattern:             req.ActionPattern,
		ResourcePattern:           req.ResourcePattern,
		EnvironmentPattern:        req.EnvironmentPattern,
		BaseScore:                 req.BaseScore,
		ScopeBulkThreshold:        req.ScopeBulkThreshold,
		ScopeBulkModifier:         req.ScopeBulkModifier,
		PIIModifier:               req.PIIModifier,
		FinancialModifier:         req.FinancialModifier,
		OffHoursModifier:          req.OffHoursModifier,
		FirstTimeModifier:         req.FirstTimeModifier,
		AutoApproveBelow:          req.AutoApproveBelow,
		RequireApprovalAbove:      req.RequireApprovalAbove,
		RequireMultiApprovalAbove: req.RequireMultiApprovalAbove,
		IsActive:                  true,
		Priority:                  req.Priority,
	}

	if policy.ResourcePattern == "" {
		policy.ResourcePattern = "*"
	}
	if policy.EnvironmentPattern == "" {
		policy.EnvironmentPattern = "*"
	}

	if err := s.actionRepo.CreateRiskPolicy(policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// UpdateRiskPolicy updates an existing risk policy
func (s *AgentActionService) UpdateRiskPolicy(policyID uuid.UUID, tenantID uuid.UUID, req *models.RiskPolicyUpdateRequest) (*models.RiskPolicy, error) {
	policy, err := s.actionRepo.GetRiskPolicyByID(policyID, tenantID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		policy.Name = *req.Name
	}
	if req.Description != nil {
		policy.Description = *req.Description
	}
	if req.ActionPattern != nil {
		policy.ActionPattern = *req.ActionPattern
	}
	if req.ResourcePattern != nil {
		policy.ResourcePattern = *req.ResourcePattern
	}
	if req.EnvironmentPattern != nil {
		policy.EnvironmentPattern = *req.EnvironmentPattern
	}
	if req.BaseScore != nil {
		policy.BaseScore = *req.BaseScore
	}
	if req.ScopeBulkThreshold != nil {
		policy.ScopeBulkThreshold = *req.ScopeBulkThreshold
	}
	if req.ScopeBulkModifier != nil {
		policy.ScopeBulkModifier = *req.ScopeBulkModifier
	}
	if req.PIIModifier != nil {
		policy.PIIModifier = *req.PIIModifier
	}
	if req.FinancialModifier != nil {
		policy.FinancialModifier = *req.FinancialModifier
	}
	if req.OffHoursModifier != nil {
		policy.OffHoursModifier = *req.OffHoursModifier
	}
	if req.FirstTimeModifier != nil {
		policy.FirstTimeModifier = *req.FirstTimeModifier
	}
	if req.AutoApproveBelow != nil {
		policy.AutoApproveBelow = req.AutoApproveBelow
	}
	if req.RequireApprovalAbove != nil {
		policy.RequireApprovalAbove = req.RequireApprovalAbove
	}
	if req.RequireMultiApprovalAbove != nil {
		policy.RequireMultiApprovalAbove = req.RequireMultiApprovalAbove
	}
	if req.IsActive != nil {
		policy.IsActive = *req.IsActive
	}
	if req.Priority != nil {
		policy.Priority = *req.Priority
	}

	if err := s.actionRepo.UpdateRiskPolicy(policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// DeleteRiskPolicy soft-deletes a risk policy
func (s *AgentActionService) DeleteRiskPolicy(policyID uuid.UUID, tenantID uuid.UUID) error {
	return s.actionRepo.DeleteRiskPolicy(policyID, tenantID)
}

// GetSettings retrieves tenant guard settings
func (s *AgentActionService) GetSettings(tenantID uuid.UUID) (*models.AgentGuardSettings, error) {
	return s.actionRepo.GetOrCreateSettings(tenantID)
}

// UpdateSettings updates tenant guard settings
func (s *AgentActionService) UpdateSettings(tenantID uuid.UUID, req *models.AgentGuardSettingsRequest) (*models.AgentGuardSettings, error) {
	settings, err := s.actionRepo.GetOrCreateSettings(tenantID)
	if err != nil {
		return nil, err
	}

	if req.AutoApproveBelow != nil {
		settings.AutoApproveBelow = *req.AutoApproveBelow
	}
	if req.RequireApprovalAbove != nil {
		settings.RequireApprovalAbove = *req.RequireApprovalAbove
	}
	if req.RequireMultiApprovalAbove != nil {
		settings.RequireMultiApprovalAbove = *req.RequireMultiApprovalAbove
	}
	if req.ApprovalTimeoutSeconds != nil {
		settings.ApprovalTimeoutSeconds = *req.ApprovalTimeoutSeconds
	}
	if req.PollingIntervalSeconds != nil {
		settings.PollingIntervalSeconds = *req.PollingIntervalSeconds
	}
	if req.BusinessHoursStart != nil {
		settings.BusinessHoursStart = *req.BusinessHoursStart
	}
	if req.BusinessHoursEnd != nil {
		settings.BusinessHoursEnd = *req.BusinessHoursEnd
	}
	if req.BusinessHoursTimezone != nil {
		settings.BusinessHoursTimezone = *req.BusinessHoursTimezone
	}
	if req.RequireBiometric != nil {
		settings.RequireBiometric = *req.RequireBiometric
	}

	if err := s.actionRepo.UpdateSettings(settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// GetAuditLog retrieves the audit log for a tenant
func (s *AgentActionService) GetAuditLog(tenantID uuid.UUID, page, perPage int) ([]models.AgentActionAuditLog, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return s.actionRepo.GetAuditLog(tenantID, page, perPage)
}

// CleanupExpiredRequests expires and cleans up old requests
func (s *AgentActionService) CleanupExpiredRequests() (int64, error) {
	return s.actionRepo.ExpireOldActionRequests()
}

// ========================================
// Internal helpers
// ========================================

func (s *AgentActionService) generateActionReqID() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return "act_" + base64.URLEncoding.EncodeToString(bytes), nil
}

// resolveTenantFromClientID maps client_id → tenant_id via tenant_mappings table
func (s *AgentActionService) resolveTenantFromClientID(clientID uuid.UUID) (uuid.UUID, error) {
	db := s.actionRepo.GetDB()
	if db == nil {
		return uuid.UUID{}, fmt.Errorf("database not initialized")
	}

	var tenantID uuid.UUID
	query := `SELECT tenant_id FROM tenant_mappings WHERE client_id = $1`
	err := db.QueryRow(query, clientID).Scan(&tenantID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("client not found in tenant_mappings: %w", err)
	}

	return tenantID, nil
}

func (s *AgentActionService) writeAuditLog(req *models.AgentActionRequest, finalStatus string, decidedBy []map[string]interface{}) {
	var decidedByArray models.JSONBArray
	for _, d := range decidedBy {
		decidedByArray = append(decidedByArray, d)
	}

	now := time.Now().Unix()
	var durationMs *int64
	if req.CreatedAt > 0 {
		d := (now - req.CreatedAt) * 1000
		durationMs = &d
	}

	entry := &models.AgentActionAuditLog{
		ID:              uuid.New(),
		TenantID:        req.TenantID,
		ActionRequestID: &req.ID,
		AgentID:         req.AgentID,
		AgentName:       req.AgentName,
		UserID:          req.UserID,
		UserEmail:       req.UserEmail,
		Action:          req.Action,
		Resource:        req.Resource,
		Detail:          req.Detail,
		Metadata:        req.Metadata,
		RiskScore:       req.RiskScore,
		RiskLevel:       req.RiskLevel,
		FinalStatus:     finalStatus,
		DecidedBy:       decidedByArray,
		RequestedAt:     req.CreatedAt,
		DecidedAt:       &now,
		ExecutionDurationMs: durationMs,
	}

	if err := s.actionRepo.CreateAuditEntry(entry); err != nil {
		fmt.Printf("Warning: failed to write audit log: %v\n", err)
	}
}

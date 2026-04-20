package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// AgentActionRepository handles agent action request and risk policy database operations
type AgentActionRepository struct {
	db *DBConnection
}

// NewAgentActionRepository creates a new agent action repository
func NewAgentActionRepository(db *DBConnection) *AgentActionRepository {
	return &AgentActionRepository{db: db}
}

// GetDB returns the underlying database connection for direct queries
func (r *AgentActionRepository) GetDB() *DBConnection {
	return r.db
}

// ========================================
// Risk Policy Operations
// ========================================

// CreateRiskPolicy creates a new risk policy for a tenant
func (r *AgentActionRepository) CreateRiskPolicy(policy *models.RiskPolicy) error {
	now := time.Now().Unix()
	policy.CreatedAt = now
	policy.UpdatedAt = now

	query := `
		INSERT INTO risk_policies (
			id, tenant_id, name, description,
			action_pattern, resource_pattern, environment_pattern,
			base_score, scope_bulk_threshold, scope_bulk_modifier,
			pii_modifier, financial_modifier, off_hours_modifier, first_time_modifier,
			auto_approve_below, require_approval_above, require_multi_approval_above,
			is_active, priority, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`

	_, err := r.db.Exec(query,
		policy.ID,
		policy.TenantID,
		policy.Name,
		policy.Description,
		policy.ActionPattern,
		policy.ResourcePattern,
		policy.EnvironmentPattern,
		policy.BaseScore,
		policy.ScopeBulkThreshold,
		policy.ScopeBulkModifier,
		policy.PIIModifier,
		policy.FinancialModifier,
		policy.OffHoursModifier,
		policy.FirstTimeModifier,
		policy.AutoApproveBelow,
		policy.RequireApprovalAbove,
		policy.RequireMultiApprovalAbove,
		policy.IsActive,
		policy.Priority,
		policy.CreatedAt,
		policy.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create risk policy: %w", err)
	}

	return nil
}

// GetRiskPoliciesByTenant retrieves all active risk policies for a tenant, ordered by priority
func (r *AgentActionRepository) GetRiskPoliciesByTenant(tenantID uuid.UUID) ([]models.RiskPolicy, error) {
	query := `
		SELECT id, tenant_id, name, description,
		       action_pattern, resource_pattern, environment_pattern,
		       base_score, scope_bulk_threshold, scope_bulk_modifier,
		       pii_modifier, financial_modifier, off_hours_modifier, first_time_modifier,
		       auto_approve_below, require_approval_above, require_multi_approval_above,
		       is_active, priority, created_at, updated_at
		FROM risk_policies
		WHERE tenant_id = $1 AND is_active = TRUE
		ORDER BY priority DESC, created_at ASC
	`

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query risk policies: %w", err)
	}
	defer rows.Close()

	var policies []models.RiskPolicy
	for rows.Next() {
		var p models.RiskPolicy
		err := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.Description,
			&p.ActionPattern, &p.ResourcePattern, &p.EnvironmentPattern,
			&p.BaseScore, &p.ScopeBulkThreshold, &p.ScopeBulkModifier,
			&p.PIIModifier, &p.FinancialModifier, &p.OffHoursModifier, &p.FirstTimeModifier,
			&p.AutoApproveBelow, &p.RequireApprovalAbove, &p.RequireMultiApprovalAbove,
			&p.IsActive, &p.Priority, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan risk policy: %w", err)
		}
		policies = append(policies, p)
	}

	return policies, nil
}

// GetRiskPolicyByID retrieves a single risk policy
func (r *AgentActionRepository) GetRiskPolicyByID(policyID uuid.UUID, tenantID uuid.UUID) (*models.RiskPolicy, error) {
	query := `
		SELECT id, tenant_id, name, description,
		       action_pattern, resource_pattern, environment_pattern,
		       base_score, scope_bulk_threshold, scope_bulk_modifier,
		       pii_modifier, financial_modifier, off_hours_modifier, first_time_modifier,
		       auto_approve_below, require_approval_above, require_multi_approval_above,
		       is_active, priority, created_at, updated_at
		FROM risk_policies
		WHERE id = $1 AND tenant_id = $2
	`

	var p models.RiskPolicy
	err := r.db.QueryRow(query, policyID, tenantID).Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Description,
		&p.ActionPattern, &p.ResourcePattern, &p.EnvironmentPattern,
		&p.BaseScore, &p.ScopeBulkThreshold, &p.ScopeBulkModifier,
		&p.PIIModifier, &p.FinancialModifier, &p.OffHoursModifier, &p.FirstTimeModifier,
		&p.AutoApproveBelow, &p.RequireApprovalAbove, &p.RequireMultiApprovalAbove,
		&p.IsActive, &p.Priority, &p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("risk policy not found")
		}
		return nil, fmt.Errorf("failed to get risk policy: %w", err)
	}

	return &p, nil
}

// UpdateRiskPolicy updates an existing risk policy
func (r *AgentActionRepository) UpdateRiskPolicy(policy *models.RiskPolicy) error {
	now := time.Now().Unix()
	policy.UpdatedAt = now

	query := `
		UPDATE risk_policies SET
			name = $1, description = $2,
			action_pattern = $3, resource_pattern = $4, environment_pattern = $5,
			base_score = $6, scope_bulk_threshold = $7, scope_bulk_modifier = $8,
			pii_modifier = $9, financial_modifier = $10, off_hours_modifier = $11, first_time_modifier = $12,
			auto_approve_below = $13, require_approval_above = $14, require_multi_approval_above = $15,
			is_active = $16, priority = $17, updated_at = $18
		WHERE id = $19 AND tenant_id = $20
	`

	result, err := r.db.Exec(query,
		policy.Name, policy.Description,
		policy.ActionPattern, policy.ResourcePattern, policy.EnvironmentPattern,
		policy.BaseScore, policy.ScopeBulkThreshold, policy.ScopeBulkModifier,
		policy.PIIModifier, policy.FinancialModifier, policy.OffHoursModifier, policy.FirstTimeModifier,
		policy.AutoApproveBelow, policy.RequireApprovalAbove, policy.RequireMultiApprovalAbove,
		policy.IsActive, policy.Priority, policy.UpdatedAt,
		policy.ID, policy.TenantID,
	)

	if err != nil {
		return fmt.Errorf("failed to update risk policy: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("risk policy not found")
	}

	return nil
}

// DeleteRiskPolicy soft-deletes a risk policy
func (r *AgentActionRepository) DeleteRiskPolicy(policyID uuid.UUID, tenantID uuid.UUID) error {
	now := time.Now().Unix()
	query := `
		UPDATE risk_policies SET is_active = FALSE, updated_at = $1
		WHERE id = $2 AND tenant_id = $3
	`

	result, err := r.db.Exec(query, now, policyID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete risk policy: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("risk policy not found")
	}

	return nil
}

// ========================================
// Agent Guard Settings Operations
// ========================================

// GetOrCreateSettings retrieves tenant settings, creating defaults if needed
func (r *AgentActionRepository) GetOrCreateSettings(tenantID uuid.UUID) (*models.AgentGuardSettings, error) {
	query := `
		SELECT id, tenant_id,
		       auto_approve_below, require_approval_above, require_multi_approval_above,
		       approval_timeout_seconds, polling_interval_seconds,
		       business_hours_start, business_hours_end, business_hours_timezone,
		       default_approver_user_id, require_biometric,
		       created_at, updated_at
		FROM agent_guard_settings
		WHERE tenant_id = $1
	`

	var s models.AgentGuardSettings
	err := r.db.QueryRow(query, tenantID).Scan(
		&s.ID, &s.TenantID,
		&s.AutoApproveBelow, &s.RequireApprovalAbove, &s.RequireMultiApprovalAbove,
		&s.ApprovalTimeoutSeconds, &s.PollingIntervalSeconds,
		&s.BusinessHoursStart, &s.BusinessHoursEnd, &s.BusinessHoursTimezone,
		&s.DefaultApproverUserID, &s.RequireBiometric,
		&s.CreatedAt, &s.UpdatedAt,
	)

	if err == nil {
		return &s, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get agent guard settings: %w", err)
	}

	// Create defaults
	now := time.Now().Unix()
	s = models.AgentGuardSettings{
		ID:                        uuid.New(),
		TenantID:                  tenantID,
		AutoApproveBelow:          30,
		RequireApprovalAbove:      31,
		RequireMultiApprovalAbove: 81,
		ApprovalTimeoutSeconds:    300,
		PollingIntervalSeconds:    5,
		BusinessHoursStart:        9,
		BusinessHoursEnd:          17,
		BusinessHoursTimezone:     "UTC",
		RequireBiometric:          true,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	insertQuery := `
		INSERT INTO agent_guard_settings (
			id, tenant_id,
			auto_approve_below, require_approval_above, require_multi_approval_above,
			approval_timeout_seconds, polling_interval_seconds,
			business_hours_start, business_hours_end, business_hours_timezone,
			default_approver_user_id, require_biometric,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (tenant_id) DO NOTHING
	`

	_, err = r.db.Exec(insertQuery,
		s.ID, s.TenantID,
		s.AutoApproveBelow, s.RequireApprovalAbove, s.RequireMultiApprovalAbove,
		s.ApprovalTimeoutSeconds, s.PollingIntervalSeconds,
		s.BusinessHoursStart, s.BusinessHoursEnd, s.BusinessHoursTimezone,
		s.DefaultApproverUserID, s.RequireBiometric,
		s.CreatedAt, s.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create default agent guard settings: %w", err)
	}

	return &s, nil
}

// UpdateSettings updates tenant agent guard settings
func (r *AgentActionRepository) UpdateSettings(settings *models.AgentGuardSettings) error {
	now := time.Now().Unix()
	settings.UpdatedAt = now

	query := `
		UPDATE agent_guard_settings SET
			auto_approve_below = $1, require_approval_above = $2, require_multi_approval_above = $3,
			approval_timeout_seconds = $4, polling_interval_seconds = $5,
			business_hours_start = $6, business_hours_end = $7, business_hours_timezone = $8,
			default_approver_user_id = $9, require_biometric = $10,
			updated_at = $11
		WHERE tenant_id = $12
	`

	_, err := r.db.Exec(query,
		settings.AutoApproveBelow, settings.RequireApprovalAbove, settings.RequireMultiApprovalAbove,
		settings.ApprovalTimeoutSeconds, settings.PollingIntervalSeconds,
		settings.BusinessHoursStart, settings.BusinessHoursEnd, settings.BusinessHoursTimezone,
		settings.DefaultApproverUserID, settings.RequireBiometric,
		settings.UpdatedAt, settings.TenantID,
	)

	if err != nil {
		return fmt.Errorf("failed to update agent guard settings: %w", err)
	}

	return nil
}

// ========================================
// Agent Action Request Operations
// ========================================

// CreateActionRequest creates a new agent action request
func (r *AgentActionRepository) CreateActionRequest(req *models.AgentActionRequest) error {
	now := time.Now().Unix()
	req.CreatedAt = now

	riskFactorsJSON, err := json.Marshal(req.RiskFactors)
	if err != nil {
		return fmt.Errorf("failed to marshal risk factors: %w", err)
	}

	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO agent_action_requests (
			id, action_req_id, tenant_id, user_id, user_email,
			agent_id, agent_name, agent_framework, session_id,
			action, resource, detail, metadata,
			risk_score, risk_level, risk_factors, matched_policy_id,
			status, approval_type, required_approvals, received_approvals,
			ciba_auth_req_id, device_token_id,
			expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
	`

	_, err = r.db.Exec(query,
		req.ID, req.ActionReqID, req.TenantID, req.UserID, req.UserEmail,
		req.AgentID, req.AgentName, req.AgentFramework, req.SessionID,
		req.Action, req.Resource, req.Detail, metadataJSON,
		req.RiskScore, req.RiskLevel, riskFactorsJSON, req.MatchedPolicyID,
		req.Status, req.ApprovalType, req.RequiredApprovals, req.ReceivedApprovals,
		req.CIBAAuthReqID, req.DeviceTokenID,
		req.ExpiresAt, req.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create agent action request: %w", err)
	}

	return nil
}

// GetActionRequestByID retrieves an agent action request by action_req_id
func (r *AgentActionRepository) GetActionRequestByID(actionReqID string) (*models.AgentActionRequest, error) {
	query := `
		SELECT id, action_req_id, tenant_id, user_id, user_email,
		       agent_id, agent_name, agent_framework, session_id,
		       action, resource, detail, metadata,
		       risk_score, risk_level, risk_factors, matched_policy_id,
		       status, approval_type, required_approvals, received_approvals,
		       ciba_auth_req_id, device_token_id,
		       expires_at, created_at, decided_at, last_polled_at
		FROM agent_action_requests
		WHERE action_req_id = $1
	`

	var req models.AgentActionRequest
	var metadataJSON, riskFactorsJSON []byte

	err := r.db.QueryRow(query, actionReqID).Scan(
		&req.ID, &req.ActionReqID, &req.TenantID, &req.UserID, &req.UserEmail,
		&req.AgentID, &req.AgentName, &req.AgentFramework, &req.SessionID,
		&req.Action, &req.Resource, &req.Detail, &metadataJSON,
		&req.RiskScore, &req.RiskLevel, &riskFactorsJSON, &req.MatchedPolicyID,
		&req.Status, &req.ApprovalType, &req.RequiredApprovals, &req.ReceivedApprovals,
		&req.CIBAAuthReqID, &req.DeviceTokenID,
		&req.ExpiresAt, &req.CreatedAt, &req.DecidedAt, &req.LastPolledAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent action request not found")
		}
		return nil, fmt.Errorf("failed to get agent action request: %w", err)
	}

	// Unmarshal JSONB fields
	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &req.Metadata)
	}
	if riskFactorsJSON != nil {
		json.Unmarshal(riskFactorsJSON, &req.RiskFactors)
	}

	return &req, nil
}

// GetPendingActionsByUser returns all non-expired pending action requests for a specific user in a tenant.
// Filters by both tenant_id and user_id so only the affected user sees their notifications.
func (r *AgentActionRepository) GetPendingActionsByUser(tenantID uuid.UUID, userID uuid.UUID) ([]models.AgentActionRequest, error) {
	now := time.Now().Unix()
	query := `
		SELECT id, action_req_id, tenant_id, user_id, user_email,
		       agent_id, agent_name, agent_framework, session_id,
		       action, resource, detail, metadata,
		       risk_score, risk_level, risk_factors, matched_policy_id,
		       status, approval_type, required_approvals, received_approvals,
		       ciba_auth_req_id, device_token_id,
		       expires_at, created_at, decided_at, last_polled_at
		FROM agent_action_requests
		WHERE tenant_id = $1 AND user_id = $2 AND status = 'pending' AND expires_at > $3
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, tenantID, userID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending actions: %w", err)
	}
	defer rows.Close()

	var results []models.AgentActionRequest
	for rows.Next() {
		var req models.AgentActionRequest
		var metadataJSON, riskFactorsJSON []byte
		var matchedPolicyID, deviceTokenID sql.NullString
		var decidedAt, lastPolledAt sql.NullInt64

		err := rows.Scan(
			&req.ID, &req.ActionReqID, &req.TenantID, &req.UserID, &req.UserEmail,
			&req.AgentID, &req.AgentName, &req.AgentFramework, &req.SessionID,
			&req.Action, &req.Resource, &req.Detail, &metadataJSON,
			&req.RiskScore, &req.RiskLevel, &riskFactorsJSON, &matchedPolicyID,
			&req.Status, &req.ApprovalType, &req.RequiredApprovals, &req.ReceivedApprovals,
			&req.CIBAAuthReqID, &deviceTokenID,
			&req.ExpiresAt, &req.CreatedAt, &decidedAt, &lastPolledAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending action: %w", err)
		}

		if matchedPolicyID.Valid {
			id := uuid.MustParse(matchedPolicyID.String)
			req.MatchedPolicyID = &id
		}
		if deviceTokenID.Valid {
			id := uuid.MustParse(deviceTokenID.String)
			req.DeviceTokenID = &id
		}
		if decidedAt.Valid {
			req.DecidedAt = &decidedAt.Int64
		}
		if lastPolledAt.Valid {
			req.LastPolledAt = &lastPolledAt.Int64
		}
		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &req.Metadata)
		}
		if riskFactorsJSON != nil {
			json.Unmarshal(riskFactorsJSON, &req.RiskFactors)
		}

		results = append(results, req)
	}

	return results, nil
}

// UpdateActionRequestStatus updates the status of an action request
func (r *AgentActionRepository) UpdateActionRequestStatus(actionReqID string, status string) error {
	now := time.Now().Unix()
	query := `
		UPDATE agent_action_requests
		SET status = $1, decided_at = $2
		WHERE action_req_id = $3
	`

	_, err := r.db.Exec(query, status, now, actionReqID)
	if err != nil {
		return fmt.Errorf("failed to update action request status: %w", err)
	}

	return nil
}

// IncrementApprovalCount increments the received_approvals count
func (r *AgentActionRepository) IncrementApprovalCount(actionReqID string) (int, int, error) {
	query := `
		UPDATE agent_action_requests
		SET received_approvals = received_approvals + 1
		WHERE action_req_id = $1
		RETURNING received_approvals, required_approvals
	`

	var received, required int
	err := r.db.QueryRow(query, actionReqID).Scan(&received, &required)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to increment approval count: %w", err)
	}

	return received, required, nil
}

// UpdateLastPolled updates the last_polled_at timestamp
func (r *AgentActionRepository) UpdateLastPolled(actionReqID string) error {
	now := time.Now().Unix()
	query := `UPDATE agent_action_requests SET last_polled_at = $1 WHERE action_req_id = $2`
	_, err := r.db.Exec(query, now, actionReqID)
	return err
}

// ExpireOldActionRequests marks expired pending requests
func (r *AgentActionRepository) ExpireOldActionRequests() (int64, error) {
	now := time.Now().Unix()
	query := `
		UPDATE agent_action_requests
		SET status = 'expired', decided_at = $1
		WHERE expires_at < $1 AND status = 'pending'
	`

	result, err := r.db.Exec(query, now)
	if err != nil {
		return 0, fmt.Errorf("failed to expire old action requests: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

// HasPriorAction checks if an agent has previously executed this action+resource combo
func (r *AgentActionRepository) HasPriorAction(agentID string, action string, resource string, tenantID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM agent_action_audit_log
			WHERE agent_id = $1 AND action = $2 AND resource = $3 AND tenant_id = $4
			AND final_status IN ('approved', 'auto_approved')
		)
	`

	var exists bool
	err := r.db.QueryRow(query, agentID, action, resource, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check prior action: %w", err)
	}

	return exists, nil
}

// ========================================
// Agent Action Decision Operations
// ========================================

// CreateDecision records a human's approval/denial
func (r *AgentActionRepository) CreateDecision(decision *models.AgentActionDecision) error {
	now := time.Now().Unix()
	decision.CreatedAt = now

	query := `
		INSERT INTO agent_action_decisions (
			id, action_request_id, approver_user_id, approver_email,
			decision, reason, biometric_verified, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(query,
		decision.ID, decision.ActionRequestID, decision.ApproverUserID, decision.ApproverEmail,
		decision.Decision, decision.Reason, decision.BiometricVerified, decision.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create decision: %w", err)
	}

	return nil
}

// ========================================
// Audit Log Operations
// ========================================

// CreateAuditEntry writes an immutable audit log entry
func (r *AgentActionRepository) CreateAuditEntry(entry *models.AgentActionAuditLog) error {
	now := time.Now().Unix()
	entry.CreatedAt = now

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal audit metadata: %w", err)
	}

	decidedByJSON, err := json.Marshal(entry.DecidedBy)
	if err != nil {
		return fmt.Errorf("failed to marshal decided_by: %w", err)
	}

	query := `
		INSERT INTO agent_action_audit_log (
			id, tenant_id, action_request_id,
			agent_id, agent_name, user_id, user_email,
			action, resource, detail, metadata,
			risk_score, risk_level, final_status, decided_by,
			requested_at, decided_at, execution_duration_ms, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	_, err = r.db.Exec(query,
		entry.ID, entry.TenantID, entry.ActionRequestID,
		entry.AgentID, entry.AgentName, entry.UserID, entry.UserEmail,
		entry.Action, entry.Resource, entry.Detail, metadataJSON,
		entry.RiskScore, entry.RiskLevel, entry.FinalStatus, decidedByJSON,
		entry.RequestedAt, entry.DecidedAt, entry.ExecutionDurationMs, entry.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create audit entry: %w", err)
	}

	return nil
}

// GetAuditLog retrieves paginated audit entries for a tenant
func (r *AgentActionRepository) GetAuditLog(tenantID uuid.UUID, page, perPage int) ([]models.AgentActionAuditLog, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM agent_action_audit_log WHERE tenant_id = $1`
	var total int
	err := r.db.QueryRow(countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit entries: %w", err)
	}

	offset := (page - 1) * perPage
	query := `
		SELECT id, tenant_id, action_request_id,
		       agent_id, agent_name, user_id, user_email,
		       action, resource, detail, metadata,
		       risk_score, risk_level, final_status, decided_by,
		       requested_at, decided_at, execution_duration_ms, created_at
		FROM agent_action_audit_log
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, tenantID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	var entries []models.AgentActionAuditLog
	for rows.Next() {
		var e models.AgentActionAuditLog
		var metadataJSON, decidedByJSON []byte

		err := rows.Scan(
			&e.ID, &e.TenantID, &e.ActionRequestID,
			&e.AgentID, &e.AgentName, &e.UserID, &e.UserEmail,
			&e.Action, &e.Resource, &e.Detail, &metadataJSON,
			&e.RiskScore, &e.RiskLevel, &e.FinalStatus, &decidedByJSON,
			&e.RequestedAt, &e.DecidedAt, &e.ExecutionDurationMs, &e.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &e.Metadata)
		}
		if decidedByJSON != nil {
			json.Unmarshal(decidedByJSON, &e.DecidedBy)
		}

		entries = append(entries, e)
	}

	return entries, total, nil
}

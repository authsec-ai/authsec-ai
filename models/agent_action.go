package models

import (
	"time"

	"github.com/google/uuid"
)

// ========================================
// Risk Policy (Tenant-configurable rules)
// ========================================

// RiskPolicy defines a rule for scoring agent actions
type RiskPolicy struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`

	// Rule matching
	Name               string `json:"name" gorm:"size:100;not null"`
	Description        string `json:"description,omitempty" gorm:"size:500"`
	ActionPattern      string `json:"action_pattern" gorm:"size:255;not null"`      // glob: "DELETE *", "SEND_EMAIL"
	ResourcePattern    string `json:"resource_pattern" gorm:"size:255;default:'*'"` // glob: "users", "billing.*"
	EnvironmentPattern string `json:"environment_pattern" gorm:"size:100;default:'*'"`

	// Scoring
	BaseScore          int `json:"base_score" gorm:"default:50;not null"`
	ScopeBulkThreshold int `json:"scope_bulk_threshold" gorm:"default:100"`
	ScopeBulkModifier  int `json:"scope_bulk_modifier" gorm:"default:30"`
	PIIModifier        int `json:"pii_modifier" gorm:"default:20"`
	FinancialModifier  int `json:"financial_modifier" gorm:"default:40"`
	OffHoursModifier   int `json:"off_hours_modifier" gorm:"default:10"`
	FirstTimeModifier  int `json:"first_time_modifier" gorm:"default:10"`

	// Threshold overrides (nil = use tenant defaults)
	AutoApproveBelow          *int `json:"auto_approve_below,omitempty"`
	RequireApprovalAbove      *int `json:"require_approval_above,omitempty"`
	RequireMultiApprovalAbove *int `json:"require_multi_approval_above,omitempty"`

	// Flags
	IsActive bool `json:"is_active" gorm:"default:true;not null;index"`
	Priority int  `json:"priority" gorm:"default:0;not null"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// TableName specifies the table name for RiskPolicy
func (RiskPolicy) TableName() string {
	return "risk_policies"
}

// ========================================
// Agent Guard Settings (Tenant defaults)
// ========================================

// AgentGuardSettings holds tenant-level defaults for risk thresholds
type AgentGuardSettings struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;uniqueIndex"`

	AutoApproveBelow          int `json:"auto_approve_below" gorm:"default:30"`
	RequireApprovalAbove      int `json:"require_approval_above" gorm:"default:31"`
	RequireMultiApprovalAbove int `json:"require_multi_approval_above" gorm:"default:81"`

	ApprovalTimeoutSeconds int    `json:"approval_timeout_seconds" gorm:"default:300"`
	PollingIntervalSeconds int    `json:"polling_interval_seconds" gorm:"default:5"`
	BusinessHoursStart     int    `json:"business_hours_start" gorm:"default:9"`
	BusinessHoursEnd       int    `json:"business_hours_end" gorm:"default:17"`
	BusinessHoursTimezone  string `json:"business_hours_timezone" gorm:"size:50;default:'UTC'"`

	DefaultApproverUserID *uuid.UUID `json:"default_approver_user_id,omitempty" gorm:"type:uuid"`
	RequireBiometric      bool       `json:"require_biometric" gorm:"default:true"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// TableName specifies the table name for AgentGuardSettings
func (AgentGuardSettings) TableName() string {
	return "agent_guard_settings"
}

// ========================================
// Agent Action Request
// ========================================

// AgentActionRequest represents a request from an AI agent that may need human approval
type AgentActionRequest struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ActionReqID string    `json:"action_req_id" gorm:"uniqueIndex;size:255;not null"`

	// Tenant + user context
	TenantID  uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	UserEmail string    `json:"user_email" gorm:"size:255;not null"`

	// Agent context (framework-agnostic)
	AgentID        string `json:"agent_id" gorm:"size:255;not null;index"`
	AgentName      string `json:"agent_name,omitempty" gorm:"size:255"`
	AgentFramework string `json:"agent_framework,omitempty" gorm:"size:100"`
	SessionID      string `json:"session_id,omitempty" gorm:"size:255;index"`

	// Action details
	Action   string     `json:"action" gorm:"size:255;not null"`
	Resource string     `json:"resource" gorm:"size:255;not null"`
	Detail   string     `json:"detail,omitempty" gorm:"type:text"`
	Metadata JSONBMap   `json:"metadata" gorm:"type:jsonb;default:'{}'"`

	// Risk evaluation result
	RiskScore       int        `json:"risk_score" gorm:"not null"`
	RiskLevel       string     `json:"risk_level" gorm:"size:20;not null"`
	RiskFactors     JSONBArray `json:"risk_factors" gorm:"type:jsonb;default:'[]'"`
	MatchedPolicyID *uuid.UUID `json:"matched_policy_id,omitempty" gorm:"type:uuid"`

	// Approval state
	Status            string `json:"status" gorm:"size:50;default:'pending';not null;index"`
	ApprovalType      string `json:"approval_type,omitempty" gorm:"size:20"`
	RequiredApprovals int    `json:"required_approvals" gorm:"default:1"`
	ReceivedApprovals int    `json:"received_approvals" gorm:"default:0"`

	// CIBA link
	CIBAAuthReqID string     `json:"ciba_auth_req_id,omitempty" gorm:"size:255"`
	DeviceTokenID *uuid.UUID `json:"device_token_id,omitempty" gorm:"type:uuid"`

	// Timestamps (Unix epoch)
	ExpiresAt    int64  `json:"expires_at" gorm:"not null;index"`
	CreatedAt    int64  `json:"created_at"`
	DecidedAt    *int64 `json:"decided_at,omitempty"`
	LastPolledAt *int64 `json:"last_polled_at,omitempty"`
}

// TableName specifies the table name for AgentActionRequest
func (AgentActionRequest) TableName() string {
	return "agent_action_requests"
}

// IsExpired checks if the action request has expired
func (a *AgentActionRequest) IsExpired() bool {
	return time.Now().Unix() > a.ExpiresAt
}

// IsPending checks if still waiting for approval
func (a *AgentActionRequest) IsPending() bool {
	return a.Status == "pending" && !a.IsExpired()
}

// IsApproved checks if the action was approved
func (a *AgentActionRequest) IsApproved() bool {
	return a.Status == "approved" || a.Status == "auto_approved"
}

// ========================================
// Agent Action Decision (multi-party)
// ========================================

// AgentActionDecision records a single approval/denial from a human
type AgentActionDecision struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ActionRequestID uuid.UUID `json:"action_request_id" gorm:"type:uuid;not null;index"`
	ApproverUserID  uuid.UUID `json:"approver_user_id" gorm:"type:uuid;not null;index"`
	ApproverEmail   string    `json:"approver_email" gorm:"size:255;not null"`

	Decision          string `json:"decision" gorm:"size:20;not null"` // approved, denied
	Reason            string `json:"reason,omitempty" gorm:"type:text"`
	BiometricVerified bool   `json:"biometric_verified" gorm:"default:false"`

	CreatedAt int64 `json:"created_at"`
}

// TableName specifies the table name for AgentActionDecision
func (AgentActionDecision) TableName() string {
	return "agent_action_decisions"
}

// ========================================
// Agent Action Audit Log
// ========================================

// AgentActionAuditLog is an immutable record of agent actions and outcomes
type AgentActionAuditLog struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID        uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	ActionRequestID *uuid.UUID `json:"action_request_id,omitempty" gorm:"type:uuid"`

	// Snapshot
	AgentID   string    `json:"agent_id" gorm:"size:255;not null;index"`
	AgentName string    `json:"agent_name,omitempty" gorm:"size:255"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	UserEmail string    `json:"user_email" gorm:"size:255;not null"`
	Action    string    `json:"action" gorm:"size:255;not null;index"`
	Resource  string    `json:"resource" gorm:"size:255;not null"`
	Detail    string    `json:"detail,omitempty" gorm:"type:text"`
	Metadata  JSONBMap  `json:"metadata" gorm:"type:jsonb;default:'{}'"`

	// Outcome
	RiskScore   int        `json:"risk_score" gorm:"not null"`
	RiskLevel   string     `json:"risk_level" gorm:"size:20;not null;index"`
	FinalStatus string     `json:"final_status" gorm:"size:50;not null;index"`
	DecidedBy   JSONBArray `json:"decided_by" gorm:"type:jsonb;default:'[]'"`

	// Timing
	RequestedAt         int64  `json:"requested_at" gorm:"not null"`
	DecidedAt           *int64 `json:"decided_at,omitempty"`
	ExecutionDurationMs *int64 `json:"execution_duration_ms,omitempty"`

	CreatedAt int64 `json:"created_at" gorm:"not null;index"`
}

// TableName specifies the table name for AgentActionAuditLog
func (AgentActionAuditLog) TableName() string {
	return "agent_action_audit_log"
}

// ========================================
// JSONB helper types
// ========================================

// JSONBMap is a map[string]interface{} that serializes to JSONB
type JSONBMap map[string]interface{}

// JSONBArray is a []interface{} that serializes to JSONB
type JSONBArray []interface{}

// ========================================
// Risk Evaluation Result (internal)
// ========================================

// RiskFactor represents a single factor contributing to the risk score
type RiskFactor struct {
	Factor string `json:"factor"` // "base", "bulk", "pii", "financial", "off_hours", "first_time"
	Score  int    `json:"score"`  // points added
	Reason string `json:"reason"` // human-readable explanation
}

// RiskEvaluation is the output of the risk engine
type RiskEvaluation struct {
	Score             int          `json:"score"`
	Level             string       `json:"level"`            // low, medium, high, critical
	Factors           []RiskFactor `json:"factors"`
	ApprovalType      string       `json:"approval_type"`    // auto, single, multi
	RequiredApprovals int          `json:"required_approvals"`
	MatchedPolicyID   *uuid.UUID   `json:"matched_policy_id,omitempty"`
}

// ========================================
// DTOs for Agent Action API
// ========================================

// AgentActionEvaluateRequest - Any agent calls this to request action approval
type AgentActionEvaluateRequest struct {
	// Tenant resolution
	ClientID string `json:"client_id" binding:"required"` // OIDC client ID → resolves tenant via tenant_mappings

	// Agent identification
	AgentID        string `json:"agent_id" binding:"required"`
	AgentName      string `json:"agent_name,omitempty"`
	AgentFramework string `json:"agent_framework,omitempty"` // langchain, crewai, mcp, custom
	SessionID      string `json:"session_id,omitempty"`

	// Action details
	Action   string                 `json:"action" binding:"required"`   // DELETE, SEND_EMAIL, TRANSFER_MONEY
	Resource string                 `json:"resource" binding:"required"` // users, billing.invoices
	Detail   string                 `json:"detail,omitempty"`            // "Delete 1M inactive users"
	Metadata map[string]interface{} `json:"metadata,omitempty"`          // {row_count: 1000000, env: "prod"}

	// User context (who owns this agent)
	UserEmail string `json:"user_email" binding:"required"`
}

// AgentActionEvaluateResponse - Returned after evaluation
type AgentActionEvaluateResponse struct {
	ActionReqID      string `json:"action_req_id"`
	Status           string `json:"status"`                  // auto_approved, pending
	RiskScore        int    `json:"risk_score"`
	RiskLevel        string `json:"risk_level"`
	ApprovalType     string `json:"approval_type"`           // auto, single, multi
	ExpiresIn        int    `json:"expires_in,omitempty"`    // seconds until expiry (if pending)
	Interval         int    `json:"interval,omitempty"`      // polling interval (if pending)
	Message          string `json:"message,omitempty"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// AgentActionStatusRequest - Agent polls for decision
type AgentActionStatusRequest struct {
	ActionReqID string `json:"action_req_id" binding:"required"`
}

// AgentActionStatusResponse - Current status of the action request
type AgentActionStatusResponse struct {
	ActionReqID      string `json:"action_req_id"`
	Status           string `json:"status"`           // pending, auto_approved, approved, denied, expired, timed_out
	RiskScore        int    `json:"risk_score"`
	RiskLevel        string `json:"risk_level"`
	Decision         string `json:"decision,omitempty"` // approved, denied (final)
	DecidedBy        string `json:"decided_by,omitempty"`
	Reason           string `json:"reason,omitempty"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// AgentActionRespondRequest - Human approves/denies via mobile app or web
type AgentActionRespondRequest struct {
	ActionReqID       string `json:"action_req_id" binding:"required"`
	Approved          bool   `json:"approved"`
	Reason            string `json:"reason,omitempty"`
	BiometricVerified bool   `json:"biometric_verified,omitempty"`
}

// AgentActionRespondResponse
type AgentActionRespondResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ========================================
// DTOs for Risk Policy Admin API
// ========================================

// RiskPolicyCreateRequest - Create a new risk policy
type RiskPolicyCreateRequest struct {
	Name               string `json:"name" binding:"required"`
	Description        string `json:"description,omitempty"`
	ActionPattern      string `json:"action_pattern" binding:"required"`
	ResourcePattern    string `json:"resource_pattern,omitempty"`
	EnvironmentPattern string `json:"environment_pattern,omitempty"`

	BaseScore          int  `json:"base_score"`
	ScopeBulkThreshold int  `json:"scope_bulk_threshold,omitempty"`
	ScopeBulkModifier  int  `json:"scope_bulk_modifier,omitempty"`
	PIIModifier        int  `json:"pii_modifier,omitempty"`
	FinancialModifier  int  `json:"financial_modifier,omitempty"`
	OffHoursModifier   int  `json:"off_hours_modifier,omitempty"`
	FirstTimeModifier  int  `json:"first_time_modifier,omitempty"`

	AutoApproveBelow          *int `json:"auto_approve_below,omitempty"`
	RequireApprovalAbove      *int `json:"require_approval_above,omitempty"`
	RequireMultiApprovalAbove *int `json:"require_multi_approval_above,omitempty"`

	Priority int `json:"priority,omitempty"`
}

// RiskPolicyUpdateRequest - Update an existing risk policy
type RiskPolicyUpdateRequest struct {
	Name               *string `json:"name,omitempty"`
	Description        *string `json:"description,omitempty"`
	ActionPattern      *string `json:"action_pattern,omitempty"`
	ResourcePattern    *string `json:"resource_pattern,omitempty"`
	EnvironmentPattern *string `json:"environment_pattern,omitempty"`

	BaseScore          *int `json:"base_score,omitempty"`
	ScopeBulkThreshold *int `json:"scope_bulk_threshold,omitempty"`
	ScopeBulkModifier  *int `json:"scope_bulk_modifier,omitempty"`
	PIIModifier        *int `json:"pii_modifier,omitempty"`
	FinancialModifier  *int `json:"financial_modifier,omitempty"`
	OffHoursModifier   *int `json:"off_hours_modifier,omitempty"`
	FirstTimeModifier  *int `json:"first_time_modifier,omitempty"`

	AutoApproveBelow          *int  `json:"auto_approve_below,omitempty"`
	RequireApprovalAbove      *int  `json:"require_approval_above,omitempty"`
	RequireMultiApprovalAbove *int  `json:"require_multi_approval_above,omitempty"`

	IsActive *bool `json:"is_active,omitempty"`
	Priority *int  `json:"priority,omitempty"`
}

// RiskPolicyResponse - Single policy in API response
type RiskPolicyResponse struct {
	Success bool        `json:"success"`
	Policy  *RiskPolicy `json:"policy,omitempty"`
	Message string      `json:"message,omitempty"`
}

// RiskPolicyListResponse - List of policies
type RiskPolicyListResponse struct {
	Success  bool         `json:"success"`
	Policies []RiskPolicy `json:"policies"`
	Total    int          `json:"total"`
}

// AgentGuardSettingsRequest - Update tenant guard settings
type AgentGuardSettingsRequest struct {
	AutoApproveBelow          *int    `json:"auto_approve_below,omitempty"`
	RequireApprovalAbove      *int    `json:"require_approval_above,omitempty"`
	RequireMultiApprovalAbove *int    `json:"require_multi_approval_above,omitempty"`
	ApprovalTimeoutSeconds    *int    `json:"approval_timeout_seconds,omitempty"`
	PollingIntervalSeconds    *int    `json:"polling_interval_seconds,omitempty"`
	BusinessHoursStart        *int    `json:"business_hours_start,omitempty"`
	BusinessHoursEnd          *int    `json:"business_hours_end,omitempty"`
	BusinessHoursTimezone     *string `json:"business_hours_timezone,omitempty"`
	RequireBiometric          *bool   `json:"require_biometric,omitempty"`
}

// AgentGuardSettingsResponse
type AgentGuardSettingsResponse struct {
	Success  bool                `json:"success"`
	Settings *AgentGuardSettings `json:"settings,omitempty"`
	Message  string              `json:"message,omitempty"`
}

// AgentAuditListResponse - Paginated audit log
type AgentAuditListResponse struct {
	Success bool                  `json:"success"`
	Entries []AgentActionAuditLog `json:"entries"`
	Total   int                   `json:"total"`
	Page    int                   `json:"page"`
	PerPage int                   `json:"per_page"`
}

// ========================================
// Agent Action Error Codes
// ========================================

const (
	AgentActionPending      = "authorization_pending" // Waiting for human
	AgentActionApproved     = "approved"              // Human approved
	AgentActionAutoApproved = "auto_approved"         // Risk low enough, auto-approved
	AgentActionDenied       = "denied"                // Human denied
	AgentActionExpired      = "expired"               // Timed out waiting
	AgentActionTimedOut     = "timed_out"             // No response from human

	AgentErrorUserNotFound  = "user_not_found"
	AgentErrorNoDevice      = "no_device_registered"
	AgentErrorInvalidAction = "invalid_action"
)

// ========================================
// Risk Level Constants
// ========================================

const (
	RiskLevelLow      = "low"
	RiskLevelMedium   = "medium"
	RiskLevelHigh     = "high"
	RiskLevelCritical = "critical"
)

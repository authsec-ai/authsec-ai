-- Agent Action Guard: Human-in-the-Loop approval for AI agent actions
-- Adds agent action tables to the master database.
-- Reuses existing CIBA push notification infrastructure (tenant_device_tokens).

-- ========================================
-- Risk Policies (tenant-configurable scoring rules)
-- ========================================

CREATE TABLE IF NOT EXISTS risk_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    -- Rule matching
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    action_pattern VARCHAR(255) NOT NULL,       -- glob pattern: "DELETE", "SEND_EMAIL", "*"
    resource_pattern VARCHAR(255) DEFAULT '*',   -- glob pattern: "users", "billing.*", "*"
    environment_pattern VARCHAR(100) DEFAULT '*', -- "production", "staging", "*"

    -- Scoring
    base_score INTEGER NOT NULL DEFAULT 50,      -- 0-100
    scope_bulk_threshold INTEGER DEFAULT 100,    -- row/item count that triggers bulk modifier
    scope_bulk_modifier INTEGER DEFAULT 30,      -- added when bulk threshold exceeded
    pii_modifier INTEGER DEFAULT 20,             -- added when resource tagged as PII
    financial_modifier INTEGER DEFAULT 40,       -- added when resource tagged as financial
    off_hours_modifier INTEGER DEFAULT 10,       -- added outside business hours
    first_time_modifier INTEGER DEFAULT 10,      -- added when agent runs this action for first time

    -- Thresholds (override tenant defaults if set)
    auto_approve_below INTEGER,                  -- NULL = use tenant default
    require_approval_above INTEGER,              -- NULL = use tenant default
    require_multi_approval_above INTEGER,        -- NULL = use tenant default

    -- Flags
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    priority INTEGER DEFAULT 0 NOT NULL,         -- higher = matched first

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    CONSTRAINT fk_risk_policy_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_risk_policies_tenant ON risk_policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_risk_policies_active ON risk_policies(is_active);
CREATE INDEX IF NOT EXISTS idx_risk_policies_action ON risk_policies(action_pattern);
CREATE UNIQUE INDEX IF NOT EXISTS idx_risk_policies_name_tenant ON risk_policies(tenant_id, name);

-- ========================================
-- Agent Guard Settings (tenant-level defaults)
-- ========================================

CREATE TABLE IF NOT EXISTS agent_guard_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE,

    -- Default thresholds
    auto_approve_below INTEGER DEFAULT 30,          -- score 0-30: auto-approve
    require_approval_above INTEGER DEFAULT 31,      -- score 31-80: single human approval
    require_multi_approval_above INTEGER DEFAULT 81, -- score 81-100: multi-party approval

    -- Timeouts
    approval_timeout_seconds INTEGER DEFAULT 300,    -- 5 minutes
    polling_interval_seconds INTEGER DEFAULT 5,

    -- Business hours (for off-hours modifier)
    business_hours_start INTEGER DEFAULT 9,    -- 09:00
    business_hours_end INTEGER DEFAULT 17,     -- 17:00
    business_hours_timezone VARCHAR(50) DEFAULT 'UTC',

    -- Defaults
    default_approver_user_id UUID,             -- fallback approver
    require_biometric BOOLEAN DEFAULT TRUE,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    CONSTRAINT fk_agent_guard_settings_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE
);

-- ========================================
-- Agent Action Requests (core approval lifecycle)
-- ========================================

CREATE TABLE IF NOT EXISTS agent_action_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_req_id VARCHAR(255) NOT NULL UNIQUE,

    -- Tenant + user context
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,                     -- the human who owns this agent session
    user_email VARCHAR(255) NOT NULL,

    -- Agent context (framework-agnostic)
    agent_id VARCHAR(255) NOT NULL,            -- caller-defined agent identifier
    agent_name VARCHAR(255),                   -- human-readable agent name
    agent_framework VARCHAR(100),              -- "langchain", "crewai", "mcp", "custom", etc.
    session_id VARCHAR(255),                   -- optional: group actions in same session

    -- Action details
    action VARCHAR(255) NOT NULL,              -- "DELETE", "SEND_EMAIL", "TRANSFER_MONEY", etc.
    resource VARCHAR(255) NOT NULL,            -- "users", "billing.invoices", etc.
    detail TEXT,                               -- human-readable description
    metadata JSONB DEFAULT '{}'::jsonb,        -- arbitrary context: row_count, amount, env, etc.

    -- Risk evaluation result
    risk_score INTEGER NOT NULL,               -- 0-100
    risk_level VARCHAR(20) NOT NULL,           -- "low", "medium", "high", "critical"
    risk_factors JSONB DEFAULT '[]'::jsonb,    -- array of {factor, score, reason}
    matched_policy_id UUID,                    -- which risk_policy matched

    -- Approval state
    status VARCHAR(50) DEFAULT 'pending' NOT NULL,  -- pending, auto_approved, approved, denied, expired, timed_out
    approval_type VARCHAR(20),                 -- "auto", "single", "multi"
    required_approvals INTEGER DEFAULT 1,
    received_approvals INTEGER DEFAULT 0,

    -- CIBA link (reuses existing tenant push infra)
    ciba_auth_req_id VARCHAR(255),             -- links to tenant_ciba_auth_requests.auth_req_id
    device_token_id UUID,                      -- links to tenant_device_tokens.id

    -- Timestamps
    expires_at BIGINT NOT NULL,
    created_at BIGINT NOT NULL,
    decided_at BIGINT,
    last_polled_at BIGINT,

    CONSTRAINT fk_agent_action_user FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_agent_action_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_action_req_id ON agent_action_requests(action_req_id);
CREATE INDEX IF NOT EXISTS idx_agent_action_tenant ON agent_action_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_agent_action_user ON agent_action_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_action_agent ON agent_action_requests(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_action_status ON agent_action_requests(status);
CREATE INDEX IF NOT EXISTS idx_agent_action_expires ON agent_action_requests(expires_at);
CREATE INDEX IF NOT EXISTS idx_agent_action_session ON agent_action_requests(session_id);
CREATE INDEX IF NOT EXISTS idx_agent_action_user_status ON agent_action_requests(user_id, status);

-- ========================================
-- Agent Action Decisions (multi-party approval support)
-- ========================================

CREATE TABLE IF NOT EXISTS agent_action_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_request_id UUID NOT NULL,
    approver_user_id UUID NOT NULL,
    approver_email VARCHAR(255) NOT NULL,

    decision VARCHAR(20) NOT NULL,             -- "approved", "denied"
    reason TEXT,                                -- optional: why approved/denied
    biometric_verified BOOLEAN DEFAULT FALSE,

    created_at BIGINT NOT NULL,

    CONSTRAINT fk_decision_action FOREIGN KEY (action_request_id)
        REFERENCES agent_action_requests(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_decision_request ON agent_action_decisions(action_request_id);
CREATE INDEX IF NOT EXISTS idx_agent_decision_approver ON agent_action_decisions(approver_user_id);

-- ========================================
-- Agent Action Audit Log (immutable)
-- ========================================

CREATE TABLE IF NOT EXISTS agent_action_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    action_request_id UUID,

    -- Snapshot of the action (immutable even if request is deleted)
    agent_id VARCHAR(255) NOT NULL,
    agent_name VARCHAR(255),
    user_id UUID NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255) NOT NULL,
    detail TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,

    -- Outcome
    risk_score INTEGER NOT NULL,
    risk_level VARCHAR(20) NOT NULL,
    final_status VARCHAR(50) NOT NULL,         -- auto_approved, approved, denied, expired, timed_out
    decided_by JSONB DEFAULT '[]'::jsonb,      -- array of {user_id, email, decision, timestamp}

    -- Timing
    requested_at BIGINT NOT NULL,
    decided_at BIGINT,
    execution_duration_ms BIGINT,              -- how long the agent waited

    created_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_audit_tenant ON agent_action_audit_log(tenant_id);
CREATE INDEX IF NOT EXISTS idx_agent_audit_agent ON agent_action_audit_log(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_audit_user ON agent_action_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_audit_action ON agent_action_audit_log(action);
CREATE INDEX IF NOT EXISTS idx_agent_audit_risk ON agent_action_audit_log(risk_level);
CREATE INDEX IF NOT EXISTS idx_agent_audit_status ON agent_action_audit_log(final_status);
CREATE INDEX IF NOT EXISTS idx_agent_audit_created ON agent_action_audit_log(created_at);

-- ========================================
-- Comments
-- ========================================

COMMENT ON TABLE risk_policies IS 'Tenant-configurable rules for scoring AI agent actions';
COMMENT ON TABLE agent_guard_settings IS 'Tenant-level defaults for risk thresholds and business hours';
COMMENT ON TABLE agent_action_requests IS 'Tracks human-in-the-loop approval requests from AI agents';
COMMENT ON TABLE agent_action_decisions IS 'Individual approve/deny votes for multi-party approval';
COMMENT ON TABLE agent_action_audit_log IS 'Immutable audit trail of all agent actions and their outcomes';

COMMENT ON COLUMN agent_action_requests.agent_id IS 'Caller-defined agent identifier (any framework)';
COMMENT ON COLUMN agent_action_requests.agent_framework IS 'Agent framework: langchain, crewai, mcp, vercel-ai, custom';
COMMENT ON COLUMN agent_action_requests.risk_score IS 'Computed risk score 0-100 from risk engine';
COMMENT ON COLUMN agent_action_requests.risk_factors IS 'JSON array of {factor, score, reason} explaining the score';
COMMENT ON COLUMN agent_action_requests.device_token_id IS 'Links to tenant_device_tokens for push notification';
COMMENT ON COLUMN agent_action_requests.approval_type IS 'auto (low risk), single (one human), multi (2+ humans)';

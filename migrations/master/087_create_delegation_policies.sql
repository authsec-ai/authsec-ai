-- Migration: 142_create_delegation_policies.sql
-- Description: Creates delegation_policies table for AI agent trust delegation governance

CREATE TABLE IF NOT EXISTS delegation_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    role_name TEXT NOT NULL,
    agent_type TEXT NOT NULL,
    allowed_permissions JSONB DEFAULT '[]'::jsonb,
    max_ttl_seconds INTEGER NOT NULL DEFAULT 3600,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_deleg_policy_tenant_role_agent UNIQUE (tenant_id, role_name, agent_type)
);

CREATE INDEX IF NOT EXISTS idx_deleg_policy_tenant_id ON delegation_policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_deleg_policy_lookup ON delegation_policies(tenant_id, role_name, agent_type, enabled);

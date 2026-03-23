-- Migration 013: Create delegation_policies and delegation_tokens tables.
-- These tables support AI agent delegation (SPIRE JWT-SVID) introduced in authsec monolith.

CREATE TABLE IF NOT EXISTS delegation_policies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    role_name           TEXT NOT NULL,
    agent_type          TEXT NOT NULL,
    allowed_permissions JSONB NOT NULL DEFAULT '[]',
    max_ttl_seconds     INT NOT NULL DEFAULT 3600,
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    client_id           UUID,
    created_by          UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_delegation_policy_tenant_role_agent UNIQUE (tenant_id, role_name, agent_type)
);

CREATE INDEX IF NOT EXISTS idx_delegation_policies_tenant_id ON delegation_policies (tenant_id);
CREATE INDEX IF NOT EXISTS idx_delegation_policies_enabled   ON delegation_policies (tenant_id, enabled);

CREATE TABLE IF NOT EXISTS delegation_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    client_id    UUID NOT NULL,
    policy_id    UUID,
    token        TEXT NOT NULL,
    spiffe_id    TEXT NOT NULL,
    permissions  JSONB NOT NULL DEFAULT '[]',
    audience     JSONB NOT NULL DEFAULT '[]',
    expires_at   TIMESTAMPTZ NOT NULL,
    delegated_by UUID NOT NULL,
    ttl_seconds  INT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_delegation_token_tenant_client UNIQUE (tenant_id, client_id)
);

CREATE INDEX IF NOT EXISTS idx_delegation_tokens_tenant_id ON delegation_tokens (tenant_id);
CREATE INDEX IF NOT EXISTS idx_delegation_tokens_client_id ON delegation_tokens (tenant_id, client_id);
CREATE INDEX IF NOT EXISTS idx_delegation_tokens_status    ON delegation_tokens (tenant_id, status);

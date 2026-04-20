-- Migration: 145_create_delegation_tokens.sql
-- Description: Creates delegation_tokens table for SDK/AI agents to pull their
--              delegated JWT-SVID tokens and permissions. Upserted by DelegateToken,
--              read by SDK via GET /uflow/sdk/delegation-token.

CREATE TABLE IF NOT EXISTS delegation_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    client_id       UUID NOT NULL,
    policy_id       UUID REFERENCES delegation_policies(id) ON DELETE SET NULL,
    token           TEXT NOT NULL,
    spiffe_id       TEXT NOT NULL,
    permissions     JSONB NOT NULL DEFAULT '[]'::jsonb,
    audience        JSONB NOT NULL DEFAULT '[]'::jsonb,
    expires_at      TIMESTAMPTZ NOT NULL,
    delegated_by    UUID NOT NULL,
    ttl_seconds     INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_delegation_token_client UNIQUE (tenant_id, client_id),
    CONSTRAINT chk_deleg_token_status CHECK (status IN ('active', 'expired', 'revoked'))
);

CREATE INDEX IF NOT EXISTS idx_deleg_token_lookup ON delegation_tokens(tenant_id, client_id, status);
CREATE INDEX IF NOT EXISTS idx_deleg_token_expires ON delegation_tokens(expires_at) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_deleg_token_policy ON delegation_tokens(policy_id);

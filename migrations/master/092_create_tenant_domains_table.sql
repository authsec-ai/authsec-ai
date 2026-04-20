-- Migration: Create tenant_domains table for custom domain support
-- This table stores all verified/unverified domains for each tenant
-- Used for Host → tenant resolution and redirect URI ownership validation

-- Create tenant_domains table
CREATE TABLE IF NOT EXISTS tenant_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain VARCHAR(255) NOT NULL,
    kind VARCHAR(32) NOT NULL DEFAULT 'custom', -- 'platform_subdomain' or 'custom'
    is_primary BOOLEAN NOT NULL DEFAULT false,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    verification_method VARCHAR(32) NOT NULL DEFAULT 'dns_txt',
    verification_token VARCHAR(255) NOT NULL,
    verification_txt_name VARCHAR(255), -- e.g., _authsec-challenge.domain.com
    verification_txt_value VARCHAR(255), -- e.g., authsec-domain-verification=<token>
    verified_at TIMESTAMP WITH TIME ZONE,
    last_checked_at TIMESTAMP WITH TIME ZONE,
    failure_reason TEXT,
    ingress_created BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    CONSTRAINT fk_tenant_domains_tenant_id FOREIGN KEY (tenant_id)
        REFERENCES tenants(tenant_id) ON DELETE CASCADE
);

-- Global domain uniqueness: no cross-tenant claims
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_domains_domain_unique
    ON tenant_domains(domain);

-- One primary domain per tenant (partial unique index)
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_domains_primary_per_tenant
    ON tenant_domains(tenant_id)
    WHERE is_primary = true;

-- Performance indexes for common queries
CREATE INDEX IF NOT EXISTS idx_tenant_domains_tenant_id_verified
    ON tenant_domains(tenant_id, is_verified);

CREATE INDEX IF NOT EXISTS idx_tenant_domains_domain_verified
    ON tenant_domains(domain, is_verified);

CREATE INDEX IF NOT EXISTS idx_tenant_domains_tenant_id_primary
    ON tenant_domains(tenant_id, is_primary);

CREATE INDEX IF NOT EXISTS idx_tenant_domains_status
    ON tenant_domains(is_verified, kind);

-- Backfill existing tenant_domains from tenants.tenant_domain
-- Mark system-owned platform subdomains as verified
INSERT INTO tenant_domains (
    tenant_id,
    domain,
    kind,
    is_primary,
    is_verified,
    verification_method,
    verification_token,
    verification_txt_name,
    verification_txt_value,
    verified_at,
    created_at,
    updated_at
)
SELECT
    tenant_id,
    LOWER(tenant_domain),
    CASE
        WHEN tenant_domain LIKE '%.authsec.dev' OR tenant_domain LIKE '%.app.authsec.dev'
        THEN 'platform_subdomain'
        ELSE 'custom'
    END,
    true, -- mark as primary since it's the only domain they had
    CASE
        WHEN tenant_domain LIKE '%.authsec.dev' OR tenant_domain LIKE '%.app.authsec.dev'
        THEN true
        ELSE false
    END,
    'dns_txt',
    '', -- empty token for backfilled domains
    NULL,
    NULL,
    CASE
        WHEN tenant_domain LIKE '%.authsec.dev' OR tenant_domain LIKE '%.app.authsec.dev'
        THEN NOW()
        ELSE NULL
    END,
    NOW(),
    NOW()
FROM tenants
WHERE tenant_domain IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM tenant_domains
    WHERE tenant_domains.tenant_id = tenants.tenant_id
  )
ON CONFLICT (domain) DO NOTHING; -- Skip if domain already exists (in case of re-runs)

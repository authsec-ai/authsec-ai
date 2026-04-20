-- SAML master DB tables (from hydra-service)
-- saml_sp_certificates: SP X.509 certificates (one per tenant)
-- saml_requests: Active SAML auth requests (10-min expiry)
-- saml_callback_states: Temporary callback state (10-min expiry)

CREATE TABLE IF NOT EXISTS saml_sp_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE,
    certificate TEXT NOT NULL,
    private_key TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_saml_sp_cert_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_saml_sp_certificates_tenant_id ON saml_sp_certificates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_saml_sp_certificates_expires_at ON saml_sp_certificates(expires_at);

CREATE TABLE IF NOT EXISTS saml_requests (
    id VARCHAR(255) PRIMARY KEY,
    login_challenge TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    client_id UUID NOT NULL,
    provider_name VARCHAR(255) NOT NULL,
    relay_state TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- Ensure client_id column exists for tables that may have been created with an older schema
ALTER TABLE saml_requests ADD COLUMN IF NOT EXISTS client_id UUID;

CREATE INDEX IF NOT EXISTS idx_saml_requests_login_challenge ON saml_requests(login_challenge);
CREATE INDEX IF NOT EXISTS idx_saml_requests_tenant_id ON saml_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_saml_requests_client_id ON saml_requests(client_id);
CREATE INDEX IF NOT EXISTS idx_saml_requests_tenant_client ON saml_requests(tenant_id, client_id);
CREATE INDEX IF NOT EXISTS idx_saml_requests_expires_at ON saml_requests(expires_at);

CREATE TABLE IF NOT EXISTS saml_callback_states (
    id TEXT PRIMARY KEY,
    redirect_to TEXT NOT NULL,
    user_email VARCHAR(255),
    user_name VARCHAR(255),
    provider_name VARCHAR(255),
    tenant_id UUID,
    client_id UUID,
    login_challenge TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- Ensure client_id column exists if table pre-existed with older schema
ALTER TABLE saml_callback_states ADD COLUMN IF NOT EXISTS client_id UUID;

CREATE INDEX IF NOT EXISTS idx_saml_callback_states_client_id ON saml_callback_states(client_id);
CREATE INDEX IF NOT EXISTS idx_saml_callback_states_tenant_client ON saml_callback_states(tenant_id, client_id);

CREATE INDEX IF NOT EXISTS idx_saml_callback_states_login_challenge ON saml_callback_states(login_challenge);
CREATE INDEX IF NOT EXISTS idx_saml_callback_states_tenant_id ON saml_callback_states(tenant_id);
CREATE INDEX IF NOT EXISTS idx_saml_callback_states_expires_at ON saml_callback_states(expires_at);

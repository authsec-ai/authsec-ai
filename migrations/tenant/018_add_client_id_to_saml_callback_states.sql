-- Add client_id to saml_callback_states for multi-client isolation
-- Also normalise tenant_id from TEXT to UUID if not already converted

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'saml_callback_states' AND column_name = 'client_id'
    ) THEN
        ALTER TABLE saml_callback_states ADD COLUMN client_id UUID;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'saml_callback_states'
          AND column_name = 'tenant_id'
          AND data_type = 'text'
    ) THEN
        ALTER TABLE saml_callback_states ALTER COLUMN tenant_id TYPE UUID USING tenant_id::UUID;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_saml_callback_states_client_id ON saml_callback_states(client_id);
CREATE INDEX IF NOT EXISTS idx_saml_callback_states_tenant_client ON saml_callback_states(tenant_id, client_id);
CREATE INDEX IF NOT EXISTS idx_saml_callback_states_login_challenge ON saml_callback_states(login_challenge);

DELETE FROM saml_callback_states WHERE expires_at < NOW();

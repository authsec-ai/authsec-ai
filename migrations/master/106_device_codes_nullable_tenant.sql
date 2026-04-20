-- Migration 106: Make tenant_id nullable in device_codes
-- Reason: CLI tool (authsec-shield) sends no client_id/tenant_domain at code-request time.
-- Tenant context is resolved from the user's browser session during the /authorize step.

-- Drop FK constraint so tenant_id can be NULL at creation
ALTER TABLE device_codes
    DROP CONSTRAINT IF EXISTS device_codes_tenant_id_fkey;

-- Allow NULL for tenant_id
ALTER TABLE device_codes
    ALTER COLUMN tenant_id DROP NOT NULL;

-- Add tenant_domain column (cached at authorize time, returned in token response)
ALTER TABLE device_codes
    ADD COLUMN IF NOT EXISTS tenant_domain TEXT;

-- Add access_token column: token is generated at /authorize time and stored here.
-- /token poll returns this value — avoids re-generating tokens on every poll.
ALTER TABLE device_codes
    ADD COLUMN IF NOT EXISTS access_token TEXT;

COMMENT ON COLUMN device_codes.tenant_id IS 'Resolved from user browser session during /authorize; NULL until then';
COMMENT ON COLUMN device_codes.tenant_domain IS 'Cached tenant domain (e.g. mycompany.authsec.ai), set at authorize time';
COMMENT ON COLUMN device_codes.access_token IS 'JWT generated at /authorize time; returned by /token once status = authorized';

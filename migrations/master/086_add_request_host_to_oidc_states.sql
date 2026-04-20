-- Migration: Add request_host column to oidc_states table
-- This column stores the actual domain where the OIDC flow was initiated
-- Used to redirect users back to the correct custom domain after OAuth callback

-- Add request_host column
ALTER TABLE oidc_states ADD COLUMN IF NOT EXISTS request_host VARCHAR(255);

COMMENT ON COLUMN oidc_states.request_host IS 'Full domain where OIDC was initiated (e.g., auth.company.com) for callback redirect';

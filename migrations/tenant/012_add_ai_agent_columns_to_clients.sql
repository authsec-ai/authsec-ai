-- Migration 014: Add AI agent delegation columns to clients table.
-- Existing tenant databases were created before these columns were added to the template.

ALTER TABLE clients ADD COLUMN IF NOT EXISTS client_type VARCHAR(255);
ALTER TABLE clients ADD COLUMN IF NOT EXISTS agent_type TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS spiffe_id TEXT;

-- Migration: 143_add_client_id_to_delegation_policies.sql
-- Description: Adds client_id to delegation_policies so autonomous workloads are tied to a registered client

ALTER TABLE delegation_policies ADD COLUMN IF NOT EXISTS client_id UUID;
CREATE INDEX IF NOT EXISTS idx_deleg_policy_client_id ON delegation_policies(client_id);

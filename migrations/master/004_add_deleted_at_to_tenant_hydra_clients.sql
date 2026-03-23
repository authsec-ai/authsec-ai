-- Add soft-delete support to tenant_hydra_clients.
-- The table was originally created without deleted_at; the Go model requires it
-- for GORM soft-delete (gorm.DeletedAt).

ALTER TABLE tenant_hydra_clients
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_tenant_hydra_clients_deleted_at
    ON tenant_hydra_clients(deleted_at);

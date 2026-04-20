ALTER TABLE tenant_hydra_clients ADD COLUMN IF NOT EXISTS deleted_at timestamp with time zone;

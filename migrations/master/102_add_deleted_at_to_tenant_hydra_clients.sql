DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tenant_hydra_clients' AND column_name = 'deleted_at'
    ) THEN
        ALTER TABLE tenant_hydra_clients ADD COLUMN deleted_at timestamp with time zone;
    END IF;
END $$;

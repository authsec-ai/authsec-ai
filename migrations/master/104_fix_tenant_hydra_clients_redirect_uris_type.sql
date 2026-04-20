-- Convert tenant_hydra_clients.redirect_uris from jsonb to text[]
-- pq.StringArray in Go serializes as PostgreSQL array {val1,val2}, not JSON ["val1","val2"]

-- PostgreSQL does not allow subqueries in ALTER COLUMN...TYPE...USING.
-- Use a rename-based approach: add new column, populate via UPDATE, swap.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tenant_hydra_clients'
          AND column_name = 'redirect_uris'
          AND data_type = 'jsonb'
    ) THEN
        -- Add temp column
        EXECUTE 'ALTER TABLE tenant_hydra_clients ADD COLUMN redirect_uris_new text[] DEFAULT ''{}''';
        -- Copy converted values
        EXECUTE 'UPDATE tenant_hydra_clients
                 SET redirect_uris_new = ARRAY(SELECT jsonb_array_elements_text(redirect_uris))
                 WHERE redirect_uris IS NOT NULL AND jsonb_typeof(redirect_uris) = ''array''';
        -- Drop old jsonb column
        EXECUTE 'ALTER TABLE tenant_hydra_clients DROP COLUMN redirect_uris';
        -- Rename new column into place
        EXECUTE 'ALTER TABLE tenant_hydra_clients RENAME COLUMN redirect_uris_new TO redirect_uris';
        RAISE NOTICE 'Converted tenant_hydra_clients.redirect_uris from jsonb to text[]';
    ELSIF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tenant_hydra_clients'
          AND column_name = 'redirect_uris'
    ) THEN
        -- Column missing entirely — add as text[]
        EXECUTE 'ALTER TABLE tenant_hydra_clients ADD COLUMN redirect_uris text[] DEFAULT ''{}''';
        RAISE NOTICE 'Added tenant_hydra_clients.redirect_uris as text[]';
    ELSE
        RAISE NOTICE 'tenant_hydra_clients.redirect_uris already text[], skipping conversion';
    END IF;
END $$;

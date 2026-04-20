-- BEGIN; (removed - app manages transactions)

-- Ensure the table exists with the latest structure.  If an older version exists,
-- we recreate it using a lock + rename strategy to preserve existing data.
DO $$
DECLARE
    table_exists boolean;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'tenant_hydra_clients'
    ) INTO table_exists;

    IF NOT table_exists THEN
        CREATE TABLE tenant_hydra_clients (
            id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
            org_id               text        NOT NULL,
            tenant_id            text        NOT NULL,
            tenant_name          text        NOT NULL,
            hydra_client_id      text        NOT NULL UNIQUE,
            hydra_client_secret  text        NOT NULL,
            client_name          text        NOT NULL,
            redirect_uris        jsonb       NOT NULL DEFAULT '[]'::jsonb,
            scopes               text[]      NOT NULL DEFAULT ARRAY['openid','profile','email']::text[],
            client_type          text        NOT NULL,
            provider_name        text,
            is_active            boolean     NOT NULL DEFAULT true,
            created_at           timestamptz NOT NULL DEFAULT NOW(),
            updated_at           timestamptz NOT NULL DEFAULT NOW(),
            created_by           text        NOT NULL DEFAULT 'system',
            updated_by           text        NOT NULL DEFAULT 'system'
        );
    ELSE
        -- Align existing table definition with the desired schema
        -- Drop legacy foreign keys that rely on UUID tenant/client IDs
        ALTER TABLE tenant_hydra_clients
            DROP CONSTRAINT IF EXISTS tenant_hydra_clients_tenant_id_fkey,
            DROP CONSTRAINT IF EXISTS tenant_hydra_clients_client_id_fkey;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'org_id';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN org_id text;
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'tenant_id' AND data_type = 'text';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ALTER COLUMN tenant_id TYPE text USING tenant_id::text;
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'tenant_name';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN tenant_name text;
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'hydra_client_secret';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN hydra_client_secret text;
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'client_name';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN client_name text;
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'redirect_uris';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN redirect_uris jsonb DEFAULT '[]'::jsonb;
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'scopes';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN scopes text[] DEFAULT ARRAY['openid','profile','email']::text[];
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'client_type';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN client_type text;
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'provider_name';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN provider_name text;
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'is_active';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN is_active boolean DEFAULT true;
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'created_by';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN created_by text DEFAULT 'system';
        END IF;

        PERFORM 1 FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
           AND column_name = 'updated_by';
        IF NOT FOUND THEN
            ALTER TABLE tenant_hydra_clients ADD COLUMN updated_by text DEFAULT 'system';
        END IF;

        -- Drop legacy columns no longer used
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = 'public' AND table_name = 'tenant_hydra_clients'
              AND column_name IN ('client_id', 'client_id_uuid', 'vault_secret_path', 'provision_status',
                                  'registered_at', 'last_synced_at', 'last_error')
        ) THEN
            ALTER TABLE tenant_hydra_clients
                DROP COLUMN IF EXISTS client_id,
                DROP COLUMN IF EXISTS client_id_uuid,
                DROP COLUMN IF EXISTS vault_secret_path,
                DROP COLUMN IF EXISTS provision_status,
                DROP COLUMN IF EXISTS registered_at,
                DROP COLUMN IF EXISTS last_synced_at,
                DROP COLUMN IF EXISTS last_error;
        END IF;

        -- Ensure NOT NULL / defaults align with new spec
        ALTER TABLE tenant_hydra_clients
            ALTER COLUMN org_id SET NOT NULL,
            ALTER COLUMN tenant_id SET NOT NULL,
            ALTER COLUMN tenant_name SET NOT NULL,
            ALTER COLUMN hydra_client_id SET NOT NULL,
            ALTER COLUMN hydra_client_secret SET NOT NULL,
            ALTER COLUMN client_name SET NOT NULL,
            ALTER COLUMN redirect_uris SET DEFAULT '[]'::jsonb,
            ALTER COLUMN redirect_uris SET NOT NULL,
            ALTER COLUMN scopes SET DEFAULT ARRAY['openid','profile','email']::text[],
            ALTER COLUMN scopes SET NOT NULL,
            ALTER COLUMN client_type SET NOT NULL,
            ALTER COLUMN is_active SET DEFAULT true,
            ALTER COLUMN is_active SET NOT NULL,
            ALTER COLUMN created_at SET DEFAULT NOW(),
            ALTER COLUMN created_at SET NOT NULL,
            ALTER COLUMN updated_at SET DEFAULT NOW(),
            ALTER COLUMN updated_at SET NOT NULL,
            ALTER COLUMN created_by SET DEFAULT 'system',
            ALTER COLUMN updated_by SET DEFAULT 'system';
    END IF;
END $$;

COMMENT ON TABLE tenant_hydra_clients IS 'Tracks Hydra client provisioning for each tenant';
COMMENT ON COLUMN tenant_hydra_clients.client_type IS 'main or oidc_provider';
COMMENT ON COLUMN tenant_hydra_clients.scopes IS 'Default Hydra scopes granted to the client';

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_hydra_clients_hydra_client_id
    ON tenant_hydra_clients (hydra_client_id);

CREATE INDEX IF NOT EXISTS idx_tenant_hydra_clients_org_tenant
    ON tenant_hydra_clients (org_id, tenant_id);

CREATE INDEX IF NOT EXISTS idx_tenant_hydra_clients_client_type
    ON tenant_hydra_clients (client_type);

DROP TRIGGER IF EXISTS update_tenant_hydra_clients_updated_at ON tenant_hydra_clients;

CREATE TRIGGER update_tenant_hydra_clients_updated_at
    BEFORE UPDATE ON tenant_hydra_clients
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- COMMIT; (removed - app manages transactions)

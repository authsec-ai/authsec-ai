
CREATE TABLE IF NOT EXISTS credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    credential_id BYTEA UNIQUE NOT NULL,
    public_key BYTEA NOT NULL,
    attestation_type VARCHAR(255),
    aaguid UUID,
    sign_count BIGINT DEFAULT 0,
    transports TEXT[],
    backup_eligible BOOLEAN DEFAULT false,
    backup_state BOOLEAN DEFAULT false,
    rp_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS otp_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    otp VARCHAR(10) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create pending_registrations table
CREATE TABLE IF NOT EXISTS pending_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    tenant_id UUID NOT NULL,
    project_id UUID NOT NULL,
    client_id UUID NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    tenant_domain VARCHAR(255) NOT NULL
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_otp_entries_email ON otp_entries(email);
CREATE INDEX IF NOT EXISTS idx_otp_entries_expires_at ON otp_entries(expires_at);
CREATE INDEX IF NOT EXISTS idx_otp_entries_verified ON otp_entries(verified);

CREATE INDEX IF NOT EXISTS idx_pending_registrations_email ON pending_registrations(email);
CREATE INDEX IF NOT EXISTS idx_pending_registrations_expires_at ON pending_registrations(expires_at);
CREATE INDEX IF NOT EXISTS idx_pending_registrations_tenant_id ON pending_registrations(tenant_id);

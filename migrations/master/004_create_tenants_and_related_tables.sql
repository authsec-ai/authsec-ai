-- Migration: Create tenants table and related tables
-- This migration creates the tenants table and tenant_mappings table

-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID UNIQUE,
    tenant_db VARCHAR(255),
    email VARCHAR(255) UNIQUE,
    username VARCHAR(255),
    password_hash TEXT,
    provider VARCHAR(50) DEFAULT 'local',
    provider_id VARCHAR(255),
    avatar TEXT,
    name VARCHAR(255),
    source VARCHAR(50),
    status VARCHAR(50) DEFAULT 'active',
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    tenant_domain VARCHAR(255) DEFAULT 'app.authsec.dev'
);

-- Create tenant_mappings table  
CREATE TABLE IF NOT EXISTS tenant_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    client_id VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create projects table (if not exists)
CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    user_id UUID,
    tenant_id UUID,
    client_id UUID,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add client_id column to projects table if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'projects' AND column_name = 'client_id') THEN
        ALTER TABLE projects ADD COLUMN client_id UUID;
    END IF;
END $$;

-- Create groups table (if not exists)
CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_tenants_tenant_id ON tenants(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenants_email ON tenants(email);
CREATE INDEX IF NOT EXISTS idx_tenants_tenant_domain ON tenants(tenant_domain);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

CREATE INDEX IF NOT EXISTS idx_tenant_mappings_tenant_id ON tenant_mappings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_mappings_client_id ON tenant_mappings(client_id);

CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id);
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'projects' AND column_name = 'client_id') THEN
        CREATE INDEX IF NOT EXISTS idx_projects_client_id ON projects(client_id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name);
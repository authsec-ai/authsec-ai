-- Migration to add tenants and projects tables if they don't exist
-- This migration ensures the basic table structure is in place

-- Create tenants table (if not exists)

-- Create projects table (if not exists)
CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
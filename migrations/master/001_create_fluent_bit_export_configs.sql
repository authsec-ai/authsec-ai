-- Master Migration 001: Create fluent_bit_export_configs table for log-aggregator service

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS fluent_bit_export_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    host TEXT NOT NULL,
    alias TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fb_export_configs_tenant_id
    ON fluent_bit_export_configs (tenant_id);

-- =====================================================
-- Migration 000: Comprehensive Base Schema (Master DB)
-- =====================================================
-- This migration creates ALL base tables required by the master database.
-- Designed to be idempotent (IF NOT EXISTS) and match production exactly.
-- Replaces fragmented initial migrations with a single comprehensive schema.
-- Does NOT include: M2M, voice auth, PKI, TOTP, CIBA, device auth tables.
-- Does NOT include: seed data (belongs in DML migrations).
-- =====================================================

-- Ensure required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================================
-- FUNCTIONS
-- =====================================================

-- Generic updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

-- SAML providers updated_at trigger function
CREATE OR REPLACE FUNCTION update_saml_providers_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

-- Generic set_updated_at trigger function
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- =====================================================
-- CORE TABLES: TENANTS
-- =====================================================

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    tenant_db TEXT,
    email TEXT NOT NULL,
    username TEXT,
    password_hash TEXT,
    provider TEXT DEFAULT 'local',
    provider_id TEXT,
    avatar TEXT,
    name TEXT,
    source TEXT,
    status TEXT,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    tenant_domain TEXT NOT NULL,
    vault_mount VARCHAR(255),
    ca_cert TEXT
);

-- =====================================================
-- CORE TABLES: USERS
-- =====================================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    project_id UUID,
    name TEXT,
    username TEXT,
    email TEXT NOT NULL,
    password_hash TEXT,
    tenant_domain TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_id TEXT,
    provider_data JSONB DEFAULT '{}'::jsonb,
    avatar_url TEXT,
    active BOOLEAN DEFAULT true,
    mfa_enabled BOOLEAN DEFAULT false NOT NULL,
    mfa_method TEXT[],
    mfa_default_method TEXT,
    mfa_enrolled_at TIMESTAMP WITH TIME ZONE,
    mfa_verified BOOLEAN DEFAULT false NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    last_login TIMESTAMP WITH TIME ZONE,
    external_id TEXT,
    sync_source TEXT,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    is_synced_user BOOLEAN DEFAULT false,
    deleted_at TIMESTAMP WITH TIME ZONE,
    role_name VARCHAR(255),
    temporary_password BOOLEAN DEFAULT false,
    password_change_required BOOLEAN DEFAULT false,
    invited_by UUID,
    invited_at TIMESTAMP WITH TIME ZONE,
    temporary_password_expires_at TIMESTAMP WITH TIME ZONE,
    is_primary_admin BOOLEAN DEFAULT false,
    is_voice_enrolled BOOLEAN DEFAULT false,
    voice_enrolled BOOLEAN DEFAULT false,
    voice_enrollment_date TIMESTAMP WITHOUT TIME ZONE,
    voice_last_verified TIMESTAMP WITHOUT TIME ZONE,
    failed_login_attempts INTEGER DEFAULT 0,
    account_locked_at TIMESTAMP WITH TIME ZONE,
    password_reset_required BOOLEAN DEFAULT false
);

-- =====================================================
-- CORE TABLES: PROJECTS
-- =====================================================

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    user_id UUID,
    tenant_id UUID,
    client_id UUID,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- =====================================================
-- CORE TABLES: CLIENTS
-- =====================================================

CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    project_id UUID NOT NULL,
    owner_id UUID NOT NULL,
    org_id UUID NOT NULL,
    name TEXT NOT NULL,
    email TEXT,
    status TEXT DEFAULT 'Active',
    tags TEXT[],
    active BOOLEAN DEFAULT true,
    last_login TIMESTAMP WITH TIME ZONE,
    mfa_enabled BOOLEAN DEFAULT false NOT NULL,
    mfa_method TEXT[],
    mfa_default_method TEXT,
    mfa_enrolled_at TIMESTAMP WITH TIME ZONE,
    mfa_verified BOOLEAN DEFAULT false,
    hydra_client_id TEXT,
    oidc_enabled BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    description TEXT
);

-- =====================================================
-- RBAC TABLES
-- =====================================================

-- Roles
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Permissions (new schema: tenant_id, resource, action text columns)
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    description TEXT,
    full_permission_string TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Resources
CREATE TABLE IF NOT EXISTS resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type VARCHAR(255) DEFAULT 'generic',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(tenant_id, name)
);

-- Groups
CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);

-- Scopes (internal RBAC scopes, separate from api_scopes)
CREATE TABLE IF NOT EXISTS scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tenant_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- RBAC JUNCTION TABLES
-- =====================================================

-- Role Permissions (M:N)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);

-- Scope Permissions (M:N)
CREATE TABLE IF NOT EXISTS scope_permissions (
    scope_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (scope_id, permission_id)
);

-- Group Roles (M:N)
CREATE TABLE IF NOT EXISTS group_roles (
    group_id UUID NOT NULL,
    role_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    tenant_id UUID,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (group_id, role_id)
);

-- Group Scopes
CREATE TABLE IF NOT EXISTS group_scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID,
    scope_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(group_id, scope_id)
);

-- Role Scopes
CREATE TABLE IF NOT EXISTS role_scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID,
    scope_name TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User Roles (M:N)
CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID DEFAULT gen_random_uuid() NOT NULL,
    role_id UUID DEFAULT gen_random_uuid() NOT NULL,
    tenant_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- User Groups (M:N)
CREATE TABLE IF NOT EXISTS user_groups (
    user_id UUID DEFAULT gen_random_uuid() NOT NULL,
    group_id UUID NOT NULL,
    tenant_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, group_id)
);

-- User Scopes
CREATE TABLE IF NOT EXISTS user_scopes (
    user_id UUID DEFAULT gen_random_uuid() NOT NULL,
    scope_name TEXT,
    tenant_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User Resources (M:N)
CREATE TABLE IF NOT EXISTS user_resources (
    user_id UUID DEFAULT gen_random_uuid() NOT NULL,
    resource_id UUID DEFAULT gen_random_uuid() NOT NULL,
    PRIMARY KEY (user_id, resource_id)
);

-- Client Roles (M:N)
CREATE TABLE IF NOT EXISTS client_roles (
    client_id UUID DEFAULT gen_random_uuid() NOT NULL,
    role_id UUID DEFAULT gen_random_uuid() NOT NULL,
    tenant_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (client_id, role_id)
);

-- Client Groups (M:N)
CREATE TABLE IF NOT EXISTS client_groups (
    client_id UUID NOT NULL,
    group_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (client_id, group_id)
);

-- Client Scopes (M:N)
CREATE TABLE IF NOT EXISTS client_scopes (
    client_id UUID NOT NULL,
    scope_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (client_id, scope_id)
);

-- Client Resources (M:N)
CREATE TABLE IF NOT EXISTS client_resources (
    client_id UUID NOT NULL,
    resource_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (client_id, resource_id)
);

-- =====================================================
-- ROLE BINDINGS & SERVICE ACCOUNTS
-- =====================================================

-- Role Bindings
CREATE TABLE IF NOT EXISTS role_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    service_account_id UUID,
    role_id UUID NOT NULL,
    scope_type TEXT DEFAULT '*',
    scope_id UUID,
    conditions JSONB DEFAULT '{}'::jsonb,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    role_name TEXT,
    username TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT check_principal CHECK (
        ((user_id IS NOT NULL) AND (service_account_id IS NULL))
        OR ((user_id IS NULL) AND (service_account_id IS NOT NULL))
    )
);

-- Service Accounts
CREATE TABLE IF NOT EXISTS service_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- API SCOPES & SCOPE MAPPINGS
-- =====================================================

-- API Scopes (OAuth external scopes)
CREATE TABLE IF NOT EXISTS api_scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API Scope Permissions (M:N)
CREATE TABLE IF NOT EXISTS api_scope_permissions (
    scope_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (scope_id, permission_id)
);

-- Scope Resource Mappings
CREATE TABLE IF NOT EXISTS scope_resource_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    scope_name TEXT NOT NULL DEFAULT '*',
    resource_name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- AUTHENTICATION & SECURITY TABLES
-- =====================================================

-- Credentials (WebAuthn - production schema)
CREATE TABLE IF NOT EXISTS credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    credential_id BYTEA NOT NULL,
    public_key BYTEA NOT NULL,
    attestation_type TEXT NOT NULL,
    aa_guid UUID,
    sign_count BIGINT DEFAULT 0 NOT NULL,
    backup_eligible BOOLEAN DEFAULT false,
    backup_state BOOLEAN DEFAULT false,
    transports TEXT[],
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    aaguid UUID
);

-- MFA Methods (production schema)
CREATE TABLE IF NOT EXISTS mfa_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    user_id UUID,
    method_type VARCHAR(50) NOT NULL,
    display_name VARCHAR(255),
    description VARCHAR(255),
    recommended BOOLEAN DEFAULT false,
    method_data JSONB,
    enabled BOOLEAN DEFAULT false,
    is_primary BOOLEAN DEFAULT false,
    verified BOOLEAN DEFAULT false,
    backup_codes TEXT,
    enrolled_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    method_subtype VARCHAR(255)
);

-- OTP Entries (production schema)
CREATE TABLE IF NOT EXISTS otp_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT,
    otp TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    verified BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);

-- Pending Registrations (production schema)
CREATE TABLE IF NOT EXISTS pending_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT,
    password_hash TEXT,
    first_name TEXT DEFAULT '',
    last_name TEXT DEFAULT '',
    tenant_id UUID,
    project_id UUID,
    client_id UUID,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    tenant_domain TEXT
);

-- WebAuthn Sessions (production schema)
CREATE TABLE IF NOT EXISTS webauthn_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_key VARCHAR(255) NOT NULL,
    challenge TEXT NOT NULL,
    user_id BYTEA NOT NULL,
    user_verification VARCHAR(50),
    extensions BYTEA,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    cred_params BYTEA,
    allowed_credential_ids BYTEA
);

-- =====================================================
-- OAUTH & OIDC TABLES
-- =====================================================

-- OAuth Sessions (production schema - PK is session_id TEXT)
CREATE TABLE IF NOT EXISTS oauth_sessions (
    session_id VARCHAR(36) PRIMARY KEY NOT NULL,
    user_email VARCHAR(255),
    user_info JSONB,
    access_token TEXT,
    refresh_token TEXT,
    authorization_code TEXT,
    token_expires_at BIGINT,
    created_at BIGINT NOT NULL,
    last_activity BIGINT NOT NULL,
    oauth_state VARCHAR(255),
    pkce_verifier TEXT,
    pkce_challenge TEXT,
    is_active BOOLEAN DEFAULT true,
    client_identifier VARCHAR(255),
    org_id VARCHAR(255),
    tenant_id VARCHAR(255),
    user_id VARCHAR(255),
    provider VARCHAR(100),
    provider_id VARCHAR(255),
    accessible_tools JSONB
);

-- OIDC Providers (production schema)
CREATE TABLE IF NOT EXISTS oidc_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_name VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    client_secret_vault_path VARCHAR(255) NOT NULL,
    authorization_url VARCHAR(500) NOT NULL,
    token_url VARCHAR(500) NOT NULL,
    userinfo_url VARCHAR(500) NOT NULL,
    scopes TEXT DEFAULT 'openid email profile',
    icon_url VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- OIDC States (production schema)
CREATE TABLE IF NOT EXISTS oidc_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state_token VARCHAR(255) NOT NULL,
    tenant_id UUID,
    tenant_domain VARCHAR(255) NOT NULL,
    provider_name VARCHAR(50) NOT NULL,
    action VARCHAR(20) NOT NULL,
    code_verifier VARCHAR(255),
    redirect_after VARCHAR(500),
    expires_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- OIDC User Identities (production schema)
CREATE TABLE IF NOT EXISTS oidc_user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    provider_name VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    profile_data JSONB,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE
);

-- =====================================================
-- SAML TABLES
-- =====================================================

-- SAML Providers (production schema)
CREATE TABLE IF NOT EXISTS saml_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    client_id UUID NOT NULL,
    provider_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    entity_id VARCHAR(500) NOT NULL,
    sso_url VARCHAR(500) NOT NULL,
    slo_url VARCHAR(500),
    certificate TEXT NOT NULL,
    metadata_url VARCHAR(500),
    name_id_format VARCHAR(255) DEFAULT 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',
    attribute_mapping JSONB,
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- SAML Callback States
CREATE TABLE IF NOT EXISTS saml_callback_states (
    id TEXT PRIMARY KEY NOT NULL,
    redirect_to TEXT NOT NULL,
    user_email VARCHAR(255),
    user_name VARCHAR(255),
    provider_name VARCHAR(255),
    tenant_id TEXT,
    login_challenge TEXT,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

-- =====================================================
-- SERVICES TABLE
-- =====================================================

CREATE TABLE IF NOT EXISTS services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    type TEXT,
    url TEXT,
    description TEXT,
    tags TEXT[],
    resource_id UUID NOT NULL,
    auth_type TEXT NOT NULL,
    auth_config TEXT,
    vault_path TEXT,
    created_by TEXT NOT NULL,
    agent_accessible BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);

-- =====================================================
-- RESOURCE METHODS
-- =====================================================

CREATE TABLE IF NOT EXISTS resource_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL,
    method VARCHAR(10) NOT NULL,
    path_pattern VARCHAR(255) NOT NULL,
    requires_admin BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(resource_id, method, path_pattern)
);

-- =====================================================
-- AUDIT & LOGGING TABLES
-- =====================================================

-- Audit Events (sequential ID)
CREATE TABLE IF NOT EXISTS audit_events (
    id BIGSERIAL PRIMARY KEY,
    request_id TEXT,
    tenant_id TEXT,
    user_id TEXT,
    action TEXT,
    resource TEXT,
    resource_id TEXT,
    method TEXT,
    path TEXT,
    user_agent TEXT,
    client_ip TEXT,
    status_code BIGINT,
    duration BIGINT,
    old_values JSONB,
    new_values JSONB,
    error TEXT,
    "timestamp" TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);

-- Audit Logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255),
    details JSONB,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    tenant_id UUID,
    event_type VARCHAR(50),
    workload_id VARCHAR(255),
    certificate_id VARCHAR(255),
    spiffe_id VARCHAR(500),
    success BOOLEAN,
    error_message TEXT,
    metadata JSONB,
    ip_address VARCHAR(100),
    user_agent TEXT
);

-- Grant Audit
CREATE TABLE IF NOT EXISTS grant_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    actor_user_id UUID,
    action TEXT,
    target_type TEXT,
    target_id UUID,
    before JSONB,
    after JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Role Assignment Requests (production schema)
CREATE TABLE IF NOT EXISTS role_assignment_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' NOT NULL,
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT role_assignment_requests_status_check CHECK (
        status IN ('pending', 'approved', 'rejected')
    )
);

-- =====================================================
-- DIRECTORY SYNC TABLES
-- =====================================================

-- Sync Configurations (production schema)
CREATE TABLE IF NOT EXISTS sync_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    client_id UUID NOT NULL,
    project_id UUID NOT NULL,
    sync_type VARCHAR(50) NOT NULL,
    config_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true NOT NULL,
    ad_server VARCHAR(500),
    ad_username VARCHAR(500),
    ad_password TEXT,
    ad_base_dn VARCHAR(500),
    ad_filter TEXT,
    ad_use_ssl BOOLEAN DEFAULT true,
    ad_skip_verify BOOLEAN DEFAULT false,
    entra_tenant_id VARCHAR(500),
    entra_client_id VARCHAR(500),
    entra_client_secret TEXT,
    entra_scopes TEXT,
    entra_skip_verify BOOLEAN DEFAULT false,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_sync_status VARCHAR(50),
    last_sync_error TEXT,
    last_sync_users_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID,
    CONSTRAINT sync_configurations_sync_type_check CHECK (
        sync_type IN ('active_directory', 'entra_id')
    )
);

-- =====================================================
-- TENANT & MIGRATION MANAGEMENT
-- =====================================================

-- Tenant Databases
CREATE TABLE IF NOT EXISTS tenant_databases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE,
    database_name VARCHAR(255) NOT NULL UNIQUE,
    migration_status VARCHAR(50) DEFAULT 'pending',
    last_migration INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tenant Mappings (production schema)
CREATE TABLE IF NOT EXISTS tenant_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    client_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);

-- Migration Logs
CREATE TABLE IF NOT EXISTS migration_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    success BOOLEAN NOT NULL,
    error_msg TEXT,
    db_type VARCHAR(20) NOT NULL,
    tenant_id UUID,
    execution_ms INTEGER,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Schema Migrations (production schema)
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- External Service Migrations
CREATE TABLE IF NOT EXISTS external_service_migrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    migration_name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, migration_name)
);

-- =====================================================
-- UNIQUE CONSTRAINTS
-- =====================================================

-- Users
DO $$ BEGIN
    ALTER TABLE users ADD CONSTRAINT users_email_tenant_unique UNIQUE (email, tenant_id);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE users ADD CONSTRAINT users_tenant_id_id_key UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- Tenants (no UNIQUE on email in production - relaxed by migration 112)
-- tenant_id is indexed but not explicitly UNIQUE-constrained in production

-- Roles
DO $$ BEGIN
    ALTER TABLE roles ADD CONSTRAINT roles_tenant_id_id_key UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE roles ADD CONSTRAINT roles_tenant_id_name_key UNIQUE (tenant_id, name);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- Permissions
DO $$ BEGIN
    ALTER TABLE permissions ADD CONSTRAINT permissions_tenant_id_id_key UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE permissions ADD CONSTRAINT permissions_tenant_id_resource_action_key UNIQUE (tenant_id, resource, action);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- Credentials
DO $$ BEGIN
    ALTER TABLE credentials ADD CONSTRAINT credentials_credential_id_unique UNIQUE (credential_id);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- MFA Methods
DO $$ BEGIN
    ALTER TABLE mfa_methods ADD CONSTRAINT mfa_methods_client_id_method_type_key UNIQUE (client_id, method_type);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- Clients
DO $$ BEGIN
    ALTER TABLE clients ADD CONSTRAINT uni_clients_hydra_client_id UNIQUE (hydra_client_id);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- Groups
DO $$ BEGIN
    ALTER TABLE groups ADD CONSTRAINT uni_groups_tenant_name UNIQUE (tenant_id, name);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- Service Accounts
DO $$ BEGIN
    ALTER TABLE service_accounts ADD CONSTRAINT service_accounts_tenant_id_id_key UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- API Scopes
DO $$ BEGIN
    ALTER TABLE api_scopes ADD CONSTRAINT uq_api_scopes_tenant_id UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE api_scopes ADD CONSTRAINT uq_api_scopes_tenant_name UNIQUE (tenant_id, name);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- Scope Resource Mappings
DO $$ BEGIN
    ALTER TABLE scope_resource_mappings ADD CONSTRAINT scope_resource_mappings_tenant_scope_resource_key UNIQUE (tenant_id, scope_name, resource_name);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- Sync Configurations
DO $$ BEGIN
    ALTER TABLE sync_configurations ADD CONSTRAINT sync_configurations_tenant_id_config_name_key UNIQUE (tenant_id, config_name);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- User Roles
DO $$ BEGIN
    ALTER TABLE user_roles ADD CONSTRAINT user_roles_user_role_tenant_key UNIQUE (user_id, role_id, tenant_id);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- WebAuthn Sessions
DO $$ BEGIN
    ALTER TABLE webauthn_sessions ADD CONSTRAINT webauthn_sessions_session_key_key UNIQUE (session_key);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- OIDC Providers
DO $$ BEGIN
    ALTER TABLE oidc_providers ADD CONSTRAINT oidc_providers_provider_name_key UNIQUE (provider_name);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- OIDC States
DO $$ BEGIN
    ALTER TABLE oidc_states ADD CONSTRAINT oidc_states_state_token_key UNIQUE (state_token);
EXCEPTION WHEN duplicate_table OR duplicate_object THEN NULL;
END $$;

-- =====================================================
-- FOREIGN KEY CONSTRAINTS
-- =====================================================

-- Permissions -> Tenants
DO $$ BEGIN
    ALTER TABLE permissions ADD CONSTRAINT permissions_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Roles -> Tenants
DO $$ BEGIN
    ALTER TABLE roles ADD CONSTRAINT roles_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Role Permissions -> Roles
DO $$ BEGIN
    ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_role_id_fkey
        FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Role Permissions -> Permissions
DO $$ BEGIN
    ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_permission_id_fkey
        FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Scope Permissions -> Permissions
DO $$ BEGIN
    ALTER TABLE scope_permissions ADD CONSTRAINT scope_permissions_permission_id_fkey
        FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- API Scope Permissions -> API Scopes
DO $$ BEGIN
    ALTER TABLE api_scope_permissions ADD CONSTRAINT api_scope_permissions_scope_id_fkey
        FOREIGN KEY (scope_id) REFERENCES api_scopes(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- API Scope Permissions -> Permissions
DO $$ BEGIN
    ALTER TABLE api_scope_permissions ADD CONSTRAINT api_scope_permissions_permission_id_fkey
        FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Client Roles -> Clients
DO $$ BEGIN
    ALTER TABLE client_roles ADD CONSTRAINT fk_client_roles_client
        FOREIGN KEY (client_id) REFERENCES clients(id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- User Groups -> Groups
DO $$ BEGIN
    ALTER TABLE user_groups ADD CONSTRAINT fk_user_groups_group
        FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- User Groups -> Users
DO $$ BEGIN
    ALTER TABLE user_groups ADD CONSTRAINT fk_user_groups_user
        FOREIGN KEY (user_id) REFERENCES users(id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- User Resources -> Users
DO $$ BEGIN
    ALTER TABLE user_resources ADD CONSTRAINT fk_user_resources_user
        FOREIGN KEY (user_id) REFERENCES users(id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- User Scopes -> Users
DO $$ BEGIN
    ALTER TABLE user_scopes ADD CONSTRAINT fk_user_scopes_user
        FOREIGN KEY (user_id) REFERENCES users(id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Role Bindings -> Users
DO $$ BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_user_fk_simple
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Role Bindings -> Roles
DO $$ BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_role_fk_simple
        FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Role Bindings -> Tenants
DO $$ BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Role Bindings -> Created By (Users)
DO $$ BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Role Bindings -> Tenant+Role compound FK
DO $$ BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_tenant_role_fk
        FOREIGN KEY (tenant_id, role_id) REFERENCES roles(tenant_id, id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Role Bindings -> Tenant+User compound FK
DO $$ BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_tenant_user_fk
        FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Role Bindings -> Tenant+Service Account compound FK
DO $$ BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_tenant_id_service_account_id_fkey
        FOREIGN KEY (tenant_id, service_account_id) REFERENCES service_accounts(tenant_id, id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Service Accounts -> Tenants
DO $$ BEGIN
    ALTER TABLE service_accounts ADD CONSTRAINT service_accounts_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Projects -> Tenants
DO $$ BEGIN
    ALTER TABLE projects ADD CONSTRAINT fk_projects_tenant_id
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Resource Methods -> Resources
DO $$ BEGIN
    ALTER TABLE resource_methods ADD CONSTRAINT resource_methods_resource_id_fkey
        FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- =====================================================
-- INDEXES
-- =====================================================

-- Users indexes
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);
CREATE INDEX IF NOT EXISTS idx_users_client_id ON users(client_id);
CREATE INDEX IF NOT EXISTS idx_users_client_email_lower ON users(client_id, lower(email));
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_email_client ON users(email, client_id);
CREATE INDEX IF NOT EXISTS idx_users_external_id ON users(external_id);
CREATE INDEX IF NOT EXISTS idx_users_mfa ON users(mfa_enabled, mfa_verified);
CREATE INDEX IF NOT EXISTS idx_users_project_id ON users(project_id);
CREATE INDEX IF NOT EXISTS idx_users_provider_status ON users(provider, active);
CREATE INDEX IF NOT EXISTS idx_users_sync_info ON users(sync_source, is_synced_user);
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant_project ON users(tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_users_timestamps ON users(created_at, updated_at);

-- Partial indexes for users
CREATE INDEX IF NOT EXISTS idx_users_account_locked ON users(account_locked_at) WHERE (account_locked_at IS NOT NULL);
CREATE INDEX IF NOT EXISTS idx_users_is_primary_admin ON users(is_primary_admin) WHERE (is_primary_admin = true);
CREATE INDEX IF NOT EXISTS idx_users_password_change_required ON users(password_change_required) WHERE (password_change_required = true);
CREATE INDEX IF NOT EXISTS idx_users_temporary_password ON users(temporary_password) WHERE (temporary_password = true);

-- Tenants indexes
CREATE INDEX IF NOT EXISTS idx_tenants_email ON tenants(email);
CREATE INDEX IF NOT EXISTS idx_tenants_provider ON tenants(provider);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
CREATE INDEX IF NOT EXISTS idx_tenants_vault_mount ON tenants(vault_mount);

-- Roles indexes (partial indexes for global/system roles)
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_global_id ON roles(id) WHERE (tenant_id IS NULL);
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_global_name ON roles(name) WHERE (tenant_id IS NULL);

-- Permissions indexes (partial indexes for global permissions)
CREATE UNIQUE INDEX IF NOT EXISTS idx_permissions_global_id ON permissions(id) WHERE (tenant_id IS NULL);
CREATE UNIQUE INDEX IF NOT EXISTS idx_permissions_global_resource_action ON permissions(resource, action) WHERE (tenant_id IS NULL);

-- Clients indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_clients_client_id ON clients(client_id);
CREATE INDEX IF NOT EXISTS idx_clients_deleted_at ON clients(deleted_at);
CREATE INDEX IF NOT EXISTS idx_clients_status ON clients(status);

-- Credentials indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_credentials_credential_id ON credentials(credential_id);

-- Groups indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_name ON groups(name);
CREATE INDEX IF NOT EXISTS idx_groups_tenant_id ON groups(tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS groups_name_tenant_unique ON groups(name, tenant_id);

-- Resources indexes
CREATE INDEX IF NOT EXISTS idx_resources_tenant_id ON resources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_resources_name ON resources(name);

-- MFA Methods indexes
CREATE INDEX IF NOT EXISTS idx_mfa_methods_client_id ON mfa_methods(client_id);
CREATE INDEX IF NOT EXISTS idx_mfa_methods_enabled ON mfa_methods(enabled);
CREATE INDEX IF NOT EXISTS idx_mfa_methods_method_type ON mfa_methods(method_type);
CREATE INDEX IF NOT EXISTS idx_mfa_methods_user_id ON mfa_methods(user_id);

-- OTP Entries indexes
CREATE INDEX IF NOT EXISTS idx_otp_entries_email ON otp_entries(email);
CREATE INDEX IF NOT EXISTS idx_otp_entries_expires_at ON otp_entries(expires_at);
CREATE INDEX IF NOT EXISTS idx_otp_entries_verified ON otp_entries(verified);

-- Pending Registrations indexes
CREATE INDEX IF NOT EXISTS idx_pending_registrations_email ON pending_registrations(email);
CREATE INDEX IF NOT EXISTS idx_pending_registrations_expires_at ON pending_registrations(expires_at);
CREATE INDEX IF NOT EXISTS idx_pending_registrations_tenant_id ON pending_registrations(tenant_id);

-- OAuth Sessions indexes
CREATE INDEX IF NOT EXISTS idx_oauth_sessions_org_id ON oauth_sessions(org_id) WHERE (is_active = true);

-- API Scopes indexes
CREATE INDEX IF NOT EXISTS idx_api_scopes_name ON api_scopes(name);
CREATE INDEX IF NOT EXISTS idx_api_scopes_tenant_id ON api_scopes(tenant_id);

-- API Scope Permissions indexes
CREATE INDEX IF NOT EXISTS idx_api_scope_permissions_scope_id ON api_scope_permissions(scope_id);
CREATE INDEX IF NOT EXISTS idx_api_scope_permissions_permission_id ON api_scope_permissions(permission_id);

-- User Roles indexes
CREATE INDEX IF NOT EXISTS idx_user_roles_tenant_id ON user_roles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_tenant ON user_roles(user_id, tenant_id);

-- User Groups indexes
CREATE INDEX IF NOT EXISTS idx_user_groups_tenant_id ON user_groups(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_groups_user_tenant ON user_groups(user_id, tenant_id);

-- User Scopes indexes
CREATE INDEX IF NOT EXISTS idx_user_scopes_tenant_id ON user_scopes(tenant_id);

-- Client Roles indexes
CREATE INDEX IF NOT EXISTS idx_client_roles_tenant_id ON client_roles(tenant_id);

-- Services indexes
CREATE INDEX IF NOT EXISTS idx_services_agent_accessible ON services(agent_accessible);
CREATE INDEX IF NOT EXISTS idx_services_created_by ON services(created_by);
CREATE INDEX IF NOT EXISTS idx_services_resource_id ON services(resource_id);
CREATE INDEX IF NOT EXISTS idx_services_type ON services(type);

-- SAML Providers indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_saml_provider_unique ON saml_providers(tenant_id, client_id, provider_name);
CREATE INDEX IF NOT EXISTS idx_saml_providers_client_id ON saml_providers(client_id);
CREATE INDEX IF NOT EXISTS idx_saml_providers_is_active ON saml_providers(is_active);
CREATE INDEX IF NOT EXISTS idx_saml_providers_sort_order ON saml_providers(sort_order);
CREATE INDEX IF NOT EXISTS idx_saml_providers_tenant_id ON saml_providers(tenant_id);

-- SAML Callback States indexes
CREATE INDEX IF NOT EXISTS idx_saml_callback_states_expires_at ON saml_callback_states(expires_at);
CREATE INDEX IF NOT EXISTS idx_saml_callback_states_id ON saml_callback_states(id);

-- WebAuthn Sessions indexes
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_created_at ON webauthn_sessions(created_at);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_expires_at ON webauthn_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_user_id ON webauthn_sessions(user_id);

-- OIDC User Identities indexes
CREATE UNIQUE INDEX IF NOT EXISTS oidc_user_identities_provider_name_provider_user_id_key ON oidc_user_identities(provider_name, provider_user_id);
CREATE UNIQUE INDEX IF NOT EXISTS oidc_user_identities_tenant_id_user_id_provider_name_key ON oidc_user_identities(tenant_id, user_id, provider_name);

-- Audit Events indexes
CREATE INDEX IF NOT EXISTS idx_audit_events_action ON audit_events(action);
CREATE INDEX IF NOT EXISTS idx_audit_events_request_id ON audit_events(request_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource ON audit_events(resource);
CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_id ON audit_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events("timestamp");
CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON audit_events(user_id);

-- Audit Logs indexes
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_event_type ON audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_success ON audit_logs(success);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_id ON audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_workload_id ON audit_logs(workload_id);

-- Migration Logs indexes
CREATE INDEX IF NOT EXISTS idx_migration_logs_version ON migration_logs(version);
CREATE INDEX IF NOT EXISTS idx_migration_logs_db_type ON migration_logs(db_type);
CREATE INDEX IF NOT EXISTS idx_migration_logs_tenant_id ON migration_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_migration_logs_success ON migration_logs(success);

-- Tenant Databases indexes
CREATE INDEX IF NOT EXISTS idx_tenant_databases_tenant_id ON tenant_databases(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_databases_status ON tenant_databases(migration_status);

-- =====================================================
-- TRIGGERS
-- =====================================================

-- User Roles updated_at trigger
DO $$ BEGIN
    CREATE TRIGGER update_user_roles_updated_at
        BEFORE UPDATE ON user_roles
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- User Groups updated_at trigger
DO $$ BEGIN
    CREATE TRIGGER update_user_groups_updated_at
        BEFORE UPDATE ON user_groups
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- User Scopes updated_at trigger
DO $$ BEGIN
    CREATE TRIGGER update_user_scopes_updated_at
        BEFORE UPDATE ON user_scopes
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Group Roles updated_at trigger
DO $$ BEGIN
    CREATE TRIGGER update_group_roles_updated_at
        BEFORE UPDATE ON group_roles
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- SAML Providers updated_at trigger
DO $$ BEGIN
    CREATE TRIGGER saml_providers_updated_at
        BEFORE UPDATE ON saml_providers
        FOR EACH ROW EXECUTE FUNCTION update_saml_providers_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Services updated_at trigger
DO $$ BEGIN
    CREATE TRIGGER update_services_updated_at
        BEFORE UPDATE ON services
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- =====================================================
-- MIGRATION COMPLETE
-- =====================================================

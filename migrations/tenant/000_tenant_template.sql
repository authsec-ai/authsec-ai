--
-- PostgreSQL database dump
--


-- Dumped from database version 17.5
-- Dumped by pg_dump version 17.6 (Homebrew)


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--



--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: -
--



--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;




--
-- Name: client_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.client_groups (
    client_id uuid NOT NULL,
    group_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: client_resources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.client_resources (
    client_id uuid NOT NULL,
    resource_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: client_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.client_roles (
    client_id uuid NOT NULL,
    role_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: client_scopes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.client_scopes (
    client_id uuid NOT NULL,
    scope_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: clients; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.clients (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid,
    owner_id uuid NOT NULL,
    org_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    email text,
    status text DEFAULT 'Active'::text,
    tags text[],
    active boolean DEFAULT true,
    last_login timestamp with time zone,
    mfa_enabled boolean DEFAULT false,
    mfa_method text[],
    mfa_default_method text,
    mfa_enrolled_at timestamp with time zone,
    mfa_verified boolean DEFAULT false,
    hydra_client_id text,
    oidc_enabled boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone
);


--
-- Name: credentials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.credentials (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_id uuid NOT NULL,
    credential_id bytea NOT NULL,
    public_key bytea NOT NULL,
    attestation_type character varying(255),
    aaguid uuid,
    sign_count bigint DEFAULT 0,
    transports text[],
    backup_eligible boolean DEFAULT false,
    backup_state boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: group_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.group_roles (
    group_id uuid NOT NULL,
    role_id uuid NOT NULL,
    tenant_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    tenant_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: mfa_methods; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mfa_methods (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    client_id uuid NOT NULL,
    method_type character varying(50) NOT NULL,
    display_name character varying(255),
    description character varying(255),
    recommended boolean DEFAULT false,
    method_data jsonb,
    method_subtype character varying(255),
    is_primary boolean DEFAULT false,
    verified boolean DEFAULT false,
    enabled boolean DEFAULT false,
    backup_codes text,
    enrolled_at timestamp with time zone DEFAULT now(),
    expires_at timestamp with time zone,
    last_used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: oauth_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    client_id uuid NOT NULL,
    access_token text,
    refresh_token text,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: otp_entries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.otp_entries (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email character varying(255) NOT NULL,
    otp character varying(10) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    verified boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: pending_registrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pending_registrations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email character varying(255) NOT NULL,
    password_hash text NOT NULL,
    first_name character varying(100),
    last_name character varying(100),
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    client_id uuid NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    tenant_domain character varying(255) NOT NULL
);


--
-- Name: permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.permissions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    role_id uuid NOT NULL,
    scope_id uuid NOT NULL,
    resource_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: projects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.projects (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    user_id uuid,
    tenant_id uuid,
    client_id uuid,
    active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: resource_methods; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.resource_methods (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    resource_id uuid NOT NULL,
    method character varying(10) NOT NULL,
    path_pattern character varying(255) NOT NULL,
    requires_admin boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: resources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.resources (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    name character varying(100) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone,
    type character varying(255) DEFAULT 'generic'::character varying
);


--
-- Name: roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.roles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    name character varying(100) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone
);


--
-- Name: scopes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.scopes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    name character varying(100) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone
);


--
-- Name: api_scopes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_scopes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: api_scope_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_scope_permissions (
    scope_id uuid NOT NULL,
    permission_id uuid NOT NULL
);


--
-- Name: services; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.services (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(100),
    url character varying(500),
    description text,
    tags text[],
    resource_id uuid,
    auth_type character varying(100) NOT NULL,
    auth_config text,
    vault_path character varying(255),
    created_by uuid NOT NULL,
    agent_accessible boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: spiffe_svids; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.spiffe_svids (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    workload_id uuid,
    svid bytea,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: spiffe_workloads; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.spiffe_workloads (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    workload_name text NOT NULL,
    selector text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: tenant_mappings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenant_mappings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: tenants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenants (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    tenant_db character varying(255),
    email character varying(255),
    username character varying(255),
    password_hash text,
    provider character varying(50) DEFAULT 'local'::character varying,
    provider_id character varying(255),
    avatar text,
    name character varying(255),
    source character varying(50),
    status character varying(50) DEFAULT 'active'::character varying,
    last_login timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    tenant_domain character varying(255) DEFAULT 'app.authsec.dev'::character varying
);


--
-- Name: user_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_groups (
    user_id uuid NOT NULL,
    group_id uuid NOT NULL,
    tenant_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: user_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_roles (
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    tenant_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: role_bindings; Type: TABLE; Schema: public; Owner: -
-- Note: This is the new RBAC table that replaces user_roles for auth-manager v1.1.2+
--

CREATE TABLE public.role_bindings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    user_id uuid,
    service_account_id uuid,
    role_id uuid NOT NULL,
    role_name text,
    username text,
    scope_type text DEFAULT '*',
    scope_id uuid,
    conditions jsonb DEFAULT '{}',
    expires_at timestamp with time zone,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: user_scopes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_scopes (
    user_id uuid NOT NULL,
    scope_id uuid NOT NULL,
    tenant_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_id uuid,
    tenant_id uuid,
    project_id uuid,
    email character varying(255) NOT NULL,
    name character varying(255),
    username character varying(255),
    password_hash text,
    tenant_domain character varying(255) DEFAULT 'app.authsec.dev'::character varying,
    provider character varying(100) DEFAULT 'local'::character varying,
    provider_id character varying(255),
    provider_data jsonb,
    avatar_url text,
    active boolean DEFAULT true,
    mfa_enabled boolean DEFAULT false,
    mfa_method text[],
    mfa_default_method character varying(50),
    mfa_enrolled_at timestamp with time zone,
    mfa_verified boolean DEFAULT false,
    external_id character varying(255),
    sync_source character varying(100),
    last_sync_at timestamp with time zone,
    is_synced_user boolean DEFAULT false,
    last_login timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: saml_providers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.saml_providers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid NOT NULL,
    provider_name character varying(255) NOT NULL,
    display_name character varying(255) NOT NULL,
    entity_id character varying(500) NOT NULL,
    sso_url character varying(500) NOT NULL,
    slo_url character varying(500),
    certificate text NOT NULL,
    metadata_url character varying(500),
    name_id_format character varying(255) DEFAULT 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress'::character varying,
    attribute_mapping jsonb,
    is_active boolean DEFAULT true,
    sort_order integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: TABLE saml_providers; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.saml_providers IS 'SAML Identity Provider (IdP) configurations - per client within tenant';


--
-- Name: COLUMN saml_providers.tenant_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.tenant_id IS 'Tenant UUID owning this provider';


--
-- Name: COLUMN saml_providers.client_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.client_id IS 'Client (app) UUID within the tenant - enables multi-client isolation';


--
-- Name: COLUMN saml_providers.provider_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.provider_name IS 'Provider identifier (MUST be lowercase with unique phrase, e.g., "okta-hr", "azure-finance") - auto-normalized via GORM hooks';


--
-- Name: COLUMN saml_providers.display_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.display_name IS 'Human-readable name shown in login UI (e.g., "Okta SAML", "Azure AD")';


--
-- Name: COLUMN saml_providers.entity_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.entity_id IS 'IdP Entity ID - unique identifier for the Identity Provider';


--
-- Name: COLUMN saml_providers.sso_url; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.sso_url IS 'Single Sign-On URL - where to send SAML authentication requests';


--
-- Name: COLUMN saml_providers.slo_url; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.slo_url IS 'Single Logout URL (optional) - for logout requests';


--
-- Name: COLUMN saml_providers.certificate; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.certificate IS 'IdP X.509 certificate in PEM format - for validating SAML response signatures';


--
-- Name: COLUMN saml_providers.metadata_url; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.metadata_url IS 'Optional URL to fetch IdP metadata for auto-configuration';


--
-- Name: COLUMN saml_providers.name_id_format; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.name_id_format IS 'Format for user identifier - usually email address';


--
-- Name: COLUMN saml_providers.attribute_mapping; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.attribute_mapping IS 'JSON mapping of SAML attributes to user fields (e.g., {"email": "email", "first_name": "firstName"})';


--
-- Name: COLUMN saml_providers.is_active; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.is_active IS 'Whether this provider is available for login';


--
-- Name: COLUMN saml_providers.sort_order; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.saml_providers.sort_order IS 'Display order in login UI (lower = higher priority)';


--
-- Name: webauthn_credentials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webauthn_credentials (
    id integer NOT NULL,
    user_id uuid,
    credential_id text NOT NULL,
    public_key text NOT NULL,
    attestation_type text,
    transports text[],
    backup_eligible boolean DEFAULT false,
    backup_state boolean DEFAULT false,
    sign_count bigint DEFAULT 0,
    user_present boolean DEFAULT false,
    user_verified boolean DEFAULT false,
    aaguid text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: webauthn_credentials_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.webauthn_credentials_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: webauthn_credentials_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.webauthn_credentials_id_seq OWNED BY public.webauthn_credentials.id;


--
-- Name: webauthn_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webauthn_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    session_key character varying(255) NOT NULL,
    challenge text NOT NULL,
    user_id bytea NOT NULL,
    user_verification character varying(50),
    extensions bytea,
    cred_params bytea,
    allowed_credential_ids bytea,
    created_at timestamp with time zone DEFAULT now(),
    expires_at timestamp with time zone NOT NULL
);


--
-- Name: webauthn_credentials id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webauthn_credentials ALTER COLUMN id SET DEFAULT nextval('public.webauthn_credentials_id_seq'::regclass);


--
-- Name: client_groups client_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_groups
    ADD CONSTRAINT client_groups_pkey PRIMARY KEY (client_id, group_id);


--
-- Name: client_resources client_resources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_resources
    ADD CONSTRAINT client_resources_pkey PRIMARY KEY (client_id, resource_id);


--
-- Name: client_roles client_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_roles
    ADD CONSTRAINT client_roles_pkey PRIMARY KEY (client_id, role_id);


--
-- Name: client_scopes client_scopes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_scopes
    ADD CONSTRAINT client_scopes_pkey PRIMARY KEY (client_id, scope_id);


--
-- Name: clients clients_client_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.clients
    ADD CONSTRAINT clients_client_id_key UNIQUE (client_id);


--
-- Name: clients clients_hydra_client_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.clients
    ADD CONSTRAINT clients_hydra_client_id_key UNIQUE (hydra_client_id);


--
-- Name: clients clients_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.clients
    ADD CONSTRAINT clients_pkey PRIMARY KEY (id);


--
-- Name: credentials credentials_credential_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.credentials
    ADD CONSTRAINT credentials_credential_id_key UNIQUE (credential_id);


--
-- Name: credentials credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.credentials
    ADD CONSTRAINT credentials_pkey PRIMARY KEY (id);


--
-- Name: group_roles group_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_roles
    ADD CONSTRAINT group_roles_pkey PRIMARY KEY (group_id, role_id);


--
-- Name: groups groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);


--
-- Name: mfa_methods mfa_methods_client_id_method_type_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mfa_methods
    ADD CONSTRAINT mfa_methods_client_id_method_type_key UNIQUE (client_id, method_type);


--
-- Name: mfa_methods mfa_methods_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mfa_methods
    ADD CONSTRAINT mfa_methods_pkey PRIMARY KEY (id);


--
-- Name: oauth_sessions oauth_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_sessions
    ADD CONSTRAINT oauth_sessions_pkey PRIMARY KEY (id);


--
-- Name: otp_entries otp_entries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.otp_entries
    ADD CONSTRAINT otp_entries_pkey PRIMARY KEY (id);


--
-- Name: pending_registrations pending_registrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pending_registrations
    ADD CONSTRAINT pending_registrations_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_role_id_scope_id_resource_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_role_id_scope_id_resource_id_key UNIQUE (role_id, scope_id, resource_id);


--
-- Name: projects projects_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_pkey PRIMARY KEY (id);


--
-- Name: resource_methods resource_methods_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_methods
    ADD CONSTRAINT resource_methods_pkey PRIMARY KEY (id);


--
-- Name: resource_methods resource_methods_resource_id_method_path_pattern_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_methods
    ADD CONSTRAINT resource_methods_resource_id_method_path_pattern_key UNIQUE (resource_id, method, path_pattern);


--
-- Name: resources resources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resources
    ADD CONSTRAINT resources_pkey PRIMARY KEY (id);


--
-- Name: resources resources_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resources
    ADD CONSTRAINT resources_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: roles roles_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: scopes scopes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.scopes
    ADD CONSTRAINT scopes_pkey PRIMARY KEY (id);


--
-- Name: scopes scopes_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.scopes
    ADD CONSTRAINT scopes_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: api_scopes api_scopes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_scopes
    ADD CONSTRAINT api_scopes_pkey PRIMARY KEY (id);


--
-- Name: api_scopes uq_api_scopes_tenant_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_scopes
    ADD CONSTRAINT uq_api_scopes_tenant_name UNIQUE (tenant_id, name);


--
-- Name: api_scopes uq_api_scopes_tenant_id; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_scopes
    ADD CONSTRAINT uq_api_scopes_tenant_id UNIQUE (tenant_id, id);


--
-- Name: api_scope_permissions api_scope_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_scope_permissions
    ADD CONSTRAINT api_scope_permissions_pkey PRIMARY KEY (scope_id, permission_id);


--
-- Name: services services_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_pkey PRIMARY KEY (id);


--
-- Name: spiffe_svids spiffe_svids_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.spiffe_svids
    ADD CONSTRAINT spiffe_svids_pkey PRIMARY KEY (id);


--
-- Name: spiffe_workloads spiffe_workloads_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.spiffe_workloads
    ADD CONSTRAINT spiffe_workloads_pkey PRIMARY KEY (id);


--
-- Name: tenant_mappings tenant_mappings_client_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_mappings
    ADD CONSTRAINT tenant_mappings_client_id_key UNIQUE (client_id);


--
-- Name: tenant_mappings tenant_mappings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_mappings
    ADD CONSTRAINT tenant_mappings_pkey PRIMARY KEY (id);


--
-- Name: tenants tenants_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_email_key UNIQUE (email);


--
-- Name: tenants tenants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_pkey PRIMARY KEY (id);


--
-- Name: tenants tenants_tenant_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_tenant_id_key UNIQUE (tenant_id);


--
-- Name: user_groups user_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_groups
    ADD CONSTRAINT user_groups_pkey PRIMARY KEY (user_id, group_id);


--
-- Name: user_groups user_groups_user_group_tenant_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_groups
    ADD CONSTRAINT user_groups_user_group_tenant_key UNIQUE (user_id, group_id, tenant_id);


--
-- Name: user_roles user_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id);


--
-- Name: user_roles user_roles_user_role_tenant_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_user_role_tenant_key UNIQUE (user_id, role_id, tenant_id);


--
-- Name: role_bindings role_bindings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_bindings
    ADD CONSTRAINT role_bindings_pkey PRIMARY KEY (id);


--
-- Name: user_scopes user_scopes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_scopes
    ADD CONSTRAINT user_scopes_pkey PRIMARY KEY (user_id, scope_id);


--
-- Name: users users_client_email_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_client_email_unique UNIQUE (client_id, email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_tenant_id_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_tenant_id_id_unique UNIQUE (tenant_id, id);


--
-- Name: saml_providers saml_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saml_providers
    ADD CONSTRAINT saml_providers_pkey PRIMARY KEY (id);


--
-- Name: webauthn_credentials webauthn_credentials_credential_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webauthn_credentials
    ADD CONSTRAINT webauthn_credentials_credential_id_key UNIQUE (credential_id);


--
-- Name: webauthn_credentials webauthn_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webauthn_credentials
    ADD CONSTRAINT webauthn_credentials_pkey PRIMARY KEY (id);


--
-- Name: webauthn_sessions webauthn_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webauthn_sessions
    ADD CONSTRAINT webauthn_sessions_pkey PRIMARY KEY (id);


--
-- Name: webauthn_sessions webauthn_sessions_session_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webauthn_sessions
    ADD CONSTRAINT webauthn_sessions_session_key_key UNIQUE (session_key);


--
-- Name: idx_api_scopes_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_scopes_tenant_id ON public.api_scopes USING btree (tenant_id);


--
-- Name: idx_api_scopes_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_scopes_name ON public.api_scopes USING btree (name);


--
-- Name: idx_api_scope_permissions_scope_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_scope_permissions_scope_id ON public.api_scope_permissions USING btree (scope_id);


--
-- Name: idx_api_scope_permissions_permission_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_scope_permissions_permission_id ON public.api_scope_permissions USING btree (permission_id);


--
-- Name: idx_client_groups_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_client_groups_client_id ON public.client_groups USING btree (client_id);


--
-- Name: idx_client_groups_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_client_groups_group_id ON public.client_groups USING btree (group_id);


--
-- Name: idx_client_resources_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_client_resources_client_id ON public.client_resources USING btree (client_id);


--
-- Name: idx_client_resources_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_client_resources_resource_id ON public.client_resources USING btree (resource_id);


--
-- Name: idx_client_roles_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_client_roles_client_id ON public.client_roles USING btree (client_id);


--
-- Name: idx_client_roles_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_client_roles_role_id ON public.client_roles USING btree (role_id);


--
-- Name: idx_client_scopes_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_client_scopes_client_id ON public.client_scopes USING btree (client_id);


--
-- Name: idx_client_scopes_scope_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_client_scopes_scope_id ON public.client_scopes USING btree (scope_id);


--
-- Name: idx_clients_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_clients_active ON public.clients USING btree (active);


--
-- Name: idx_clients_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_clients_client_id ON public.clients USING btree (client_id);


--
-- Name: idx_clients_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_clients_deleted_at ON public.clients USING btree (deleted_at);


--
-- Name: idx_clients_hydra_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_clients_hydra_client_id ON public.clients USING btree (hydra_client_id);


--
-- Name: idx_clients_org_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_clients_org_id ON public.clients USING btree (org_id);


--
-- Name: idx_clients_owner_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_clients_owner_id ON public.clients USING btree (owner_id);


--
-- Name: idx_clients_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_clients_project_id ON public.clients USING btree (project_id);


--
-- Name: idx_clients_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_clients_status ON public.clients USING btree (status);


--
-- Name: idx_clients_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_clients_tenant_id ON public.clients USING btree (tenant_id);


--
-- Name: idx_credentials_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_credentials_client_id ON public.credentials USING btree (client_id);


--
-- Name: idx_group_roles_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_group_roles_group_id ON public.group_roles USING btree (group_id);


--
-- Name: idx_group_roles_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_group_roles_role_id ON public.group_roles USING btree (role_id);


--
-- Name: idx_group_roles_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_group_roles_tenant_id ON public.group_roles USING btree (tenant_id);


--
-- Name: idx_mfa_methods_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mfa_methods_client_id ON public.mfa_methods USING btree (client_id);


--
-- Name: idx_mfa_methods_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mfa_methods_enabled ON public.mfa_methods USING btree (enabled) WHERE (enabled = true);


--
-- Name: idx_mfa_methods_method_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mfa_methods_method_type ON public.mfa_methods USING btree (method_type);


--
-- Name: idx_mfa_methods_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mfa_methods_user_id ON public.mfa_methods USING btree (user_id);


--
-- Name: idx_oauth_sessions_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_sessions_client_id ON public.oauth_sessions USING btree (client_id);


--
-- Name: idx_oauth_sessions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_sessions_user_id ON public.oauth_sessions USING btree (user_id);


--
-- Name: idx_otp_entries_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_otp_entries_email ON public.otp_entries USING btree (email);


--
-- Name: idx_otp_entries_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_otp_entries_expires_at ON public.otp_entries USING btree (expires_at);


--
-- Name: idx_otp_entries_verified; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_otp_entries_verified ON public.otp_entries USING btree (verified);


--
-- Name: idx_pending_registrations_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pending_registrations_email ON public.pending_registrations USING btree (email);


--
-- Name: idx_pending_registrations_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pending_registrations_expires_at ON public.pending_registrations USING btree (expires_at);


--
-- Name: idx_pending_registrations_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pending_registrations_tenant_id ON public.pending_registrations USING btree (tenant_id);


--
-- Name: idx_permissions_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_permissions_resource_id ON public.permissions USING btree (resource_id);


--
-- Name: idx_permissions_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_permissions_role_id ON public.permissions USING btree (role_id);


--
-- Name: idx_permissions_scope_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_permissions_scope_id ON public.permissions USING btree (scope_id);


--
-- Name: idx_projects_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_active ON public.projects USING btree (active);


--
-- Name: idx_projects_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_client_id ON public.projects USING btree (client_id);


--
-- Name: idx_projects_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_tenant_id ON public.projects USING btree (tenant_id);


--
-- Name: idx_projects_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_user_id ON public.projects USING btree (user_id);


--
-- Name: idx_services_agent_accessible; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_services_agent_accessible ON public.services USING btree (agent_accessible);


--
-- Name: idx_services_created_by; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_services_created_by ON public.services USING btree (created_by);


--
-- Name: idx_services_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_services_type ON public.services USING btree (type);


--
-- Name: idx_saml_provider_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_saml_provider_unique ON public.saml_providers USING btree (tenant_id, client_id, provider_name);


--
-- Name: idx_saml_providers_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_saml_providers_tenant_id ON public.saml_providers USING btree (tenant_id);


--
-- Name: idx_saml_providers_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_saml_providers_client_id ON public.saml_providers USING btree (client_id);


--
-- Name: idx_saml_providers_is_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_saml_providers_is_active ON public.saml_providers USING btree (is_active);


--
-- Name: idx_saml_providers_sort_order; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_saml_providers_sort_order ON public.saml_providers USING btree (sort_order);


--
-- Name: idx_tenant_mappings_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenant_mappings_client_id ON public.tenant_mappings USING btree (client_id);


--
-- Name: idx_tenant_mappings_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenant_mappings_tenant_id ON public.tenant_mappings USING btree (tenant_id);


--
-- Name: idx_tenants_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenants_email ON public.tenants USING btree (email);


--
-- Name: idx_tenants_tenant_domain; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenants_tenant_domain ON public.tenants USING btree (tenant_domain);


--
-- Name: idx_tenants_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenants_tenant_id ON public.tenants USING btree (tenant_id);


--
-- Name: idx_user_groups_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_groups_group_id ON public.user_groups USING btree (group_id);


--
-- Name: idx_user_groups_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_groups_tenant_id ON public.user_groups USING btree (tenant_id);


--
-- Name: idx_user_groups_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_groups_user_id ON public.user_groups USING btree (user_id);


--
-- Name: idx_user_groups_user_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_groups_user_tenant ON public.user_groups USING btree (user_id, tenant_id);


--
-- Name: idx_user_roles_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_roles_role_id ON public.user_roles USING btree (role_id);


--
-- Name: idx_user_roles_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_roles_tenant_id ON public.user_roles USING btree (tenant_id);


--
-- Name: idx_user_roles_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_roles_user_id ON public.user_roles USING btree (user_id);


--
-- Name: idx_user_roles_user_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_roles_user_tenant ON public.user_roles USING btree (user_id, tenant_id);


--
-- Name: idx_user_scopes_scope_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_scopes_scope_id ON public.user_scopes USING btree (scope_id);


--
-- Name: idx_user_scopes_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_scopes_tenant_id ON public.user_scopes USING btree (tenant_id);


--
-- Name: idx_user_scopes_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_scopes_user_id ON public.user_scopes USING btree (user_id);


--
-- Name: idx_users_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_active ON public.users USING btree (active);


--
-- Name: idx_users_client_email_lower; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_client_email_lower ON public.users USING btree (client_id, lower((email)::text));


--
-- Name: idx_users_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_client_id ON public.users USING btree (client_id);


--
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_email ON public.users USING btree (email);


--
-- Name: idx_users_external_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_external_id ON public.users USING btree (external_id);


--
-- Name: idx_users_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_project_id ON public.users USING btree (project_id);


--
-- Name: idx_users_provider; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_provider ON public.users USING btree (provider);


--
-- Name: idx_users_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_tenant_id ON public.users USING btree (tenant_id);


--
-- Name: idx_webauthn_sessions_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webauthn_sessions_expires_at ON public.webauthn_sessions USING btree (expires_at);


--
-- Name: idx_webauthn_sessions_session_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webauthn_sessions_session_key ON public.webauthn_sessions USING btree (session_key);


--
-- Name: idx_webauthn_sessions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webauthn_sessions_user_id ON public.webauthn_sessions USING btree (user_id);


--
-- Name: group_roles update_group_roles_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_group_roles_updated_at BEFORE UPDATE ON public.group_roles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: saml_providers saml_providers_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER saml_providers_updated_at BEFORE UPDATE ON public.saml_providers FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: user_groups update_user_groups_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_user_groups_updated_at BEFORE UPDATE ON public.user_groups FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: user_roles update_user_roles_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_user_roles_updated_at BEFORE UPDATE ON public.user_roles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: user_scopes update_user_scopes_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_user_scopes_updated_at BEFORE UPDATE ON public.user_scopes FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: api_scope_permissions api_scope_permissions_scope_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_scope_permissions
    ADD CONSTRAINT api_scope_permissions_scope_id_fkey FOREIGN KEY (scope_id) REFERENCES public.api_scopes(id) ON DELETE CASCADE;


--
-- Name: api_scope_permissions api_scope_permissions_permission_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_scope_permissions
    ADD CONSTRAINT api_scope_permissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES public.permissions(id) ON DELETE CASCADE;


--
-- Name: client_groups client_groups_client_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_groups
    ADD CONSTRAINT client_groups_client_id_fkey FOREIGN KEY (client_id) REFERENCES public.clients(id) ON DELETE CASCADE;


--
-- Name: client_groups client_groups_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_groups
    ADD CONSTRAINT client_groups_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.groups(id) ON DELETE CASCADE;


--
-- Name: client_resources client_resources_client_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_resources
    ADD CONSTRAINT client_resources_client_id_fkey FOREIGN KEY (client_id) REFERENCES public.clients(id) ON DELETE CASCADE;


--
-- Name: client_resources client_resources_resource_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_resources
    ADD CONSTRAINT client_resources_resource_id_fkey FOREIGN KEY (resource_id) REFERENCES public.resources(id) ON DELETE CASCADE;


--
-- Name: client_roles client_roles_client_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_roles
    ADD CONSTRAINT client_roles_client_id_fkey FOREIGN KEY (client_id) REFERENCES public.clients(id) ON DELETE CASCADE;


--
-- Name: client_roles client_roles_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_roles
    ADD CONSTRAINT client_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: client_scopes client_scopes_client_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_scopes
    ADD CONSTRAINT client_scopes_client_id_fkey FOREIGN KEY (client_id) REFERENCES public.clients(id) ON DELETE CASCADE;


--
-- Name: client_scopes client_scopes_scope_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_scopes
    ADD CONSTRAINT client_scopes_scope_id_fkey FOREIGN KEY (scope_id) REFERENCES public.scopes(id) ON DELETE CASCADE;


--
-- Name: group_roles group_roles_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_roles
    ADD CONSTRAINT group_roles_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.groups(id) ON DELETE CASCADE;


--
-- Name: group_roles group_roles_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_roles
    ADD CONSTRAINT group_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: mfa_methods mfa_methods_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mfa_methods
    ADD CONSTRAINT mfa_methods_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: permissions permissions_resource_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_resource_id_fkey FOREIGN KEY (resource_id) REFERENCES public.resources(id) ON DELETE CASCADE;


--
-- Name: permissions permissions_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: permissions permissions_scope_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_scope_id_fkey FOREIGN KEY (scope_id) REFERENCES public.scopes(id) ON DELETE CASCADE;


--
-- Name: resource_methods resource_methods_resource_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_methods
    ADD CONSTRAINT resource_methods_resource_id_fkey FOREIGN KEY (resource_id) REFERENCES public.resources(id) ON DELETE CASCADE;


--
-- Name: services services_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: user_groups user_groups_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_groups
    ADD CONSTRAINT user_groups_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.groups(id) ON DELETE CASCADE;


--
-- Name: user_groups user_groups_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_groups
    ADD CONSTRAINT user_groups_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_roles user_roles_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: user_roles user_roles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: role_bindings role_bindings_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_bindings
    ADD CONSTRAINT role_bindings_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: role_bindings role_bindings_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_bindings
    ADD CONSTRAINT role_bindings_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: user_scopes user_scopes_scope_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_scopes
    ADD CONSTRAINT user_scopes_scope_id_fkey FOREIGN KEY (scope_id) REFERENCES public.scopes(id) ON DELETE CASCADE;


--
-- Name: user_scopes user_scopes_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_scopes
    ADD CONSTRAINT user_scopes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

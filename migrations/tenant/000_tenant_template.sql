--
-- PostgreSQL database dump
--


-- Dumped from database version 16.1
-- Dumped by pg_dump version 18.3 (Ubuntu 18.3-1.pgdg24.04+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', 'public', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: backup_foreign_keys(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.backup_foreign_keys() RETURNS TABLE(constraint_name text, table_name text, column_name text, foreign_table_name text, foreign_column_name text, delete_rule text, update_rule text)
    LANGUAGE plpgsql
    AS $$

BEGIN

    RETURN QUERY

    SELECT

        tc.constraint_name::text,

        tc.table_name::text,

        kcu.column_name::text,

        ccu.table_name::text,

        ccu.column_name::text,

        rc.delete_rule::text,

        rc.update_rule::text

    FROM information_schema.table_constraints tc

    JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name

    JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = tc.constraint_name

    JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name

    WHERE tc.constraint_type = 'FOREIGN KEY'

    AND (ccu.table_name = 'tenants' OR tc.table_name = 'tenants');

END $$;



--
-- Name: check_ca_expiration(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.check_ca_expiration() RETURNS trigger
    LANGUAGE plpgsql
    AS $$


BEGIN


    IF NEW.not_after < NOW() AND NEW.status = 'active' THEN


        NEW.status = 'expired';


    END IF;


    RETURN NEW;


END;


$$;



--
-- Name: check_cert_expiration(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.check_cert_expiration() RETURNS trigger
    LANGUAGE plpgsql
    AS $$


BEGIN


    IF NEW.not_after < NOW() AND NEW.status = 'active' THEN


        NEW.status = 'expired';


    END IF;


    RETURN NEW;


END;


$$;



--
-- Name: cleanup_expired_device_codes(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.cleanup_expired_device_codes() RETURNS integer
    LANGUAGE plpgsql
    AS $$

DECLARE

    deleted_count INTEGER;

    current_epoch BIGINT;

BEGIN

    current_epoch := EXTRACT(EPOCH FROM NOW())::BIGINT;



    DELETE FROM device_codes

    WHERE expires_at < (current_epoch - 86400)  -- 24 hours ago

    AND status IN ('expired', 'consumed', 'denied');



    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    RETURN deleted_count;

END;

$$;



--
-- Name: FUNCTION cleanup_expired_device_codes(); Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON FUNCTION public.cleanup_expired_device_codes() IS 'Deletes device codes older than 24 hours that are expired/consumed/denied';


--
-- Name: cleanup_expired_voice_active_sessions(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.cleanup_expired_voice_active_sessions() RETURNS integer
    LANGUAGE plpgsql
    AS $$

DECLARE

    deleted_count INTEGER;

BEGIN

    UPDATE voice_active_sessions

    SET is_active = false,

        revoked_at = NOW(),

        revoked_reason = 'expired'

    WHERE is_active = true

    AND expires_at < NOW();



    DELETE FROM voice_active_sessions

    WHERE is_active = false

    AND (revoked_at < NOW() - INTERVAL '30 days' OR expires_at < NOW() - INTERVAL '30 days');



    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    RETURN deleted_count;

END;

$$;



--
-- Name: FUNCTION cleanup_expired_voice_active_sessions(); Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON FUNCTION public.cleanup_expired_voice_active_sessions() IS 'Marks expired sessions as inactive and deletes old inactive sessions';


--
-- Name: cleanup_expired_voice_sessions(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.cleanup_expired_voice_sessions() RETURNS integer
    LANGUAGE plpgsql
    AS $$

DECLARE

    deleted_count INTEGER;

    current_epoch BIGINT;

BEGIN

    current_epoch := EXTRACT(EPOCH FROM NOW())::BIGINT;



    DELETE FROM voice_sessions

    WHERE expires_at < (current_epoch - 3600)  -- 1 hour ago

    AND status IN ('expired', 'failed', 'verified');



    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    RETURN deleted_count;

END;

$$;



--
-- Name: FUNCTION cleanup_expired_voice_sessions(); Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON FUNCTION public.cleanup_expired_voice_sessions() IS 'Deletes voice sessions older than 1 hour that are expired/failed/verified';


--
-- Name: set_updated_at(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$


BEGIN


  NEW.updated_at = NOW();


  RETURN NEW;


END;


$$;



--
-- Name: update_device_codes_updated_at(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.update_device_codes_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$

BEGIN

    NEW.updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT;

    RETURN NEW;

END;

$$;



--
-- Name: update_m2m_agent_updated_at(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.update_m2m_agent_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$


BEGIN


    NEW.updated_at = NOW();


    RETURN NEW;


END;


$$;



--
-- Name: update_m2m_updated_at(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.update_m2m_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$


BEGIN


    NEW.updated_at = NOW();


    RETURN NEW;


END;


$$;



--
-- Name: update_oidc_providers_updated_at(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.update_oidc_providers_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$

BEGIN

    NEW.updated_at = CURRENT_TIMESTAMP;

    RETURN NEW;

END;

$$;



--
-- Name: update_oidc_user_identities_updated_at(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.update_oidc_user_identities_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$

BEGIN

    NEW.updated_at = CURRENT_TIMESTAMP;

    RETURN NEW;

END;

$$;



--
-- Name: update_saml_providers_updated_at(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.update_saml_providers_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$

BEGIN

    NEW.updated_at = CURRENT_TIMESTAMP;

    RETURN NEW;

END;

$$;



--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;



--
-- Name: update_voice_active_sessions_updated_at(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.update_voice_active_sessions_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$

BEGIN

    NEW.updated_at = NOW();

    RETURN NEW;

END;

$$;



--
-- Name: update_voice_identity_links_updated_at(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.update_voice_identity_links_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$

BEGIN

    NEW.updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT;

    RETURN NEW;

END;

$$;



--
-- Name: update_voice_sessions_updated_at(); Type: FUNCTION; Schema: public; Owner: authprod
--

CREATE FUNCTION public.update_voice_sessions_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$

BEGIN

    NEW.updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT;

    RETURN NEW;

END;

$$;



SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: agents; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.agents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    spiffe_id character varying(512) NOT NULL,
    attested_at timestamp without time zone,
    serial_number character varying(255),
    banned boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    tenant_id character varying(255),
    node_id character varying(255),
    attestation_type character varying(100),
    node_selectors jsonb,
    certificate_serial character varying(255),
    status character varying(50) DEFAULT 'active'::character varying,
    last_seen timestamp without time zone,
    cluster_name character varying(255),
    last_heartbeat timestamp without time zone
);



--
-- Name: api_scope_permissions; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.api_scope_permissions (
    scope_id uuid NOT NULL,
    permission_id uuid NOT NULL
);



--
-- Name: TABLE api_scope_permissions; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.api_scope_permissions IS 'Maps API Scopes to internal RBAC Permissions (M:N)';


--
-- Name: api_scopes; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.api_scopes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now()
);



--
-- Name: TABLE api_scopes; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.api_scopes IS 'OAuth API Scopes - external contracts that map to internal RBAC permissions';


--
-- Name: COLUMN api_scopes.name; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.api_scopes.name IS 'OAuth scope name, e.g., files:read, project:write';


--
-- Name: attestation_policies; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.attestation_policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    attestation_type character varying(50) NOT NULL,
    selector_rules jsonb DEFAULT '{}'::jsonb NOT NULL,
    vault_role character varying(255) NOT NULL,
    ttl integer DEFAULT 3600 NOT NULL,
    priority integer DEFAULT 0 NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);



--
-- Name: TABLE attestation_policies; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.attestation_policies IS 'Policies for workload attestation and certificate issuance';


--
-- Name: COLUMN attestation_policies.selector_rules; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.attestation_policies.selector_rules IS 'JSONB containing selector matching rules';


--
-- Name: COLUMN attestation_policies.vault_role; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.attestation_policies.vault_role IS 'Vault PKI role to use for certificate issuance';


--
-- Name: COLUMN attestation_policies.ttl; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.attestation_policies.ttl IS 'Certificate time-to-live in seconds';


--
-- Name: COLUMN attestation_policies.priority; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.attestation_policies.priority IS 'Policy priority for matching (higher = higher priority)';


--
-- Name: audit_events; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.audit_events (
    id bigint NOT NULL,
    request_id text,
    tenant_id text,
    user_id text,
    action text,
    resource text,
    resource_id text,
    method text,
    path text,
    user_agent text,
    client_ip text,
    status_code bigint,
    duration bigint,
    old_values jsonb,
    new_values jsonb,
    error text,
    "timestamp" timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);



--
-- Name: audit_events_id_seq; Type: SEQUENCE; Schema: public; Owner: authprod
--

CREATE SEQUENCE public.audit_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;



--
-- Name: audit_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: authprod
--

ALTER SEQUENCE public.audit_events_id_seq OWNED BY public.audit_events.id;


--
-- Name: audit_logs; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.audit_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    action character varying(50) NOT NULL,
    resource_type character varying(50) NOT NULL,
    resource_id character varying(255),
    details jsonb,
    created_at timestamp without time zone DEFAULT now(),
    tenant_id uuid,
    event_type character varying(50),
    workload_id character varying(255),
    certificate_id character varying(255),
    spiffe_id character varying(500),
    success boolean,
    error_message text,
    metadata jsonb,
    ip_address character varying(100),
    user_agent text
);



--
-- Name: auth_agents; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.auth_agents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    type character varying(10) NOT NULL,
    name character varying(255) NOT NULL,
    auth_provider character varying(50),
    mfa_methods text[],
    api_key character varying(255) NOT NULL,
    api_secret_hash character varying(255) NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT auth_agents_status_check CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('inactive'::character varying)::text, ('deleted'::character varying)::text]))),
    CONSTRAINT auth_agents_type_check CHECK (((type)::text = ANY (ARRAY[('voice'::character varying)::text, ('web'::character varying)::text])))
);



--
-- Name: TABLE auth_agents; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.auth_agents IS 'Configuration storage for voice and web agents that use existing authentication flows';


--
-- Name: COLUMN auth_agents.type; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.auth_agents.type IS 'Type of agent: voice (telephony) or web (chat widget)';


--
-- Name: COLUMN auth_agents.auth_provider; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.auth_agents.auth_provider IS 'OAuth provider or device auth method to use';


--
-- Name: COLUMN auth_agents.mfa_methods; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.auth_agents.mfa_methods IS 'Array of enabled MFA methods for this agent';


--
-- Name: COLUMN auth_agents.api_key; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.auth_agents.api_key IS 'Public API key for agent authentication (shown to admin)';


--
-- Name: COLUMN auth_agents.api_secret_hash; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.auth_agents.api_secret_hash IS 'Bcrypt hash of API secret (never returned in responses)';


--
-- Name: certificate_revocation_list; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.certificate_revocation_list (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_id uuid NOT NULL,
    cert_id uuid NOT NULL,
    serial_number character varying(255) NOT NULL,
    revocation_date timestamp with time zone DEFAULT now() NOT NULL,
    revocation_reason character varying(100) NOT NULL,
    crl_number bigint NOT NULL,
    revoked_by uuid,
    notes text,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT certificate_revocation_list_revocation_reason_check CHECK (((revocation_reason)::text = ANY (ARRAY[('unspecified'::character varying)::text, ('key_compromise'::character varying)::text, ('ca_compromise'::character varying)::text, ('affiliation_changed'::character varying)::text, ('superseded'::character varying)::text, ('cessation_of_operation'::character varying)::text, ('certificate_hold'::character varying)::text, ('remove_from_crl'::character varying)::text, ('privilege_withdrawn'::character varying)::text, ('aa_compromise'::character varying)::text])))
);



--
-- Name: TABLE certificate_revocation_list; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.certificate_revocation_list IS 'List of revoked certificates per tenant CA';


--
-- Name: ciba_auth_requests; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.ciba_auth_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    auth_req_id character varying(255) NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    user_email character varying(255) NOT NULL,
    client_id uuid,
    device_token_id uuid NOT NULL,
    binding_message character varying(255),
    scopes jsonb DEFAULT '[]'::jsonb,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    biometric_verified boolean DEFAULT false,
    expires_at bigint NOT NULL,
    created_at bigint NOT NULL,
    responded_at bigint,
    last_polled_at bigint
);



--
-- Name: TABLE ciba_auth_requests; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.ciba_auth_requests IS 'CIBA authentication requests (push notification based auth)';


--
-- Name: COLUMN ciba_auth_requests.auth_req_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.ciba_auth_requests.auth_req_id IS 'Unique request ID returned to client for polling';


--
-- Name: COLUMN ciba_auth_requests.binding_message; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.ciba_auth_requests.binding_message IS 'Message shown to user in push notification';


--
-- Name: ciba_requests; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.ciba_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    auth_req_id character varying(255) NOT NULL,
    login_hint character varying(255) NOT NULL,
    binding_message character varying(255),
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    expires_at bigint NOT NULL,
    created_at bigint NOT NULL,
    completed_at bigint
);



--
-- Name: TABLE ciba_requests; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.ciba_requests IS 'Tracks CIBA backchannel authentication requests from voice agents';


--
-- Name: COLUMN ciba_requests.auth_req_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.ciba_requests.auth_req_id IS 'Okta authentication request ID returned from /bc/authorize';


--
-- Name: COLUMN ciba_requests.binding_message; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.ciba_requests.binding_message IS 'Message displayed on user device (e.g., "Voice Agent Login")';


--
-- Name: client_groups; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.client_groups (
    client_id uuid NOT NULL,
    group_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);



--
-- Name: client_roles; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.client_roles (
    client_id uuid DEFAULT gen_random_uuid() NOT NULL,
    role_id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);



--
-- Name: clients; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.clients (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    owner_id uuid NOT NULL,
    org_id uuid NOT NULL,
    name text NOT NULL,
    email text,
    status text DEFAULT 'Active'::text,
    tags text[],
    active boolean DEFAULT true,
    last_login timestamp with time zone,
    mfa_enabled boolean DEFAULT false NOT NULL,
    mfa_method text[],
    mfa_default_method text,
    mfa_enrolled_at timestamp with time zone,
    mfa_verified boolean DEFAULT false,
    hydra_client_id text,
    oidc_enabled boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    description text,
    deleted boolean DEFAULT false,
    client_type character varying(255),
    agent_type text,
    spiffe_id text
);



--
-- Name: COLUMN clients.description; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.clients.description IS 'Description of the client application or service';


--
-- Name: credentials; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.credentials (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_id uuid NOT NULL,
    credential_id bytea NOT NULL,
    public_key bytea NOT NULL,
    attestation_type text NOT NULL,
    aa_guid uuid,
    sign_count bigint DEFAULT 0 NOT NULL,
    backup_eligible boolean DEFAULT false,
    backup_state boolean DEFAULT false,
    transports text[],
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    aaguid uuid,
    rp_id character varying(225)
);



--
-- Name: delegation_policies; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.delegation_policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    role_name text NOT NULL,
    agent_type text NOT NULL,
    allowed_permissions jsonb DEFAULT '[]'::jsonb,
    max_ttl_seconds integer DEFAULT 3600 NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    client_id uuid,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT pk_delegation_policies PRIMARY KEY (id),
    CONSTRAINT uq_deleg_policy_tenant_role_agent UNIQUE (tenant_id, role_name, agent_type)
);



--
-- Name: delegation_tokens; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.delegation_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid NOT NULL,
    policy_id uuid,
    token text NOT NULL,
    spiffe_id text NOT NULL,
    permissions jsonb DEFAULT '[]'::jsonb NOT NULL,
    audience jsonb DEFAULT '[]'::jsonb NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    delegated_by uuid NOT NULL,
    ttl_seconds integer NOT NULL,
    status text DEFAULT 'active'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_deleg_token_status CHECK ((status = ANY (ARRAY['active'::text, 'expired'::text, 'revoked'::text])))
);



--
-- Name: device_auth_challenges; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.device_auth_challenges (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid NOT NULL,
    challenge text NOT NULL,
    verified boolean DEFAULT false,
    expires_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    verified_at timestamp without time zone
);



--
-- Name: TABLE device_auth_challenges; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.device_auth_challenges IS 'Authentication challenges for device verification';


--
-- Name: device_codes; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.device_codes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid,
    device_code character varying(128) NOT NULL,
    user_code character varying(16) NOT NULL,
    verification_uri text NOT NULL,
    verification_uri_complete text,
    user_id uuid,
    user_email text,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    scopes jsonb DEFAULT '[]'::jsonb,
    device_info jsonb,
    expires_at bigint NOT NULL,
    last_polled_at bigint,
    authorized_at bigint,
    created_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint,
    updated_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint,
    CONSTRAINT chk_device_codes_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('authorized'::character varying)::text, ('denied'::character varying)::text, ('expired'::character varying)::text, ('consumed'::character varying)::text])))
);



--
-- Name: TABLE device_codes; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.device_codes IS 'OAuth 2.0 Device Authorization Grant (RFC 8628) - stores device flow authorization requests';


--
-- Name: COLUMN device_codes.device_code; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.device_codes.device_code IS 'Long secret code for device polling (128 chars)';


--
-- Name: COLUMN device_codes.user_code; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.device_codes.user_code IS 'Short human-readable code shown to user (8-16 chars, e.g., WDJB-MJHT)';


--
-- Name: COLUMN device_codes.verification_uri; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.device_codes.verification_uri IS 'URL where user activates device (e.g., https://authsec.dev/activate)';


--
-- Name: COLUMN device_codes.status; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.device_codes.status IS 'Authorization state: pending (waiting), authorized (approved), denied (rejected), expired (timeout), consumed (token issued)';


--
-- Name: COLUMN device_codes.scopes; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.device_codes.scopes IS 'JSON array of requested OAuth scopes (e.g., ["openid", "email", "profile"])';


--
-- Name: COLUMN device_codes.expires_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.device_codes.expires_at IS 'Unix epoch timestamp (seconds) when this device code expires';


--
-- Name: COLUMN device_codes.created_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.device_codes.created_at IS 'Unix epoch timestamp (seconds) when created';


--
-- Name: COLUMN device_codes.updated_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.device_codes.updated_at IS 'Unix epoch timestamp (seconds) when last updated';


--
-- Name: device_sessions; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.device_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_id uuid NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    session_token character varying(255),
    ip_address character varying(45),
    active boolean DEFAULT true,
    expires_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    last_activity timestamp without time zone
);



--
-- Name: TABLE device_sessions; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.device_sessions IS 'Active device sessions';


--
-- Name: device_tokens; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.device_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    device_token character varying(500) NOT NULL,
    platform character varying(20) NOT NULL,
    device_name character varying(100),
    device_model character varying(100),
    app_version character varying(20),
    os_version character varying(20),
    is_active boolean DEFAULT true NOT NULL,
    last_used bigint,
    created_at bigint NOT NULL,
    updated_at bigint NOT NULL
);



--
-- Name: TABLE device_tokens; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.device_tokens IS 'FCM/APNS device tokens for push notifications';


--
-- Name: COLUMN device_tokens.device_token; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.device_tokens.device_token IS 'FCM token (Android) or APNS token (iOS)';


--
-- Name: external_service_migrations; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.external_service_migrations (
    id integer NOT NULL,
    tenant_id uuid NOT NULL,
    migration_name character varying(255) NOT NULL,
    applied_at timestamp with time zone DEFAULT now()
);



--
-- Name: external_service_migrations_id_seq; Type: SEQUENCE; Schema: public; Owner: authprod
--

CREATE SEQUENCE public.external_service_migrations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;



--
-- Name: external_service_migrations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: authprod
--

ALTER SEQUENCE public.external_service_migrations_id_seq OWNED BY public.external_service_migrations.id;


--
-- Name: fluent_bit_export_configs; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.fluent_bit_export_configs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id text NOT NULL,
    name text NOT NULL,
    host text NOT NULL,
    alias text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);



--
-- Name: grant_audit; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.grant_audit (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid,
    actor_user_id uuid,
    action text,
    target_type text,
    target_id uuid,
    before jsonb,
    after jsonb,
    created_at timestamp with time zone DEFAULT now()
);



--
-- Name: group_roles; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.group_roles (
    group_id uuid NOT NULL,
    role_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    tenant_id uuid,
    updated_at timestamp with time zone DEFAULT now()
);



--
-- Name: group_scopes; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.group_scopes (
    group_id uuid,
    scope_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    id uuid DEFAULT gen_random_uuid() NOT NULL
);



--
-- Name: groups; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.groups (
    tenant_id uuid,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    id uuid DEFAULT gen_random_uuid() NOT NULL
);



--
-- Name: hydra_client; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_client (
    id character varying(255) NOT NULL,
    client_name text NOT NULL,
    client_secret text NOT NULL,
    scope text NOT NULL,
    owner text NOT NULL,
    policy_uri text NOT NULL,
    tos_uri text NOT NULL,
    client_uri text NOT NULL,
    logo_uri text NOT NULL,
    client_secret_expires_at integer DEFAULT 0 NOT NULL,
    sector_identifier_uri text NOT NULL,
    jwks text NOT NULL,
    jwks_uri text NOT NULL,
    token_endpoint_auth_method character varying(25) DEFAULT ''::character varying NOT NULL,
    request_object_signing_alg character varying(10) DEFAULT ''::character varying NOT NULL,
    userinfo_signed_response_alg character varying(10) DEFAULT ''::character varying NOT NULL,
    subject_type character varying(15) DEFAULT ''::character varying NOT NULL,
    pk_deprecated integer NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    frontchannel_logout_uri text DEFAULT ''::text NOT NULL,
    frontchannel_logout_session_required boolean DEFAULT false NOT NULL,
    backchannel_logout_uri text DEFAULT ''::text NOT NULL,
    backchannel_logout_session_required boolean DEFAULT false NOT NULL,
    metadata text NOT NULL,
    token_endpoint_auth_signing_alg character varying(10) DEFAULT ''::character varying NOT NULL,
    authorization_code_grant_access_token_lifespan bigint,
    authorization_code_grant_id_token_lifespan bigint,
    authorization_code_grant_refresh_token_lifespan bigint,
    client_credentials_grant_access_token_lifespan bigint,
    implicit_grant_access_token_lifespan bigint,
    implicit_grant_id_token_lifespan bigint,
    jwt_bearer_grant_access_token_lifespan bigint,
    password_grant_access_token_lifespan bigint,
    password_grant_refresh_token_lifespan bigint,
    refresh_token_grant_id_token_lifespan bigint,
    refresh_token_grant_access_token_lifespan bigint,
    refresh_token_grant_refresh_token_lifespan bigint,
    pk uuid,
    registration_access_token_signature character varying(128) DEFAULT ''::character varying NOT NULL,
    nid uuid NOT NULL,
    redirect_uris jsonb NOT NULL,
    grant_types jsonb NOT NULL,
    response_types jsonb NOT NULL,
    audience jsonb NOT NULL,
    allowed_cors_origins jsonb NOT NULL,
    contacts jsonb NOT NULL,
    request_uris jsonb NOT NULL,
    post_logout_redirect_uris jsonb DEFAULT '[]'::jsonb NOT NULL,
    access_token_strategy character varying(10) DEFAULT ''::character varying NOT NULL,
    skip_consent boolean DEFAULT false NOT NULL,
    skip_logout_consent boolean
);



--
-- Name: hydra_client_pk_seq; Type: SEQUENCE; Schema: public; Owner: authprod
--

CREATE SEQUENCE public.hydra_client_pk_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;



--
-- Name: hydra_client_pk_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: authprod
--

ALTER SEQUENCE public.hydra_client_pk_seq OWNED BY public.hydra_client.pk_deprecated;


--
-- Name: hydra_jwk; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_jwk (
    sid character varying(255) NOT NULL,
    kid character varying(255) NOT NULL,
    version integer DEFAULT 0 NOT NULL,
    keydata text NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    pk_deprecated integer NOT NULL,
    pk uuid NOT NULL,
    nid uuid NOT NULL
);



--
-- Name: hydra_jwk_pk_seq; Type: SEQUENCE; Schema: public; Owner: authprod
--

CREATE SEQUENCE public.hydra_jwk_pk_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;



--
-- Name: hydra_jwk_pk_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: authprod
--

ALTER SEQUENCE public.hydra_jwk_pk_seq OWNED BY public.hydra_jwk.pk_deprecated;


--
-- Name: hydra_oauth2_access; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_access (
    signature character varying(255) NOT NULL,
    request_id character varying(40) NOT NULL,
    requested_at timestamp without time zone DEFAULT now() NOT NULL,
    client_id character varying(255) NOT NULL,
    scope text NOT NULL,
    granted_scope text NOT NULL,
    form_data text NOT NULL,
    session_data text NOT NULL,
    subject character varying(255) DEFAULT ''::character varying NOT NULL,
    active boolean DEFAULT true NOT NULL,
    requested_audience text DEFAULT ''::text,
    granted_audience text DEFAULT ''::text,
    challenge_id character varying(40),
    nid uuid NOT NULL,
    expires_at timestamp without time zone
);



--
-- Name: hydra_oauth2_authentication_session; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_authentication_session (
    id character varying(40) NOT NULL,
    authenticated_at timestamp without time zone,
    subject character varying(255) NOT NULL,
    remember boolean DEFAULT false NOT NULL,
    nid uuid NOT NULL,
    identity_provider_session_id character varying(40)
);



--
-- Name: hydra_oauth2_code; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_code (
    signature character varying(255) NOT NULL,
    request_id character varying(40) NOT NULL,
    requested_at timestamp without time zone DEFAULT now() NOT NULL,
    client_id character varying(255) NOT NULL,
    scope text NOT NULL,
    granted_scope text NOT NULL,
    form_data text NOT NULL,
    session_data text NOT NULL,
    subject character varying(255) DEFAULT ''::character varying NOT NULL,
    active boolean DEFAULT true NOT NULL,
    requested_audience text DEFAULT ''::text,
    granted_audience text DEFAULT ''::text,
    challenge_id character varying(40),
    nid uuid NOT NULL,
    expires_at timestamp without time zone
);



--
-- Name: hydra_oauth2_flow; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_flow (
    login_challenge character varying(40) NOT NULL,
    login_verifier character varying(40) NOT NULL,
    login_csrf character varying(40) NOT NULL,
    subject character varying(255) NOT NULL,
    request_url text NOT NULL,
    login_skip boolean NOT NULL,
    client_id character varying(255) NOT NULL,
    requested_at timestamp without time zone DEFAULT now() NOT NULL,
    login_initialized_at timestamp without time zone,
    oidc_context jsonb DEFAULT '{}'::jsonb NOT NULL,
    login_session_id character varying(40),
    state integer NOT NULL,
    login_remember boolean DEFAULT false NOT NULL,
    login_remember_for integer NOT NULL,
    login_error text,
    acr text DEFAULT ''::text NOT NULL,
    login_authenticated_at timestamp without time zone,
    login_was_used boolean DEFAULT false NOT NULL,
    forced_subject_identifier character varying(255) DEFAULT ''::character varying NOT NULL,
    context jsonb DEFAULT '{}'::jsonb NOT NULL,
    consent_challenge_id character varying(40),
    consent_skip boolean DEFAULT false NOT NULL,
    consent_verifier character varying(40),
    consent_csrf character varying(40),
    consent_remember boolean DEFAULT false NOT NULL,
    consent_remember_for integer,
    consent_handled_at timestamp without time zone,
    consent_error text,
    session_access_token jsonb DEFAULT '{}'::jsonb NOT NULL,
    session_id_token jsonb DEFAULT '{}'::jsonb NOT NULL,
    consent_was_used boolean DEFAULT false NOT NULL,
    nid uuid NOT NULL,
    requested_scope jsonb NOT NULL,
    requested_at_audience jsonb DEFAULT '[]'::jsonb,
    amr jsonb DEFAULT '[]'::jsonb,
    granted_scope jsonb,
    granted_at_audience jsonb DEFAULT '[]'::jsonb,
    login_extend_session_lifespan boolean DEFAULT false NOT NULL,
    identity_provider_session_id character varying(40),
    CONSTRAINT hydra_oauth2_flow_check CHECK (((state = 128) OR (state = 129) OR (state = 1) OR ((state = 2) AND ((login_remember IS NOT NULL) AND (login_remember_for IS NOT NULL) AND (login_error IS NOT NULL) AND (acr IS NOT NULL) AND (login_was_used IS NOT NULL) AND (context IS NOT NULL) AND (amr IS NOT NULL))) OR ((state = 3) AND ((login_remember IS NOT NULL) AND (login_remember_for IS NOT NULL) AND (login_error IS NOT NULL) AND (acr IS NOT NULL) AND (login_was_used IS NOT NULL) AND (context IS NOT NULL) AND (amr IS NOT NULL))) OR ((state = 4) AND ((login_remember IS NOT NULL) AND (login_remember_for IS NOT NULL) AND (login_error IS NOT NULL) AND (acr IS NOT NULL) AND (login_was_used IS NOT NULL) AND (context IS NOT NULL) AND (amr IS NOT NULL) AND (consent_challenge_id IS NOT NULL) AND (consent_verifier IS NOT NULL) AND (consent_skip IS NOT NULL) AND (consent_csrf IS NOT NULL))) OR ((state = 5) AND ((login_remember IS NOT NULL) AND (login_remember_for IS NOT NULL) AND (login_error IS NOT NULL) AND (acr IS NOT NULL) AND (login_was_used IS NOT NULL) AND (context IS NOT NULL) AND (amr IS NOT NULL) AND (consent_challenge_id IS NOT NULL) AND (consent_verifier IS NOT NULL) AND (consent_skip IS NOT NULL) AND (consent_csrf IS NOT NULL))) OR ((state = 6) AND ((login_remember IS NOT NULL) AND (login_remember_for IS NOT NULL) AND (login_error IS NOT NULL) AND (acr IS NOT NULL) AND (login_was_used IS NOT NULL) AND (context IS NOT NULL) AND (amr IS NOT NULL) AND (consent_challenge_id IS NOT NULL) AND (consent_verifier IS NOT NULL) AND (consent_skip IS NOT NULL) AND (consent_csrf IS NOT NULL) AND (granted_scope IS NOT NULL) AND (consent_remember IS NOT NULL) AND (consent_remember_for IS NOT NULL) AND (consent_error IS NOT NULL) AND (session_access_token IS NOT NULL) AND (session_id_token IS NOT NULL) AND (consent_was_used IS NOT NULL)))))
);



--
-- Name: hydra_oauth2_jti_blacklist; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_jti_blacklist (
    signature character varying(64) NOT NULL,
    expires_at timestamp without time zone DEFAULT now() NOT NULL,
    nid uuid NOT NULL
);



--
-- Name: hydra_oauth2_logout_request; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_logout_request (
    challenge character varying(36) NOT NULL,
    verifier character varying(36) NOT NULL,
    subject character varying(255) NOT NULL,
    sid character varying(36) NOT NULL,
    client_id character varying(255),
    request_url text NOT NULL,
    redir_url text NOT NULL,
    was_used boolean DEFAULT false NOT NULL,
    accepted boolean DEFAULT false NOT NULL,
    rejected boolean DEFAULT false NOT NULL,
    rp_initiated boolean DEFAULT false NOT NULL,
    nid uuid NOT NULL,
    expires_at timestamp without time zone,
    requested_at timestamp without time zone
);



--
-- Name: hydra_oauth2_obfuscated_authentication_session; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_obfuscated_authentication_session (
    subject character varying(255) NOT NULL,
    client_id character varying(255) NOT NULL,
    subject_obfuscated character varying(255) NOT NULL,
    nid uuid NOT NULL
);



--
-- Name: hydra_oauth2_oidc; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_oidc (
    signature character varying(255) NOT NULL,
    request_id character varying(40) NOT NULL,
    requested_at timestamp without time zone DEFAULT now() NOT NULL,
    client_id character varying(255) NOT NULL,
    scope text NOT NULL,
    granted_scope text NOT NULL,
    form_data text NOT NULL,
    session_data text NOT NULL,
    subject character varying(255) DEFAULT ''::character varying NOT NULL,
    active boolean DEFAULT true NOT NULL,
    requested_audience text DEFAULT ''::text,
    granted_audience text DEFAULT ''::text,
    challenge_id character varying(40),
    nid uuid NOT NULL,
    expires_at timestamp without time zone
);



--
-- Name: hydra_oauth2_pkce; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_pkce (
    signature character varying(255) NOT NULL,
    request_id character varying(40) NOT NULL,
    requested_at timestamp without time zone DEFAULT now() NOT NULL,
    client_id character varying(255) NOT NULL,
    scope text NOT NULL,
    granted_scope text NOT NULL,
    form_data text NOT NULL,
    session_data text NOT NULL,
    subject character varying(255) NOT NULL,
    active boolean DEFAULT true NOT NULL,
    requested_audience text DEFAULT ''::text,
    granted_audience text DEFAULT ''::text,
    challenge_id character varying(40),
    nid uuid NOT NULL,
    expires_at timestamp without time zone
);



--
-- Name: hydra_oauth2_refresh; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_refresh (
    signature character varying(255) NOT NULL,
    request_id character varying(40) NOT NULL,
    requested_at timestamp without time zone DEFAULT now() NOT NULL,
    client_id character varying(255) NOT NULL,
    scope text NOT NULL,
    granted_scope text NOT NULL,
    form_data text NOT NULL,
    session_data text NOT NULL,
    subject character varying(255) DEFAULT ''::character varying NOT NULL,
    active boolean DEFAULT true NOT NULL,
    requested_audience text DEFAULT ''::text,
    granted_audience text DEFAULT ''::text,
    challenge_id character varying(40),
    nid uuid NOT NULL,
    expires_at timestamp without time zone,
    first_used_at timestamp without time zone,
    access_token_signature character varying(255) DEFAULT NULL::character varying
);



--
-- Name: hydra_oauth2_trusted_jwt_bearer_issuer; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.hydra_oauth2_trusted_jwt_bearer_issuer (
    id uuid NOT NULL,
    issuer character varying(255) NOT NULL,
    subject character varying(255) NOT NULL,
    scope text NOT NULL,
    key_set character varying(255) NOT NULL,
    key_id character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    expires_at timestamp without time zone DEFAULT now() NOT NULL,
    nid uuid NOT NULL,
    allow_any_subject boolean DEFAULT false NOT NULL
);



--
-- Name: join_tokens; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.join_tokens (
    token character varying(255) NOT NULL,
    agent_id character varying(255),
    expiry timestamp without time zone NOT NULL,
    used boolean DEFAULT false,
    used_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT now()
);



--
-- Name: m2m_agent_attestations; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_agent_attestations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agent_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    workload_id uuid NOT NULL,
    attestation_type character varying(50) NOT NULL,
    attestation_data jsonb NOT NULL,
    status character varying(50) NOT NULL,
    verified_at timestamp with time zone DEFAULT now(),
    verified_by character varying(255),
    failure_reason text,
    error_details jsonb,
    issued_certificate_id uuid,
    ip_address character varying(50),
    user_agent text,
    metadata jsonb,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT valid_attestation_status CHECK (((status)::text = ANY (ARRAY[('success'::character varying)::text, ('failed'::character varying)::text, ('invalid'::character varying)::text, ('expired'::character varying)::text])))
);



--
-- Name: TABLE m2m_agent_attestations; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_agent_attestations IS 'Audit log of all agent attestation attempts';


--
-- Name: m2m_agent_certificate_renewals; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_agent_certificate_renewals (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agent_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    workload_id uuid NOT NULL,
    old_certificate_id uuid,
    old_certificate_expires_at timestamp with time zone,
    new_certificate_id uuid,
    new_certificate_expires_at timestamp with time zone,
    renewal_type character varying(50) NOT NULL,
    renewal_reason character varying(255),
    status character varying(50) DEFAULT 'initiated'::character varying NOT NULL,
    initiated_at timestamp with time zone DEFAULT now(),
    completed_at timestamp with time zone,
    failure_reason text,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    next_retry_at timestamp with time zone,
    metadata jsonb,
    CONSTRAINT valid_renewal_status CHECK (((status)::text = ANY (ARRAY[('initiated'::character varying)::text, ('in-progress'::character varying)::text, ('completed'::character varying)::text, ('failed'::character varying)::text, ('cancelled'::character varying)::text]))),
    CONSTRAINT valid_renewal_type CHECK (((renewal_type)::text = ANY (ARRAY[('automatic'::character varying)::text, ('manual'::character varying)::text, ('forced'::character varying)::text, ('emergency'::character varying)::text])))
);



--
-- Name: TABLE m2m_agent_certificate_renewals; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_agent_certificate_renewals IS 'Tracks automatic certificate renewal operations';


--
-- Name: m2m_agent_deployment_tokens; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_agent_deployment_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    workload_id uuid NOT NULL,
    token character varying(512) NOT NULL,
    token_hash character varying(255) NOT NULL,
    max_uses integer DEFAULT 1,
    current_uses integer DEFAULT 0,
    expires_at timestamp with time zone NOT NULL,
    required_attestation_type character varying(50),
    required_platform character varying(50),
    required_labels jsonb,
    status character varying(50) DEFAULT 'active'::character varying,
    revoked_at timestamp with time zone,
    revoked_reason text,
    used_by_agent_id uuid,
    used_at timestamp with time zone,
    used_from_ip character varying(50),
    created_at timestamp with time zone DEFAULT now(),
    created_by uuid,
    CONSTRAINT valid_token_status CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('used'::character varying)::text, ('expired'::character varying)::text, ('revoked'::character varying)::text]))),
    CONSTRAINT valid_uses CHECK ((current_uses <= max_uses))
);



--
-- Name: TABLE m2m_agent_deployment_tokens; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_agent_deployment_tokens IS 'One-time tokens for agent bootstrap (like SPIRE join tokens)';


--
-- Name: COLUMN m2m_agent_deployment_tokens.token; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.m2m_agent_deployment_tokens.token IS 'One-time use token for agent initial registration';


--
-- Name: m2m_agent_heartbeats; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_agent_heartbeats (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agent_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    "timestamp" timestamp with time zone DEFAULT now(),
    health_status character varying(50) NOT NULL,
    cpu_usage_percent double precision,
    memory_usage_mb integer,
    disk_usage_percent double precision,
    certificate_valid boolean,
    certificate_expires_in_hours integer,
    requests_served_count integer DEFAULT 0,
    errors_count integer DEFAULT 0,
    metadata jsonb,
    CONSTRAINT valid_heartbeat_health_status CHECK (((health_status)::text = ANY (ARRAY[('healthy'::character varying)::text, ('unhealthy'::character varying)::text, ('degraded'::character varying)::text])))
);



--
-- Name: TABLE m2m_agent_heartbeats; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_agent_heartbeats IS 'Heartbeat data from M2M agents for health monitoring';


--
-- Name: m2m_agent_policies; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_agent_policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    workload_id uuid,
    policy_name character varying(255) NOT NULL,
    description text,
    priority integer DEFAULT 100,
    auto_renew_enabled boolean DEFAULT true,
    renew_before_days integer DEFAULT 7,
    max_certificate_ttl_days integer DEFAULT 90,
    min_certificate_ttl_days integer DEFAULT 1,
    attestation_required boolean DEFAULT true,
    attestation_interval_hours integer DEFAULT 24,
    allowed_attestation_types character varying(100)[],
    heartbeat_required boolean DEFAULT true,
    heartbeat_interval_seconds integer DEFAULT 60,
    heartbeat_timeout_seconds integer DEFAULT 300,
    require_mtls boolean DEFAULT true,
    allowed_platforms character varying(50)[],
    blocked_platforms character varying(50)[],
    max_renewal_attempts_per_hour integer DEFAULT 10,
    max_attestation_attempts_per_hour integer DEFAULT 20,
    enabled boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    created_by uuid
);



--
-- Name: TABLE m2m_agent_policies; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_agent_policies IS 'Configuration policies for agent behavior';


--
-- Name: m2m_agents; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_agents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    workload_id uuid NOT NULL,
    agent_id character varying(255) NOT NULL,
    agent_version character varying(50) DEFAULT '1.0.0'::character varying NOT NULL,
    hostname character varying(255),
    platform character varying(50) NOT NULL,
    platform_metadata jsonb,
    attestation_type character varying(50) NOT NULL,
    attestation_data jsonb,
    last_attestation_time timestamp with time zone,
    attestation_status character varying(50) DEFAULT 'pending'::character varying,
    current_certificate_id uuid,
    certificate_fingerprint character varying(255),
    certificate_expires_at timestamp with time zone,
    auto_renew_enabled boolean DEFAULT true,
    renew_before_days integer DEFAULT 7,
    status character varying(50) DEFAULT 'active'::character varying,
    health_status character varying(50) DEFAULT 'unknown'::character varying,
    last_heartbeat timestamp with time zone,
    heartbeat_interval_seconds integer DEFAULT 60,
    api_endpoint character varying(500),
    tls_enabled boolean DEFAULT true,
    labels jsonb,
    annotations jsonb,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    created_by uuid,
    CONSTRAINT valid_attestation_status CHECK (((attestation_status)::text = ANY (ARRAY[('pending'::character varying)::text, ('verified'::character varying)::text, ('failed'::character varying)::text, ('expired'::character varying)::text]))),
    CONSTRAINT valid_attestation_type CHECK (((attestation_type)::text = ANY (ARRAY[('k8s-sa'::character varying)::text, ('aws-iam'::character varying)::text, ('gcp-iam'::character varying)::text, ('azure-msi'::character varying)::text, ('unix-process'::character varying)::text, ('docker'::character varying)::text, ('none'::character varying)::text]))),
    CONSTRAINT valid_health_status CHECK (((health_status)::text = ANY (ARRAY[('healthy'::character varying)::text, ('unhealthy'::character varying)::text, ('unknown'::character varying)::text, ('degraded'::character varying)::text]))),
    CONSTRAINT valid_platform CHECK (((platform)::text = ANY (ARRAY[('kubernetes'::character varying)::text, ('docker'::character varying)::text, ('vm'::character varying)::text, ('bare-metal'::character varying)::text, ('serverless'::character varying)::text]))),
    CONSTRAINT valid_status CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('inactive'::character varying)::text, ('suspended'::character varying)::text, ('terminated'::character varying)::text])))
);



--
-- Name: TABLE m2m_agents; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_agents IS 'M2M agents deployed on workloads for automatic certificate management and attestation';


--
-- Name: COLUMN m2m_agents.agent_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.m2m_agents.agent_id IS 'Unique identifier generated by the agent itself';


--
-- Name: COLUMN m2m_agents.platform_metadata; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.m2m_agents.platform_metadata IS 'Platform-specific data: K8s pod/namespace, VM instance ID, etc.';


--
-- Name: COLUMN m2m_agents.attestation_type; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.m2m_agents.attestation_type IS 'Method used to prove workload identity';


--
-- Name: m2m_attestation_logs; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_attestation_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    credential_id uuid,
    workload_id uuid,
    tenant_id uuid,
    auth_method character varying(50) NOT NULL,
    cert_serial_number character varying(255),
    cert_fingerprint character varying(64),
    cert_common_name character varying(255),
    client_ip inet,
    user_agent text,
    request_id character varying(255),
    success boolean NOT NULL,
    error_code character varying(100),
    error_message text,
    jwt_issued boolean DEFAULT false,
    attestation_data jsonb,
    attested_at timestamp with time zone DEFAULT now(),
    response_time_ms integer,
    CONSTRAINT m2m_attestation_logs_auth_method_check CHECK (((auth_method)::text = ANY (ARRAY[('mtls'::character varying)::text, ('client_credentials'::character varying)::text, ('mtls_jwt'::character varying)::text, ('api_key'::character varying)::text])))
);



--
-- Name: TABLE m2m_attestation_logs; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_attestation_logs IS 'Audit log of all M2M authentication attempts';


--
-- Name: m2m_audit_events; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_audit_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    event_type character varying(100) NOT NULL,
    event_category character varying(50) NOT NULL,
    severity character varying(20) DEFAULT 'info'::character varying NOT NULL,
    actor_type character varying(50) NOT NULL,
    actor_id uuid,
    actor_name character varying(255),
    target_type character varying(50),
    target_id uuid,
    target_name character varying(255),
    description text NOT NULL,
    metadata jsonb,
    request_id character varying(255),
    client_ip inet,
    user_agent text,
    event_time timestamp with time zone DEFAULT now(),
    CONSTRAINT m2m_audit_events_actor_type_check CHECK (((actor_type)::text = ANY (ARRAY[('admin_user'::character varying)::text, ('system'::character varying)::text, ('workload'::character varying)::text, ('api_client'::character varying)::text, ('automation'::character varying)::text]))),
    CONSTRAINT m2m_audit_events_event_category_check CHECK (((event_category)::text = ANY (ARRAY[('ca_management'::character varying)::text, ('workload_management'::character varying)::text, ('certificate_lifecycle'::character varying)::text, ('authentication'::character varying)::text, ('authorization'::character varying)::text, ('system'::character varying)::text]))),
    CONSTRAINT m2m_audit_events_severity_check CHECK (((severity)::text = ANY (ARRAY[('debug'::character varying)::text, ('info'::character varying)::text, ('warning'::character varying)::text, ('error'::character varying)::text, ('critical'::character varying)::text])))
);



--
-- Name: TABLE m2m_audit_events; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_audit_events IS 'High-level audit trail for M2M system operations';


--
-- Name: m2m_certificates; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_certificates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workload_id uuid NOT NULL,
    ca_id uuid NOT NULL,
    common_name character varying(255) NOT NULL,
    subject_alternative_names text[],
    certificate text NOT NULL,
    certificate_chain text,
    private_key_path character varying(500) NOT NULL,
    public_key text NOT NULL,
    serial_number character varying(255) NOT NULL,
    fingerprint_sha256 character varying(64) NOT NULL,
    key_algorithm character varying(50) DEFAULT 'RSA'::character varying NOT NULL,
    key_size integer DEFAULT 2048 NOT NULL,
    signature_algorithm character varying(50) DEFAULT 'SHA256-RSA'::character varying NOT NULL,
    not_before timestamp with time zone NOT NULL,
    not_after timestamp with time zone NOT NULL,
    auto_renew boolean DEFAULT true,
    renewal_threshold_days integer DEFAULT 30,
    last_used_at timestamp with time zone,
    usage_count bigint DEFAULT 0,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    revoked_at timestamp with time zone,
    revocation_reason character varying(255),
    revoked_by uuid,
    replaced_by_cert_id uuid,
    issued_by uuid,
    issued_at timestamp with time zone DEFAULT now(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT m2m_certificates_check CHECK ((not_after > not_before)),
    CONSTRAINT m2m_certificates_renewal_threshold_days_check CHECK (((renewal_threshold_days > 0) AND (renewal_threshold_days < 365))),
    CONSTRAINT m2m_certificates_status_check CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('revoked'::character varying)::text, ('expired'::character varying)::text, ('renewed'::character varying)::text])))
);



--
-- Name: TABLE m2m_certificates; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_certificates IS 'Issued mTLS certificates for workload authentication';


--
-- Name: m2m_credentials; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_credentials (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workload_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    client_id character varying(255) NOT NULL,
    client_secret_hash character varying(255),
    jwt_issuer character varying(500) NOT NULL,
    jwt_audience text[] NOT NULL,
    jwt_subject character varying(500) NOT NULL,
    jwt_ttl_seconds integer DEFAULT 3600 NOT NULL,
    jwt_max_ttl_seconds integer DEFAULT 86400 NOT NULL,
    roles uuid[] DEFAULT '{}'::uuid[],
    scopes text[] DEFAULT '{}'::text[],
    permissions jsonb DEFAULT '[]'::jsonb,
    allowed_grant_types text[] DEFAULT '{client_credentials}'::text[],
    require_mtls boolean DEFAULT true,
    require_jwt boolean DEFAULT false,
    ip_whitelist inet[],
    rate_limit_per_minute integer DEFAULT 1000,
    last_token_issued_at timestamp with time zone,
    total_tokens_issued bigint DEFAULT 0,
    active boolean DEFAULT true,
    expires_at timestamp with time zone,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    last_rotated_at timestamp with time zone,
    CONSTRAINT m2m_credentials_check CHECK (((jwt_ttl_seconds > 0) AND (jwt_ttl_seconds <= jwt_max_ttl_seconds))),
    CONSTRAINT m2m_credentials_rate_limit_per_minute_check CHECK ((rate_limit_per_minute > 0))
);



--
-- Name: TABLE m2m_credentials; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_credentials IS 'Service account credentials with JWT configuration and RBAC';


--
-- Name: m2m_workloads; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.m2m_workloads (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    ca_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    display_name character varying(500),
    description text,
    service_type character varying(100),
    environment character varying(50),
    identity_uri character varying(500) NOT NULL,
    dns_names text[],
    ip_addresses inet[],
    k8s_namespace character varying(255),
    k8s_service_account character varying(255),
    k8s_pod_label_selectors jsonb,
    cloud_instance_metadata jsonb,
    requires_mfa boolean DEFAULT false,
    ip_whitelist inet[],
    allowed_scopes text[],
    active boolean DEFAULT true,
    approved boolean DEFAULT false,
    approved_by uuid,
    approved_at timestamp with time zone,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    last_attested_at timestamp with time zone
);



--
-- Name: TABLE m2m_workloads; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.m2m_workloads IS 'Registered machine identities (services) requiring M2M authentication';


--
-- Name: mfa_methods; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.mfa_methods (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_id uuid NOT NULL,
    user_id uuid,
    method_type character varying(50) NOT NULL,
    display_name character varying(255),
    description character varying(255),
    recommended boolean DEFAULT false,
    method_data jsonb,
    enabled boolean DEFAULT false,
    is_primary boolean DEFAULT false,
    verified boolean DEFAULT false,
    backup_codes text,
    enrolled_at timestamp with time zone,
    expires_at timestamp with time zone,
    last_used_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    method_subtype character varying(255)
);



--
-- Name: COLUMN mfa_methods.backup_codes; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.mfa_methods.backup_codes IS 'Encrypted TOTP backup codes (TEXT type for encrypted string storage)';


--
-- Name: migration_logs; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.migration_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    version bigint NOT NULL,
    name character varying(255) NOT NULL,
    executed_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    success boolean DEFAULT false NOT NULL,
    error_msg text,
    db_type character varying(50) NOT NULL,
    tenant_id character varying(255),
    execution_ms bigint DEFAULT 0 NOT NULL
);



--
-- Name: networks; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.networks (
    id uuid NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);



--
-- Name: oauth_sessions; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.oauth_sessions (
    session_id character varying(36) NOT NULL,
    user_email character varying(255),
    user_info jsonb,
    access_token text,
    refresh_token text,
    authorization_code text,
    token_expires_at bigint,
    created_at bigint NOT NULL,
    last_activity bigint NOT NULL,
    oauth_state character varying(255),
    pkce_verifier text,
    pkce_challenge text,
    is_active boolean DEFAULT true,
    client_identifier character varying(255),
    org_id character varying(255),
    tenant_id character varying(255),
    user_id character varying(255),
    provider character varying(100),
    provider_id character varying(255),
    accessible_tools jsonb
);



--
-- Name: oidc_providers; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.oidc_providers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    provider_name character varying(50) NOT NULL,
    display_name character varying(100) NOT NULL,
    client_id character varying(255) NOT NULL,
    client_secret_vault_path character varying(255) NOT NULL,
    authorization_url character varying(500) NOT NULL,
    token_url character varying(500) NOT NULL,
    userinfo_url character varying(500) NOT NULL,
    scopes text DEFAULT 'openid email profile'::text,
    icon_url character varying(500),
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);



--
-- Name: TABLE oidc_providers; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.oidc_providers IS 'Platform-level OIDC provider configurations (Google, GitHub, Microsoft)';


--
-- Name: COLUMN oidc_providers.provider_name; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.oidc_providers.provider_name IS 'Unique identifier: google, github, microsoft';


--
-- Name: COLUMN oidc_providers.client_secret_vault_path; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.oidc_providers.client_secret_vault_path IS 'HashiCorp Vault path where client_secret is stored';


--
-- Name: oidc_states; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.oidc_states (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    state_token character varying(255) NOT NULL,
    tenant_id uuid,
    tenant_domain character varying(255) NOT NULL,
    provider_name character varying(50) NOT NULL,
    action character varying(20) NOT NULL,
    code_verifier character varying(255),
    redirect_after character varying(500),
    expires_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    request_host character varying(255)
);



--
-- Name: TABLE oidc_states; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.oidc_states IS 'Short-lived OIDC state storage for secure OAuth flow';


--
-- Name: COLUMN oidc_states.state_token; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.oidc_states.state_token IS 'Random token passed to OAuth provider and verified on callback';


--
-- Name: COLUMN oidc_states.code_verifier; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.oidc_states.code_verifier IS 'PKCE code verifier for enhanced security';


--
-- Name: COLUMN oidc_states.request_host; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.oidc_states.request_host IS 'Full domain where OIDC was initiated (e.g., auth.company.com) for callback redirect';


--
-- Name: oidc_user_identities; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.oidc_user_identities (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    user_id uuid NOT NULL,
    provider_name character varying(50) NOT NULL,
    provider_user_id character varying(255) NOT NULL,
    email character varying(255),
    profile_data jsonb,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    last_login_at timestamp with time zone
);



--
-- Name: TABLE oidc_user_identities; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.oidc_user_identities IS 'Links OIDC provider identities to tenant users';


--
-- Name: COLUMN oidc_user_identities.provider_user_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.oidc_user_identities.provider_user_id IS 'Unique user ID from provider (Google sub, GitHub id)';


--
-- Name: otp_entries; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.otp_entries (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email text,
    otp text,
    expires_at timestamp with time zone,
    verified boolean DEFAULT false,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);



--
-- Name: pending_registrations; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.pending_registrations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email text,
    password_hash text,
    first_name text DEFAULT ''::text,
    last_name text DEFAULT ''::text,
    tenant_id uuid,
    project_id uuid,
    client_id uuid,
    expires_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    tenant_domain text
);



--
-- Name: permissions; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.permissions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    resource text NOT NULL,
    action text NOT NULL,
    description text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    full_permission_string text
);



--
-- Name: projects; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.projects (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text NOT NULL,
    description text,
    user_id uuid,
    tenant_id uuid,
    active boolean DEFAULT true,
    client_id uuid,
    id uuid DEFAULT gen_random_uuid() NOT NULL
);



--
-- Name: TABLE projects; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.projects IS 'Projects table with UUID primary key for shared-models compatibility';


--
-- Name: COLUMN projects.id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.projects.id IS 'UUID primary key for shared-models compatibility';


--
-- Name: resources; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.resources (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    name character varying(100) NOT NULL,
    description text,
    type character varying(255) DEFAULT 'generic'::character varying,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone
);



--
-- Name: role_assignment_requests; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.role_assignment_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    requested_at timestamp with time zone DEFAULT now() NOT NULL,
    reviewed_at timestamp with time zone,
    reviewed_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT role_assignment_requests_status_check CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('approved'::character varying)::text, ('rejected'::character varying)::text])))
);



--
-- Name: TABLE role_assignment_requests; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.role_assignment_requests IS 'End-user role assignment requests requiring admin approval';


--
-- Name: COLUMN role_assignment_requests.status; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.role_assignment_requests.status IS 'Request status: pending, approved, rejected';


--
-- Name: role_bindings; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.role_bindings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    user_id uuid,
    service_account_id uuid,
    role_id uuid NOT NULL,
    scope_type text DEFAULT '*'::text,
    scope_id uuid,
    conditions jsonb DEFAULT '{}'::jsonb,
    expires_at timestamp with time zone,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    role_name text,
    username text,
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT check_principal CHECK ((((user_id IS NOT NULL) AND (service_account_id IS NULL)) OR ((user_id IS NULL) AND (service_account_id IS NOT NULL))))
);



--
-- Name: role_permissions; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.role_permissions (
    role_id uuid NOT NULL,
    permission_id uuid NOT NULL
);



--
-- Name: role_scopes; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.role_scopes (
    role_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    scope_name text
);



--
-- Name: roles; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.roles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    is_system boolean DEFAULT false,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);



--
-- Name: saml_callback_states; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.saml_callback_states (
    id text NOT NULL,
    redirect_to text NOT NULL,
    user_email character varying(255),
    user_name character varying(255),
    provider_name character varying(255),
    tenant_id uuid,
    client_id uuid,
    login_challenge text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamp without time zone NOT NULL
);



--
-- Name: saml_providers; Type: TABLE; Schema: public; Owner: authprod
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
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);



--
-- Name: TABLE saml_providers; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.saml_providers IS 'SAML Identity Provider (IdP) configurations - per client within tenant';


--
-- Name: COLUMN saml_providers.tenant_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.tenant_id IS 'Tenant UUID owning this provider';


--
-- Name: COLUMN saml_providers.client_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.client_id IS 'Client (app) UUID within the tenant - enables multi-client isolation';


--
-- Name: COLUMN saml_providers.provider_name; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.provider_name IS 'Provider identifier (MUST be lowercase with unique phrase, e.g., "okta-hr", "azure-finance") - auto-normalized via GORM hooks';


--
-- Name: COLUMN saml_providers.display_name; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.display_name IS 'Human-readable name shown in login UI (e.g., "Okta SAML", "Azure AD")';


--
-- Name: COLUMN saml_providers.entity_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.entity_id IS 'IdP Entity ID - unique identifier for the Identity Provider';


--
-- Name: COLUMN saml_providers.sso_url; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.sso_url IS 'Single Sign-On URL - where to send SAML authentication requests';


--
-- Name: COLUMN saml_providers.slo_url; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.slo_url IS 'Single Logout URL (optional) - for logout requests';


--
-- Name: COLUMN saml_providers.certificate; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.certificate IS 'IdP X.509 certificate in PEM format - for validating SAML response signatures';


--
-- Name: COLUMN saml_providers.metadata_url; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.metadata_url IS 'Optional URL to fetch IdP metadata for auto-configuration';


--
-- Name: COLUMN saml_providers.name_id_format; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.name_id_format IS 'Format for user identifier - usually email address';


--
-- Name: COLUMN saml_providers.attribute_mapping; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.attribute_mapping IS 'JSON mapping of SAML attributes to user fields (e.g., {"email": "email", "first_name": "firstName"})';


--
-- Name: COLUMN saml_providers.is_active; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.is_active IS 'Whether this provider is available for login';


--
-- Name: COLUMN saml_providers.sort_order; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.saml_providers.sort_order IS 'Display order in login UI (lower = higher priority)';


--
-- Name: saml_requests; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.saml_requests (
    id character varying(255) NOT NULL,
    login_challenge text NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid NOT NULL,
    provider_name character varying(255) NOT NULL,
    relay_state text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamp without time zone NOT NULL
);



--
-- Name: saml_sp_certificates; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.saml_sp_certificates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    certificate text NOT NULL,
    private_key text NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamp without time zone NOT NULL
);



--
-- Name: schema_migration; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.schema_migration (
    version character varying(48) NOT NULL,
    version_self integer DEFAULT 0 NOT NULL
);



--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.schema_migrations (
    version integer NOT NULL,
    name text NOT NULL,
    applied_at timestamp with time zone DEFAULT now()
);



--
-- Name: scope_permissions; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.scope_permissions (
    scope_id uuid NOT NULL,
    permission_id uuid NOT NULL
);



--
-- Name: scope_resource_mappings; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.scope_resource_mappings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    scope_name text DEFAULT '*'::text NOT NULL,
    resource_name text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);



--
-- Name: TABLE scope_resource_mappings; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.scope_resource_mappings IS 'Mappings between scopes and resources';


--
-- Name: COLUMN scope_resource_mappings.scope_name; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.scope_resource_mappings.scope_name IS 'Name of the scope (e.g., "read", "write", "*")';


--
-- Name: COLUMN scope_resource_mappings.resource_name; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.scope_resource_mappings.resource_name IS 'Name of the resource this scope applies to';


--
-- Name: scopes; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.scopes (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone
);



--
-- Name: service_accounts; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.service_accounts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);



--
-- Name: services; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.services (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    type text,
    url text,
    description text,
    tags text[],
    resource_id uuid NOT NULL,
    auth_type text NOT NULL,
    auth_config text,
    vault_path text,
    created_by text NOT NULL,
    agent_accessible boolean DEFAULT true,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);



--
-- Name: spiffe_svids; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.spiffe_svids (
    id character varying(255) NOT NULL,
    workload_id character varying(255) NOT NULL,
    spiffe_id character varying(512) NOT NULL,
    x509_svid bytea NOT NULL,
    private_key_encrypted bytea NOT NULL,
    bundle bytea NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    issued_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    source character varying(50) DEFAULT 'spire'::character varying
);



--
-- Name: spiffe_workloads; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.spiffe_workloads (
    id character varying(255) NOT NULL,
    spiffe_id character varying(512) NOT NULL,
    type character varying(50) NOT NULL,
    selectors jsonb DEFAULT '[]'::jsonb NOT NULL,
    attestation_status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    last_attested_at timestamp with time zone,
    metadata jsonb DEFAULT '{}'::jsonb
);



--
-- Name: svid_metadata; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.svid_metadata (
    id character varying(255) NOT NULL,
    workload_id character varying(255) NOT NULL,
    spiffe_id text NOT NULL,
    serial_number character varying(255) NOT NULL,
    issued_at timestamp without time zone NOT NULL,
    expires_at timestamp without time zone NOT NULL,
    issuer text NOT NULL,
    subject text NOT NULL,
    source character varying(50) NOT NULL,
    fetched_by character varying(255),
    fetched_at timestamp without time zone NOT NULL,
    is_expired boolean DEFAULT false,
    renewal_count integer DEFAULT 0,
    last_renewal_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);



--
-- Name: sync_configurations; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.sync_configurations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid NOT NULL,
    project_id uuid NOT NULL,
    sync_type character varying(50) NOT NULL,
    config_name character varying(255) NOT NULL,
    description text,
    is_active boolean DEFAULT true NOT NULL,
    ad_server character varying(500),
    ad_username character varying(500),
    ad_password text,
    ad_base_dn character varying(500),
    ad_filter text,
    ad_use_ssl boolean DEFAULT true,
    ad_skip_verify boolean DEFAULT false,
    entra_tenant_id character varying(500),
    entra_client_id character varying(500),
    entra_client_secret text,
    entra_scopes text,
    entra_skip_verify boolean DEFAULT false,
    last_sync_at timestamp with time zone,
    last_sync_status character varying(50),
    last_sync_error text,
    last_sync_users_count integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    created_by uuid,
    CONSTRAINT sync_configurations_sync_type_check CHECK (((sync_type)::text = ANY (ARRAY[('active_directory'::character varying)::text, ('entra_id'::character varying)::text])))
);



--
-- Name: TABLE sync_configurations; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.sync_configurations IS 'Stores Active Directory and Entra ID sync configurations with encrypted credentials';


--
-- Name: COLUMN sync_configurations.sync_type; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.sync_configurations.sync_type IS 'Type of directory sync: active_directory or entra_id';


--
-- Name: COLUMN sync_configurations.ad_password; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.sync_configurations.ad_password IS 'Encrypted AD service account password';


--
-- Name: COLUMN sync_configurations.entra_client_secret; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.sync_configurations.entra_client_secret IS 'Encrypted Entra ID client secret';


--
-- Name: tenant_ca_policies; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenant_ca_policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    default_cert_ttl_days integer DEFAULT 90 NOT NULL,
    max_cert_ttl_days integer DEFAULT 365 NOT NULL,
    min_key_size integer DEFAULT 2048 NOT NULL,
    allowed_key_algorithms text[] DEFAULT '{RSA,ECDSA}'::text[],
    allowed_signature_algorithms text[] DEFAULT '{SHA256-RSA,SHA384-RSA,SHA256-ECDSA}'::text[],
    enable_auto_renewal boolean DEFAULT true,
    renewal_threshold_days integer DEFAULT 30,
    enable_ocsp boolean DEFAULT true,
    enable_crl boolean DEFAULT true,
    crl_update_frequency_hours integer DEFAULT 24,
    require_admin_approval boolean DEFAULT true,
    require_dual_approval boolean DEFAULT false,
    enforce_mtls boolean DEFAULT true,
    allow_wildcard_dns boolean DEFAULT false,
    max_san_entries integer DEFAULT 10,
    log_all_attestations boolean DEFAULT true,
    alert_on_failed_attestations boolean DEFAULT true,
    failed_attestation_threshold integer DEFAULT 5,
    enable_api_cert_issuance boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    updated_by uuid,
    CONSTRAINT tenant_ca_policies_check CHECK (((default_cert_ttl_days > 0) AND (default_cert_ttl_days <= max_cert_ttl_days))),
    CONSTRAINT tenant_ca_policies_check1 CHECK (((renewal_threshold_days > 0) AND (renewal_threshold_days < default_cert_ttl_days))),
    CONSTRAINT tenant_ca_policies_max_san_entries_check CHECK ((max_san_entries > 0)),
    CONSTRAINT tenant_ca_policies_min_key_size_check CHECK ((min_key_size >= 2048))
);



--
-- Name: TABLE tenant_ca_policies; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.tenant_ca_policies IS 'Tenant-specific policies for CA operations and certificate issuance';


--
-- Name: tenant_certificate_authorities; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenant_certificate_authorities (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    common_name character varying(255) NOT NULL,
    issuer character varying(500) NOT NULL,
    organization character varying(255),
    organizational_unit character varying(255),
    country character varying(2),
    root_certificate text NOT NULL,
    certificate_chain text,
    private_key_path character varying(500) NOT NULL,
    public_key text NOT NULL,
    serial_number character varying(255) NOT NULL,
    key_algorithm character varying(50) DEFAULT 'RSA'::character varying NOT NULL,
    key_size integer DEFAULT 4096 NOT NULL,
    signature_algorithm character varying(50) DEFAULT 'SHA256-RSA'::character varying NOT NULL,
    not_before timestamp with time zone NOT NULL,
    not_after timestamp with time zone NOT NULL,
    crl_distribution_points text[],
    ocsp_server_urls text[],
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    revoked_at timestamp with time zone,
    revocation_reason character varying(255),
    created_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT tenant_certificate_authorities_check CHECK ((not_after > not_before)),
    CONSTRAINT tenant_certificate_authorities_key_algorithm_check CHECK (((key_algorithm)::text = ANY (ARRAY[('RSA'::character varying)::text, ('ECDSA'::character varying)::text, ('Ed25519'::character varying)::text]))),
    CONSTRAINT tenant_certificate_authorities_status_check CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('revoked'::character varying)::text, ('expired'::character varying)::text, ('suspended'::character varying)::text])))
);



--
-- Name: TABLE tenant_certificate_authorities; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.tenant_certificate_authorities IS 'Tenant-specific root CAs for issuing M2M certificates';


--
-- Name: tenant_ciba_auth_requests; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenant_ciba_auth_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    auth_req_id character varying(255) NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    user_email character varying(255) NOT NULL,
    client_id uuid,
    device_token_id uuid NOT NULL,
    binding_message character varying(255),
    scopes jsonb DEFAULT '[]'::jsonb,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    biometric_verified boolean DEFAULT false,
    expires_at bigint NOT NULL,
    created_at bigint NOT NULL,
    responded_at bigint,
    last_polled_at bigint
);



--
-- Name: TABLE tenant_ciba_auth_requests; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.tenant_ciba_auth_requests IS 'Tracks CIBA push notification authentication requests for tenant users';


--
-- Name: COLUMN tenant_ciba_auth_requests.auth_req_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_ciba_auth_requests.auth_req_id IS 'Unique request ID for CIBA authentication flow';


--
-- Name: COLUMN tenant_ciba_auth_requests.device_token_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_ciba_auth_requests.device_token_id IS 'Device to send push notification to';


--
-- Name: COLUMN tenant_ciba_auth_requests.binding_message; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_ciba_auth_requests.binding_message IS 'Message displayed on user device during approval';


--
-- Name: COLUMN tenant_ciba_auth_requests.scopes; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_ciba_auth_requests.scopes IS 'OAuth scopes requested by client';


--
-- Name: COLUMN tenant_ciba_auth_requests.status; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_ciba_auth_requests.status IS 'Request status: pending, approved, denied, expired, consumed';


--
-- Name: COLUMN tenant_ciba_auth_requests.biometric_verified; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_ciba_auth_requests.biometric_verified IS 'Whether user used biometric verification';


--
-- Name: COLUMN tenant_ciba_auth_requests.responded_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_ciba_auth_requests.responded_at IS 'When user approved/denied the request';


--
-- Name: COLUMN tenant_ciba_auth_requests.last_polled_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_ciba_auth_requests.last_polled_at IS 'When client last polled for token';


--
-- Name: tenant_databases; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenant_databases (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id character varying(255) NOT NULL,
    database_name character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    last_migration bigint,
    migration_status character varying(50) DEFAULT 'pending'::character varying,
    created_by uuid
);



--
-- Name: tenant_device_tokens; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenant_device_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    device_token character varying(500) NOT NULL,
    platform character varying(20) NOT NULL,
    device_name character varying(100),
    device_model character varying(100),
    app_version character varying(20),
    os_version character varying(20),
    is_active boolean DEFAULT true NOT NULL,
    last_used bigint,
    created_at bigint NOT NULL,
    updated_at bigint NOT NULL
);



--
-- Name: TABLE tenant_device_tokens; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.tenant_device_tokens IS 'Stores push notification device tokens for tenant users';


--
-- Name: COLUMN tenant_device_tokens.device_token; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_device_tokens.device_token IS 'FCM/APNS push notification token from mobile device';


--
-- Name: COLUMN tenant_device_tokens.platform; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_device_tokens.platform IS 'Mobile platform: ios or android';


--
-- Name: COLUMN tenant_device_tokens.device_name; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_device_tokens.device_name IS 'User-friendly device name (e.g., "John''s iPhone")';


--
-- Name: COLUMN tenant_device_tokens.device_model; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_device_tokens.device_model IS 'Device model (e.g., "iPhone 14 Pro")';


--
-- Name: COLUMN tenant_device_tokens.app_version; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_device_tokens.app_version IS 'AuthSec Mobile app version';


--
-- Name: COLUMN tenant_device_tokens.os_version; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_device_tokens.os_version IS 'Operating system version';


--
-- Name: COLUMN tenant_device_tokens.last_used; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_device_tokens.last_used IS 'Unix timestamp of last authentication using this device';


--
-- Name: tenant_domains; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenant_domains (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    domain character varying(255) NOT NULL,
    kind character varying(32) DEFAULT 'custom'::character varying NOT NULL,
    is_primary boolean DEFAULT false NOT NULL,
    is_verified boolean DEFAULT false NOT NULL,
    verification_method character varying(32) DEFAULT 'dns_txt'::character varying NOT NULL,
    verification_token character varying(255) NOT NULL,
    verification_txt_name character varying(255),
    verification_txt_value character varying(255),
    verified_at timestamp with time zone,
    last_checked_at timestamp with time zone,
    failure_reason text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    created_by uuid,
    updated_by uuid,
    ingress_created boolean DEFAULT false NOT NULL
);



--
-- Name: tenant_hydra_clients; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenant_hydra_clients (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id text NOT NULL,
    hydra_client_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    org_id text NOT NULL,
    tenant_name text NOT NULL,
    hydra_client_secret text NOT NULL,
    client_name text NOT NULL,
    redirect_uris text[] DEFAULT '{}'::text[] NOT NULL,
    scopes text[] DEFAULT ARRAY['openid'::text, 'profile'::text, 'email'::text] NOT NULL,
    client_type text NOT NULL,
    provider_name text,
    is_active boolean DEFAULT true NOT NULL,
    created_by text DEFAULT 'system'::text,
    updated_by text DEFAULT 'system'::text,
    deleted_at timestamp with time zone
);



--
-- Name: TABLE tenant_hydra_clients; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.tenant_hydra_clients IS 'Tracks Hydra client provisioning for each tenant';


--
-- Name: COLUMN tenant_hydra_clients.scopes; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_hydra_clients.scopes IS 'Default Hydra scopes granted to the client';


--
-- Name: COLUMN tenant_hydra_clients.client_type; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_hydra_clients.client_type IS 'main or oidc_provider';


--
-- Name: tenant_mappings; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenant_mappings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);



--
-- Name: tenant_totp_backup_codes; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenant_totp_backup_codes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    code character varying(64) NOT NULL,
    is_used boolean DEFAULT false NOT NULL,
    created_at bigint NOT NULL,
    used_at bigint
);



--
-- Name: TABLE tenant_totp_backup_codes; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.tenant_totp_backup_codes IS 'Stores backup recovery codes for tenant user TOTP devices';


--
-- Name: COLUMN tenant_totp_backup_codes.code; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_totp_backup_codes.code IS 'SHA-1 hash of backup recovery code';


--
-- Name: COLUMN tenant_totp_backup_codes.is_used; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_totp_backup_codes.is_used IS 'Whether backup code has been used';


--
-- Name: COLUMN tenant_totp_backup_codes.used_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_totp_backup_codes.used_at IS 'When backup code was used';


--
-- Name: tenant_totp_secrets; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenant_totp_secrets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    secret character varying(64) NOT NULL,
    device_name character varying(100),
    device_type character varying(50) DEFAULT 'generic'::character varying,
    last_used bigint,
    is_active boolean DEFAULT true NOT NULL,
    is_primary boolean DEFAULT false,
    created_at bigint NOT NULL,
    updated_at bigint NOT NULL
);



--
-- Name: TABLE tenant_totp_secrets; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.tenant_totp_secrets IS 'Stores TOTP authenticator secrets for tenant users';


--
-- Name: COLUMN tenant_totp_secrets.secret; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_totp_secrets.secret IS 'Base32 encoded TOTP secret (never exposed in API responses)';


--
-- Name: COLUMN tenant_totp_secrets.device_name; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_totp_secrets.device_name IS 'User-friendly device name for TOTP authenticator';


--
-- Name: COLUMN tenant_totp_secrets.device_type; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_totp_secrets.device_type IS 'Type of TOTP authenticator app';


--
-- Name: COLUMN tenant_totp_secrets.last_used; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_totp_secrets.last_used IS 'Unix timestamp of last TOTP verification';


--
-- Name: COLUMN tenant_totp_secrets.is_primary; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.tenant_totp_secrets.is_primary IS 'Primary device for TOTP login';


--
-- Name: tenants; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.tenants (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    tenant_db text,
    email text NOT NULL,
    username text,
    password_hash text,
    provider text DEFAULT 'local'::text,
    provider_id text,
    avatar text,
    name text,
    source text,
    status text,
    last_login timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    tenant_domain text NOT NULL,
    vault_mount character varying(255),
    ca_cert text,
    migration_status character varying(50) DEFAULT 'pending'::character varying,
    last_migration integer
);



--
-- Name: totp_backup_codes; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.totp_backup_codes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    code character varying(64) NOT NULL,
    is_used boolean DEFAULT false NOT NULL,
    created_at bigint NOT NULL,
    used_at bigint
);



--
-- Name: TABLE totp_backup_codes; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.totp_backup_codes IS 'Stores recovery codes for TOTP 2FA';


--
-- Name: COLUMN totp_backup_codes.code; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.totp_backup_codes.code IS 'SHA1-hashed recovery code (never exposed in plain)';


--
-- Name: totp_secrets; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.totp_secrets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    secret character varying(64) NOT NULL,
    device_name character varying(100) NOT NULL,
    device_type character varying(50) DEFAULT 'generic'::character varying,
    last_used bigint,
    is_active boolean DEFAULT true NOT NULL,
    is_primary boolean DEFAULT false NOT NULL,
    created_at bigint NOT NULL,
    updated_at bigint NOT NULL
);



--
-- Name: TABLE totp_secrets; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.totp_secrets IS 'Stores TOTP authenticator devices registered by users for 2FA';


--
-- Name: COLUMN totp_secrets.secret; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.totp_secrets.secret IS 'Base32-encoded TOTP secret (never exposed in API responses)';


--
-- Name: trust_bundle_cas; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.trust_bundle_cas (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    bundle_id uuid NOT NULL,
    ca_id uuid NOT NULL,
    ca_type character varying(50) DEFAULT 'root'::character varying NOT NULL,
    is_primary boolean DEFAULT false,
    priority integer DEFAULT 100,
    trust_level character varying(50) DEFAULT 'full'::character varying NOT NULL,
    allowed_purposes text[] DEFAULT '{client_auth,server_auth}'::text[],
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    added_at timestamp with time zone DEFAULT now(),
    added_by uuid,
    updated_at timestamp with time zone DEFAULT now(),
    removed_at timestamp with time zone,
    removed_by uuid
);



--
-- Name: TABLE trust_bundle_cas; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.trust_bundle_cas IS 'Many-to-many relationship between trust bundles and certificate authorities';


--
-- Name: trust_bundles; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.trust_bundles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    trust_domain character varying(255) NOT NULL,
    bundle_name character varying(255) NOT NULL,
    description text,
    bundle_version character varying(50) NOT NULL,
    sequence_number bigint NOT NULL,
    is_active boolean DEFAULT true,
    refresh_interval_seconds integer DEFAULT 3600 NOT NULL,
    max_age_seconds integer DEFAULT 7200 NOT NULL,
    valid_from timestamp with time zone DEFAULT now() NOT NULL,
    valid_until timestamp with time zone,
    bundle_format character varying(50) DEFAULT 'spiffe'::character varying NOT NULL,
    checksum_sha256 character varying(64),
    signature text,
    signed_by_ca_id uuid,
    distribution_endpoints text[] DEFAULT '{}'::text[],
    auto_distribute boolean DEFAULT true,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    activated_at timestamp with time zone,
    activated_by uuid
);



--
-- Name: TABLE trust_bundles; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.trust_bundles IS 'Trust bundles containing collections of trusted CAs for SPIFFE-based M2M authentication';


--
-- Name: user_auth_preferences; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.user_auth_preferences (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    preferred_method character varying(50) DEFAULT 'device_code'::character varying NOT NULL,
    okta_verify_enrolled boolean DEFAULT false NOT NULL,
    okta_user_id character varying(255),
    created_at bigint NOT NULL,
    updated_at bigint NOT NULL
);



--
-- Name: TABLE user_auth_preferences; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.user_auth_preferences IS 'Stores user preferences for authentication methods';


--
-- Name: COLUMN user_auth_preferences.preferred_method; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.user_auth_preferences.preferred_method IS 'User preferred auth: ciba (push), device_code, or totp';


--
-- Name: user_groups; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.user_groups (
    user_id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    group_id uuid NOT NULL
);



--
-- Name: user_resources; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.user_resources (
    user_id uuid DEFAULT gen_random_uuid() NOT NULL,
    resource_id uuid DEFAULT gen_random_uuid() NOT NULL
);



--
-- Name: user_roles; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.user_roles (
    user_id uuid DEFAULT gen_random_uuid() NOT NULL,
    role_id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);



--
-- Name: TABLE user_roles; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.user_roles IS 'User-role assignments (no FK constraint to allow role assignment during registration)';


--
-- Name: user_scopes; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.user_scopes (
    user_id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    scope_name text
);



--
-- Name: users; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid,
    name text,
    username text,
    email text NOT NULL,
    password_hash text,
    tenant_domain text NOT NULL,
    provider text NOT NULL,
    provider_id text,
    provider_data jsonb DEFAULT '{}'::jsonb,
    avatar_url text,
    active boolean DEFAULT true,
    mfa_enabled boolean DEFAULT false NOT NULL,
    mfa_method text[],
    mfa_default_method text,
    mfa_enrolled_at timestamp with time zone,
    mfa_verified boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    last_login timestamp with time zone,
    external_id text,
    sync_source text,
    last_sync_at timestamp with time zone,
    is_synced_user boolean DEFAULT false,
    deleted_at timestamp with time zone,
    role_name character varying(255),
    temporary_password boolean DEFAULT false,
    password_change_required boolean DEFAULT false,
    invited_by uuid,
    invited_at timestamp with time zone,
    temporary_password_expires_at timestamp with time zone,
    is_primary_admin boolean DEFAULT false,
    is_voice_enrolled boolean DEFAULT false,
    voice_enrolled boolean DEFAULT false,
    voice_enrollment_date timestamp without time zone,
    voice_last_verified timestamp without time zone,
    failed_login_attempts integer DEFAULT 0,
    account_locked_at timestamp with time zone,
    password_reset_required boolean DEFAULT false
);



--
-- Name: COLUMN users.provider_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.provider_id IS 'OAuth/SSO provider ID (nullable for custom registration)';


--
-- Name: COLUMN users.mfa_method; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.mfa_method IS 'Supported values: otp, totp, webauthn, voice (array)';


--
-- Name: COLUMN users.temporary_password; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.temporary_password IS 'Indicates if user is using a temporary password from admin invite';


--
-- Name: COLUMN users.password_change_required; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.password_change_required IS 'Forces user to change password on next login';


--
-- Name: COLUMN users.invited_by; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.invited_by IS 'UUID of admin who invited this user';


--
-- Name: COLUMN users.invited_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.invited_at IS 'Timestamp when user was invited';


--
-- Name: COLUMN users.temporary_password_expires_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.temporary_password_expires_at IS 'Timestamp when temporary password expires';


--
-- Name: COLUMN users.is_primary_admin; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.is_primary_admin IS 'Indicates if this user is the primary admin who cannot be deleted. Each tenant should have at least one primary admin.';


--
-- Name: COLUMN users.failed_login_attempts; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.failed_login_attempts IS 'Number of consecutive failed login attempts';


--
-- Name: COLUMN users.account_locked_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.account_locked_at IS 'Timestamp when account was locked due to too many failed attempts';


--
-- Name: COLUMN users.password_reset_required; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.users.password_reset_required IS 'Flag indicating user must reset password before next login';


--
-- Name: v_agent_attestation_summary; Type: VIEW; Schema: public; Owner: authprod
--

CREATE VIEW public.v_agent_attestation_summary AS
 SELECT a.id AS agent_id,
    a.agent_id AS agent_identifier,
    a.tenant_id,
    a.workload_id,
    w.name AS workload_name,
    a.attestation_type,
    a.attestation_status,
    a.last_attestation_time,
    count(att.id) AS total_attestations,
    sum(
        CASE
            WHEN ((att.status)::text = 'success'::text) THEN 1
            ELSE 0
        END) AS successful_attestations,
    sum(
        CASE
            WHEN ((att.status)::text = 'failed'::text) THEN 1
            ELSE 0
        END) AS failed_attestations
   FROM ((public.m2m_agents a
     JOIN public.m2m_workloads w ON ((a.workload_id = w.id)))
     LEFT JOIN public.m2m_agent_attestations att ON ((a.id = att.agent_id)))
  GROUP BY a.id, a.agent_id, a.tenant_id, a.workload_id, w.name, a.attestation_type, a.attestation_status, a.last_attestation_time;



--
-- Name: v_agents_requiring_renewal; Type: VIEW; Schema: public; Owner: authprod
--

CREATE VIEW public.v_agents_requiring_renewal AS
 SELECT a.id,
    a.agent_id,
    a.tenant_id,
    a.workload_id,
    w.name AS workload_name,
    a.hostname,
    a.platform,
    a.certificate_expires_at,
    a.renew_before_days,
    (a.certificate_expires_at - ((a.renew_before_days || ' days'::text))::interval) AS renewal_threshold,
    a.last_heartbeat,
    a.health_status,
    a.auto_renew_enabled
   FROM (public.m2m_agents a
     JOIN public.m2m_workloads w ON ((a.workload_id = w.id)))
  WHERE (((a.status)::text = 'active'::text) AND (a.auto_renew_enabled = true) AND (a.certificate_expires_at IS NOT NULL) AND (a.certificate_expires_at <= (now() + ((a.renew_before_days || ' days'::text))::interval)));



--
-- Name: v_unhealthy_agents; Type: VIEW; Schema: public; Owner: authprod
--

CREATE VIEW public.v_unhealthy_agents AS
 SELECT a.id,
    a.agent_id,
    a.tenant_id,
    a.workload_id,
    w.name AS workload_name,
    a.hostname,
    a.platform,
    a.last_heartbeat,
    a.heartbeat_interval_seconds,
    (EXTRACT(epoch FROM (now() - a.last_heartbeat)))::integer AS seconds_since_heartbeat,
    a.health_status
   FROM (public.m2m_agents a
     JOIN public.m2m_workloads w ON ((a.workload_id = w.id)))
  WHERE (((a.status)::text = 'active'::text) AND ((a.last_heartbeat IS NULL) OR (a.last_heartbeat < (now() - (((a.heartbeat_interval_seconds * 3) || ' seconds'::text))::interval))));



--
-- Name: voice_active_sessions; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.voice_active_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid,
    user_id uuid NOT NULL,
    user_email text NOT NULL,
    session_id character varying(128) NOT NULL,
    voice_platform character varying(50),
    voice_user_id text,
    device_info jsonb DEFAULT '{}'::jsonb,
    device_name text,
    access_token_hash character varying(64),
    refresh_token_hash character varying(64),
    is_active boolean DEFAULT true,
    revoked_reason character varying(100),
    login_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint,
    last_activity_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint,
    expires_at bigint NOT NULL,
    revoked_at bigint,
    created_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint,
    updated_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint
);



--
-- Name: TABLE voice_active_sessions; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.voice_active_sessions IS 'Tracks active JWT sessions from voice authentication for device management and logout';


--
-- Name: COLUMN voice_active_sessions.session_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_active_sessions.session_id IS 'Unique session identifier, used as jti claim in JWT';


--
-- Name: COLUMN voice_active_sessions.access_token_hash; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_active_sessions.access_token_hash IS 'SHA256 hash of access token for revocation checking';


--
-- Name: COLUMN voice_active_sessions.is_active; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_active_sessions.is_active IS 'Whether session is active (false = logged out/revoked)';


--
-- Name: voice_auth_logs; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.voice_auth_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid,
    tenant_id uuid NOT NULL,
    client_id uuid,
    event_type character varying(50) NOT NULL,
    confidence_score double precision,
    ip_address character varying(45),
    user_agent text,
    success boolean DEFAULT false,
    error_message text,
    created_at timestamp without time zone DEFAULT now()
);



--
-- Name: TABLE voice_auth_logs; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.voice_auth_logs IS 'Audit log for voice authentication events';


--
-- Name: voice_auth_sessions; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.voice_auth_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid,
    email character varying(255) NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid,
    challenge text NOT NULL,
    challenge_type character varying(50) DEFAULT 'enrollment'::character varying,
    audio_received boolean DEFAULT false,
    verified boolean DEFAULT false,
    verification_score double precision,
    expires_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);



--
-- Name: TABLE voice_auth_sessions; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.voice_auth_sessions IS 'Tracks voice enrollment and verification sessions';


--
-- Name: COLUMN voice_auth_sessions.challenge; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_auth_sessions.challenge IS 'Random phrase user must speak for enrollment/verification';


--
-- Name: voice_identity_links; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.voice_identity_links (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    voice_platform character varying(50) NOT NULL,
    voice_user_id text NOT NULL,
    voice_user_name text,
    user_id uuid NOT NULL,
    user_email text NOT NULL,
    is_active boolean DEFAULT true,
    link_method character varying(50),
    last_used_at bigint,
    linked_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint,
    created_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint,
    updated_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint
);



--
-- Name: TABLE voice_identity_links; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.voice_identity_links IS 'Permanent links between voice assistant accounts and user accounts for passwordless auth';


--
-- Name: COLUMN voice_identity_links.voice_user_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_identity_links.voice_user_id IS 'Platform-specific user ID (e.g., Alexa user amzn1.account.xxx)';


--
-- Name: COLUMN voice_identity_links.is_active; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_identity_links.is_active IS 'Whether link is active (user can deactivate)';


--
-- Name: COLUMN voice_identity_links.last_used_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_identity_links.last_used_at IS 'Unix epoch timestamp (seconds) when last used for authentication';


--
-- Name: COLUMN voice_identity_links.linked_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_identity_links.linked_at IS 'Unix epoch timestamp (seconds) when link was created';


--
-- Name: voice_profiles; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.voice_profiles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid,
    voice_embedding bytea NOT NULL,
    enrollment_samples integer DEFAULT 0,
    quality_score double precision DEFAULT 0.0,
    active boolean DEFAULT true,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);



--
-- Name: TABLE voice_profiles; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.voice_profiles IS 'Stores user voice biometric embeddings for authentication';


--
-- Name: COLUMN voice_profiles.voice_embedding; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_profiles.voice_embedding IS 'Encrypted voice feature vector (speaker embedding)';


--
-- Name: voice_sessions; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.voice_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    client_id uuid,
    session_token character varying(128) NOT NULL,
    voice_otp character varying(10) NOT NULL,
    otp_attempts integer DEFAULT 0,
    voice_platform character varying(50),
    voice_user_id text,
    device_info jsonb,
    user_id uuid,
    user_email text,
    status character varying(20) DEFAULT 'initiated'::character varying NOT NULL,
    linked_device_code character varying(128),
    scopes jsonb DEFAULT '[]'::jsonb,
    pending_approval boolean DEFAULT false,
    approval_status character varying(20) DEFAULT NULL::character varying,
    approved_at timestamp without time zone,
    approved_by uuid,
    expires_at bigint NOT NULL,
    verified_at bigint,
    created_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint,
    updated_at bigint DEFAULT (EXTRACT(epoch FROM now()))::bigint,
    CONSTRAINT chk_voice_otp_attempts CHECK (((otp_attempts >= 0) AND (otp_attempts <= 5))),
    CONSTRAINT chk_voice_sessions_status CHECK (((status)::text = ANY (ARRAY[('initiated'::character varying)::text, ('verified'::character varying)::text, ('expired'::character varying)::text, ('failed'::character varying)::text])))
);



--
-- Name: TABLE voice_sessions; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.voice_sessions IS 'Voice authentication sessions for voice assistant integration (Alexa, Google, Siri)';


--
-- Name: COLUMN voice_sessions.session_token; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_sessions.session_token IS 'Secret token identifying this voice session';


--
-- Name: COLUMN voice_sessions.voice_otp; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_sessions.voice_otp IS 'Numeric code spoken to user for verification (e.g., 8532)';


--
-- Name: COLUMN voice_sessions.status; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_sessions.status IS 'Session state: initiated, verified, expired, failed';


--
-- Name: COLUMN voice_sessions.linked_device_code; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_sessions.linked_device_code IS 'Optional link to device authorization flow';


--
-- Name: COLUMN voice_sessions.pending_approval; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_sessions.pending_approval IS 'Whether this voice auth request is waiting for user approval';


--
-- Name: COLUMN voice_sessions.approval_status; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_sessions.approval_status IS 'Approval status: pending, approved, denied';


--
-- Name: COLUMN voice_sessions.expires_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_sessions.expires_at IS 'Unix epoch timestamp (seconds) when this session expires';


--
-- Name: COLUMN voice_sessions.created_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voice_sessions.created_at IS 'Unix epoch timestamp (seconds) when created';


--
-- Name: voiceprints; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.voiceprints (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    user_id uuid NOT NULL,
    client_id uuid NOT NULL,
    voice_sample bytea,
    voice_features text NOT NULL,
    audio_format character varying(10) NOT NULL,
    sample_duration double precision,
    enrolled_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    last_verified_at timestamp without time zone,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);



--
-- Name: TABLE voiceprints; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.voiceprints IS 'Stores voice biometric data (voiceprints) for user authentication';


--
-- Name: COLUMN voiceprints.voice_sample; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voiceprints.voice_sample IS 'Raw audio data in binary format';


--
-- Name: COLUMN voiceprints.voice_features; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voiceprints.voice_features IS 'JSON string containing extracted voice features (MFCC, pitch, formants, etc.)';


--
-- Name: COLUMN voiceprints.audio_format; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voiceprints.audio_format IS 'Audio format of the sample (wav, mp3, ogg, etc.)';


--
-- Name: COLUMN voiceprints.sample_duration; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voiceprints.sample_duration IS 'Duration of the audio sample in seconds';


--
-- Name: COLUMN voiceprints.enrolled_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voiceprints.enrolled_at IS 'Timestamp when the voiceprint was enrolled';


--
-- Name: COLUMN voiceprints.last_verified_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voiceprints.last_verified_at IS 'Timestamp of the last successful verification';


--
-- Name: COLUMN voiceprints.active; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.voiceprints.active IS 'Whether this voiceprint is currently active for authentication';


--
-- Name: webauthn_credentials; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.webauthn_credentials (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    credential_id text NOT NULL,
    public_key text NOT NULL,
    sign_count bigint DEFAULT 0,
    aaguid text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    attestation_type text,
    transports text[],
    backup_eligible boolean DEFAULT false,
    backup_state boolean DEFAULT false,
    user_present boolean DEFAULT false,
    user_verified boolean DEFAULT false
);



--
-- Name: webauthn_sessions; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.webauthn_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    session_key character varying(255) NOT NULL,
    challenge text NOT NULL,
    user_id bytea NOT NULL,
    user_verification character varying(50),
    extensions bytea,
    created_at timestamp with time zone DEFAULT now(),
    expires_at timestamp with time zone NOT NULL,
    cred_params bytea,
    allowed_credential_ids bytea
);



--
-- Name: TABLE webauthn_sessions; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON TABLE public.webauthn_sessions IS 'Stores WebAuthn session data for registration and authentication ceremonies';


--
-- Name: COLUMN webauthn_sessions.session_key; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.webauthn_sessions.session_key IS 'Unique session identifier in format: operation:email:tenant_id';


--
-- Name: COLUMN webauthn_sessions.challenge; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.webauthn_sessions.challenge IS 'Base64-encoded WebAuthn challenge';


--
-- Name: COLUMN webauthn_sessions.user_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.webauthn_sessions.user_id IS 'Binary user identifier for the WebAuthn ceremony';


--
-- Name: COLUMN webauthn_sessions.user_verification; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.webauthn_sessions.user_verification IS 'Required user verification level (required, preferred, discouraged)';


--
-- Name: COLUMN webauthn_sessions.extensions; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.webauthn_sessions.extensions IS 'JSON-encoded WebAuthn extensions data';


--
-- Name: COLUMN webauthn_sessions.expires_at; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.webauthn_sessions.expires_at IS 'Session expiration timestamp (typically 10 minutes from creation)';


--
-- Name: COLUMN webauthn_sessions.cred_params; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.webauthn_sessions.cred_params IS 'JSON-encoded credential parameters for registration';


--
-- Name: COLUMN webauthn_sessions.allowed_credential_ids; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.webauthn_sessions.allowed_credential_ids IS 'JSON-encoded list of allowed credential IDs for authentication';


--
-- Name: workload_entries; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.workload_entries (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    spiffe_id character varying(512) NOT NULL,
    parent_id character varying(512) NOT NULL,
    selectors jsonb NOT NULL,
    ttl integer DEFAULT 3600,
    admin boolean DEFAULT false,
    federates_with text[],
    downstream boolean DEFAULT false,
    dns_names text[],
    spire_entry_id character varying(255),
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    tenant_id uuid NOT NULL
);



--
-- Name: COLUMN workload_entries.tenant_id; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON COLUMN public.workload_entries.tenant_id IS 'Tenant that owns this workload entry (added for multi-tenancy)';


--
-- Name: workloads; Type: TABLE; Schema: public; Owner: authprod
--

CREATE TABLE public.workloads (
    id character varying(255) NOT NULL,
    tenant_id character varying(255) NOT NULL,
    spiffe_id character varying(500) NOT NULL,
    selectors jsonb NOT NULL,
    vault_role character varying(255) NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    attestation_type character varying(50) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);



--
-- Name: audit_events id; Type: DEFAULT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.audit_events ALTER COLUMN id SET DEFAULT nextval('public.audit_events_id_seq'::regclass);


--
-- Name: external_service_migrations id; Type: DEFAULT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.external_service_migrations ALTER COLUMN id SET DEFAULT nextval('public.external_service_migrations_id_seq'::regclass);


--
-- Name: hydra_client pk_deprecated; Type: DEFAULT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_client ALTER COLUMN pk_deprecated SET DEFAULT nextval('public.hydra_client_pk_seq'::regclass);


--
-- Name: hydra_jwk pk_deprecated; Type: DEFAULT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_jwk ALTER COLUMN pk_deprecated SET DEFAULT nextval('public.hydra_jwk_pk_seq'::regclass);


--
-- Name: api_scope_permissions api_scope_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.api_scope_permissions
    ADD CONSTRAINT api_scope_permissions_pkey PRIMARY KEY (scope_id, permission_id);


--
-- Name: api_scopes api_scopes_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.api_scopes
    ADD CONSTRAINT api_scopes_pkey PRIMARY KEY (id);


--
-- Name: attestation_policies attestation_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.attestation_policies
    ADD CONSTRAINT attestation_policies_pkey PRIMARY KEY (id);


--
-- Name: audit_events audit_events_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.audit_events
    ADD CONSTRAINT audit_events_pkey PRIMARY KEY (id);


--
-- Name: audit_logs audit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);


--
-- Name: auth_agents auth_agents_api_key_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.auth_agents
    ADD CONSTRAINT auth_agents_api_key_key UNIQUE (api_key);


--
-- Name: auth_agents auth_agents_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.auth_agents
    ADD CONSTRAINT auth_agents_pkey PRIMARY KEY (id);


--
-- Name: certificate_revocation_list certificate_revocation_list_ca_id_serial_number_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.certificate_revocation_list
    ADD CONSTRAINT certificate_revocation_list_ca_id_serial_number_key UNIQUE (ca_id, serial_number);


--
-- Name: certificate_revocation_list certificate_revocation_list_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.certificate_revocation_list
    ADD CONSTRAINT certificate_revocation_list_pkey PRIMARY KEY (id);


--
-- Name: ciba_auth_requests ciba_auth_requests_auth_req_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.ciba_auth_requests
    ADD CONSTRAINT ciba_auth_requests_auth_req_id_key UNIQUE (auth_req_id);


--
-- Name: ciba_auth_requests ciba_auth_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.ciba_auth_requests
    ADD CONSTRAINT ciba_auth_requests_pkey PRIMARY KEY (id);


--
-- Name: ciba_requests ciba_requests_auth_req_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.ciba_requests
    ADD CONSTRAINT ciba_requests_auth_req_id_key UNIQUE (auth_req_id);


--
-- Name: clients clients_hydra_client_id_key; Type: INDEX; Schema: public; Owner: -
-- Partial unique index: allows empty/NULL hydra_client_id (e.g. AI agent clients)
--

CREATE UNIQUE INDEX clients_hydra_client_id_key ON public.clients (hydra_client_id) WHERE hydra_client_id != '';


--
-- Name: clients clients_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.clients
    ADD CONSTRAINT clients_pkey PRIMARY KEY (id);


--
-- Name: credentials credentials_credential_id_unique; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.credentials
    ADD CONSTRAINT credentials_credential_id_unique UNIQUE (credential_id);


--
-- Name: credentials credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.credentials
    ADD CONSTRAINT credentials_pkey PRIMARY KEY (id);


--
-- Name: delegation_tokens delegation_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.delegation_tokens
    ADD CONSTRAINT delegation_tokens_pkey PRIMARY KEY (id);


--
-- Name: device_auth_challenges device_auth_challenges_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_auth_challenges
    ADD CONSTRAINT device_auth_challenges_pkey PRIMARY KEY (id);


--
-- Name: device_codes device_codes_device_code_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_codes
    ADD CONSTRAINT device_codes_device_code_key UNIQUE (device_code);


--
-- Name: device_codes device_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_codes
    ADD CONSTRAINT device_codes_pkey PRIMARY KEY (id);


--
-- Name: device_codes device_codes_user_code_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_codes
    ADD CONSTRAINT device_codes_user_code_key UNIQUE (user_code);


--
-- Name: device_sessions device_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_pkey PRIMARY KEY (id);


--
-- Name: device_sessions device_sessions_session_token_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_session_token_key UNIQUE (session_token);


--
-- Name: device_tokens device_tokens_device_token_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_tokens
    ADD CONSTRAINT device_tokens_device_token_key UNIQUE (device_token);


--
-- Name: device_tokens device_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_tokens
    ADD CONSTRAINT device_tokens_pkey PRIMARY KEY (id);


--
-- Name: external_service_migrations external_service_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.external_service_migrations
    ADD CONSTRAINT external_service_migrations_pkey PRIMARY KEY (id);


--
-- Name: external_service_migrations external_service_migrations_tenant_id_migration_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.external_service_migrations
    ADD CONSTRAINT external_service_migrations_tenant_id_migration_name_key UNIQUE (tenant_id, migration_name);


--
-- Name: tenant_device_tokens fk_tenant_device_token; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_device_tokens
    ADD CONSTRAINT fk_tenant_device_token UNIQUE (device_token, tenant_id);


--
-- Name: fluent_bit_export_configs fluent_bit_export_configs_alias_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.fluent_bit_export_configs
    ADD CONSTRAINT fluent_bit_export_configs_alias_key UNIQUE (alias);


--
-- Name: fluent_bit_export_configs fluent_bit_export_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.fluent_bit_export_configs
    ADD CONSTRAINT fluent_bit_export_configs_pkey PRIMARY KEY (id);


--
-- Name: grant_audit grant_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.grant_audit
    ADD CONSTRAINT grant_audit_pkey PRIMARY KEY (id);


--
-- Name: group_roles group_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.group_roles
    ADD CONSTRAINT group_roles_pkey PRIMARY KEY (group_id, role_id);


--
-- Name: group_scopes group_scopes_group_id_scope_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.group_scopes
    ADD CONSTRAINT group_scopes_group_id_scope_id_key UNIQUE (group_id, scope_id);


--
-- Name: group_scopes group_scopes_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.group_scopes
    ADD CONSTRAINT group_scopes_pkey PRIMARY KEY (id);


--
-- Name: groups groups_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);


--
-- Name: hydra_client hydra_client_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_client
    ADD CONSTRAINT hydra_client_pkey PRIMARY KEY (id, nid);


--
-- Name: hydra_jwk hydra_jwk_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_jwk
    ADD CONSTRAINT hydra_jwk_pkey PRIMARY KEY (pk);


--
-- Name: hydra_oauth2_access hydra_oauth2_access_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_access
    ADD CONSTRAINT hydra_oauth2_access_pkey PRIMARY KEY (signature);


--
-- Name: hydra_oauth2_authentication_session hydra_oauth2_authentication_session_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_authentication_session
    ADD CONSTRAINT hydra_oauth2_authentication_session_pkey PRIMARY KEY (id);


--
-- Name: hydra_oauth2_code hydra_oauth2_code_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_code
    ADD CONSTRAINT hydra_oauth2_code_pkey PRIMARY KEY (signature);


--
-- Name: hydra_oauth2_flow hydra_oauth2_flow_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_flow
    ADD CONSTRAINT hydra_oauth2_flow_pkey PRIMARY KEY (login_challenge);


--
-- Name: hydra_oauth2_jti_blacklist hydra_oauth2_jti_blacklist_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_jti_blacklist
    ADD CONSTRAINT hydra_oauth2_jti_blacklist_pkey PRIMARY KEY (signature, nid);


--
-- Name: hydra_oauth2_logout_request hydra_oauth2_logout_request_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_logout_request
    ADD CONSTRAINT hydra_oauth2_logout_request_pkey PRIMARY KEY (challenge);


--
-- Name: hydra_oauth2_obfuscated_authentication_session hydra_oauth2_obfuscated_authentication_session_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_obfuscated_authentication_session
    ADD CONSTRAINT hydra_oauth2_obfuscated_authentication_session_pkey PRIMARY KEY (subject, client_id, nid);


--
-- Name: hydra_oauth2_oidc hydra_oauth2_oidc_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_oidc
    ADD CONSTRAINT hydra_oauth2_oidc_pkey PRIMARY KEY (signature);


--
-- Name: hydra_oauth2_pkce hydra_oauth2_pkce_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_pkce
    ADD CONSTRAINT hydra_oauth2_pkce_pkey PRIMARY KEY (signature);


--
-- Name: hydra_oauth2_refresh hydra_oauth2_refresh_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_refresh
    ADD CONSTRAINT hydra_oauth2_refresh_pkey PRIMARY KEY (signature);


--
-- Name: hydra_oauth2_trusted_jwt_bearer_issuer hydra_oauth2_trusted_jwt_bearer_issue_issuer_subject_key_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_trusted_jwt_bearer_issuer
    ADD CONSTRAINT hydra_oauth2_trusted_jwt_bearer_issue_issuer_subject_key_id_key UNIQUE (issuer, subject, key_id, nid);


--
-- Name: hydra_oauth2_trusted_jwt_bearer_issuer hydra_oauth2_trusted_jwt_bearer_issuer_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_trusted_jwt_bearer_issuer
    ADD CONSTRAINT hydra_oauth2_trusted_jwt_bearer_issuer_pkey PRIMARY KEY (id);


--
-- Name: join_tokens join_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.join_tokens
    ADD CONSTRAINT join_tokens_pkey PRIMARY KEY (token);


--
-- Name: m2m_agent_attestations m2m_agent_attestations_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_attestations
    ADD CONSTRAINT m2m_agent_attestations_pkey PRIMARY KEY (id);


--
-- Name: m2m_agent_certificate_renewals m2m_agent_certificate_renewals_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_certificate_renewals
    ADD CONSTRAINT m2m_agent_certificate_renewals_pkey PRIMARY KEY (id);


--
-- Name: m2m_agent_deployment_tokens m2m_agent_deployment_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_deployment_tokens
    ADD CONSTRAINT m2m_agent_deployment_tokens_pkey PRIMARY KEY (id);


--
-- Name: m2m_agent_deployment_tokens m2m_agent_deployment_tokens_token_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_deployment_tokens
    ADD CONSTRAINT m2m_agent_deployment_tokens_token_key UNIQUE (token);


--
-- Name: m2m_agent_heartbeats m2m_agent_heartbeats_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_heartbeats
    ADD CONSTRAINT m2m_agent_heartbeats_pkey PRIMARY KEY (id);


--
-- Name: m2m_agent_policies m2m_agent_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_policies
    ADD CONSTRAINT m2m_agent_policies_pkey PRIMARY KEY (id);


--
-- Name: m2m_agents m2m_agents_agent_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agents
    ADD CONSTRAINT m2m_agents_agent_id_key UNIQUE (agent_id);


--
-- Name: m2m_agents m2m_agents_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agents
    ADD CONSTRAINT m2m_agents_pkey PRIMARY KEY (id);


--
-- Name: m2m_attestation_logs m2m_attestation_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_attestation_logs
    ADD CONSTRAINT m2m_attestation_logs_pkey PRIMARY KEY (id);


--
-- Name: m2m_audit_events m2m_audit_events_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_audit_events
    ADD CONSTRAINT m2m_audit_events_pkey PRIMARY KEY (id);


--
-- Name: m2m_certificates m2m_certificates_fingerprint_sha256_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_certificates
    ADD CONSTRAINT m2m_certificates_fingerprint_sha256_key UNIQUE (fingerprint_sha256);


--
-- Name: m2m_certificates m2m_certificates_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_certificates
    ADD CONSTRAINT m2m_certificates_pkey PRIMARY KEY (id);


--
-- Name: m2m_certificates m2m_certificates_serial_number_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_certificates
    ADD CONSTRAINT m2m_certificates_serial_number_key UNIQUE (serial_number);


--
-- Name: m2m_credentials m2m_credentials_client_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_credentials
    ADD CONSTRAINT m2m_credentials_client_id_key UNIQUE (client_id);


--
-- Name: m2m_credentials m2m_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_credentials
    ADD CONSTRAINT m2m_credentials_pkey PRIMARY KEY (id);


--
-- Name: m2m_credentials m2m_credentials_workload_id_client_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_credentials
    ADD CONSTRAINT m2m_credentials_workload_id_client_id_key UNIQUE (workload_id, client_id);


--
-- Name: m2m_workloads m2m_workloads_ca_id_identity_uri_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_workloads
    ADD CONSTRAINT m2m_workloads_ca_id_identity_uri_key UNIQUE (ca_id, identity_uri);


--
-- Name: m2m_workloads m2m_workloads_identity_uri_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_workloads
    ADD CONSTRAINT m2m_workloads_identity_uri_key UNIQUE (identity_uri);


--
-- Name: m2m_workloads m2m_workloads_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_workloads
    ADD CONSTRAINT m2m_workloads_pkey PRIMARY KEY (id);


--
-- Name: m2m_workloads m2m_workloads_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_workloads
    ADD CONSTRAINT m2m_workloads_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: mfa_methods mfa_methods_client_id_method_type_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.mfa_methods
    ADD CONSTRAINT mfa_methods_client_id_method_type_key UNIQUE (client_id, method_type);


--
-- Name: CONSTRAINT mfa_methods_client_id_method_type_key ON mfa_methods; Type: COMMENT; Schema: public; Owner: authprod
--

COMMENT ON CONSTRAINT mfa_methods_client_id_method_type_key ON public.mfa_methods IS 'Ensures one instance of each MFA method type per client';


--
-- Name: mfa_methods mfa_methods_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.mfa_methods
    ADD CONSTRAINT mfa_methods_pkey PRIMARY KEY (id);


--
-- Name: migration_logs migration_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.migration_logs
    ADD CONSTRAINT migration_logs_pkey PRIMARY KEY (id);


--
-- Name: networks networks_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.networks
    ADD CONSTRAINT networks_pkey PRIMARY KEY (id);


--
-- Name: oauth_sessions oauth_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.oauth_sessions
    ADD CONSTRAINT oauth_sessions_pkey PRIMARY KEY (session_id);


--
-- Name: oidc_providers oidc_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.oidc_providers
    ADD CONSTRAINT oidc_providers_pkey PRIMARY KEY (id);


--
-- Name: oidc_providers oidc_providers_provider_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.oidc_providers
    ADD CONSTRAINT oidc_providers_provider_name_key UNIQUE (provider_name);


--
-- Name: oidc_states oidc_states_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.oidc_states
    ADD CONSTRAINT oidc_states_pkey PRIMARY KEY (id);


--
-- Name: oidc_states oidc_states_state_token_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.oidc_states
    ADD CONSTRAINT oidc_states_state_token_key UNIQUE (state_token);


--
-- Name: oidc_user_identities oidc_user_identities_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.oidc_user_identities
    ADD CONSTRAINT oidc_user_identities_pkey PRIMARY KEY (id);


--
-- Name: otp_entries otp_entries_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.otp_entries
    ADD CONSTRAINT otp_entries_pkey PRIMARY KEY (id);


--
-- Name: pending_registrations pending_registrations_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.pending_registrations
    ADD CONSTRAINT pending_registrations_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_tenant_resource_action_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_tenant_resource_action_key UNIQUE (tenant_id, resource, action);


--
-- Name: projects projects_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_pkey PRIMARY KEY (id);


--
-- Name: resources resources_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.resources
    ADD CONSTRAINT resources_pkey PRIMARY KEY (id);


--
-- Name: resources resources_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.resources
    ADD CONSTRAINT resources_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: role_assignment_requests role_assignment_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.role_assignment_requests
    ADD CONSTRAINT role_assignment_requests_pkey PRIMARY KEY (id);


--
-- Name: role_bindings role_bindings_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.role_bindings
    ADD CONSTRAINT role_bindings_pkey PRIMARY KEY (id);


--
-- Name: role_permissions role_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_pkey PRIMARY KEY (role_id, permission_id);


--
-- Name: role_scopes role_scopes_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.role_scopes
    ADD CONSTRAINT role_scopes_pkey PRIMARY KEY (id);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: roles roles_tenant_id_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_tenant_id_id_key UNIQUE (tenant_id, id);


--
-- Name: roles roles_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: roles roles_tenant_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_tenant_name_key UNIQUE (tenant_id, name);


--
-- Name: saml_callback_states saml_callback_states_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.saml_callback_states
    ADD CONSTRAINT saml_callback_states_pkey PRIMARY KEY (id);


--
-- Name: saml_providers saml_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.saml_providers
    ADD CONSTRAINT saml_providers_pkey PRIMARY KEY (id);


--
-- Name: saml_requests saml_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.saml_requests
    ADD CONSTRAINT saml_requests_pkey PRIMARY KEY (id);


--
-- Name: saml_sp_certificates saml_sp_certificates_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.saml_sp_certificates
    ADD CONSTRAINT saml_sp_certificates_pkey PRIMARY KEY (id);


--
-- Name: saml_sp_certificates saml_sp_certificates_tenant_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.saml_sp_certificates
    ADD CONSTRAINT saml_sp_certificates_tenant_id_key UNIQUE (tenant_id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: scope_permissions scope_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.scope_permissions
    ADD CONSTRAINT scope_permissions_pkey PRIMARY KEY (scope_id, permission_id);


--
-- Name: scope_resource_mappings scope_resource_mappings_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.scope_resource_mappings
    ADD CONSTRAINT scope_resource_mappings_pkey PRIMARY KEY (id);


--
-- Name: scope_resource_mappings scope_resource_mappings_tenant_scope_resource_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.scope_resource_mappings
    ADD CONSTRAINT scope_resource_mappings_tenant_scope_resource_key UNIQUE (tenant_id, scope_name, resource_name);


--
-- Name: scopes scopes_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.scopes
    ADD CONSTRAINT scopes_name_key UNIQUE (name) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: scopes scopes_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.scopes
    ADD CONSTRAINT scopes_pkey PRIMARY KEY (id);


--
-- Name: scopes scopes_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.scopes
    ADD CONSTRAINT scopes_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: scopes scopes_tenant_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.scopes
    ADD CONSTRAINT scopes_tenant_name_key UNIQUE (tenant_id, name);


--
-- Name: service_accounts service_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.service_accounts
    ADD CONSTRAINT service_accounts_pkey PRIMARY KEY (id);


--
-- Name: service_accounts service_accounts_tenant_id_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.service_accounts
    ADD CONSTRAINT service_accounts_tenant_id_id_key UNIQUE (tenant_id, id);


--
-- Name: services services_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_pkey PRIMARY KEY (id);


--
-- Name: spiffe_svids spiffe_svids_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.spiffe_svids
    ADD CONSTRAINT spiffe_svids_pkey PRIMARY KEY (id);


--
-- Name: spiffe_workloads spiffe_workloads_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.spiffe_workloads
    ADD CONSTRAINT spiffe_workloads_pkey PRIMARY KEY (id);


--
-- Name: spiffe_workloads spiffe_workloads_spiffe_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.spiffe_workloads
    ADD CONSTRAINT spiffe_workloads_spiffe_id_key UNIQUE (spiffe_id);


--
-- Name: svid_metadata svid_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.svid_metadata
    ADD CONSTRAINT svid_metadata_pkey PRIMARY KEY (id);


--
-- Name: sync_configurations sync_configurations_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.sync_configurations
    ADD CONSTRAINT sync_configurations_pkey PRIMARY KEY (id);


--
-- Name: sync_configurations sync_configurations_tenant_id_config_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.sync_configurations
    ADD CONSTRAINT sync_configurations_tenant_id_config_name_key UNIQUE (tenant_id, config_name);


--
-- Name: tenant_ca_policies tenant_ca_policies_ca_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_ca_policies
    ADD CONSTRAINT tenant_ca_policies_ca_id_key UNIQUE (ca_id);


--
-- Name: tenant_ca_policies tenant_ca_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_ca_policies
    ADD CONSTRAINT tenant_ca_policies_pkey PRIMARY KEY (id);


--
-- Name: tenant_certificate_authorities tenant_certificate_authorities_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_certificate_authorities
    ADD CONSTRAINT tenant_certificate_authorities_pkey PRIMARY KEY (id);


--
-- Name: tenant_certificate_authorities tenant_certificate_authorities_serial_number_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_certificate_authorities
    ADD CONSTRAINT tenant_certificate_authorities_serial_number_key UNIQUE (serial_number);


--
-- Name: tenant_certificate_authorities tenant_certificate_authorities_tenant_id_common_name_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_certificate_authorities
    ADD CONSTRAINT tenant_certificate_authorities_tenant_id_common_name_key UNIQUE (tenant_id, common_name);


--
-- Name: tenant_ciba_auth_requests tenant_ciba_auth_requests_auth_req_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_ciba_auth_requests
    ADD CONSTRAINT tenant_ciba_auth_requests_auth_req_id_key UNIQUE (auth_req_id);


--
-- Name: tenant_ciba_auth_requests tenant_ciba_auth_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_ciba_auth_requests
    ADD CONSTRAINT tenant_ciba_auth_requests_pkey PRIMARY KEY (id);


--
-- Name: tenant_databases tenant_databases_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_databases
    ADD CONSTRAINT tenant_databases_pkey PRIMARY KEY (id);


--
-- Name: tenant_device_tokens tenant_device_tokens_device_token_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_device_tokens
    ADD CONSTRAINT tenant_device_tokens_device_token_key UNIQUE (device_token);


--
-- Name: tenant_device_tokens tenant_device_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_device_tokens
    ADD CONSTRAINT tenant_device_tokens_pkey PRIMARY KEY (id);


--
-- Name: tenant_domains tenant_domains_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_domains
    ADD CONSTRAINT tenant_domains_pkey PRIMARY KEY (id);


--
-- Name: tenant_hydra_clients tenant_hydra_clients_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_hydra_clients
    ADD CONSTRAINT tenant_hydra_clients_pkey PRIMARY KEY (id);


--
-- Name: tenant_mappings tenant_mappings_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_mappings
    ADD CONSTRAINT tenant_mappings_pkey PRIMARY KEY (id);


--
-- Name: tenant_totp_backup_codes tenant_totp_backup_codes_code_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_totp_backup_codes
    ADD CONSTRAINT tenant_totp_backup_codes_code_key UNIQUE (code);


--
-- Name: tenant_totp_backup_codes tenant_totp_backup_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_totp_backup_codes
    ADD CONSTRAINT tenant_totp_backup_codes_pkey PRIMARY KEY (id);


--
-- Name: tenant_totp_secrets tenant_totp_secrets_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_totp_secrets
    ADD CONSTRAINT tenant_totp_secrets_pkey PRIMARY KEY (id);


--
-- Name: tenant_totp_secrets tenant_totp_secrets_user_id_tenant_id_is_primary_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_totp_secrets
    ADD CONSTRAINT tenant_totp_secrets_user_id_tenant_id_is_primary_key UNIQUE (user_id, tenant_id, is_primary);


--
-- Name: tenants tenants_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_pkey PRIMARY KEY (id);


--
-- Name: totp_backup_codes totp_backup_codes_code_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.totp_backup_codes
    ADD CONSTRAINT totp_backup_codes_code_key UNIQUE (code);


--
-- Name: totp_backup_codes totp_backup_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.totp_backup_codes
    ADD CONSTRAINT totp_backup_codes_pkey PRIMARY KEY (id);


--
-- Name: totp_secrets totp_secrets_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.totp_secrets
    ADD CONSTRAINT totp_secrets_pkey PRIMARY KEY (id);


--
-- Name: trust_bundle_cas trust_bundle_cas_bundle_id_ca_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.trust_bundle_cas
    ADD CONSTRAINT trust_bundle_cas_bundle_id_ca_id_key UNIQUE (bundle_id, ca_id);


--
-- Name: trust_bundle_cas trust_bundle_cas_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.trust_bundle_cas
    ADD CONSTRAINT trust_bundle_cas_pkey PRIMARY KEY (id);


--
-- Name: trust_bundles trust_bundles_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.trust_bundles
    ADD CONSTRAINT trust_bundles_pkey PRIMARY KEY (id);


--
-- Name: trust_bundles trust_bundles_tenant_id_trust_domain_bundle_version_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.trust_bundles
    ADD CONSTRAINT trust_bundles_tenant_id_trust_domain_bundle_version_key UNIQUE (tenant_id, trust_domain, bundle_version);


--
-- Name: trust_bundles trust_bundles_tenant_id_trust_domain_sequence_number_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.trust_bundles
    ADD CONSTRAINT trust_bundles_tenant_id_trust_domain_sequence_number_key UNIQUE (tenant_id, trust_domain, sequence_number);


--
-- Name: groups uni_groups_tenant_name; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT uni_groups_tenant_name UNIQUE (tenant_id, name);


--
-- Name: tenant_databases uni_tenant_databases_database_name; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_databases
    ADD CONSTRAINT uni_tenant_databases_database_name UNIQUE (database_name);


--
-- Name: tenant_databases uni_tenant_databases_tenant_id; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_databases
    ADD CONSTRAINT uni_tenant_databases_tenant_id UNIQUE (tenant_id);


--
-- Name: tenants uni_tenants_email; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT uni_tenants_email UNIQUE (email);


--
-- Name: tenants uni_tenants_tenant_domain; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT uni_tenants_tenant_domain UNIQUE (tenant_domain);


--
-- Name: tenants uni_tenants_tenant_id; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT uni_tenants_tenant_id UNIQUE (tenant_id);


--
-- Name: m2m_agent_policies unique_policy_per_workload; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_policies
    ADD CONSTRAINT unique_policy_per_workload UNIQUE (tenant_id, workload_id, policy_name);


--
-- Name: api_scopes uq_api_scopes_tenant_id; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.api_scopes
    ADD CONSTRAINT uq_api_scopes_tenant_id UNIQUE (tenant_id, id);


--
-- Name: api_scopes uq_api_scopes_tenant_name; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.api_scopes
    ADD CONSTRAINT uq_api_scopes_tenant_name UNIQUE (tenant_id, name);




--
-- Name: tenant_device_tokens uq_tenant_device_id_tenant; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_device_tokens
    ADD CONSTRAINT uq_tenant_device_id_tenant UNIQUE (id, tenant_id);


--
-- Name: voice_identity_links uq_voice_identity_tenant_platform_user; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_identity_links
    ADD CONSTRAINT uq_voice_identity_tenant_platform_user UNIQUE (tenant_id, voice_platform, voice_user_id);


--
-- Name: user_auth_preferences user_auth_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_auth_preferences
    ADD CONSTRAINT user_auth_preferences_pkey PRIMARY KEY (id);


--
-- Name: user_auth_preferences user_auth_preferences_user_id_tenant_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_auth_preferences
    ADD CONSTRAINT user_auth_preferences_user_id_tenant_id_key UNIQUE (user_id, tenant_id);


--
-- Name: user_groups user_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_groups
    ADD CONSTRAINT user_groups_pkey PRIMARY KEY (user_id, group_id);


--
-- Name: user_resources user_resources_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_resources
    ADD CONSTRAINT user_resources_pkey PRIMARY KEY (user_id, resource_id);


--
-- Name: user_roles user_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id);


--
-- Name: user_roles user_roles_user_role_tenant_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_user_role_tenant_key UNIQUE (user_id, role_id, tenant_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_tenant_id_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_tenant_id_id_key UNIQUE (tenant_id, id);


--
-- Name: voice_active_sessions voice_active_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_active_sessions
    ADD CONSTRAINT voice_active_sessions_pkey PRIMARY KEY (id);


--
-- Name: voice_active_sessions voice_active_sessions_session_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_active_sessions
    ADD CONSTRAINT voice_active_sessions_session_id_key UNIQUE (session_id);


--
-- Name: voice_auth_logs voice_auth_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_auth_logs
    ADD CONSTRAINT voice_auth_logs_pkey PRIMARY KEY (id);


--
-- Name: voice_auth_sessions voice_auth_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_auth_sessions
    ADD CONSTRAINT voice_auth_sessions_pkey PRIMARY KEY (id);


--
-- Name: voice_identity_links voice_identity_links_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_identity_links
    ADD CONSTRAINT voice_identity_links_pkey PRIMARY KEY (id);


--
-- Name: voice_profiles voice_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_profiles
    ADD CONSTRAINT voice_profiles_pkey PRIMARY KEY (id);


--
-- Name: voice_profiles voice_profiles_user_id_tenant_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_profiles
    ADD CONSTRAINT voice_profiles_user_id_tenant_id_key UNIQUE (user_id, tenant_id);


--
-- Name: voice_sessions voice_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_sessions
    ADD CONSTRAINT voice_sessions_pkey PRIMARY KEY (id);


--
-- Name: voice_sessions voice_sessions_session_token_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_sessions
    ADD CONSTRAINT voice_sessions_session_token_key UNIQUE (session_token);


--
-- Name: voiceprints voiceprints_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voiceprints
    ADD CONSTRAINT voiceprints_pkey PRIMARY KEY (id);


--
-- Name: webauthn_credentials webauthn_credentials_credential_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.webauthn_credentials
    ADD CONSTRAINT webauthn_credentials_credential_id_key UNIQUE (credential_id);


--
-- Name: webauthn_credentials webauthn_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.webauthn_credentials
    ADD CONSTRAINT webauthn_credentials_pkey PRIMARY KEY (id);


--
-- Name: webauthn_sessions webauthn_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.webauthn_sessions
    ADD CONSTRAINT webauthn_sessions_pkey PRIMARY KEY (id);


--
-- Name: webauthn_sessions webauthn_sessions_session_key_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.webauthn_sessions
    ADD CONSTRAINT webauthn_sessions_session_key_key UNIQUE (session_key);


--
-- Name: workload_entries workload_entries_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.workload_entries
    ADD CONSTRAINT workload_entries_pkey PRIMARY KEY (id);


--
-- Name: workload_entries workload_entries_spiffe_id_key; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.workload_entries
    ADD CONSTRAINT workload_entries_spiffe_id_key UNIQUE (spiffe_id);


--
-- Name: workloads workloads_pkey; Type: CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.workloads
    ADD CONSTRAINT workloads_pkey PRIMARY KEY (id);


--
-- Name: groups_name_tenant_unique; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX groups_name_tenant_unique ON public.groups USING btree (name, tenant_id);


--
-- Name: hydra_client_idx_id_uq; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX hydra_client_idx_id_uq ON public.hydra_client USING btree (id, nid);


--
-- Name: hydra_jwk_nid_sid_created_at_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_jwk_nid_sid_created_at_idx ON public.hydra_jwk USING btree (nid, sid, created_at);


--
-- Name: hydra_jwk_nid_sid_kid_created_at_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_jwk_nid_sid_kid_created_at_idx ON public.hydra_jwk USING btree (nid, sid, kid, created_at);


--
-- Name: hydra_jwk_sid_kid_nid_key; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX hydra_jwk_sid_kid_nid_key ON public.hydra_jwk USING btree (sid, kid, nid);


--
-- Name: hydra_oauth2_access_challenge_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_access_challenge_id_idx ON public.hydra_oauth2_access USING btree (challenge_id);


--
-- Name: hydra_oauth2_access_client_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_access_client_id_idx ON public.hydra_oauth2_access USING btree (client_id, nid);


--
-- Name: hydra_oauth2_access_request_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_access_request_id_idx ON public.hydra_oauth2_access USING btree (request_id, nid);


--
-- Name: hydra_oauth2_access_requested_at_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_access_requested_at_idx ON public.hydra_oauth2_access USING btree (requested_at, nid);


--
-- Name: hydra_oauth2_authentication_session_sub_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_authentication_session_sub_idx ON public.hydra_oauth2_authentication_session USING btree (subject, nid);


--
-- Name: hydra_oauth2_code_challenge_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_code_challenge_id_idx ON public.hydra_oauth2_code USING btree (challenge_id, nid);


--
-- Name: hydra_oauth2_code_client_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_code_client_id_idx ON public.hydra_oauth2_code USING btree (client_id, nid);


--
-- Name: hydra_oauth2_flow_cid_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_flow_cid_idx ON public.hydra_oauth2_flow USING btree (client_id, nid);


--
-- Name: hydra_oauth2_flow_client_id_subject_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_flow_client_id_subject_idx ON public.hydra_oauth2_flow USING btree (client_id, nid, subject);


--
-- Name: hydra_oauth2_flow_consent_challenge_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX hydra_oauth2_flow_consent_challenge_idx ON public.hydra_oauth2_flow USING btree (consent_challenge_id);


--
-- Name: hydra_oauth2_flow_login_session_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_flow_login_session_id_idx ON public.hydra_oauth2_flow USING btree (login_session_id, nid);


--
-- Name: hydra_oauth2_flow_previous_consents_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_flow_previous_consents_idx ON public.hydra_oauth2_flow USING btree (subject, client_id, nid, consent_skip, consent_error, consent_remember);


--
-- Name: hydra_oauth2_flow_sub_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_flow_sub_idx ON public.hydra_oauth2_flow USING btree (subject, nid);


--
-- Name: hydra_oauth2_jti_blacklist_expires_at_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_jti_blacklist_expires_at_idx ON public.hydra_oauth2_jti_blacklist USING btree (expires_at, nid);


--
-- Name: hydra_oauth2_logout_request_client_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_logout_request_client_id_idx ON public.hydra_oauth2_logout_request USING btree (client_id, nid);


--
-- Name: hydra_oauth2_logout_request_veri_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX hydra_oauth2_logout_request_veri_idx ON public.hydra_oauth2_logout_request USING btree (verifier);


--
-- Name: hydra_oauth2_obfuscated_authentication_session_so_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX hydra_oauth2_obfuscated_authentication_session_so_idx ON public.hydra_oauth2_obfuscated_authentication_session USING btree (client_id, subject_obfuscated, nid);


--
-- Name: hydra_oauth2_oidc_challenge_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_oidc_challenge_id_idx ON public.hydra_oauth2_oidc USING btree (challenge_id);


--
-- Name: hydra_oauth2_oidc_client_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_oidc_client_id_idx ON public.hydra_oauth2_oidc USING btree (client_id, nid);


--
-- Name: hydra_oauth2_pkce_challenge_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_pkce_challenge_id_idx ON public.hydra_oauth2_pkce USING btree (challenge_id);


--
-- Name: hydra_oauth2_pkce_client_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_pkce_client_id_idx ON public.hydra_oauth2_pkce USING btree (client_id, nid);


--
-- Name: hydra_oauth2_refresh_challenge_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_refresh_challenge_id_idx ON public.hydra_oauth2_refresh USING btree (challenge_id);


--
-- Name: hydra_oauth2_refresh_client_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_refresh_client_id_idx ON public.hydra_oauth2_refresh USING btree (client_id, nid);


--
-- Name: hydra_oauth2_refresh_request_id_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_refresh_request_id_idx ON public.hydra_oauth2_refresh USING btree (request_id);


--
-- Name: hydra_oauth2_refresh_requested_at_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_refresh_requested_at_idx ON public.hydra_oauth2_refresh USING btree (nid, requested_at);


--
-- Name: hydra_oauth2_trusted_jwt_bearer_issuer_expires_at_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_trusted_jwt_bearer_issuer_expires_at_idx ON public.hydra_oauth2_trusted_jwt_bearer_issuer USING btree (expires_at);


--
-- Name: hydra_oauth2_trusted_jwt_bearer_issuer_nid_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX hydra_oauth2_trusted_jwt_bearer_issuer_nid_idx ON public.hydra_oauth2_trusted_jwt_bearer_issuer USING btree (id, nid);


--
-- Name: idx_api_scope_permissions_permission_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_api_scope_permissions_permission_id ON public.api_scope_permissions USING btree (permission_id);


--
-- Name: idx_api_scope_permissions_scope_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_api_scope_permissions_scope_id ON public.api_scope_permissions USING btree (scope_id);


--
-- Name: idx_api_scopes_name; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_api_scopes_name ON public.api_scopes USING btree (name);


--
-- Name: idx_api_scopes_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_api_scopes_tenant_id ON public.api_scopes USING btree (tenant_id);


--
-- Name: idx_attestation_policies_enabled; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_attestation_policies_enabled ON public.attestation_policies USING btree (tenant_id, attestation_type, enabled) WHERE (enabled = true);


--
-- Name: idx_attestation_policies_priority; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_attestation_policies_priority ON public.attestation_policies USING btree (tenant_id, priority DESC);


--
-- Name: idx_attestation_policies_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_attestation_policies_tenant_id ON public.attestation_policies USING btree (tenant_id);


--
-- Name: idx_audit_action; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_action ON public.audit_logs USING btree (action);


--
-- Name: idx_audit_created_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_created_at ON public.audit_logs USING btree (created_at);


--
-- Name: idx_audit_event_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_event_type ON public.audit_logs USING btree (event_type);


--
-- Name: idx_audit_events_action; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_events_action ON public.audit_events USING btree (action);


--
-- Name: idx_audit_events_request_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_events_request_id ON public.audit_events USING btree (request_id);


--
-- Name: idx_audit_events_resource; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_events_resource ON public.audit_events USING btree (resource);


--
-- Name: idx_audit_events_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_events_tenant_id ON public.audit_events USING btree (tenant_id);


--
-- Name: idx_audit_events_timestamp; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_events_timestamp ON public.audit_events USING btree ("timestamp");


--
-- Name: idx_audit_events_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_events_user_id ON public.audit_events USING btree (user_id);


--
-- Name: idx_audit_success; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_success ON public.audit_logs USING btree (success);


--
-- Name: idx_audit_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_tenant_id ON public.audit_logs USING btree (tenant_id);


--
-- Name: idx_audit_workload_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_audit_workload_id ON public.audit_logs USING btree (workload_id);


--
-- Name: idx_auth_agents_api_key; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_auth_agents_api_key ON public.auth_agents USING btree (api_key);


--
-- Name: idx_auth_agents_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_auth_agents_status ON public.auth_agents USING btree (status);


--
-- Name: idx_auth_agents_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_auth_agents_tenant_id ON public.auth_agents USING btree (tenant_id);


--
-- Name: idx_auth_agents_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_auth_agents_type ON public.auth_agents USING btree (type);


--
-- Name: idx_auth_pref_method; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_auth_pref_method ON public.user_auth_preferences USING btree (preferred_method);


--
-- Name: idx_auth_pref_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_auth_pref_user ON public.user_auth_preferences USING btree (user_id);


--
-- Name: idx_backup_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_backup_tenant ON public.totp_backup_codes USING btree (tenant_id);


--
-- Name: idx_backup_used; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_backup_used ON public.totp_backup_codes USING btree (is_used);


--
-- Name: idx_backup_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_backup_user ON public.totp_backup_codes USING btree (user_id);


--
-- Name: idx_ciba_auth_expires; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_ciba_auth_expires ON public.ciba_auth_requests USING btree (expires_at);


--
-- Name: idx_ciba_auth_req_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_ciba_auth_req_id ON public.ciba_requests USING btree (auth_req_id);


--
-- Name: idx_ciba_auth_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_ciba_auth_status ON public.ciba_auth_requests USING btree (status);


--
-- Name: idx_ciba_auth_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_ciba_auth_tenant ON public.ciba_auth_requests USING btree (tenant_id);


--
-- Name: idx_ciba_auth_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_ciba_auth_user ON public.ciba_auth_requests USING btree (user_id);


--
-- Name: idx_ciba_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_ciba_expires_at ON public.ciba_requests USING btree (expires_at);


--
-- Name: idx_ciba_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_ciba_status ON public.ciba_requests USING btree (status);


--
-- Name: idx_ciba_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_ciba_user ON public.ciba_requests USING btree (user_id);


--
-- Name: idx_client_roles_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_client_roles_tenant_id ON public.client_roles USING btree (tenant_id);


--
-- Name: idx_clients_deleted; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_deleted ON public.clients USING btree (deleted);


--
-- Name: idx_clients_deleted_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_deleted_at ON public.clients USING btree (deleted_at);


--
-- Name: idx_clients_hydra_client_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_hydra_client_id ON public.clients USING btree (hydra_client_id) WHERE ((hydra_client_id IS NOT NULL) AND (hydra_client_id <> ''::text));


--
-- Name: idx_clients_oidc_enabled; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_oidc_enabled ON public.clients USING btree (oidc_enabled);


--
-- Name: idx_clients_org_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_org_id ON public.clients USING btree (org_id);


--
-- Name: idx_clients_owner; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_owner ON public.clients USING btree (owner_id);


--
-- Name: idx_clients_owner_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_owner_id ON public.clients USING btree (owner_id);


--
-- Name: idx_clients_project_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_project_id ON public.clients USING btree (project_id);


--
-- Name: idx_clients_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_status ON public.clients USING btree (status);


--
-- Name: idx_clients_tags; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_tags ON public.clients USING btree (tags);


--
-- Name: idx_clients_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_tenant_id ON public.clients USING btree (tenant_id);


--
-- Name: idx_clients_tenant_org; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_clients_tenant_org ON public.clients USING btree (tenant_id, org_id);


--
-- Name: idx_credentials_credential_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_credentials_credential_id ON public.credentials USING btree (credential_id);


--
-- Name: idx_crl_ca_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_crl_ca_id ON public.certificate_revocation_list USING btree (ca_id);


--
-- Name: idx_crl_number; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_crl_number ON public.certificate_revocation_list USING btree (ca_id, crl_number);


--
-- Name: idx_crl_serial; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_crl_serial ON public.certificate_revocation_list USING btree (serial_number);


--
-- Name: idx_deleg_policy_client_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_deleg_policy_client_id ON public.delegation_policies USING btree (client_id);


--
-- Name: idx_deleg_policy_lookup; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_deleg_policy_lookup ON public.delegation_policies USING btree (tenant_id, role_name, agent_type, enabled);


--
-- Name: idx_deleg_policy_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_deleg_policy_tenant_id ON public.delegation_policies USING btree (tenant_id);


--
-- Name: idx_deleg_token_expires; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_deleg_token_expires ON public.delegation_tokens USING btree (expires_at) WHERE (status = 'active'::text);


--
-- Name: idx_deleg_token_lookup; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_deleg_token_lookup ON public.delegation_tokens USING btree (tenant_id, client_id, status);


--
-- Name: idx_deleg_token_policy; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_deleg_token_policy ON public.delegation_tokens USING btree (policy_id);


--
-- Name: idx_device_challenges_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_challenges_active ON public.device_auth_challenges USING btree (device_id, verified, expires_at);


--
-- Name: idx_device_codes_device_code; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_codes_device_code ON public.device_codes USING btree (device_code);


--
-- Name: idx_device_codes_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_codes_expires_at ON public.device_codes USING btree (expires_at);


--
-- Name: idx_device_codes_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_codes_status ON public.device_codes USING btree (status);


--
-- Name: idx_device_codes_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_codes_tenant_id ON public.device_codes USING btree (tenant_id);


--
-- Name: idx_device_codes_user_code; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_codes_user_code ON public.device_codes USING btree (user_code);


--
-- Name: idx_device_codes_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_codes_user_id ON public.device_codes USING btree (user_id);


--
-- Name: idx_device_sessions_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_sessions_active ON public.device_sessions USING btree (device_id, active);


--
-- Name: idx_device_sessions_token; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_sessions_token ON public.device_sessions USING btree (session_token) WHERE (active = true);


--
-- Name: idx_device_tokens_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_tokens_active ON public.device_tokens USING btree (is_active);


--
-- Name: idx_device_tokens_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_tokens_tenant ON public.device_tokens USING btree (tenant_id);


--
-- Name: idx_device_tokens_token; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_tokens_token ON public.device_tokens USING btree (device_token);


--
-- Name: idx_device_tokens_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_device_tokens_user ON public.device_tokens USING btree (user_id);


--
-- Name: idx_fb_export_configs_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_fb_export_configs_tenant_id ON public.fluent_bit_export_configs USING btree (tenant_id);


--
-- Name: idx_grant_audit_actor_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_grant_audit_actor_user_id ON public.grant_audit USING btree (actor_user_id);


--
-- Name: idx_grant_audit_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_grant_audit_tenant_id ON public.grant_audit USING btree (tenant_id);


--
-- Name: idx_group_roles_group_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_group_roles_group_id ON public.group_roles USING btree (group_id);


--
-- Name: idx_group_roles_role_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_group_roles_role_id ON public.group_roles USING btree (role_id);


--
-- Name: idx_group_roles_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_group_roles_tenant_id ON public.group_roles USING btree (tenant_id);


--
-- Name: idx_group_scopes_group_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_group_scopes_group_id ON public.group_scopes USING btree (group_id);


--
-- Name: idx_group_scopes_scope_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_group_scopes_scope_id ON public.group_scopes USING btree (scope_id);


--
-- Name: idx_groups_name; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_groups_name ON public.groups USING btree (name);


--
-- Name: idx_groups_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_groups_tenant_id ON public.groups USING btree (tenant_id);


--
-- Name: idx_m2m_agent_attestations_agent_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_attestations_agent_id ON public.m2m_agent_attestations USING btree (agent_id);


--
-- Name: idx_m2m_agent_attestations_created_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_attestations_created_at ON public.m2m_agent_attestations USING btree (created_at DESC);


--
-- Name: idx_m2m_agent_attestations_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_attestations_status ON public.m2m_agent_attestations USING btree (status);


--
-- Name: idx_m2m_agent_attestations_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_attestations_tenant_id ON public.m2m_agent_attestations USING btree (tenant_id);


--
-- Name: idx_m2m_agent_attestations_workload_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_attestations_workload_id ON public.m2m_agent_attestations USING btree (workload_id);


--
-- Name: idx_m2m_agent_heartbeats_agent_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_heartbeats_agent_id ON public.m2m_agent_heartbeats USING btree (agent_id);


--
-- Name: idx_m2m_agent_heartbeats_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_heartbeats_tenant_id ON public.m2m_agent_heartbeats USING btree (tenant_id);


--
-- Name: idx_m2m_agent_heartbeats_timestamp; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_heartbeats_timestamp ON public.m2m_agent_heartbeats USING btree ("timestamp" DESC);


--
-- Name: idx_m2m_agent_policies_enabled; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_policies_enabled ON public.m2m_agent_policies USING btree (enabled);


--
-- Name: idx_m2m_agent_policies_priority; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_policies_priority ON public.m2m_agent_policies USING btree (priority DESC);


--
-- Name: idx_m2m_agent_policies_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_policies_tenant_id ON public.m2m_agent_policies USING btree (tenant_id);


--
-- Name: idx_m2m_agent_policies_workload_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_policies_workload_id ON public.m2m_agent_policies USING btree (workload_id);


--
-- Name: idx_m2m_agent_renewals_agent_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_renewals_agent_id ON public.m2m_agent_certificate_renewals USING btree (agent_id);


--
-- Name: idx_m2m_agent_renewals_initiated_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_renewals_initiated_at ON public.m2m_agent_certificate_renewals USING btree (initiated_at DESC);


--
-- Name: idx_m2m_agent_renewals_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_renewals_status ON public.m2m_agent_certificate_renewals USING btree (status);


--
-- Name: idx_m2m_agent_renewals_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_renewals_tenant_id ON public.m2m_agent_certificate_renewals USING btree (tenant_id);


--
-- Name: idx_m2m_agent_tokens_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_tokens_expires_at ON public.m2m_agent_deployment_tokens USING btree (expires_at);


--
-- Name: idx_m2m_agent_tokens_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_tokens_status ON public.m2m_agent_deployment_tokens USING btree (status);


--
-- Name: idx_m2m_agent_tokens_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_tokens_tenant_id ON public.m2m_agent_deployment_tokens USING btree (tenant_id);


--
-- Name: idx_m2m_agent_tokens_token_hash; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_tokens_token_hash ON public.m2m_agent_deployment_tokens USING btree (token_hash);


--
-- Name: idx_m2m_agent_tokens_workload_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agent_tokens_workload_id ON public.m2m_agent_deployment_tokens USING btree (workload_id);


--
-- Name: idx_m2m_agents_agent_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agents_agent_id ON public.m2m_agents USING btree (agent_id);


--
-- Name: idx_m2m_agents_certificate_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agents_certificate_expires_at ON public.m2m_agents USING btree (certificate_expires_at);


--
-- Name: idx_m2m_agents_health_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agents_health_status ON public.m2m_agents USING btree (health_status);


--
-- Name: idx_m2m_agents_last_heartbeat; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agents_last_heartbeat ON public.m2m_agents USING btree (last_heartbeat);


--
-- Name: idx_m2m_agents_platform; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agents_platform ON public.m2m_agents USING btree (platform);


--
-- Name: idx_m2m_agents_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agents_status ON public.m2m_agents USING btree (status);


--
-- Name: idx_m2m_agents_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agents_tenant_id ON public.m2m_agents USING btree (tenant_id);


--
-- Name: idx_m2m_agents_workload_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_agents_workload_id ON public.m2m_agents USING btree (workload_id);


--
-- Name: idx_m2m_attestations_cert_serial; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_attestations_cert_serial ON public.m2m_attestation_logs USING btree (cert_serial_number);


--
-- Name: idx_m2m_attestations_credential_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_attestations_credential_id ON public.m2m_attestation_logs USING btree (credential_id);


--
-- Name: idx_m2m_attestations_success; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_attestations_success ON public.m2m_attestation_logs USING btree (success);


--
-- Name: idx_m2m_attestations_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_attestations_tenant_id ON public.m2m_attestation_logs USING btree (tenant_id);


--
-- Name: idx_m2m_attestations_time; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_attestations_time ON public.m2m_attestation_logs USING btree (attested_at);


--
-- Name: idx_m2m_attestations_workload_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_attestations_workload_id ON public.m2m_attestation_logs USING btree (workload_id);


--
-- Name: idx_m2m_audit_actor; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_audit_actor ON public.m2m_audit_events USING btree (actor_type, actor_id);


--
-- Name: idx_m2m_audit_event_category; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_audit_event_category ON public.m2m_audit_events USING btree (event_category);


--
-- Name: idx_m2m_audit_event_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_audit_event_type ON public.m2m_audit_events USING btree (event_type);


--
-- Name: idx_m2m_audit_severity; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_audit_severity ON public.m2m_audit_events USING btree (severity);


--
-- Name: idx_m2m_audit_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_audit_tenant_id ON public.m2m_audit_events USING btree (tenant_id);


--
-- Name: idx_m2m_audit_time; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_audit_time ON public.m2m_audit_events USING btree (event_time);


--
-- Name: idx_m2m_certs_ca_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_certs_ca_id ON public.m2m_certificates USING btree (ca_id);


--
-- Name: idx_m2m_certs_expiry; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_certs_expiry ON public.m2m_certificates USING btree (not_after) WHERE ((status)::text = 'active'::text);


--
-- Name: idx_m2m_certs_fingerprint; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_certs_fingerprint ON public.m2m_certificates USING btree (fingerprint_sha256);


--
-- Name: idx_m2m_certs_renewal; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_certs_renewal ON public.m2m_certificates USING btree (not_after, auto_renew) WHERE (((status)::text = 'active'::text) AND (auto_renew = true));


--
-- Name: idx_m2m_certs_serial; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_certs_serial ON public.m2m_certificates USING btree (serial_number);


--
-- Name: idx_m2m_certs_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_certs_status ON public.m2m_certificates USING btree (status);


--
-- Name: idx_m2m_certs_workload_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_certs_workload_id ON public.m2m_certificates USING btree (workload_id);


--
-- Name: idx_m2m_creds_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_creds_active ON public.m2m_credentials USING btree (active) WHERE (active = true);


--
-- Name: idx_m2m_creds_client_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_creds_client_id ON public.m2m_credentials USING btree (client_id);


--
-- Name: idx_m2m_creds_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_creds_tenant_id ON public.m2m_credentials USING btree (tenant_id);


--
-- Name: idx_m2m_creds_workload_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_creds_workload_id ON public.m2m_credentials USING btree (workload_id);


--
-- Name: idx_m2m_workloads_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_workloads_active ON public.m2m_workloads USING btree (active) WHERE (active = true);


--
-- Name: idx_m2m_workloads_ca_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_workloads_ca_id ON public.m2m_workloads USING btree (ca_id);


--
-- Name: idx_m2m_workloads_identity_uri; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_workloads_identity_uri ON public.m2m_workloads USING btree (identity_uri);


--
-- Name: idx_m2m_workloads_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_m2m_workloads_tenant_id ON public.m2m_workloads USING btree (tenant_id);


--
-- Name: idx_mfa_methods_client_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_mfa_methods_client_id ON public.mfa_methods USING btree (client_id);


--
-- Name: idx_mfa_methods_enabled; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_mfa_methods_enabled ON public.mfa_methods USING btree (enabled);


--
-- Name: idx_mfa_methods_method_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_mfa_methods_method_type ON public.mfa_methods USING btree (method_type);


--
-- Name: idx_mfa_methods_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_mfa_methods_type ON public.mfa_methods USING btree (method_type);


--
-- Name: idx_mfa_methods_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_mfa_methods_user_id ON public.mfa_methods USING btree (user_id);


--
-- Name: idx_migration_logs_db_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_migration_logs_db_type ON public.migration_logs USING btree (db_type);


--
-- Name: idx_migration_logs_success; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_migration_logs_success ON public.migration_logs USING btree (success);


--
-- Name: idx_migration_logs_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_migration_logs_tenant_id ON public.migration_logs USING btree (tenant_id);


--
-- Name: idx_migration_logs_version; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_migration_logs_version ON public.migration_logs USING btree (version);


--
-- Name: idx_oauth_sessions_org_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_oauth_sessions_org_id ON public.oauth_sessions USING btree (org_id) WHERE (is_active = true);


--
-- Name: idx_oidc_identities_provider; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_oidc_identities_provider ON public.oidc_user_identities USING btree (provider_name, provider_user_id);


--
-- Name: idx_oidc_identities_provider_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_oidc_identities_provider_user ON public.oidc_user_identities USING btree (provider_name, provider_user_id);


--
-- Name: idx_oidc_identities_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_oidc_identities_tenant ON public.oidc_user_identities USING btree (tenant_id);


--
-- Name: idx_oidc_identities_tenant_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_oidc_identities_tenant_user ON public.oidc_user_identities USING btree (tenant_id, user_id);


--
-- Name: idx_oidc_identities_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_oidc_identities_user ON public.oidc_user_identities USING btree (tenant_id, user_id);


--
-- Name: idx_oidc_providers_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_oidc_providers_active ON public.oidc_providers USING btree (is_active);


--
-- Name: idx_oidc_states_expires; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_oidc_states_expires ON public.oidc_states USING btree (expires_at);


--
-- Name: idx_oidc_states_token; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_oidc_states_token ON public.oidc_states USING btree (state_token);


--
-- Name: idx_otp_entries_email; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_otp_entries_email ON public.otp_entries USING btree (email);


--
-- Name: idx_otp_entries_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_otp_entries_expires_at ON public.otp_entries USING btree (expires_at);


--
-- Name: idx_otp_entries_verified; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_otp_entries_verified ON public.otp_entries USING btree (verified);


--
-- Name: idx_parent_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_parent_id ON public.workload_entries USING btree (parent_id);


--
-- Name: idx_pending_registrations_email; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_pending_registrations_email ON public.pending_registrations USING btree (email);


--
-- Name: idx_pending_registrations_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_pending_registrations_expires_at ON public.pending_registrations USING btree (expires_at);


--
-- Name: idx_pending_registrations_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_pending_registrations_tenant_id ON public.pending_registrations USING btree (tenant_id);


--
-- Name: idx_permissions_global_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_permissions_global_id ON public.permissions USING btree (id) WHERE (tenant_id IS NULL);


--
-- Name: idx_permissions_global_resource_action; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_permissions_global_resource_action ON public.permissions USING btree (resource, action) WHERE (tenant_id IS NULL);


--
-- Name: idx_permissions_resource; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_permissions_resource ON public.permissions USING btree (resource);


--
-- Name: idx_permissions_tenant_resource_action_unique; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_permissions_tenant_resource_action_unique ON public.permissions USING btree (tenant_id, resource, action);


--
-- Name: idx_projects_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_projects_active ON public.projects USING btree (active);


--
-- Name: idx_projects_client_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_projects_client_id ON public.projects USING btree (client_id);


--
-- Name: idx_projects_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_projects_tenant_id ON public.projects USING btree (tenant_id);


--
-- Name: idx_projects_timestamps; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_projects_timestamps ON public.projects USING btree (created_at, updated_at);


--
-- Name: idx_projects_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_projects_user_id ON public.projects USING btree (user_id);


--
-- Name: idx_resources_name; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_resources_name ON public.resources USING btree (name);


--
-- Name: idx_resources_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_resources_tenant_id ON public.resources USING btree (tenant_id);


--
-- Name: idx_role_assignment_requests_role_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_assignment_requests_role_id ON public.role_assignment_requests USING btree (role_id);


--
-- Name: idx_role_assignment_requests_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_assignment_requests_status ON public.role_assignment_requests USING btree (status);


--
-- Name: idx_role_assignment_requests_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_assignment_requests_tenant_id ON public.role_assignment_requests USING btree (tenant_id);


--
-- Name: idx_role_assignment_requests_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_assignment_requests_user_id ON public.role_assignment_requests USING btree (user_id);


--
-- Name: idx_role_bindings_role_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_bindings_role_id ON public.role_bindings USING btree (role_id);


--
-- Name: idx_role_bindings_service_account_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_bindings_service_account_id ON public.role_bindings USING btree (service_account_id);


--
-- Name: idx_role_bindings_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_bindings_tenant_id ON public.role_bindings USING btree (tenant_id);


--
-- Name: idx_role_bindings_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_bindings_user_id ON public.role_bindings USING btree (user_id);


--
-- Name: idx_role_permissions_permission_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_permissions_permission_id ON public.role_permissions USING btree (permission_id);


--
-- Name: idx_role_permissions_role_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_permissions_role_id ON public.role_permissions USING btree (role_id);


--
-- Name: idx_role_scopes_role_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_role_scopes_role_id ON public.role_scopes USING btree (role_id);


--
-- Name: idx_roles_global_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_roles_global_id ON public.roles USING btree (id) WHERE (tenant_id IS NULL);


--
-- Name: idx_roles_global_name; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_roles_global_name ON public.roles USING btree (name) WHERE (tenant_id IS NULL);


--
-- Name: idx_roles_name; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_roles_name ON public.roles USING btree (name);


--
-- Name: idx_roles_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_roles_tenant_id ON public.roles USING btree (tenant_id);


--
-- Name: idx_saml_callback_states_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_saml_callback_states_expires_at ON public.saml_callback_states USING btree (expires_at);


--
-- Name: idx_saml_callback_states_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_saml_callback_states_id ON public.saml_callback_states USING btree (id);


--
-- Name: idx_saml_provider_unique; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_saml_provider_unique ON public.saml_providers USING btree (tenant_id, client_id, provider_name);


--
-- Name: idx_saml_providers_client_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_saml_providers_client_id ON public.saml_providers USING btree (client_id);


--
-- Name: idx_saml_providers_is_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_saml_providers_is_active ON public.saml_providers USING btree (is_active);


--
-- Name: idx_saml_providers_sort_order; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_saml_providers_sort_order ON public.saml_providers USING btree (sort_order);


--
-- Name: idx_saml_providers_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_saml_providers_tenant_id ON public.saml_providers USING btree (tenant_id);


--
-- Name: idx_scopes_global_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_scopes_global_id ON public.scopes USING btree (id) WHERE (tenant_id IS NULL);


--
-- Name: idx_scopes_global_name; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_scopes_global_name ON public.scopes USING btree (name) WHERE (tenant_id IS NULL);


--
-- Name: idx_scopes_name; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_scopes_name ON public.scopes USING btree (name);


--
-- Name: idx_scopes_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_scopes_tenant_id ON public.scopes USING btree (tenant_id);


--
-- Name: idx_service_accounts_global_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_service_accounts_global_id ON public.service_accounts USING btree (id) WHERE (tenant_id IS NULL);


--
-- Name: idx_service_accounts_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_service_accounts_tenant_id ON public.service_accounts USING btree (tenant_id);


--
-- Name: idx_services_agent_accessible; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_services_agent_accessible ON public.services USING btree (agent_accessible);


--
-- Name: idx_services_created_by; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_services_created_by ON public.services USING btree (created_by);


--
-- Name: idx_services_resource_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_services_resource_id ON public.services USING btree (resource_id);


--
-- Name: idx_services_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_services_type ON public.services USING btree (type);


--
-- Name: idx_spiffe_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_spiffe_id ON public.workload_entries USING btree (spiffe_id);


--
-- Name: idx_sync_configs_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_sync_configs_active ON public.sync_configurations USING btree (is_active);


--
-- Name: idx_sync_configs_client_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_sync_configs_client_id ON public.sync_configurations USING btree (client_id);


--
-- Name: idx_sync_configs_sync_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_sync_configs_sync_type ON public.sync_configurations USING btree (sync_type);


--
-- Name: idx_sync_configs_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_sync_configs_tenant_id ON public.sync_configurations USING btree (tenant_id);


--
-- Name: idx_sync_configs_tenant_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_sync_configs_tenant_type ON public.sync_configurations USING btree (tenant_id, sync_type);


--
-- Name: idx_tenant_backup_code; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_backup_code ON public.tenant_totp_backup_codes USING btree (code);


--
-- Name: idx_tenant_backup_created_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_backup_created_at ON public.tenant_totp_backup_codes USING btree (created_at);


--
-- Name: idx_tenant_backup_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_backup_tenant ON public.tenant_totp_backup_codes USING btree (tenant_id);


--
-- Name: idx_tenant_backup_used; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_backup_used ON public.tenant_totp_backup_codes USING btree (is_used);


--
-- Name: idx_tenant_backup_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_backup_user ON public.tenant_totp_backup_codes USING btree (user_id);


--
-- Name: idx_tenant_backup_user_unused; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_backup_user_unused ON public.tenant_totp_backup_codes USING btree (user_id, is_used);


--
-- Name: idx_tenant_ca_policies_ca_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_ca_policies_ca_id ON public.tenant_ca_policies USING btree (ca_id);


--
-- Name: idx_tenant_ca_policies_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_ca_policies_tenant_id ON public.tenant_ca_policies USING btree (tenant_id);


--
-- Name: idx_tenant_cas_expiry; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_cas_expiry ON public.tenant_certificate_authorities USING btree (not_after) WHERE ((status)::text = 'active'::text);


--
-- Name: idx_tenant_cas_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_cas_status ON public.tenant_certificate_authorities USING btree (status);


--
-- Name: idx_tenant_cas_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_cas_tenant_id ON public.tenant_certificate_authorities USING btree (tenant_id);


--
-- Name: idx_tenant_ciba_auth_req_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_ciba_auth_req_id ON public.tenant_ciba_auth_requests USING btree (auth_req_id);


--
-- Name: idx_tenant_ciba_created_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_ciba_created_at ON public.tenant_ciba_auth_requests USING btree (created_at);


--
-- Name: idx_tenant_ciba_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_ciba_expires_at ON public.tenant_ciba_auth_requests USING btree (expires_at);


--
-- Name: idx_tenant_ciba_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_ciba_status ON public.tenant_ciba_auth_requests USING btree (status);


--
-- Name: idx_tenant_ciba_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_ciba_tenant ON public.tenant_ciba_auth_requests USING btree (tenant_id);


--
-- Name: idx_tenant_ciba_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_ciba_user ON public.tenant_ciba_auth_requests USING btree (user_id);


--
-- Name: idx_tenant_ciba_user_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_ciba_user_status ON public.tenant_ciba_auth_requests USING btree (user_id, status);


--
-- Name: idx_tenant_databases_last_migration; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_databases_last_migration ON public.tenant_databases USING btree (last_migration);


--
-- Name: idx_tenant_databases_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_databases_status ON public.tenant_databases USING btree (migration_status);


--
-- Name: idx_tenant_databases_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_databases_tenant_id ON public.tenant_databases USING btree (tenant_id);


--
-- Name: idx_tenant_device_token_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_device_token_active ON public.tenant_device_tokens USING btree (is_active);


--
-- Name: idx_tenant_device_token_device_token; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_device_token_device_token ON public.tenant_device_tokens USING btree (device_token);


--
-- Name: idx_tenant_device_token_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_device_token_tenant ON public.tenant_device_tokens USING btree (tenant_id);


--
-- Name: idx_tenant_device_token_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_device_token_user ON public.tenant_device_tokens USING btree (user_id);


--
-- Name: idx_tenant_domains_domain_unique; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_tenant_domains_domain_unique ON public.tenant_domains USING btree (domain);


--
-- Name: idx_tenant_domains_domain_verified; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_domains_domain_verified ON public.tenant_domains USING btree (domain, is_verified);


--
-- Name: idx_tenant_domains_primary_per_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_tenant_domains_primary_per_tenant ON public.tenant_domains USING btree (tenant_id) WHERE (is_primary = true);


--
-- Name: idx_tenant_domains_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_domains_status ON public.tenant_domains USING btree (is_verified, kind);


--
-- Name: idx_tenant_domains_tenant_id_primary; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_domains_tenant_id_primary ON public.tenant_domains USING btree (tenant_id, is_primary);


--
-- Name: idx_tenant_domains_tenant_id_verified; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_domains_tenant_id_verified ON public.tenant_domains USING btree (tenant_id, is_verified);


--
-- Name: idx_tenant_hydra_clients_client_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_hydra_clients_client_type ON public.tenant_hydra_clients USING btree (client_type);


--
-- Name: idx_tenant_hydra_clients_hydra_client_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_tenant_hydra_clients_hydra_client_id ON public.tenant_hydra_clients USING btree (hydra_client_id);


--
-- Name: idx_tenant_hydra_clients_org_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_hydra_clients_org_tenant ON public.tenant_hydra_clients USING btree (org_id, tenant_id);


--
-- Name: idx_tenant_mappings_client_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_tenant_mappings_client_id ON public.tenant_mappings USING btree (client_id);


--
-- Name: idx_tenant_mappings_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_mappings_tenant ON public.tenant_mappings USING btree (tenant_id);


--
-- Name: idx_tenant_mappings_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_mappings_tenant_id ON public.tenant_mappings USING btree (tenant_id);


--
-- Name: idx_tenant_totp_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_totp_active ON public.tenant_totp_secrets USING btree (is_active);


--
-- Name: idx_tenant_totp_created_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_totp_created_at ON public.tenant_totp_secrets USING btree (created_at);


--
-- Name: idx_tenant_totp_primary; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_totp_primary ON public.tenant_totp_secrets USING btree (is_primary);


--
-- Name: idx_tenant_totp_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_totp_tenant ON public.tenant_totp_secrets USING btree (tenant_id);


--
-- Name: idx_tenant_totp_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_totp_user ON public.tenant_totp_secrets USING btree (user_id);


--
-- Name: idx_tenant_totp_user_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenant_totp_user_active ON public.tenant_totp_secrets USING btree (user_id, is_active);


--
-- Name: idx_tenants_email; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenants_email ON public.tenants USING btree (email);


--
-- Name: idx_tenants_provider; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenants_provider ON public.tenants USING btree (provider);


--
-- Name: idx_tenants_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenants_status ON public.tenants USING btree (status);


--
-- Name: idx_tenants_tenant_domain; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenants_tenant_domain ON public.tenants USING btree (tenant_domain);


--
-- Name: idx_tenants_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenants_tenant_id ON public.tenants USING btree (tenant_id);


--
-- Name: idx_tenants_vault_mount; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_tenants_vault_mount ON public.tenants USING btree (vault_mount);


--
-- Name: idx_token_expiry; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_token_expiry ON public.join_tokens USING btree (expiry);


--
-- Name: idx_totp_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_totp_active ON public.totp_secrets USING btree (is_active, is_primary);


--
-- Name: idx_totp_primary_device; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX idx_totp_primary_device ON public.totp_secrets USING btree (user_id, tenant_id) WHERE (is_primary = true);


--
-- Name: idx_totp_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_totp_tenant ON public.totp_secrets USING btree (tenant_id);


--
-- Name: idx_totp_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_totp_user ON public.totp_secrets USING btree (user_id);


--
-- Name: idx_trust_bundle_cas_bundle_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_trust_bundle_cas_bundle_id ON public.trust_bundle_cas USING btree (bundle_id);


--
-- Name: idx_trust_bundle_cas_ca_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_trust_bundle_cas_ca_id ON public.trust_bundle_cas USING btree (ca_id);


--
-- Name: idx_trust_bundle_cas_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_trust_bundle_cas_status ON public.trust_bundle_cas USING btree (status);


--
-- Name: idx_trust_bundles_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_trust_bundles_active ON public.trust_bundles USING btree (tenant_id, trust_domain, is_active) WHERE (is_active = true);


--
-- Name: idx_trust_bundles_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_trust_bundles_tenant_id ON public.trust_bundles USING btree (tenant_id);


--
-- Name: idx_trust_bundles_trust_domain; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_trust_bundles_trust_domain ON public.trust_bundles USING btree (trust_domain);


--
-- Name: idx_user_groups_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_user_groups_tenant_id ON public.user_groups USING btree (tenant_id);


--
-- Name: idx_user_groups_user_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_user_groups_user_tenant ON public.user_groups USING btree (user_id, tenant_id);


--
-- Name: idx_user_roles_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_user_roles_tenant_id ON public.user_roles USING btree (tenant_id);


--
-- Name: idx_user_roles_user_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_user_roles_user_tenant ON public.user_roles USING btree (user_id, tenant_id);


--
-- Name: idx_user_scopes_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_user_scopes_tenant_id ON public.user_scopes USING btree (tenant_id);


--
-- Name: idx_users_account_locked; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_account_locked ON public.users USING btree (account_locked_at) WHERE (account_locked_at IS NOT NULL);


--
-- Name: idx_users_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_active ON public.users USING btree (active);


--
-- Name: idx_users_client_email; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_client_email ON public.users USING btree (client_id, email);


--
-- Name: idx_users_client_email_lower; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_client_email_lower ON public.users USING btree (client_id, lower(email));


--
-- Name: idx_users_client_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_client_id ON public.users USING btree (client_id);


--
-- Name: idx_users_deleted_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_deleted_at ON public.users USING btree (deleted_at);


--
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_email ON public.users USING btree (email);


--
-- Name: idx_users_email_client; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_email_client ON public.users USING btree (email, client_id);


--
-- Name: idx_users_email_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_email_tenant ON public.users USING btree (email, tenant_id);


--
-- Name: idx_users_external_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_external_id ON public.users USING btree (external_id);


--
-- Name: idx_users_is_primary_admin; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_is_primary_admin ON public.users USING btree (is_primary_admin) WHERE (is_primary_admin = true);


--
-- Name: idx_users_mfa; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_mfa ON public.users USING btree (mfa_enabled, mfa_verified);


--
-- Name: idx_users_password_change_required; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_password_change_required ON public.users USING btree (password_change_required) WHERE (password_change_required = true);


--
-- Name: idx_users_project_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_project_id ON public.users USING btree (project_id);


--
-- Name: idx_users_provider; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_provider ON public.tenants USING btree (provider);


--
-- Name: idx_users_provider_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_provider_status ON public.users USING btree (provider, active);


--
-- Name: idx_users_sync_info; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_sync_info ON public.users USING btree (sync_source, is_synced_user);


--
-- Name: idx_users_temp_password_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_temp_password_expires_at ON public.users USING btree (temporary_password_expires_at) WHERE (temporary_password_expires_at IS NOT NULL);


--
-- Name: idx_users_temporary_password; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_temporary_password ON public.users USING btree (temporary_password) WHERE (temporary_password = true);


--
-- Name: idx_users_tenant_email; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_tenant_email ON public.users USING btree (tenant_id, email);


--
-- Name: idx_users_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_tenant_id ON public.users USING btree (tenant_id);


--
-- Name: idx_users_tenant_project; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_tenant_project ON public.users USING btree (tenant_id, project_id);


--
-- Name: idx_users_timestamps; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_users_timestamps ON public.users USING btree (created_at, updated_at);


--
-- Name: idx_voice_active_sessions_access_token_hash; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_active_sessions_access_token_hash ON public.voice_active_sessions USING btree (access_token_hash);


--
-- Name: idx_voice_active_sessions_is_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_active_sessions_is_active ON public.voice_active_sessions USING btree (is_active);


--
-- Name: idx_voice_active_sessions_session_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_active_sessions_session_id ON public.voice_active_sessions USING btree (session_id);


--
-- Name: idx_voice_active_sessions_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_active_sessions_tenant_id ON public.voice_active_sessions USING btree (tenant_id);


--
-- Name: idx_voice_active_sessions_tenant_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_active_sessions_tenant_user ON public.voice_active_sessions USING btree (tenant_id, user_id);


--
-- Name: idx_voice_active_sessions_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_active_sessions_user_id ON public.voice_active_sessions USING btree (user_id);


--
-- Name: idx_voice_auth_logs_created; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_auth_logs_created ON public.voice_auth_logs USING btree (created_at);


--
-- Name: idx_voice_auth_logs_event_type; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_auth_logs_event_type ON public.voice_auth_logs USING btree (event_type);


--
-- Name: idx_voice_auth_logs_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_auth_logs_tenant ON public.voice_auth_logs USING btree (tenant_id);


--
-- Name: idx_voice_auth_logs_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_auth_logs_user ON public.voice_auth_logs USING btree (user_id);


--
-- Name: idx_voice_auth_sessions_email; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_auth_sessions_email ON public.voice_auth_sessions USING btree (email);


--
-- Name: idx_voice_auth_sessions_expires; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_auth_sessions_expires ON public.voice_auth_sessions USING btree (expires_at);


--
-- Name: idx_voice_auth_sessions_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_auth_sessions_tenant ON public.voice_auth_sessions USING btree (tenant_id);


--
-- Name: idx_voice_auth_sessions_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_auth_sessions_user ON public.voice_auth_sessions USING btree (user_id);


--
-- Name: idx_voice_identity_links_is_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_identity_links_is_active ON public.voice_identity_links USING btree (is_active);


--
-- Name: idx_voice_identity_links_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_identity_links_tenant_id ON public.voice_identity_links USING btree (tenant_id);


--
-- Name: idx_voice_identity_links_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_identity_links_user_id ON public.voice_identity_links USING btree (user_id);


--
-- Name: idx_voice_identity_links_voice_platform_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_identity_links_voice_platform_user ON public.voice_identity_links USING btree (voice_platform, voice_user_id);


--
-- Name: idx_voice_profiles_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_profiles_tenant ON public.voice_profiles USING btree (tenant_id);


--
-- Name: idx_voice_profiles_user_tenant; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_profiles_user_tenant ON public.voice_profiles USING btree (user_id, tenant_id);


--
-- Name: idx_voice_sessions_approval_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_sessions_approval_status ON public.voice_sessions USING btree (approval_status);


--
-- Name: idx_voice_sessions_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_sessions_expires_at ON public.voice_sessions USING btree (expires_at);


--
-- Name: idx_voice_sessions_pending_approval; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_sessions_pending_approval ON public.voice_sessions USING btree (tenant_id, pending_approval) WHERE (pending_approval = true);


--
-- Name: idx_voice_sessions_session_token; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_sessions_session_token ON public.voice_sessions USING btree (session_token);


--
-- Name: idx_voice_sessions_status; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_sessions_status ON public.voice_sessions USING btree (status);


--
-- Name: idx_voice_sessions_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_sessions_tenant_id ON public.voice_sessions USING btree (tenant_id);


--
-- Name: idx_voice_sessions_voice_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voice_sessions_voice_user_id ON public.voice_sessions USING btree (voice_user_id);


--
-- Name: idx_voiceprints_active; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voiceprints_active ON public.voiceprints USING btree (active);


--
-- Name: idx_voiceprints_tenant_user; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voiceprints_tenant_user ON public.voiceprints USING btree (tenant_id, user_id);


--
-- Name: idx_voiceprints_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_voiceprints_user_id ON public.voiceprints USING btree (user_id);


--
-- Name: idx_webauthn_credentials_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_webauthn_credentials_user_id ON public.webauthn_credentials USING btree (user_id);


--
-- Name: idx_webauthn_sessions_created_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_webauthn_sessions_created_at ON public.webauthn_sessions USING btree (created_at);


--
-- Name: idx_webauthn_sessions_expires_at; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_webauthn_sessions_expires_at ON public.webauthn_sessions USING btree (expires_at);


--
-- Name: idx_webauthn_sessions_user_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_webauthn_sessions_user_id ON public.webauthn_sessions USING btree (user_id);


--
-- Name: idx_workload_entries_tenant_id; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_workload_entries_tenant_id ON public.workload_entries USING btree (tenant_id);


--
-- Name: idx_workload_entries_tenant_parent; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX idx_workload_entries_tenant_parent ON public.workload_entries USING btree (tenant_id, parent_id);


--
-- Name: oidc_user_identities_provider_name_provider_user_id_key; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX oidc_user_identities_provider_name_provider_user_id_key ON public.oidc_user_identities USING btree (provider_name, provider_user_id);


--
-- Name: oidc_user_identities_tenant_id_user_id_provider_name_key; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX oidc_user_identities_tenant_id_user_id_provider_name_key ON public.oidc_user_identities USING btree (tenant_id, user_id, provider_name);


--
-- Name: roles_name_tenant_unique; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX roles_name_tenant_unique ON public.roles USING btree (name, tenant_id);


--
-- Name: schema_migration_version_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX schema_migration_version_idx ON public.schema_migration USING btree (version);


--
-- Name: schema_migration_version_self_idx; Type: INDEX; Schema: public; Owner: authprod
--

CREATE INDEX schema_migration_version_self_idx ON public.schema_migration USING btree (version_self);


--
-- Name: scopes_name_tenant_unique; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX scopes_name_tenant_unique ON public.scopes USING btree (name, tenant_id);


--
-- Name: uq_delegation_token_client; Type: INDEX; Schema: public; Owner: authprod
--

CREATE UNIQUE INDEX uq_delegation_token_client ON public.delegation_tokens USING btree (tenant_id, client_id);


--
-- Name: tenant_certificate_authorities check_ca_expiration_trigger; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER check_ca_expiration_trigger BEFORE UPDATE ON public.tenant_certificate_authorities FOR EACH ROW EXECUTE FUNCTION public.check_ca_expiration();


--
-- Name: m2m_certificates check_cert_expiration_trigger; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER check_cert_expiration_trigger BEFORE UPDATE ON public.m2m_certificates FOR EACH ROW EXECUTE FUNCTION public.check_cert_expiration();


--
-- Name: oidc_providers oidc_providers_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER oidc_providers_updated_at BEFORE UPDATE ON public.oidc_providers FOR EACH ROW EXECUTE FUNCTION public.update_oidc_providers_updated_at();


--
-- Name: oidc_user_identities oidc_user_identities_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER oidc_user_identities_updated_at BEFORE UPDATE ON public.oidc_user_identities FOR EACH ROW EXECUTE FUNCTION public.update_oidc_user_identities_updated_at();


--
-- Name: saml_providers saml_providers_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER saml_providers_updated_at BEFORE UPDATE ON public.saml_providers FOR EACH ROW EXECUTE FUNCTION public.update_saml_providers_updated_at();


--
-- Name: device_codes trigger_device_codes_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER trigger_device_codes_updated_at BEFORE UPDATE ON public.device_codes FOR EACH ROW EXECUTE FUNCTION public.update_device_codes_updated_at();


--
-- Name: m2m_agent_policies trigger_m2m_agent_policies_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER trigger_m2m_agent_policies_updated_at BEFORE UPDATE ON public.m2m_agent_policies FOR EACH ROW EXECUTE FUNCTION public.update_m2m_agent_updated_at();


--
-- Name: m2m_agents trigger_m2m_agents_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER trigger_m2m_agents_updated_at BEFORE UPDATE ON public.m2m_agents FOR EACH ROW EXECUTE FUNCTION public.update_m2m_agent_updated_at();


--
-- Name: voice_active_sessions trigger_voice_active_sessions_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER trigger_voice_active_sessions_updated_at BEFORE UPDATE ON public.voice_active_sessions FOR EACH ROW EXECUTE FUNCTION public.update_voice_active_sessions_updated_at();


--
-- Name: voice_identity_links trigger_voice_identity_links_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER trigger_voice_identity_links_updated_at BEFORE UPDATE ON public.voice_identity_links FOR EACH ROW EXECUTE FUNCTION public.update_voice_identity_links_updated_at();


--
-- Name: voice_sessions trigger_voice_sessions_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER trigger_voice_sessions_updated_at BEFORE UPDATE ON public.voice_sessions FOR EACH ROW EXECUTE FUNCTION public.update_voice_sessions_updated_at();


--
-- Name: group_roles update_group_roles_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_group_roles_updated_at BEFORE UPDATE ON public.group_roles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: m2m_certificates update_m2m_certificates_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_m2m_certificates_updated_at BEFORE UPDATE ON public.m2m_certificates FOR EACH ROW EXECUTE FUNCTION public.update_m2m_updated_at();


--
-- Name: m2m_credentials update_m2m_credentials_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_m2m_credentials_updated_at BEFORE UPDATE ON public.m2m_credentials FOR EACH ROW EXECUTE FUNCTION public.update_m2m_updated_at();


--
-- Name: m2m_workloads update_m2m_workloads_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_m2m_workloads_updated_at BEFORE UPDATE ON public.m2m_workloads FOR EACH ROW EXECUTE FUNCTION public.update_m2m_updated_at();


--
-- Name: services update_services_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_services_updated_at BEFORE UPDATE ON public.services FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: tenant_ca_policies update_tenant_ca_policies_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_tenant_ca_policies_updated_at BEFORE UPDATE ON public.tenant_ca_policies FOR EACH ROW EXECUTE FUNCTION public.update_m2m_updated_at();


--
-- Name: tenant_certificate_authorities update_tenant_cas_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_tenant_cas_updated_at BEFORE UPDATE ON public.tenant_certificate_authorities FOR EACH ROW EXECUTE FUNCTION public.update_m2m_updated_at();


--
-- Name: tenant_hydra_clients update_tenant_hydra_clients_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_tenant_hydra_clients_updated_at BEFORE UPDATE ON public.tenant_hydra_clients FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: user_groups update_user_groups_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_user_groups_updated_at BEFORE UPDATE ON public.user_groups FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: user_roles update_user_roles_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_user_roles_updated_at BEFORE UPDATE ON public.user_roles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: user_scopes update_user_scopes_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER update_user_scopes_updated_at BEFORE UPDATE ON public.user_scopes FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: users users_set_updated_at; Type: TRIGGER; Schema: public; Owner: authprod
--

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: api_scope_permissions api_scope_permissions_scope_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.api_scope_permissions
    ADD CONSTRAINT api_scope_permissions_scope_id_fkey FOREIGN KEY (scope_id) REFERENCES public.api_scopes(id) ON DELETE CASCADE;


--
-- Name: certificate_revocation_list certificate_revocation_list_ca_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.certificate_revocation_list
    ADD CONSTRAINT certificate_revocation_list_ca_id_fkey FOREIGN KEY (ca_id) REFERENCES public.tenant_certificate_authorities(id) ON DELETE CASCADE;


--
-- Name: certificate_revocation_list certificate_revocation_list_cert_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.certificate_revocation_list
    ADD CONSTRAINT certificate_revocation_list_cert_id_fkey FOREIGN KEY (cert_id) REFERENCES public.m2m_certificates(id) ON DELETE CASCADE;


--
-- Name: delegation_tokens delegation_tokens_policy_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.delegation_tokens
    ADD CONSTRAINT delegation_tokens_policy_id_fkey FOREIGN KEY (policy_id) REFERENCES public.delegation_policies(id);


--
-- Name: device_codes device_codes_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_codes
    ADD CONSTRAINT device_codes_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: user_auth_preferences fk_auth_pref_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_auth_preferences
    ADD CONSTRAINT fk_auth_pref_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: user_auth_preferences fk_auth_pref_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_auth_preferences
    ADD CONSTRAINT fk_auth_pref_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id) ON DELETE CASCADE;


--
-- Name: totp_backup_codes fk_backup_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.totp_backup_codes
    ADD CONSTRAINT fk_backup_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: totp_backup_codes fk_backup_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.totp_backup_codes
    ADD CONSTRAINT fk_backup_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id) ON DELETE CASCADE;


--
-- Name: ciba_auth_requests fk_ciba_auth_device; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.ciba_auth_requests
    ADD CONSTRAINT fk_ciba_auth_device FOREIGN KEY (device_token_id) REFERENCES public.device_tokens(id) ON DELETE CASCADE;


--
-- Name: ciba_auth_requests fk_ciba_auth_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.ciba_auth_requests
    ADD CONSTRAINT fk_ciba_auth_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: ciba_auth_requests fk_ciba_auth_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.ciba_auth_requests
    ADD CONSTRAINT fk_ciba_auth_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id) ON DELETE CASCADE;


--
-- Name: ciba_requests fk_ciba_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.ciba_requests
    ADD CONSTRAINT fk_ciba_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: ciba_requests fk_ciba_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.ciba_requests
    ADD CONSTRAINT fk_ciba_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id) ON DELETE CASCADE;


--
-- Name: device_sessions fk_device_sessions_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT fk_device_sessions_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: device_sessions fk_device_sessions_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT fk_device_sessions_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: device_tokens fk_device_token_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_tokens
    ADD CONSTRAINT fk_device_token_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: device_tokens fk_device_token_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.device_tokens
    ADD CONSTRAINT fk_device_token_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id) ON DELETE CASCADE;


--
-- Name: tenant_totp_backup_codes fk_tenant_backup_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_totp_backup_codes
    ADD CONSTRAINT fk_tenant_backup_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: tenant_totp_backup_codes fk_tenant_backup_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_totp_backup_codes
    ADD CONSTRAINT fk_tenant_backup_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id) ON DELETE CASCADE;


--
-- Name: tenant_ciba_auth_requests fk_tenant_ciba_device; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_ciba_auth_requests
    ADD CONSTRAINT fk_tenant_ciba_device FOREIGN KEY (device_token_id, tenant_id) REFERENCES public.tenant_device_tokens(id, tenant_id) ON DELETE CASCADE;


--
-- Name: tenant_ciba_auth_requests fk_tenant_ciba_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_ciba_auth_requests
    ADD CONSTRAINT fk_tenant_ciba_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: tenant_ciba_auth_requests fk_tenant_ciba_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_ciba_auth_requests
    ADD CONSTRAINT fk_tenant_ciba_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id) ON DELETE CASCADE;


--
-- Name: tenant_device_tokens fk_tenant_device_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_device_tokens
    ADD CONSTRAINT fk_tenant_device_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: tenant_device_tokens fk_tenant_device_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_device_tokens
    ADD CONSTRAINT fk_tenant_device_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id) ON DELETE CASCADE;


--
-- Name: tenant_domains fk_tenant_domains_tenant_id; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_domains
    ADD CONSTRAINT fk_tenant_domains_tenant_id FOREIGN KEY (tenant_id) REFERENCES public.tenants(tenant_id) ON DELETE CASCADE;


--
-- Name: tenant_totp_secrets fk_tenant_totp_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_totp_secrets
    ADD CONSTRAINT fk_tenant_totp_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: tenant_totp_secrets fk_tenant_totp_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_totp_secrets
    ADD CONSTRAINT fk_tenant_totp_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id) ON DELETE CASCADE;


--
-- Name: totp_secrets fk_totp_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.totp_secrets
    ADD CONSTRAINT fk_totp_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: totp_secrets fk_totp_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.totp_secrets
    ADD CONSTRAINT fk_totp_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id) ON DELETE CASCADE;


--
-- Name: user_groups fk_user_groups_group; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_groups
    ADD CONSTRAINT fk_user_groups_group FOREIGN KEY (group_id) REFERENCES public.groups(id) ON DELETE CASCADE;


--
-- Name: user_groups fk_user_groups_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_groups
    ADD CONSTRAINT fk_user_groups_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: user_resources fk_user_resources_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_resources
    ADD CONSTRAINT fk_user_resources_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: user_scopes fk_user_scopes_user; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.user_scopes
    ADD CONSTRAINT fk_user_scopes_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: workload_entries fk_workload_entry_tenant; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.workload_entries
    ADD CONSTRAINT fk_workload_entry_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: grant_audit grant_audit_actor_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.grant_audit
    ADD CONSTRAINT grant_audit_actor_user_id_fkey FOREIGN KEY (actor_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: grant_audit grant_audit_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.grant_audit
    ADD CONSTRAINT grant_audit_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: hydra_client hydra_client_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_client
    ADD CONSTRAINT hydra_client_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_jwk hydra_jwk_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_jwk
    ADD CONSTRAINT hydra_jwk_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_access hydra_oauth2_access_challenge_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_access
    ADD CONSTRAINT hydra_oauth2_access_challenge_id_fk FOREIGN KEY (challenge_id) REFERENCES public.hydra_oauth2_flow(consent_challenge_id) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_access hydra_oauth2_access_client_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_access
    ADD CONSTRAINT hydra_oauth2_access_client_id_fk FOREIGN KEY (client_id, nid) REFERENCES public.hydra_client(id, nid) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_access hydra_oauth2_access_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_access
    ADD CONSTRAINT hydra_oauth2_access_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_authentication_session hydra_oauth2_authentication_session_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_authentication_session
    ADD CONSTRAINT hydra_oauth2_authentication_session_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_code hydra_oauth2_code_challenge_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_code
    ADD CONSTRAINT hydra_oauth2_code_challenge_id_fk FOREIGN KEY (challenge_id) REFERENCES public.hydra_oauth2_flow(consent_challenge_id) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_code hydra_oauth2_code_client_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_code
    ADD CONSTRAINT hydra_oauth2_code_client_id_fk FOREIGN KEY (client_id, nid) REFERENCES public.hydra_client(id, nid) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_code hydra_oauth2_code_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_code
    ADD CONSTRAINT hydra_oauth2_code_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_flow hydra_oauth2_flow_client_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_flow
    ADD CONSTRAINT hydra_oauth2_flow_client_id_fk FOREIGN KEY (client_id, nid) REFERENCES public.hydra_client(id, nid) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_flow hydra_oauth2_flow_login_session_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_flow
    ADD CONSTRAINT hydra_oauth2_flow_login_session_id_fk FOREIGN KEY (login_session_id) REFERENCES public.hydra_oauth2_authentication_session(id) ON DELETE SET NULL;


--
-- Name: hydra_oauth2_flow hydra_oauth2_flow_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_flow
    ADD CONSTRAINT hydra_oauth2_flow_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_jti_blacklist hydra_oauth2_jti_blacklist_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_jti_blacklist
    ADD CONSTRAINT hydra_oauth2_jti_blacklist_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_logout_request hydra_oauth2_logout_request_client_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_logout_request
    ADD CONSTRAINT hydra_oauth2_logout_request_client_id_fk FOREIGN KEY (client_id, nid) REFERENCES public.hydra_client(id, nid) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_logout_request hydra_oauth2_logout_request_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_logout_request
    ADD CONSTRAINT hydra_oauth2_logout_request_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_obfuscated_authentication_session hydra_oauth2_obfuscated_authentication_session_client_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_obfuscated_authentication_session
    ADD CONSTRAINT hydra_oauth2_obfuscated_authentication_session_client_id_fk FOREIGN KEY (client_id, nid) REFERENCES public.hydra_client(id, nid) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_obfuscated_authentication_session hydra_oauth2_obfuscated_authentication_session_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_obfuscated_authentication_session
    ADD CONSTRAINT hydra_oauth2_obfuscated_authentication_session_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_oidc hydra_oauth2_oidc_challenge_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_oidc
    ADD CONSTRAINT hydra_oauth2_oidc_challenge_id_fk FOREIGN KEY (challenge_id) REFERENCES public.hydra_oauth2_flow(consent_challenge_id) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_oidc hydra_oauth2_oidc_client_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_oidc
    ADD CONSTRAINT hydra_oauth2_oidc_client_id_fk FOREIGN KEY (client_id, nid) REFERENCES public.hydra_client(id, nid) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_oidc hydra_oauth2_oidc_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_oidc
    ADD CONSTRAINT hydra_oauth2_oidc_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_pkce hydra_oauth2_pkce_challenge_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_pkce
    ADD CONSTRAINT hydra_oauth2_pkce_challenge_id_fk FOREIGN KEY (challenge_id) REFERENCES public.hydra_oauth2_flow(consent_challenge_id) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_pkce hydra_oauth2_pkce_client_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_pkce
    ADD CONSTRAINT hydra_oauth2_pkce_client_id_fk FOREIGN KEY (client_id, nid) REFERENCES public.hydra_client(id, nid) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_pkce hydra_oauth2_pkce_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_pkce
    ADD CONSTRAINT hydra_oauth2_pkce_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_refresh hydra_oauth2_refresh_challenge_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_refresh
    ADD CONSTRAINT hydra_oauth2_refresh_challenge_id_fk FOREIGN KEY (challenge_id) REFERENCES public.hydra_oauth2_flow(consent_challenge_id) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_refresh hydra_oauth2_refresh_client_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_refresh
    ADD CONSTRAINT hydra_oauth2_refresh_client_id_fk FOREIGN KEY (client_id, nid) REFERENCES public.hydra_client(id, nid) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_refresh hydra_oauth2_refresh_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_refresh
    ADD CONSTRAINT hydra_oauth2_refresh_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: hydra_oauth2_trusted_jwt_bearer_issuer hydra_oauth2_trusted_jwt_bearer_issuer_key_set_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_trusted_jwt_bearer_issuer
    ADD CONSTRAINT hydra_oauth2_trusted_jwt_bearer_issuer_key_set_fkey FOREIGN KEY (key_set, key_id, nid) REFERENCES public.hydra_jwk(sid, kid, nid) ON DELETE CASCADE;


--
-- Name: hydra_oauth2_trusted_jwt_bearer_issuer hydra_oauth2_trusted_jwt_bearer_issuer_nid_fk_idx; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.hydra_oauth2_trusted_jwt_bearer_issuer
    ADD CONSTRAINT hydra_oauth2_trusted_jwt_bearer_issuer_nid_fk_idx FOREIGN KEY (nid) REFERENCES public.networks(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: m2m_agent_attestations m2m_agent_attestations_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_attestations
    ADD CONSTRAINT m2m_agent_attestations_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.m2m_agents(id) ON DELETE CASCADE;


--
-- Name: m2m_agent_attestations m2m_agent_attestations_issued_certificate_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_attestations
    ADD CONSTRAINT m2m_agent_attestations_issued_certificate_id_fkey FOREIGN KEY (issued_certificate_id) REFERENCES public.m2m_certificates(id) ON DELETE SET NULL;


--
-- Name: m2m_agent_attestations m2m_agent_attestations_workload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_attestations
    ADD CONSTRAINT m2m_agent_attestations_workload_id_fkey FOREIGN KEY (workload_id) REFERENCES public.m2m_workloads(id) ON DELETE CASCADE;


--
-- Name: m2m_agent_certificate_renewals m2m_agent_certificate_renewals_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_certificate_renewals
    ADD CONSTRAINT m2m_agent_certificate_renewals_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.m2m_agents(id) ON DELETE CASCADE;


--
-- Name: m2m_agent_certificate_renewals m2m_agent_certificate_renewals_new_certificate_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_certificate_renewals
    ADD CONSTRAINT m2m_agent_certificate_renewals_new_certificate_id_fkey FOREIGN KEY (new_certificate_id) REFERENCES public.m2m_certificates(id) ON DELETE SET NULL;


--
-- Name: m2m_agent_certificate_renewals m2m_agent_certificate_renewals_old_certificate_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_certificate_renewals
    ADD CONSTRAINT m2m_agent_certificate_renewals_old_certificate_id_fkey FOREIGN KEY (old_certificate_id) REFERENCES public.m2m_certificates(id) ON DELETE SET NULL;


--
-- Name: m2m_agent_certificate_renewals m2m_agent_certificate_renewals_workload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_certificate_renewals
    ADD CONSTRAINT m2m_agent_certificate_renewals_workload_id_fkey FOREIGN KEY (workload_id) REFERENCES public.m2m_workloads(id) ON DELETE CASCADE;


--
-- Name: m2m_agent_deployment_tokens m2m_agent_deployment_tokens_used_by_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_deployment_tokens
    ADD CONSTRAINT m2m_agent_deployment_tokens_used_by_agent_id_fkey FOREIGN KEY (used_by_agent_id) REFERENCES public.m2m_agents(id) ON DELETE SET NULL;


--
-- Name: m2m_agent_deployment_tokens m2m_agent_deployment_tokens_workload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_deployment_tokens
    ADD CONSTRAINT m2m_agent_deployment_tokens_workload_id_fkey FOREIGN KEY (workload_id) REFERENCES public.m2m_workloads(id) ON DELETE CASCADE;


--
-- Name: m2m_agent_heartbeats m2m_agent_heartbeats_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_heartbeats
    ADD CONSTRAINT m2m_agent_heartbeats_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.m2m_agents(id) ON DELETE CASCADE;


--
-- Name: m2m_agent_policies m2m_agent_policies_workload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agent_policies
    ADD CONSTRAINT m2m_agent_policies_workload_id_fkey FOREIGN KEY (workload_id) REFERENCES public.m2m_workloads(id) ON DELETE CASCADE;


--
-- Name: m2m_agents m2m_agents_current_certificate_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agents
    ADD CONSTRAINT m2m_agents_current_certificate_id_fkey FOREIGN KEY (current_certificate_id) REFERENCES public.m2m_certificates(id) ON DELETE SET NULL;


--
-- Name: m2m_agents m2m_agents_workload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_agents
    ADD CONSTRAINT m2m_agents_workload_id_fkey FOREIGN KEY (workload_id) REFERENCES public.m2m_workloads(id) ON DELETE CASCADE;


--
-- Name: m2m_attestation_logs m2m_attestation_logs_credential_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_attestation_logs
    ADD CONSTRAINT m2m_attestation_logs_credential_id_fkey FOREIGN KEY (credential_id) REFERENCES public.m2m_credentials(id) ON DELETE SET NULL;


--
-- Name: m2m_attestation_logs m2m_attestation_logs_workload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_attestation_logs
    ADD CONSTRAINT m2m_attestation_logs_workload_id_fkey FOREIGN KEY (workload_id) REFERENCES public.m2m_workloads(id) ON DELETE SET NULL;


--
-- Name: m2m_certificates m2m_certificates_ca_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_certificates
    ADD CONSTRAINT m2m_certificates_ca_id_fkey FOREIGN KEY (ca_id) REFERENCES public.tenant_certificate_authorities(id) ON DELETE RESTRICT;


--
-- Name: m2m_certificates m2m_certificates_replaced_by_cert_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_certificates
    ADD CONSTRAINT m2m_certificates_replaced_by_cert_id_fkey FOREIGN KEY (replaced_by_cert_id) REFERENCES public.m2m_certificates(id);


--
-- Name: m2m_certificates m2m_certificates_workload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_certificates
    ADD CONSTRAINT m2m_certificates_workload_id_fkey FOREIGN KEY (workload_id) REFERENCES public.m2m_workloads(id) ON DELETE CASCADE;


--
-- Name: m2m_credentials m2m_credentials_workload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_credentials
    ADD CONSTRAINT m2m_credentials_workload_id_fkey FOREIGN KEY (workload_id) REFERENCES public.m2m_workloads(id) ON DELETE CASCADE;


--
-- Name: m2m_workloads m2m_workloads_ca_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.m2m_workloads
    ADD CONSTRAINT m2m_workloads_ca_id_fkey FOREIGN KEY (ca_id) REFERENCES public.tenant_certificate_authorities(id) ON DELETE RESTRICT;


--
-- Name: role_permissions role_permissions_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: scope_permissions scope_permissions_permission_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.scope_permissions
    ADD CONSTRAINT scope_permissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES public.permissions(id) ON DELETE CASCADE;


--
-- Name: scope_permissions scope_permissions_scope_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.scope_permissions
    ADD CONSTRAINT scope_permissions_scope_id_fkey FOREIGN KEY (scope_id) REFERENCES public.scopes(id) ON DELETE CASCADE;


--
-- Name: scopes scopes_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.scopes
    ADD CONSTRAINT scopes_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: service_accounts service_accounts_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.service_accounts
    ADD CONSTRAINT service_accounts_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: tenant_ca_policies tenant_ca_policies_ca_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.tenant_ca_policies
    ADD CONSTRAINT tenant_ca_policies_ca_id_fkey FOREIGN KEY (ca_id) REFERENCES public.tenant_certificate_authorities(id) ON DELETE CASCADE;


--
-- Name: trust_bundle_cas trust_bundle_cas_bundle_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.trust_bundle_cas
    ADD CONSTRAINT trust_bundle_cas_bundle_id_fkey FOREIGN KEY (bundle_id) REFERENCES public.trust_bundles(id) ON DELETE CASCADE;


--
-- Name: trust_bundle_cas trust_bundle_cas_ca_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.trust_bundle_cas
    ADD CONSTRAINT trust_bundle_cas_ca_id_fkey FOREIGN KEY (ca_id) REFERENCES public.tenant_certificate_authorities(id) ON DELETE CASCADE;


--
-- Name: trust_bundles trust_bundles_signed_by_ca_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.trust_bundles
    ADD CONSTRAINT trust_bundles_signed_by_ca_id_fkey FOREIGN KEY (signed_by_ca_id) REFERENCES public.tenant_certificate_authorities(id);


--
-- Name: voice_active_sessions voice_active_sessions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_active_sessions
    ADD CONSTRAINT voice_active_sessions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: voice_identity_links voice_identity_links_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_identity_links
    ADD CONSTRAINT voice_identity_links_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: voice_sessions voice_sessions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: authprod
--

ALTER TABLE ONLY public.voice_sessions
    ADD CONSTRAINT voice_sessions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--



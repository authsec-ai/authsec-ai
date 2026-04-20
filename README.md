# AuthSec – Identity & Access Management Platform

![Go Version](https://img.shields.io/badge/Go-1.25-blue)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

AuthSec is a unified Go service for the complete identity lifecycle: authentication, MFA, OIDC federation, RBAC, SCIM provisioning, client management, external-service credentials, and SPIFFE/SPIRE workload identity — all served from a single binary.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Modules](#modules)
- [Quick Start](#quick-start)
- [Environment Variables](#environment-variables)
- [API Route Map](#api-route-map)
  - [Core Auth & User Flow (`/authsec/uflow`)](#core-auth--user-flow-authsecuflow)
  - [WebAuthn / Passkeys (`/authsec/webauthn`)](#webauthn--passkeys-authsecwebauthn)
  - [Client Management (`/authsec/clientms`)](#client-management-authsecclientms)
  - [Hydra Manager (`/authsec/hmgr`)](#hydra-manager-authsechmgr)
  - [OIDC Config Manager (`/authsec/oocmgr`)](#oidc-config-manager-authsecoocmgr)
  - [Auth Manager (`/authsec/authmgr`)](#auth-manager-authsecauthmgr)
  - [External Services (`/authsec/exsvc`)](#external-services-authsecexsvc)
  - [SPIRE Headless (`/authsec/spire`)](#spire-headless-authsecspire)
  - [Migration Management (`/authsec/migration`)](#migration-management-authsecmigration)
  - [Well-Known Endpoints](#well-known-endpoints)
  - [Metrics](#metrics)
- [Authentication & Middleware](#authentication--middleware)
- [Database Configuration](#database-configuration)
- [Internal Package Layout](#internal-package-layout)
- [Background Workers](#background-workers)
- [Building & Running](#building--running)
- [Contributing](#contributing)

---

## Architecture Overview

```text
┌────────────────────────────────────────────────────────────────────┐
│                          authsec  (port 7468)                      │
│                                                                    │
│  /authsec/uflow/*        – Auth, RBAC, OIDC federation, SCIM      │
│  /authsec/webauthn/*     – Passkeys, TOTP, SMS MFA                │
│  /authsec/clientms/*     – Client lifecycle management            │
│  /authsec/hmgr/*         – Ory Hydra login/consent, SAML SSO      │
│  /authsec/oocmgr/*       – OIDC provider config & Hydra sync      │
│  /authsec/authmgr/*      – JWT verification, RBAC checks          │
│  /authsec/exsvc/*        – External service registry              │
│  /authsec/spire/*        – SPIFFE workload identity               │
│  /authsec/migration/*    – Database migration management          │
│                                                                    │
│  /.well-known/*          – OIDC discovery (RFC 8414 root path)    │
│  /metrics                – Prometheus metrics                     │
└────────────────────────────────────────────────────────────────────┘
         │
         ├── PostgreSQL (primary DB + per-tenant DBs)
         ├── HashiCorp Vault (secrets, OIDC provider credentials)
         └── Redis (optional – permission cache, session cache)
```

All HTTP routes are served from a single `gin.Engine`. Each module's routes live under its own sub-prefix so paths are globally unique.

---

## Modules

| Module | Sub-prefix | Description |
| --- | --- | --- |
| Core Auth & User Flow | `/authsec/uflow` | Admin/enduser login, RBAC, OIDC federation, SCIM, TOTP, CIBA, voice auth |
| WebAuthn / Passkeys | `/authsec/webauthn` | WebAuthn/FIDO2 passkeys, TOTP setup, SMS MFA |
| Client Management | `/authsec/clientms` | Hydra client lifecycle management |
| Hydra Manager | `/authsec/hmgr` | Ory Hydra login/consent, SAML SSO, token exchange |
| OIDC Config Manager | `/authsec/oocmgr` | OIDC provider config, Hydra client sync, SAML providers |
| Auth Manager | `/authsec/authmgr` | JWT verify/issue, RBAC permission checks, group management |
| External Services | `/authsec/exsvc` | External service registry with Vault-backed credentials |
| SPIRE Headless | `/authsec/spire` | SPIFFE/SPIRE workload identity, OIDC token exchange, cloud federation (AWS/Azure/GCP), RBAC/ABAC policy engine |
| Migration Management | `/authsec/migration` | Database migration management (master DB + per-tenant DB) |

---

## Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL 14+ (primary DB + tenant DBs)
- HashiCorp Vault (optional but recommended for OIDC secrets)
- Redis (optional, for caching)

### Run locally

```bash
# Copy and edit environment variables
cp .env.example .env

# Build
go build -o authsec ./cmd/

# Run
./authsec
```

Or with `go run`:

```bash
go run ./cmd/
```

The server starts on port **7468** by default.

---

## Environment Variables

### Required

| Variable | Description | Example |
| --- | --- | --- |
| `DB_NAME` | PostgreSQL database name | `authsec_db` |
| `DB_USER` | Database username | `authsec` |
| `DB_PASSWORD` | Database password | `changeme` |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `WEBAUTHN_RP_NAME` | WebAuthn relying party display name | `AuthSec` |
| `WEBAUTHN_RP_ID` | WebAuthn relying party ID (must match origin's hostname) | `app.authsec.dev` |
| `WEBAUTHN_ORIGIN` | Allowed WebAuthn origin | `https://app.authsec.dev` |

### Optional – Core Service

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `7468` | HTTP listen port |
| `GIN_MODE` | `debug` | Gin run mode (`debug` / `release` / `test`) |
| `ENVIRONMENT` | `development` | Runtime label used by tenant domain checks (`development` / `production`) |
| `DB_SCHEMA` | `public` | PostgreSQL schema |
| `JWT_SECRET` | `""` | Primary JWT signing secret (ext-service routes, SPIFFE delegate) |
| `JWT_DEF_SECRET` | — | Default JWT signing secret (admin / platform tokens) |
| `JWT_SDK_SECRET` | — | SDK JWT signing secret |
| `BASE_URL` | `https://app.authsec.dev` | Base URL for OIDC callbacks and email links |
| `TENANT_DOMAIN_SUFFIX` | — | Suffix for auto-generated tenant sub-domains |
| `REDIS_URL` | `""` | Redis connection URL (e.g. `redis://localhost:6379`) |
| `ICP_SERVICE_URL` | `http://localhost:7001` | ICP/PKI provisioning service |
| `REQUIRE_SERVER_AUTH` | `true` | Enforce inter-service auth check (`false` to disable in dev) |
| `SKIP_MIGRATIONS` | `false` | Set to `true` to skip master DB migrations at startup |

### Optional – CORS

| Variable | Default | Description |
| --- | --- | --- |
| `CORS_ALLOWED_ORIGINS` | (auto-detect from `WEBAUTHN_ORIGIN`) | Comma-separated allowed origins |
| `CORS_ALLOWED_METHODS` | `GET,POST,PUT,PATCH,DELETE,OPTIONS` | Allowed HTTP methods |
| `CORS_ALLOWED_HEADERS` | `Origin,Content-Type,Authorization,…` | Allowed request headers |

### Optional – Encryption Keys

| Variable | Description |
| --- | --- |
| `TOTP_ENCRYPTION_KEY` | 64-hex-char AES-256 key for encrypting TOTP secrets at rest (required in production) |
| `SYNC_CONFIG_ENCRYPTION_KEY` | 64-hex-char AES-256 key for encrypting AD/Entra sync configurations at rest |

### Optional – Twilio (SMS MFA / Voice)

| Variable | Description |
| --- | --- |
| `TWILIO_ACCOUNT_SID` | Twilio account SID (e.g. `ACxxxxxxxx`) |
| `TWILIO_AUTH_TOKEN` | Twilio auth token |
| `TWILIO_FROM_NUMBER` | Sender phone number for SMS OTPs (e.g. `+10000000000`) |

### Optional – External Integrations

| Variable | Description |
| --- | --- |
| `VAULT_ADDR` | HashiCorp Vault address (default: `http://localhost:8200`) |
| `VAULT_TOKEN` | Vault root/service token |
| `HYDRA_ADMIN_URL` | Ory Hydra admin API (default: `http://localhost:4445`) |
| `HYDRA_PUBLIC_URL` | Ory Hydra public API (default: `http://localhost:4444`) |
| `REACT_APP_URL` | Frontend app URL for redirects |
| `IDENTITY_PROVIDER_URL` | Identity provider base URL |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` | SMTP for email notifications |
| `GOOGLE_CLIENT_SECRET` | Google OIDC client secret (fallback if Vault unavailable) |
| `GITHUB_CLIENT_SECRET` | GitHub OIDC client secret |
| `MICROSOFT_CLIENT_SECRET` | Microsoft OIDC client secret |
| `HUBSPOT_ACCESS_TOKEN` | HubSpot CRM integration token |

### Optional – OIDC Token Validation

| Variable | Description |
| --- | --- |
| `AUTH_EXPECT_ISS` | Expected `iss` claim when validating incoming OIDC tokens (empty = skip) |
| `AUTH_EXPECT_AUD` | Expected `aud` claim when validating incoming OIDC tokens (empty = skip) |

### Optional – SPIFFE / SVID OIDC

Required only when SPIFFE workload identity / delegate endpoints are used.

| Variable | Description |
| --- | --- |
| `SPIFFE_OIDC_ISSUER` | Issuer URL embedded in SPIFFE OIDC tokens |
| `SPIFFE_JWKS_KEY_ID` | Key ID used in the JWKS endpoint |
| `SPIFFE_RSA_PRIVATE_KEY_B64` | Base64-encoded PEM RSA private key for signing SPIFFE JWTs |
| `SPIFFE_TRUST_DOMAIN` | SPIFFE trust domain (e.g. `spiffe://example.org`) |

### Optional – Okta CIBA

Required only when Okta is used as a CIBA provider.

| Variable | Description |
| --- | --- |
| `OKTA_DOMAIN` | Okta domain (e.g. `dev-12345678.okta.com`) |
| `OKTA_CLIENT_ID` | Okta application client ID |
| `OKTA_CLIENT_SECRET` | Okta application client secret |
| `OKTA_ISSUER` | Okta issuer URL |
| `OKTA_API_TOKEN` | Okta API token for admin operations |

---

## API Route Map

All application routes are under the `/authsec` prefix (except OIDC discovery and `/metrics`).

### Core Auth & User Flow (`/authsec/uflow`)

#### Health

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/uflow/health` | Comprehensive health check |
| `GET` | `/authsec/uflow/health/tenant/:tenant_id` | Single tenant DB health |
| `GET` | `/authsec/uflow/health/tenants` | All tenant DBs health |

#### Admin Authentication (`/authsec/uflow/auth/admin`)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/uflow/auth/admin/challenge` | Get auth challenge |
| `POST` | `/authsec/uflow/auth/admin/login/precheck` | Pre-login check |
| `POST` | `/authsec/uflow/auth/admin/login/bootstrap` | Bootstrap first admin |
| `POST` | `/authsec/uflow/auth/admin/login` | Admin login |
| `POST` | `/authsec/uflow/auth/admin/login-hybrid` | Hybrid login |
| `POST` | `/authsec/uflow/auth/admin/register` | Register admin |
| `POST` | `/authsec/uflow/auth/admin/complete-registration` | Complete registration |
| `POST` | `/authsec/uflow/auth/admin/forgot-password` | Initiate password reset |
| `POST` | `/authsec/uflow/auth/admin/forgot-password/verify-otp` | Verify OTP |
| `POST` | `/authsec/uflow/auth/admin/forgot-password/reset` | Reset password |

#### End-User Authentication (`/authsec/uflow/auth/enduser`)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/uflow/auth/enduser/challenge` | Get challenge |
| `POST` | `/authsec/uflow/auth/enduser/initiate-registration` | Start registration |
| `POST` | `/authsec/uflow/auth/enduser/verify-otp` | Verify OTP + complete registration |
| `POST` | `/authsec/uflow/auth/enduser/login/precheck` | Pre-login check |
| `POST` | `/authsec/uflow/auth/enduser/webauthn-callback` | WebAuthn assertion callback |
| `POST` | `/authsec/uflow/auth/enduser/delegate-svid` | Delegate SPIFFE SVID |

#### Device Authorization Grant – RFC 8628 (`/authsec/uflow/auth/device`)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/authsec/uflow/auth/device/code` | Public | Device requests code |
| `POST` | `/authsec/uflow/auth/device/token` | Public | Device polls for token |
| `GET` | `/authsec/uflow/auth/device/activate/info` | Public | Get device info for UI |
| `POST` | `/authsec/uflow/auth/device/verify` | JWT | User authorises device |
| `GET` | `/authsec/uflow/activate` | Public | Activation UI page |

#### Voice Authentication (`/authsec/uflow/auth/voice`)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/authsec/uflow/auth/voice/initiate` | Public | Initiate voice auth |
| `POST` | `/authsec/uflow/auth/voice/verify` | Public | Verify voice OTP |
| `POST` | `/authsec/uflow/auth/voice/token` | Public | Get token with credentials |
| `POST` | `/authsec/uflow/auth/voice/link` | JWT | Link voice assistant |
| `POST` | `/authsec/uflow/auth/voice/unlink` | JWT | Unlink voice assistant |
| `GET` | `/authsec/uflow/auth/voice/links` | JWT | List linked assistants |
| `GET` | `/authsec/uflow/auth/voice/device-pending` | JWT | Get pending device codes |
| `POST` | `/authsec/uflow/auth/voice/device-approve` | JWT | Approve/deny device code |

#### TOTP – Platform (`/authsec/uflow/auth/totp`)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/authsec/uflow/auth/totp/login` | Public | Login with TOTP |
| `POST` | `/authsec/uflow/auth/totp/device-approve` | Public | Approve device with TOTP |
| `POST` | `/authsec/uflow/auth/totp/register` | JWT | Register TOTP device |
| `POST` | `/authsec/uflow/auth/totp/confirm` | JWT | Confirm TOTP registration |
| `POST` | `/authsec/uflow/auth/totp/verify` | JWT | Verify TOTP code |
| `GET` | `/authsec/uflow/auth/totp/devices` | JWT | List registered devices |
| `POST` | `/authsec/uflow/auth/totp/device/delete` | JWT | Delete TOTP device |
| `POST` | `/authsec/uflow/auth/totp/device/primary` | JWT | Set primary device |
| `POST` | `/authsec/uflow/auth/totp/backup/regenerate` | JWT | Regenerate backup codes |

#### CIBA – Platform (`/authsec/uflow/auth/ciba`)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/authsec/uflow/auth/ciba/initiate` | Public | Initiate CIBA flow |
| `POST` | `/authsec/uflow/auth/ciba/token` | Public | Poll for CIBA token |
| `POST` | `/authsec/uflow/auth/ciba/respond` | JWT | Respond to CIBA request |
| `POST` | `/authsec/uflow/auth/ciba/register-device` | JWT | Register push device |
| `GET` | `/authsec/uflow/auth/ciba/devices` | JWT | List push devices |
| `DELETE` | `/authsec/uflow/auth/ciba/devices/:device_id` | JWT | Delete push device |

#### Tenant TOTP / CIBA (`/authsec/uflow/auth/tenant`)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/authsec/uflow/auth/tenant/totp/login` | Public | Tenant TOTP login |
| `POST` | `/authsec/uflow/auth/tenant/totp/register` | JWT+Tenant | Register tenant TOTP device |
| `POST` | `/authsec/uflow/auth/tenant/totp/confirm` | JWT+Tenant | Confirm device |
| `GET` | `/authsec/uflow/auth/tenant/totp/devices` | JWT+Tenant | List devices |
| `POST` | `/authsec/uflow/auth/tenant/totp/devices/delete` | JWT+Tenant | Delete device |
| `POST` | `/authsec/uflow/auth/tenant/totp/devices/primary` | JWT+Tenant | Set primary |
| `POST` | `/authsec/uflow/auth/tenant/ciba/initiate` | Public | Initiate tenant CIBA |
| `POST` | `/authsec/uflow/auth/tenant/ciba/token` | Public | Poll tenant CIBA token |
| `POST` | `/authsec/uflow/auth/tenant/ciba/respond` | JWT+Tenant | Respond |
| `POST` | `/authsec/uflow/auth/tenant/ciba/register-device` | JWT+Tenant | Register device |
| `GET` | `/authsec/uflow/auth/tenant/ciba/requests` | JWT+Tenant | List pending requests |
| `GET` | `/authsec/uflow/auth/tenant/ciba/devices` | JWT+Tenant | List devices |
| `DELETE` | `/authsec/uflow/auth/tenant/ciba/devices/:device_id` | JWT+Tenant | Delete device |

#### OIDC Federation (`/authsec/uflow/oidc`)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/authsec/uflow/oidc/providers` | Public | List OIDC providers |
| `POST` | `/authsec/uflow/oidc/initiate` | Public | Initiate OIDC flow |
| `POST` | `/authsec/uflow/oidc/register/initiate` | Public | Initiate OIDC registration |
| `POST` | `/authsec/uflow/oidc/login/initiate` | Public | Initiate OIDC login |
| `GET` | `/authsec/uflow/oidc/callback` | Public | OIDC callback |
| `POST` | `/authsec/uflow/oidc/exchange-code` | Public | Exchange auth code |
| `POST` | `/authsec/uflow/oidc/complete-registration` | Public | Complete OIDC registration |
| `GET` | `/authsec/uflow/oidc/check-tenant` | Public | Check tenant exists |
| `POST` | `/authsec/uflow/oidc/auth-url` | Public | Get auth URL |
| `POST` | `/authsec/uflow/oidc/link` | JWT+Tenant | Link OIDC identity |
| `GET` | `/authsec/uflow/oidc/identities` | JWT+Tenant | List linked identities |
| `DELETE` | `/authsec/uflow/oidc/unlink/:provider` | JWT+Tenant | Unlink identity |

#### End-User Self-Service (`/authsec/uflow/user`)

Public endpoints (no auth required):

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/authsec/uflow/user/login` | Custom login |
| `POST` | `/authsec/uflow/user/login/status` | Login status |
| `POST` | `/authsec/uflow/user/saml/login` | SAML login |
| `POST` | `/authsec/uflow/user/register/initiate` | Initiate registration |
| `POST` | `/authsec/uflow/user/register/complete` | Complete registration |
| `POST` | `/authsec/uflow/user/register` | Direct registration |
| `POST` | `/authsec/uflow/user/forgot-password` | Request password reset |
| `POST` | `/authsec/uflow/user/forgot-password/verify-otp` | Verify reset OTP |
| `POST` | `/authsec/uflow/user/forgot-password/reset` | Reset password |
| `POST` | `/authsec/uflow/user/oidc/login` | OIDC login |

Authenticated endpoints (JWT + tenant required):

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/authsec/uflow/user/clients/register` | Register client |
| `GET` | `/authsec/uflow/user/clients` | List clients |
| `GET` | `/authsec/uflow/user/enduser/:tenant_id/:user_id` | Get end-user |
| `PUT` | `/authsec/uflow/user/enduser/:tenant_id/:user_id` | Update end-user |
| `DELETE` | `/authsec/uflow/user/enduser/:tenant_id/:user_id` | Delete end-user |
| `GET` | `/authsec/uflow/user/permissions` | My permissions |
| `GET` | `/authsec/uflow/user/permissions/effective` | My effective permissions |
| `GET` | `/authsec/uflow/user/permissions/check` | Check permission |
| `POST` | `/authsec/uflow/user/rbac/roles` | Create role |
| `GET` | `/authsec/uflow/user/rbac/roles` | List roles |
| `PUT` | `/authsec/uflow/user/rbac/roles/:role_id` | Update role |
| `DELETE` | `/authsec/uflow/user/rbac/roles/:role_id` | Delete role |
| `POST` | `/authsec/uflow/user/rbac/bindings` | Assign role |
| `GET` | `/authsec/uflow/user/rbac/bindings` | List bindings |
| `POST` | `/authsec/uflow/user/rbac/policy/check` | Policy decision check |
| `GET` | `/authsec/uflow/user/scopes` | List user scopes |
| `POST` | `/authsec/uflow/user/scopes` | Add scope |
| `POST` | `/authsec/uflow/user/api_scopes` | Create API scope |
| `GET` | `/authsec/uflow/user/api_scopes` | List API scopes |
| `POST` | `/authsec/uflow/user/groups/users/add` | Add user to group |
| `POST` | `/authsec/uflow/user/groups/users/remove` | Remove user from group |
| `GET` | `/authsec/uflow/user/groups/users` | My groups |

#### Admin Management (`/authsec/uflow/admin`)

All admin endpoints require `JWT + admin:access + tenant validation`.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/uflow/admin/tenants` | List tenants |
| `POST` | `/authsec/uflow/admin/tenants` | Create tenant |
| `PUT` | `/authsec/uflow/admin/tenants/:tenant_id` | Update tenant |
| `DELETE` | `/authsec/uflow/admin/tenants/:tenant_id` | Delete tenant |
| `GET` | `/authsec/uflow/admin/tenants/:tenant_id/users` | Get tenant users |
| `GET/POST` | `/authsec/uflow/admin/users/list` | List admin users |
| `DELETE` | `/authsec/uflow/admin/users/:user_id` | Delete admin user |
| `POST` | `/authsec/uflow/admin/enduser/list` | List end-users by tenant |
| `POST` | `/authsec/uflow/admin/invite` | Invite admin |
| `POST` | `/authsec/uflow/admin/invite/cancel` | Cancel invite |
| `POST` | `/authsec/uflow/admin/invite/resend` | Resend invite |
| `GET` | `/authsec/uflow/admin/invite/pending` | List pending invites |
| `POST` | `/authsec/uflow/admin/tenants/:tenant_id/domains` | Create domain |
| `GET` | `/authsec/uflow/admin/tenants/:tenant_id/domains` | List domains |
| `POST` | `/authsec/uflow/admin/tenants/:tenant_id/domains/:domain_id/verify` | Verify domain |
| `POST` | `/authsec/uflow/admin/oidc/providers` | Get all OIDC providers |
| `PUT` | `/authsec/uflow/admin/oidc/providers/:provider` | Update provider |
| `POST` | `/authsec/uflow/admin/projects` | Create project |
| `GET` | `/authsec/uflow/admin/projects` | List projects |
| `POST` | `/authsec/uflow/admin/groups` | Add user-defined groups |
| `POST` | `/authsec/uflow/admin/ad/sync` | Sync Active Directory users |
| `POST` | `/authsec/uflow/admin/entra/sync` | Sync Entra ID users |
| `POST` | `/authsec/uflow/admin/sync-configs/create` | Create sync config |
| `POST` | `/authsec/uflow/admin/scim/generate-token` | Generate SCIM token |

#### Admin RBAC (`/authsec/uflow/admin` – scoped bindings)

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/authsec/uflow/admin/roles` | Create role |
| `GET` | `/authsec/uflow/admin/roles` | List roles |
| `GET` | `/authsec/uflow/admin/roles/:role_id` | Get role by ID |
| `PUT` | `/authsec/uflow/admin/roles/:role_id` | Update role |
| `DELETE` | `/authsec/uflow/admin/roles/:role_id` | Delete role |
| `POST` | `/authsec/uflow/admin/bindings` | Assign role (scoped) |
| `GET` | `/authsec/uflow/admin/bindings` | List bindings |
| `POST` | `/authsec/uflow/admin/permissions` | Register permission |
| `GET` | `/authsec/uflow/admin/permissions` | List permissions |
| `DELETE` | `/authsec/uflow/admin/permissions/:id` | Delete permission |
| `GET` | `/authsec/uflow/admin/permissions/resources` | List resources |
| `GET/POST/PUT/DELETE` | `/authsec/uflow/admin/scopes` | Scope management |
| `GET` | `/authsec/uflow/admin/scopes/mappings` | Get scope mappings |
| `POST` | `/authsec/uflow/admin/policy/check` | Admin PDP check |
| `GET/POST/PUT/DELETE` | `/authsec/uflow/admin/api_scopes` | API scope management |

#### SCIM 2.0 (`/authsec/uflow/scim/v2`)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/authsec/uflow/scim/v2/ServiceProviderConfig` | Public | Service provider config |
| `GET` | `/authsec/uflow/scim/v2/Schemas` | Public | SCIM schemas |
| `GET` | `/authsec/uflow/scim/v2/ResourceTypes` | Public | Resource types |
| `GET/POST/PUT/PATCH/DELETE` | `/authsec/uflow/scim/v2/:client_id/:project_id/Users` | JWT+Tenant | User provisioning |
| `GET/POST/PUT/PATCH/DELETE` | `/authsec/uflow/scim/v2/:client_id/:project_id/Groups` | JWT+Tenant | Group provisioning |
| `GET/POST/PUT/PATCH/DELETE` | `/authsec/uflow/scim/v2/admin/Users` | JWT+Admin | Admin user provisioning |

---

### WebAuthn / Passkeys (`/authsec/webauthn`)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/webauthn/health` | Health check |
| `POST` | `/authsec/webauthn/admin/mfa/status` | Admin MFA status |
| `POST` | `/authsec/webauthn/admin/mfa/loginStatus` | Admin MFA login status |
| `GET` | `/authsec/webauthn/admin/mfa/loginStatus` | Admin MFA login status (GET) |
| `POST` | `/authsec/webauthn/admin/beginRegistration` | Begin admin WebAuthn registration |
| `POST` | `/authsec/webauthn/admin/finishRegistration` | Finish admin registration |
| `POST` | `/authsec/webauthn/admin/beginAuthentication` | Begin admin authentication |
| `POST` | `/authsec/webauthn/admin/finishAuthentication` | Finish admin authentication |
| `POST` | `/authsec/webauthn/enduser/mfa/status` | End-user MFA status |
| `POST` | `/authsec/webauthn/enduser/mfa/loginStatus` | End-user MFA login status |
| `GET` | `/authsec/webauthn/enduser/mfa/loginStatus` | End-user MFA login status (GET) |
| `POST` | `/authsec/webauthn/enduser/beginRegistration` | Begin end-user registration |
| `POST` | `/authsec/webauthn/enduser/finishRegistration` | Finish end-user registration |
| `POST` | `/authsec/webauthn/enduser/beginAuthentication` | Begin end-user authentication |
| `POST` | `/authsec/webauthn/enduser/finishAuthentication` | Finish end-user authentication |
| `POST` | `/authsec/webauthn/beginRegistration` | Registration (flat) |
| `POST` | `/authsec/webauthn/beginAuthentication` | Authentication (flat) |
| `POST` | `/authsec/webauthn/finishRegistration` | Finish registration (flat) |
| `POST` | `/authsec/webauthn/finishAuthentication` | Finish authentication (flat) |
| `POST` | `/authsec/webauthn/biometric/verifyBegin` | Begin biometric verify |
| `POST` | `/authsec/webauthn/biometric/verifyFinish` | Finish biometric verify |
| `POST` | `/authsec/webauthn/biometric/beginSetup` | Begin biometric setup |
| `POST` | `/authsec/webauthn/biometric/confirmSetup` | Confirm biometric setup |
| `POST` | `/authsec/webauthn/biometric/beginLoginSetup` | Begin login biometric setup |
| `POST` | `/authsec/webauthn/biometric/confirmLoginSetup` | Confirm login biometric setup |
| `POST` | `/authsec/webauthn/biometric/verifyLoginBegin` | Begin login biometric verify |
| `POST` | `/authsec/webauthn/biometric/verifyLoginFinish` | Finish login biometric verify |
| `POST` | `/authsec/webauthn/totp/beginLoginSetup` | Begin TOTP login setup |
| `POST` | `/authsec/webauthn/totp/beginSetup` | Begin TOTP setup |
| `POST` | `/authsec/webauthn/totp/confirmLoginSetup` | Confirm TOTP login setup |
| `POST` | `/authsec/webauthn/totp/confirmSetup` | Confirm TOTP setup |
| `POST` | `/authsec/webauthn/totp/verifyLogin` | Verify TOTP (login flow) |
| `POST` | `/authsec/webauthn/totp/verify` | Verify TOTP |
| `POST` | `/authsec/webauthn/sms/beginSetup` | Begin SMS setup |
| `POST` | `/authsec/webauthn/sms/confirmSetup` | Confirm SMS setup |
| `POST` | `/authsec/webauthn/sms/requestCode` | Request SMS code |
| `POST` | `/authsec/webauthn/sms/verify` | Verify SMS code |

---

### Client Management (`/authsec/clientms`)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/authsec/clientms/health` | Public | Health check |
| `GET` | `/authsec/clientms/swagger` | Public | API documentation |
| `GET` | `/authsec/clientms/swagger/doc.json` | Public | OpenAPI spec |
| `GET` | `/authsec/clientms/tenants/:tenantId/clients/getClients` | JWT | List clients |
| `POST` | `/authsec/clientms/tenants/:tenantId/clients/getClients` | JWT | List clients (POST) |
| `GET` | `/authsec/clientms/tenants/:tenantId/clients/:id` | JWT | Get client |
| `POST` | `/authsec/clientms/tenants/:tenantId/clients/create` | JWT | Create client |
| `PUT` | `/authsec/clientms/tenants/:tenantId/clients/:id` | JWT | Replace client |
| `PATCH` | `/authsec/clientms/tenants/:tenantId/clients/:id` | JWT | Edit client |
| `PATCH` | `/authsec/clientms/tenants/:tenantId/clients/:id/soft-delete` | JWT | Soft delete |
| `DELETE` | `/authsec/clientms/tenants/:tenantId/clients/:id` | JWT | Hard delete |
| `POST` | `/authsec/clientms/tenants/:tenantId/clients/delete-complete` | JWT | Complete delete (cascade) |
| `PATCH` | `/authsec/clientms/tenants/:tenantId/clients/:id/activate` | JWT | Activate client |
| `PATCH` | `/authsec/clientms/tenants/:tenantId/clients/:id/deactivate` | JWT | Deactivate client |
| `POST` | `/authsec/clientms/tenants/:tenantId/clients/set-status` | JWT | Set status |
| `GET` | `/authsec/clientms/admin/clients/` | JWT+Admin | Cross-tenant client list |

---

### Hydra Manager (`/authsec/hmgr`)

#### Public Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/hmgr/health` | Health check |
| `GET` | `/authsec/hmgr/login` | Login redirect |
| `GET` | `/authsec/hmgr/consent` | Hydra consent handler |
| `GET` | `/authsec/hmgr/challenge` | Login challenge |
| `GET` | `/authsec/hmgr/login/page-data` | Login page data |
| `POST` | `/authsec/hmgr/auth/initiate/:provider` | Initiate OIDC auth |
| `POST` | `/authsec/hmgr/auth/callback` | OIDC callback |
| `POST` | `/authsec/hmgr/auth/exchange-token` | Exchange token |
| `POST` | `/authsec/hmgr/saml/initiate/:provider` | Initiate SAML |
| `POST` | `/authsec/hmgr/saml/acs` | SAML ACS (shared) |
| `POST` | `/authsec/hmgr/saml/acs/:tenant_id/:client_id` | SAML ACS (client-specific) |
| `GET` | `/authsec/hmgr/saml/metadata/:tenant_id/:client_id` | SAML metadata |
| `POST` | `/authsec/hmgr/saml/test-provider` | Test SAML provider |

#### Admin Endpoints (JWT required)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/hmgr/admin/profile` | Get profile |
| `PUT` | `/authsec/hmgr/admin/profile` | Update profile |
| `GET/POST/PUT/DELETE` | `/authsec/hmgr/admin/users` | User management |
| `GET/POST/PUT/DELETE` | `/authsec/hmgr/admin/tenants` | Tenant management |
| `GET/POST/PUT/DELETE` | `/authsec/hmgr/admin/saml-providers` | SAML provider management |
| `GET/POST/PUT/DELETE` | `/authsec/hmgr/admin/roles` | Role management |
| `GET/POST` | `/authsec/hmgr/admin/permissions` | Permission management |
| `POST` | `/authsec/hmgr/admin/users/:id/roles` | Assign role |
| `DELETE` | `/authsec/hmgr/admin/users/:id/roles/:role_id` | Remove role |

---

### OIDC Config Manager (`/authsec/oocmgr`)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/oocmgr/health` | Health check |
| `POST` | `/authsec/oocmgr/configure-complete-oidc` | Complete OIDC config |
| `POST` | `/authsec/oocmgr/tenant/create-base-client` | Create base tenant client |
| `POST` | `/authsec/oocmgr/tenant/check-exists` | Check tenant exists |
| `POST` | `/authsec/oocmgr/tenant/list-all` | List all tenants |
| `POST` | `/authsec/oocmgr/tenant/delete-complete` | Delete complete tenant config |
| `POST` | `/authsec/oocmgr/tenant/update-complete` | Update complete tenant config |
| `POST` | `/authsec/oocmgr/tenant/login-page-data` | Get login page data |
| `POST` | `/authsec/oocmgr/config/edit` | Edit configuration |
| `POST` | `/authsec/oocmgr/oidc/add-provider` | Add OIDC provider |
| `POST` | `/authsec/oocmgr/oidc/get-config` | Get OIDC config |
| `POST` | `/authsec/oocmgr/oidc/get-provider` | Get provider |
| `POST` | `/authsec/oocmgr/oidc/get-provider-secret` | Get provider secret |
| `POST` | `/authsec/oocmgr/oidc/update-provider` | Update provider |
| `POST` | `/authsec/oocmgr/oidc/delete-provider` | Delete provider |
| `POST` | `/authsec/oocmgr/oidc/templates` | Get provider templates |
| `POST` | `/authsec/oocmgr/oidc/validate` | Validate OIDC config |
| `GET/POST` | `/authsec/oocmgr/oidc/show-auth-providers` | List auth providers |
| `POST` | `/authsec/oocmgr/oidc/raw-hydra-dump` | Raw Hydra data dump (JWT) |
| `POST` | `/authsec/oocmgr/oidc/edit-client-auth-provider` | Edit auth provider |
| `POST` | `/authsec/oocmgr/saml/add-provider` | Add SAML provider |
| `POST` | `/authsec/oocmgr/saml/list-providers` | List SAML providers |
| `POST` | `/authsec/oocmgr/saml/get-provider` | Get SAML provider |
| `POST` | `/authsec/oocmgr/saml/update-provider` | Update SAML provider |
| `POST` | `/authsec/oocmgr/saml/delete-provider` | Delete SAML provider |
| `POST` | `/authsec/oocmgr/saml/templates` | Get SAML templates |
| `POST` | `/authsec/oocmgr/hydra-clients/list` | List Hydra clients |
| `POST` | `/authsec/oocmgr/hydra-clients/get-by-tenant` | Get Hydra clients by tenant |
| `POST` | `/authsec/oocmgr/hydra-clients/sync` | Sync Hydra clients |
| `POST` | `/authsec/oocmgr/test/oidc-flow` | Test OIDC flow |
| `POST` | `/authsec/oocmgr/stats/tenant` | Get tenant stats |
| `POST` | `/authsec/oocmgr/clients/getClients` | Get clients by tenant |

---

### Auth Manager (`/authsec/authmgr`)

Provides JWT verification, RBAC permission checks, and group management.

#### Health & Token Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/authmgr/health` | Health check |
| `POST` | `/authsec/authmgr/token/verify` | Verify JWT token |
| `POST` | `/authsec/authmgr/token/generate` | Generate JWT token |
| `POST` | `/authsec/authmgr/token/oidc` | Exchange for OIDC token |

#### Admin Endpoints (`/authsec/authmgr/admin`, JWT required)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/authmgr/admin/profile` | Get profile |
| `GET` | `/authsec/authmgr/admin/auth-status` | Auth status |
| `GET` | `/authsec/authmgr/admin/validate/token` | Validate token |
| `GET` | `/authsec/authmgr/admin/validate/scope` | Validate scope |
| `GET` | `/authsec/authmgr/admin/validate/resource` | Validate resource |
| `POST` | `/authsec/authmgr/admin/validate/permissions` | Validate permissions |
| `GET` | `/authsec/authmgr/admin/check/permission` | Check permission |
| `GET` | `/authsec/authmgr/admin/check/role` | Check role |
| `GET` | `/authsec/authmgr/admin/check/role-resource` | Check role resource |
| `GET` | `/authsec/authmgr/admin/check/permission-scoped` | Check scoped permission |
| `GET` | `/authsec/authmgr/admin/check/oauth-scope` | Check OAuth scope |
| `GET` | `/authsec/authmgr/admin/permissions` | List user permissions |
| `POST` | `/authsec/authmgr/admin/groups` | Create group |
| `GET` | `/authsec/authmgr/admin/groups` | List groups |
| `GET` | `/authsec/authmgr/admin/groups/:id` | Get group |
| `PUT` | `/authsec/authmgr/admin/groups/:id` | Update group |
| `DELETE` | `/authsec/authmgr/admin/groups/:id` | Delete group |
| `POST` | `/authsec/authmgr/admin/groups/:id/users` | Add users to group |
| `DELETE` | `/authsec/authmgr/admin/groups/:id/users` | Remove users from group |
| `GET` | `/authsec/authmgr/admin/groups/:id/users` | List group users |

#### User Endpoints (`/authsec/authmgr/user`, JWT required)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/authmgr/user/profile` | Get profile |
| `GET` | `/authsec/authmgr/user/auth-status` | Auth status |
| `GET` | `/authsec/authmgr/user/validate/token` | Validate token |
| `GET` | `/authsec/authmgr/user/validate/scope` | Validate scope |
| `GET` | `/authsec/authmgr/user/validate/resource` | Validate resource |
| `POST` | `/authsec/authmgr/user/validate/permissions` | Validate permissions |
| `GET` | `/authsec/authmgr/user/check/permission` | Check permission |
| `GET` | `/authsec/authmgr/user/check/role` | Check role |
| `GET` | `/authsec/authmgr/user/check/role-resource` | Check role resource |
| `GET` | `/authsec/authmgr/user/check/permission-scoped` | Check scoped permission |
| `GET` | `/authsec/authmgr/user/check/oauth-scope` | Check OAuth scope |
| `GET` | `/authsec/authmgr/user/permissions` | List user permissions |

---

### External Services (`/authsec/exsvc`)

Manages registered external service integrations with Vault-backed credentials.

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/authsec/exsvc/health` | Public | Health check |
| `GET` | `/authsec/exsvc/debug/auth` | JWT | Debug JWT claims |
| `GET` | `/authsec/exsvc/debug/test` | JWT | Connectivity test |
| `GET` | `/authsec/exsvc/debug/token` | JWT | Inspect token context |
| `POST` | `/authsec/exsvc/services` | JWT + `external-service:create` | Register external service |
| `GET` | `/authsec/exsvc/services` | JWT + `external-service:read` | List external services |
| `GET` | `/authsec/exsvc/services/:id` | JWT + `external-service:read` | Get external service |
| `PUT` | `/authsec/exsvc/services/:id` | JWT + `external-service:update` | Update external service |
| `DELETE` | `/authsec/exsvc/services/:id` | JWT + `external-service:delete` | Delete external service |
| `GET` | `/authsec/exsvc/services/:id/credentials` | JWT + `external-service:credentials` | Get service credentials |

---

### SPIRE Headless (`/authsec/spire`)

Provides SPIFFE workload identity, OIDC token issuance with cloud federation (AWS/Azure/GCP), and a built-in RBAC/ABAC policy engine. Connects to a SPIRE server via the SPIFFE Workload API socket when available; degrades gracefully if the socket is absent.

#### Health & Discovery

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/authsec/spire/health` | Public | Health check |
| `GET` | `/authsec/spire/.well-known/openid-configuration` | Public | OIDC discovery document |
| `GET` | `/authsec/spire/.well-known/jwks.json` | Public | JWK Set (SPIRE signing keys) |

#### Registry

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/authsec/spire/registry/workloads` | — | Register a SPIFFE workload |
| `PUT` | `/authsec/spire/registry/workloads/:id` | — | Update a registered workload |
| `DELETE` | `/authsec/spire/registry/workloads/:id` | — | Delete a workload registration |
| `GET` | `/authsec/spire/registry/workloads` | — | List all registered workloads |

#### OIDC Token Operations

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/authsec/spire/oidc/token` | — | Exchange credentials for an OIDC token |
| `POST` | `/authsec/spire/oidc/introspect` | — | Introspect a token |
| `POST` | `/authsec/spire/oidc/revoke` | — | Revoke a token |
| `POST` | `/authsec/spire/oidc/exchange/spiffe` | — | Exchange a SPIFFE SVID for an OIDC token |
| `POST` | `/authsec/spire/oidc/issue/jwt-svid` | — | Issue a JWT-SVID |
| `POST` | `/authsec/spire/oidc/exchange/cloud` | — | Generic cloud token exchange |
| `POST` | `/authsec/spire/oidc/exchange/aws` | — | Exchange for AWS STS credentials |
| `POST` | `/authsec/spire/oidc/exchange/azure` | — | Exchange for Azure AD token |
| `POST` | `/authsec/spire/oidc/exchange/gcp` | — | Exchange for GCP access token |

#### Policy Engine

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/authsec/spire/policy` | — | Create a policy |
| `GET` | `/authsec/spire/policy` | — | List policies |
| `GET` | `/authsec/spire/policy/:id` | — | Get a policy |
| `PUT` | `/authsec/spire/policy/:id` | — | Update a policy |
| `DELETE` | `/authsec/spire/policy/:id` | — | Delete a policy |
| `POST` | `/authsec/spire/policy/evaluate` | — | Evaluate a policy (single) |
| `POST` | `/authsec/spire/policy/batch-evaluate` | — | Evaluate multiple policies in batch |
| `POST` | `/authsec/spire/policy/test` | — | Dry-run a policy without persisting |

#### Role Bindings

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/authsec/spire/roles/bind` | — | Bind a role to a subject |
| `POST` | `/authsec/spire/roles/unbind` | — | Remove a role binding |
| `GET` | `/authsec/spire/roles/bindings` | — | List role bindings |

#### Audit

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/authsec/spire/audit/logs` | — | Query audit log entries |
| `GET` | `/authsec/spire/audit/logs/export` | — | Export audit logs |

---

### Migration Management (`/authsec/migration`)

Manages master and per-tenant database migrations. All endpoints require JWT authentication.

#### Master Database

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/authsec/migration/migrations/master/run` | Execute all pending master DB migrations |
| `GET` | `/authsec/migration/migrations/master/status` | Get master DB migration status |

#### Tenant Databases

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/authsec/migration/tenants` | List all tenants and their migration status |
| `POST` | `/authsec/migration/tenants/create-db` | Create a tenant database and kick off migrations async |
| `POST` | `/authsec/migration/tenants/migrate-all` | Run migrations for all tenants not yet completed |
| `POST` | `/authsec/migration/tenants/:tenant_id/migrations/run` | Run migrations for a specific tenant |
| `GET` | `/authsec/migration/tenants/:tenant_id/migrations/status` | Get migration status for a specific tenant |

---

### Well-Known Endpoints

Required at the root path by RFC 8414. These cannot be moved.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/.well-known/openid-configuration` | OIDC discovery document |
| `GET` | `/.well-known/jwks.json` | JWK Set (public signing keys) |

### Metrics

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/metrics` | Prometheus metrics |

---

## Authentication & Middleware

All routes with **JWT** authentication use `AuthMiddleware` from `middlewares/auth.go`. The middleware:

1. Extracts the `Authorization: Bearer <token>` header.
2. Validates the JWT signature against the configured `JWT_DEF_SECRET` / `JWT_SDK_SECRET`.
3. Accepts tokens with issuer `authsec-ai/auth-manager`.
4. Sets claims into the gin context (`user_id`, `tenant_id`, `project_id`, `client_id`, `email`, `roles`, `scopes`).

Routes marked **JWT+Tenant** additionally pass through `ValidateTenantFromToken()` which ensures the token's `tenant_id` claim matches the tenant being accessed.

Routes marked **JWT+Admin** also enforce `Require("admin", "access")`.

Permission-gated routes (e.g. `external-service:create`) use `Require(resource, action)` which performs a live RBAC check against the tenant database.

---

## Database Configuration

AuthSec uses two categories of database connections:

### Primary Database

Configured via `DB_*` environment variables. Holds:

- Admin users, tenants, projects
- Platform RBAC tables (`roles`, `permissions`, `role_bindings`, …)
- WebAuthn sessions
- Audit log
- Tenant registry (maps tenant IDs to their database names)

### Per-Tenant Databases

Each tenant gets its own PostgreSQL database. The connection is resolved dynamically using `config.GetTenantGORMDB(tenantID)` which looks up the database name in the primary DB's `tenants` table and opens a pooled connection.

Tenant tables include: end-users, OIDC identities, client registrations, external services, TOTP/SMS MFA state.

### Migrations

Master DB migrations run automatically at startup (unless `SKIP_MIGRATIONS=true`). SQL files live under:

```text
migrations/
├── master/          – master DB schema (applied at boot)
│   ├── 000_comprehensive_base_schema.sql
│   ├── 001_create_fluent_bit_export_configs.sql
│   ├── 002_add_migration_tracking_to_tenants.sql
│   ├── 1004_dml_001_initial_data.sql
│   └── 1005_dml_002_test_data.sql
├── tenant/          – per-tenant DB schema (applied via migration API)
│   ├── 000_tenant_template.sql
│   ├── 001–010_*.sql
│   └── ...
└── permissions/
    └── master/      – RBAC permission seed migrations (appended to master run)
        └── 079–200_*.sql
```

The runner (`internal/migration/runner.go`) tracks applied migrations in `migration_logs` (master DB), supports retry logic (3 attempts per file), and handles dollar-quoted PostgreSQL functions safely.

Tenant databases are provisioned on demand via the `/authsec/migration` API — see [Migration Management](#migration-management-authsecmigration).

---

## Internal Package Layout

```text
authsec/
├── cmd/main.go               – entry point, initialises all components
├── config/                   – configuration, DB connections, Vault, WebAuthn setup
├── controllers/              – HTTP handlers organised by concern
│   ├── admin/                – admin-facing handlers (auth, tenants, RBAC, migration, …)
│   │   ├── admin_auth_controller.go
│   │   ├── admin_user_controller.go
│   │   ├── migration_controller.go
│   │   ├── permission_controller.go
│   │   ├── roles_scoped_bindings_controller.go
│   │   └── ...
│   ├── enduser/              – end-user self-service handlers
│   │   ├── enduser_auth_controller.go
│   │   ├── enduser_controller.go
│   │   ├── totp_controller.go
│   │   └── ...
│   ├── platform/             – cross-cutting / platform handlers
│   │   ├── authmgr_controller.go
│   │   ├── clients_controller.go
│   │   ├── extsvc_controller.go
│   │   ├── hmgr_controller.go
│   │   ├── oocmgr_controller.go
│   │   └── ...
│   └── shared/               – shared helpers, health, AD/Entra sync
│       ├── health_controller.go
│       ├── ad_controller.go
│       └── entra_controller.go
├── handlers/                 – WebAuthn/FIDO2 handlers
│   ├── webauthn_handler.go
│   ├── admin_webauthn_handler.go
│   ├── enduser_webauthn_handler.go
│   ├── totp_handler.go
│   └── sms_handler.go
├── internal/
│   ├── authmgr/
│   │   ├── models/rbac.go    – GORM models for RBAC tables
│   │   └── repo/rbac_repository.go
│   ├── hydra/models/         – Hydra client / SAML models
│   ├── migration/            – migration runner, models, DB utilities
│   │   ├── runner.go         – versioned SQL runner with retry + migration_logs
│   │   ├── models.go
│   │   └── db_utils.go       – ConnectToTenantDB, CreateDatabase, IsValidDatabaseName
│   ├── oocmgr/               – OIDC config manager repository + services
│   ├── session/              – WebAuthn PostgreSQL session store
│   └── clients/              – ICP PKI client, auth methods
├── migrations/               – SQL migration files (master + tenant + permissions)
├── middlewares/              – CORS, JWT auth, rate limiting, tenant validation
├── models/                   – shared GORM models
├── monitoring/               – Prometheus metrics, audit log, structured logging
├── repository/               – shared repositories (MFA, clients RBAC, extsvc)
├── routes/routes.go          – central route registration
├── services/                 – business logic services
└── vault/                    – HashiCorp Vault client interface
```

---

## Background Workers

The following goroutines start automatically at boot:

| Worker | Interval | Purpose |
| --- | --- | --- |
| Audit log cleanup | 24 hours | Removes audit events older than 90 days |
| System metrics | 30 seconds | Updates Prometheus system gauges (goroutines, memory, …) |
| PKI retry worker | 5 minutes | Retries failed ICP/PKI provisioning operations |
| WebAuthn session GC | At startup | Cleans expired WebAuthn challenge sessions |

---

## Building & Running

### Build

```bash
go build -o authsec ./cmd/
```

### Docker

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o authsec ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/authsec .
EXPOSE 7468
CMD ["./authsec"]
```

### Health check

```bash
curl http://localhost:7468/authsec/uflow/health
```

Expected response:

```json
{"status": "healthy", "database": "connected", "timestamp": "..."}
```

---

## Contributing

Contributions are welcome! Please:

1. Fork the repository and create a feature branch.
2. Ensure `go build ./...` and `go test ./...` pass before opening a pull request.
3. Keep pull requests focused — one feature or fix per PR.
4. Follow the existing code style (standard Go formatting via `gofmt`).

For significant changes, open an issue first to discuss the approach.

---

## License

Apache License 2.0 — see [LICENSE](LICENSE) for the full text.

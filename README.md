# AuthSec — Enterprise Identity & Access Management Platform

AuthSec is an open-source, enterprise-grade IAM platform that consolidates authentication, authorization, and identity lifecycle management into a single deployable stack. It supports passkeys, TOTP, voice auth, CIBA, SCIM, OIDC federation, AD/Entra sync, SPIFFE workload identity, and more — all behind a single Go binary with a React frontend, orchestrated with Docker Compose.

---

## Repository Layout

This monorepo contains everything needed to self-host AuthSec:

```
authsec-ai/
├── authsec/        # Go backend — single binary serving all IAM modules
├── authsec-ui/     # React 19 + TypeScript frontend
└── onprem/         # Docker Compose stack + Nginx + setup scripts
```

Each sub-directory has its own detailed README:

- [authsec/README.md](authsec/README.md) — Backend architecture, API reference, environment variables
- [authsec-ui/README.md](authsec-ui/README.md) — Frontend setup, scripts, environment variables
- [onprem/README.md](onprem/README.md) — Full deployment guide (Docker Compose, TLS, production hardening)
- [onprem/TROUBLESHOOTING.md](onprem/TROUBLESHOOTING.md) — Common issues and fixes

---

## Architecture Overview

```
                        Internet
                           │
                     ┌─────▼──────┐
                     │   Nginx    │  :80 / :443
                     └─────┬──────┘
          ┌────────────────┼──────────────────┐
          │                │                  │
   ┌──────▼──────┐  ┌──────▼──────┐  ┌───────▼───────┐
   │  AuthSec UI │  │  AuthSec    │  │   Ory Hydra   │
   │  React SPA  │  │  Go binary  │  │  OAuth2/OIDC  │
   │   :3000     │  │   :7468     │  │  :4444/:4445  │
   └─────────────┘  └──────┬──────┘  └───────────────┘
                           │
           ┌───────────────┼──────────────┐
           │               │              │
    ┌──────▼──────┐ ┌──────▼──────┐ ┌────▼────┐
    │  PostgreSQL │ │    Redis    │ │  Vault  │
    │  master +   │ │  sessions / │ │ secrets │
    │  per-tenant │ │ rate-limit  │ │         │
    └─────────────┘ └─────────────┘ └─────────┘
```

**AuthSec backend modules** (all served from the single binary):

| Route prefix | Module | Responsibility |
|---|---|---|
| `/authsec/uflow/` | User Flow | Login, registration, OIDC federation, SCIM, TOTP, CIBA |
| `/authsec/webauthn/` | WebAuthn | FIDO2 passkeys, TOTP setup, SMS MFA |
| `/authsec/authmgr/` | Auth Manager | JWT validation, RBAC permission checks, group management |
| `/authsec/clientms/` | Client Manager | OAuth2 client lifecycle (Hydra-backed) |
| `/authsec/hmgr/` | Hydra Manager | Hydra login/consent, SAML SSO, token exchange |
| `/authsec/oocmgr/` | OIDC Config Mgr | External OIDC/SAML provider config and sync |
| `/authsec/exsvc/` | External Services | External service registry with Vault-backed credentials |
| `/authsec/spire/` | SPIRE | SPIFFE workload identity, cloud federation (AWS/Azure/GCP) |
| `/authsec/sdkmgr/` | SDK Manager | AI agent SDK, MCP auth, playground |
| `/authsec/migration/` | Migration | Master and per-tenant database migrations |
| `/.well-known/` | OIDC Discovery | JWKS, OpenID config (RFC 8414) |
| `/oauth2/`, `/userinfo` | OAuth2 | Hydra OAuth2/OIDC endpoints (RFC 6749) |

---

## Quickstart — Docker Compose

The fastest way to get a full stack running locally.

**Prerequisites:** Docker Engine 24.0+ and Docker Compose v2.20+ (plugin), 4 GB RAM minimum.

```bash
cd onprem
bash setup.sh
```

The setup script will:
1. Check prerequisites
2. Prompt for deployment type and admin credentials
3. Generate all secrets (JWT, encryption keys, Hydra secrets)
4. Write a `.env` file
5. Pull images and start all services
6. Run the master database migration
7. Create the first admin account

Once complete, open **http://localhost** in your browser.

For manual setup or production deployment (custom domain, TLS), see [onprem/README.md](onprem/README.md).

---

## Local Development

### Backend (Go)

**Requirements:** Go 1.25+, PostgreSQL 14+

```bash
cd authsec
cp .env.example .env
# Edit .env — set DB_* and required secrets
go build -o authsec ./cmd/
./authsec
# API available at http://localhost:7468
```

See [authsec/README.md](authsec/README.md) for the full environment variable reference and API route map.

### Frontend (React)

**Requirements:** Node.js >= 20.0.0, npm >= 10.0.0

```bash
cd authsec-ui
npm install
cp .env.example .env
# Set VITE_API_URL=http://localhost:7468
npm run dev
# UI available at http://localhost:5173
```

See [authsec-ui/README.md](authsec-ui/README.md) for all scripts and configuration options.

---

## Services at a Glance

| Service | Image | Purpose | Exposed Port |
|---|---|---|---|
| `nginx` | nginx:alpine | Reverse proxy, TLS termination | 80, 443 |
| `authsec` | authsec-ai/authsec | Go IAM backend | internal :7468 |
| `ui` | authsec-ai/authsec-ui | React frontend | internal :3000 |
| `postgres` | postgres:15 | Master + per-tenant databases | internal |
| `hydra` | oryd/hydra | OAuth2 / OIDC server | internal :4444/:4445 |
| `redis` | redis:7 | Session cache, rate limiting | internal |
| `vault` | hashicorp/vault | OIDC provider secrets | internal |

---

## Feature Highlights

- **Authentication methods** — Password, TOTP, SMS OTP, FIDO2/WebAuthn passkeys, voice authentication, CIBA
- **Federation** — OIDC/SAML identity providers, Google/GitHub/Microsoft social login, AD/LDAP, Microsoft Entra ID
- **Authorization** — Scoped RBAC, role bindings, permission checks, API scopes
- **Multi-tenancy** — Isolated per-tenant PostgreSQL databases with dynamic provisioning
- **Standards compliance** — OAuth 2.0, OIDC, SCIM 2.0, FIDO2/WebAuthn, SPIFFE/SPIRE, Device Authorization (RFC 8628)
- **AI agent identity** — SPIFFE workload identity, MCP auth, JWT-SVID, cloud federation (AWS/Azure/GCP)
- **Operational** — Prometheus metrics, structured audit logging, rate limiting, token blacklisting, Vault secrets management

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25, Gin, GORM, PostgreSQL 15 |
| Frontend | React 19, TypeScript 5.8, Vite 6, Redux Toolkit, Tailwind CSS 4, Radix UI |
| OAuth2/OIDC | Ory Hydra |
| Secrets | HashiCorp Vault |
| Cache | Redis 7 |
| Identity | SPIFFE/SPIRE |
| Proxy | Nginx |
| Container | Docker Compose |

---

## Production Deployment

For a production deployment with a custom domain and TLS:

1. Point your domain's DNS A record to your server's IP
2. Update `.env` with production values (`BASE_URL`, `HYDRA_PUBLIC_URL`, `WEBAUTHN_ORIGIN`, `ENVIRONMENT=production`)
3. Obtain TLS certificates (Let's Encrypt / Certbot)
4. Uncomment TLS blocks in `onprem/nginx/nginx.conf`
5. Restart Nginx: `docker compose restart nginx`

Full step-by-step instructions: [onprem/README.md](onprem/README.md)

---

## Health Checks

```bash
curl http://localhost/health
curl http://localhost/authsec/uflow/health
curl http://localhost:4444/health/ready
```

---

## Contributing

1. Fork the repo and create a feature branch
2. Follow existing code style and conventions
3. Add tests for new functionality
4. Open a pull request with a clear description

Bug reports and feature requests are welcome via GitHub Issues.

---

## License

AuthSec is licensed under the [Apache 2.0 License](authsec/LICENSE).

# AuthSec — On-Premise Deployment Guide

AuthSec is an open-source identity and access management platform for AI agents and human users. This guide covers running the full stack locally or on a self-hosted VM using Docker Compose.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Environment Variables](#environment-variables)
5. [First-Time Setup](#first-time-setup)
6. [API Routes](#api-routes)
7. [Production Deployment](#production-deployment)
8. [Maintenance](#maintenance)
9. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

```text
Browser / API Client
        │
        ▼
   ┌─────────┐
   │  Nginx  │  :80 / :443
   └────┬────┘
        │
        ├── /              ──► UI (React frontend, :3000)
        ├── /authsec/*     ──► AuthSec backend (:7468)
        ├── /.well-known/* ──► AuthSec backend (:7468)
        ├── /oauth2/*      ──► Hydra public API (:4444)
        └── /userinfo      ──► Hydra public API (:4444)

   ┌──────────────────────────────────────────────────┐
   │              AuthSec Backend (:7468)             │
   │  (Go monolith — all modules merged into one      │
   │   binary, one port, one Docker image)            │
   │                                                  │
   │  /authsec/uflow/*    – auth, OIDC, JWT, TOTP,    │
   │                         WebAuthn, SCIM           │
   │  /authsec/authmgr/*  – RBAC, token validation    │
   │  /authsec/clientms/* – OAuth client management   │
   │  /authsec/hmgr/*     – Hydra proxy/login/consent │
   │  /authsec/oocmgr/*   – OIDC provider config      │
   │  /authsec/sdkmgr/*   – AI agent SDK management   │
   │  /authsec/exsvc/*    – external integrations     │
   │  /authsec/webauthn/* – passkey flows              │
   │  /authsec/spire/*    – SPIFFE workload identity   │
   │  /authsec/migration/*– DB migration management   │
   └──────────────────────────────────────────────────┘

   Infrastructure:
     postgres   – master DB + dynamic per-tenant DBs
     hydra      – Ory Hydra OAuth2/OIDC server
     redis      – session cache, rate limiting
```

**The UI is the entry point for all user-facing flows.** It handles client-side routing and calls the AuthSec backend via `/authsec/*` API paths. You do not interact with the backend directly in normal usage.

**`OOC_MANAGER_URL` and `AUTH_MANAGER_URL` both point back to the AuthSec service itself** — those modules are merged into the monolith and calls are handled in-process.

---

## Prerequisites

| Requirement | Version |
| --- | --- |
| Docker Engine | 24.0+ |
| Docker Compose | v2.20+ (plugin) |
| RAM | 4 GB minimum, 8 GB recommended |
| Disk | 20 GB minimum |
| OS | Linux (Ubuntu 22.04+ recommended), macOS, or WSL2 |

### Install Docker (Ubuntu)

```bash
sudo apt update && sudo apt install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
  | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update && sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker $USER && newgrp docker
```

---

## Quick Start

The fastest way to get running is the interactive setup script. It asks you a handful of questions, auto-generates all secrets, starts the stack, runs migrations, and creates your first admin account.

```bash
# 1. Clone the repo
git clone https://github.com/authsec-ai/onprem.git
cd onprem

# 2. Run the setup script
bash setup.sh
```

That's it. The script handles everything and prints your URLs when done.

> **What the script does:** checks prerequisites → asks deployment type (local/production) and admin credentials → auto-generates all JWT/encryption/Hydra secrets → writes `.env` → pulls images → starts services → waits for health → runs the master DB migration → creates the first admin account.

### Manual setup (advanced)

If you prefer to configure things yourself:

```bash
# Copy and edit the env file
cp .env.example .env
$EDITOR .env   # at minimum fill in DB_PASSWORD and the JWT/encryption keys

# Pull images and start
docker compose pull
docker compose up -d

# Wait for services to be healthy (~60–90s on first run)
docker compose ps

# Open the UI
open http://localhost
```

> **Note:** On first startup AuthSec automatically runs master DB migrations and builds the tenant DB template in the background (~30–60 seconds). No manual migration step is needed.

---

## Environment Variables

Copy `.env.example` to `.env` and fill in the values. The table below summarises what is required vs optional.

### Required — service will not start without these

| Variable | Description |
| --- | --- |
| `DB_PASSWORD` | PostgreSQL password for the `authsec` user |
| `JWT_DEF_SECRET` | JWT signing secret (32+ chars) |
| `JWT_SECRET` | JWT signing secret (32+ chars) |
| `JWT_SDK_SECRET` | SDK JWT signing secret (32+ chars) |
| `TOTP_ENCRYPTION_KEY` | AES key for encrypting TOTP secrets (32+ chars) |
| `SYNC_CONFIG_ENCRYPTION_KEY` | Config sync encryption key (32+ chars) |
| `SESSION_SECRET` | Session signing key (32+ chars) |
| `HYDRA_SECRETS_SYSTEM` | Hydra system secret (32+ chars) |
| `HYDRA_SECRETS_COOKIE` | Hydra cookie secret (32+ chars) |
| `HYDRA_DSN` | Hydra postgres DSN — must match `DB_PASSWORD` |

Generate all secrets at once:

```bash
for var in JWT_DEF_SECRET JWT_SECRET JWT_SDK_SECRET TOTP_ENCRYPTION_KEY \
           SYNC_CONFIG_ENCRYPTION_KEY SESSION_SECRET \
           HYDRA_SECRETS_SYSTEM HYDRA_SECRETS_COOKIE; do
  echo "$var=$(openssl rand -hex 32)"
done
```

### Service URLs (have sensible defaults for local dev)

| Variable | Default | Notes |
| --- | --- | --- |
| `BASE_URL` | `http://localhost` | Public-facing root URL |
| `HYDRA_PUBLIC_URL` | `http://localhost:4444` | Hydra public OAuth2 URL (browser-accessible) |
| `REACT_APP_URL` | `http://localhost:3000` | Frontend SPA URL |
| `TENANT_DOMAIN_SUFFIX` | `localhost` | Suffix for per-tenant workspace domains |

### Optional features

| Variable | Feature |
| --- | --- |
| `GOOGLE_CLIENT_SECRET` | Google social login |
| `GITHUB_CLIENT_SECRET` | GitHub social login |
| `MICROSOFT_CLIENT_SECRET` | Microsoft social login |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` | Email OTP, registration, forgot-password |
| `VAULT_ADDR` / `VAULT_TOKEN` | HashiCorp Vault for storing OIDC provider secrets |
| `ICP_SERVICE_URL` | SPIFFE/SPIRE for AI agent workload identity (default `http://spire-headless:7001`) |
| `SPIFFE_OIDC_ISSUER` etc. | JWT-SVID OIDC issuer configuration |
| `OKTA_*` | Okta CIBA integration |

---

## First-Time Setup

Once the stack is running, open the UI at **<http://localhost>** and complete the initial setup through the browser.

### Migrate all tenant DBs (after tenants have been created)

When new tenants register, AuthSec creates their database dynamically. After upgrading to a new version, apply tenant migrations across all existing tenant databases:

```bash
curl -X POST http://localhost/authsec/migration/tenants/migrate-all \
  -H "Authorization: Bearer <admin-jwt>"
```

### 4. Verify health

```bash
curl http://localhost/health
# → {"status":"ok"}

curl http://localhost/authsec/uflow/health
# → {"status":"ok"}

curl http://localhost:4444/health/ready
# → {}
```

---

## API Routes

All backend API calls go through nginx at `http://localhost` (or your domain). The UI calls these paths internally — you only need to know them if integrating directly or debugging.

| Prefix | Module | Examples |
| --- | --- | --- |
| `/authsec/uflow/` | Auth, OIDC, JWT, TOTP, WebAuthn | `POST /authsec/uflow/auth/login` |
| `/authsec/authmgr/` | RBAC, token validation | `GET /authsec/authmgr/roles` |
| `/authsec/clientms/` | OAuth client management | `POST /authsec/clientms/clients` |
| `/authsec/hmgr/` | Hydra login/consent proxy | `GET /authsec/hmgr/login` |
| `/authsec/oocmgr/` | OIDC provider config | `GET /authsec/oocmgr/providers` |
| `/authsec/sdkmgr/` | AI agent SDK management | `POST /authsec/sdkmgr/agents` |
| `/authsec/exsvc/` | External integrations | — |
| `/authsec/webauthn/` | Passkey flows | `POST /authsec/webauthn/register/begin` |
| `/authsec/spire/` | SPIFFE workload identity | — |
| `/authsec/migration/` | DB migration management | `POST /authsec/migration/migrations/master/run` |
| `/.well-known/openid-configuration` | OIDC discovery | RFC 8414 |
| `/.well-known/jwks.json` | Public key set | RFC 7517 |
| `/oauth2/*` | Hydra OAuth2 (auth, token, revoke) | RFC 6749 |
| `/userinfo` | Hydra OIDC userinfo | RFC 9068 |
| `/metrics` | Prometheus scrape endpoint | — |

---

## Production Deployment

### 1. DNS setup

Point your domain's A record to your VM's public IP. You need at minimum:

- `app.yourdomain.com` — the main application (UI + API)

Optionally a separate subdomain for Hydra if you want it on its own origin:

- `oauth.yourdomain.com` — Hydra public OAuth2

### 2. Update `.env` for production

```bash
# Public URLs
BASE_URL=https://app.yourdomain.com
HYDRA_PUBLIC_URL=https://oauth.yourdomain.com   # or https://app.yourdomain.com/oauth2
REACT_APP_URL=https://app.yourdomain.com

# Domains for WebAuthn RP (no protocol/port)
DOMAIN=yourdomain.com
APP_DOMAIN=app.yourdomain.com
WEBAUTHN_ORIGIN=https://app.yourdomain.com

# Tighten up security
ENVIRONMENT=production
GIN_MODE=release
REQUIRE_SERVER_AUTH=true
CORS_ALLOW_ORIGIN=https://app.yourdomain.com

TENANT_DOMAIN_SUFFIX=yourdomain.com

# Real SMTP provider
SMTP_HOST=smtp.yourmailprovider.com
SMTP_PORT=587
SMTP_USER=noreply@yourdomain.com
SMTP_PASSWORD=your-smtp-password
```

### 3. TLS with Let's Encrypt

```bash
# Install certbot
sudo apt install -y certbot

# Obtain certificate (nginx must be stopped or not yet started)
sudo certbot certonly --standalone -d app.yourdomain.com

# Set permissions so nginx container can read certs
sudo chmod -R 755 /etc/letsencrypt/live/ /etc/letsencrypt/archive/
```

Uncomment the TLS volume mounts and server block in `nginx/nginx.conf`, then update `server_name` and certificate paths.

```bash
# Restart nginx to pick up the new config
docker compose restart nginx
```

Auto-renew certs:

```bash
# Add to root crontab
echo "0 3 * * * certbot renew --quiet && docker compose -f /path/to/onprem/docker-compose.yaml restart nginx" \
  | sudo tee -a /etc/cron.d/certbot-renew
```

### 4. Firewall

```bash
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw --force enable
```

The postgres, redis, and Hydra admin ports are bound to `127.0.0.1` in `docker-compose.yaml` and are not exposed to the internet.

---

## Maintenance

### View logs

```bash
docker compose logs -f                  # all services
docker compose logs -f authsec          # backend only
docker compose logs -f authsec --tail=100
```

### Restart a service

```bash
docker compose restart authsec
```

### Update to a new version

```bash
docker compose pull
docker compose up -d

# Apply any new tenant migrations
curl -X POST http://localhost/authsec/migration/tenants/migrate-all \
  -H "Authorization: Bearer <admin-jwt>"
```

### Backup the database

```bash
# Dump the master DB
docker compose exec postgres pg_dump -U authsec kloudone_db > backup_$(date +%Y%m%d).sql

# Dump all databases (includes tenant DBs)
docker compose exec postgres pg_dumpall -U authsec > backup_all_$(date +%Y%m%d).sql
```

### Restore from backup

```bash
docker compose exec -T postgres psql -U authsec kloudone_db < backup_20260101.sql
```

### Check resource usage

```bash
docker stats
```

### Hard reset (destructive — deletes all data)

```bash
docker compose down -v        # stops containers and removes volumes
docker compose up -d          # fresh start
```

---

## Troubleshooting

### Services are unhealthy / not starting

```bash
# Check which services aren't up
docker compose ps

# Tail logs for the failing service
docker compose logs --tail=50 authsec
docker compose logs --tail=50 hydra
docker compose logs --tail=50 postgres
```

**Common causes:**

- Missing required env vars (blank `JWT_DEF_SECRET`, `DB_PASSWORD`, etc.) — check `.env`
- `hydra-migrate` failed — run `docker compose logs hydra-migrate` and verify `HYDRA_DSN` matches `DB_PASSWORD`
- Postgres not yet ready — `authsec` and `hydra` wait for `postgres` to be healthy, but on slow machines the `start_period` may need increasing

### Database connection refused

```bash
# Confirm postgres is healthy
docker compose ps postgres

# Test connectivity from inside the authsec container
docker compose exec authsec wget -qO- http://postgres:5432 || echo "port open"

# Connect directly
docker compose exec postgres psql -U authsec -d kloudone_db -c "SELECT 1"
```

### CORS errors in the browser

Ensure `CORS_ALLOW_ORIGIN` in `.env` includes your frontend origin exactly (no trailing slash):

```bash
CORS_ALLOW_ORIGIN=http://localhost:3000
```

### WebAuthn / Passkeys not working

- `WEBAUTHN_RP_ID` must be the **hostname only** (no `https://`, no port): e.g. `localhost` or `app.yourdomain.com`
- `WEBAUTHN_ORIGIN` must be the **full origin**: e.g. `http://localhost` or `https://app.yourdomain.com`
- WebAuthn requires a Secure Context — on localhost HTTP works; on any other hostname you must use HTTPS

### Hydra OAuth2 flows failing

```bash
# Check Hydra is healthy
curl http://localhost:4444/health/ready

# Check Hydra logs
docker compose logs hydra

# Verify login/consent URLs are reachable from Hydra's perspective
# (they point to http://authsec:7468/authsec/hmgr/* inside the Docker network)
docker compose exec hydra wget -qO- http://authsec:7468/authsec/uflow/health
```

### Port 80/443 already in use

```bash
sudo lsof -i :80
sudo lsof -i :443
# Stop whatever is using the port, then:
docker compose up -d nginx
```

---

## Security Checklist (Production)

- [ ] All secrets in `.env` are randomly generated — no placeholder values
- [ ] `DB_PASSWORD` is strong and unique
- [ ] `REQUIRE_SERVER_AUTH=true`
- [ ] `GIN_MODE=release`, `ENVIRONMENT=production`
- [ ] `CORS_ALLOW_ORIGIN` is set to your exact domain (not `*`)
- [ ] TLS certificates are installed and auto-renewing
- [ ] Firewall allows only 22, 80, 443 — all other ports are blocked
- [ ] Postgres, Redis, and Hydra admin port (`4445`) are NOT exposed publicly
- [ ] SSH uses key-based authentication only
- [ ] Regular database backups are scheduled

---

## Optional Add-Ons

### HashiCorp Vault

Vault is included in the stack and starts automatically. It runs in **dev mode** — auto-initialised, never sealed, root token set to `VAULT_TOKEN` from your `.env`.

| Endpoint | Purpose |
| --- | --- |
| `http://localhost:8200` | Vault UI — log in with `VAULT_TOKEN` |
| `http://vault:8200` | Internal address used by authsec |

**What Vault stores:** OIDC provider client secrets (Google, Microsoft, GitHub). These can be set via the Vault UI or CLI after the stack is running:

```bash
# Set a provider secret in Vault
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=$(grep VAULT_TOKEN .env | cut -d= -f2)

vault kv put secret/google client_secret=your-google-secret
vault kv put secret/microsoft client_secret=your-microsoft-secret
vault kv put secret/github client_secret=your-github-secret
```

> **Dev mode caveat:** Vault dev mode uses in-memory storage. Secrets stored in Vault are lost when the `vault` container restarts. For persistent secrets across restarts, set `GOOGLE_CLIENT_SECRET`, `MICROSOFT_CLIENT_SECRET`, and `GITHUB_CLIENT_SECRET` directly in `.env` — authsec falls back to env vars when Vault keys are absent.
>
> For production with persistent Vault storage, see [TROUBLESHOOTING.md — Vault](TROUBLESHOOTING.md).

### SPIRE / spire-headless

Required only for the AI agent SPIFFE workload identity delegation feature. When enabled, AuthSec delegates SVID issuance to the spire-headless service.

```bash
ICP_SERVICE_URL=http://spire-headless:7001
```

Add the `spire-headless` service to your compose override file and set the `SPIFFE_*` variables in `.env`.

---

**Last updated:** March 2026
**Version:** 2.0.0 (monolith)

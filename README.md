# Agent-first Open Source Identity layer for Agents and Autonomous AI.

[Website](https://authsec.ai) · [Documentation](https://authsec.ai/docs) · [Blog](https://authsec.ai/blogs)

Agent-first Open Source Identity layer for Agents and Autonomous AI.

It is built for the age of agents and autonomous AI — where agents need to authenticate without browsers, delegate user permissions, talk to each other securely, and plug into MCP servers — while still covering every enterprise IAM need (SSO, RBAC, passkeys, SCIM, and more).

**This repository contains the backend service and deployment scripts.** The UI lives in a separate repository: [authsec-ai/Authsec-ui](https://github.com/authsec-ai/Authsec-ui).

---


## Get Started

### Docker Compose (Recommended)

The fastest way to run the full AuthSec stack locally or on a server.

**Prerequisites:** Docker Engine 24.0+, Docker Compose v2.20+, 4 GB RAM

```bash
git clone https://github.com/authsec-ai/authsec-ai.git
cd authsec-ai/scripts
bash setup.sh
```

Open **<http://localhost>** once the script finishes.

#### What does `setup.sh` install?

The script brings up a complete, production-ready stack:

| Service | What it does |
| --- | --- |
| **AuthSec backend** | Go binary — auth flows, RBAC, OIDC, SCIM, CIBA, SPIFFE, MCP, migration |
| **[AuthSec UI](https://github.com/authsec-ai/Authsec-ui)** | React 19 admin and end-user portal (separate repo, image pulled automatically) |
| **Ory Hydra** | Standards-compliant OAuth 2.0 / OIDC / SAML server |
| **PostgreSQL** | Master database + dynamic per-tenant databases |
| **Redis** | Session cache and rate limiting |
| **HashiCorp Vault** | Secrets storage for OIDC provider credentials |
| **Nginx** | Reverse proxy with optional TLS termination |

The script automatically:

1. Checks that Docker and Compose are installed
2. Asks for your deployment type and admin credentials
3. Generates all secrets (JWT keys, encryption keys, Hydra system/cookie secrets)
4. Writes a `.env` file
5. Pulls images and starts all services
6. Runs the master database migration
7. Creates your first admin account

For manual setup, custom domains, TLS, and production hardening — see [scripts/README.md](scripts/README.md).

---


## What AuthSec Solves

### 1. Headless Authentication for Agents (Voice / CIBA)

Modern AI agents can't open a browser login page. AuthSec supports **CIBA (Client Initiated Backchannel Authentication)** — an OAuth 2.0 extension (RFC 9126) that decouples *where* authentication is initiated from *where* the user approves it.

With CIBA:

- An agent initiates an auth request on behalf of a user using a channel for example voice
- The user receives an out-of-band prompt (voice call, push notification, SMS) to approve on a mobile app
- The agent receives tokens once the user approves — no browser redirect, no UI required

This powers voice-activated agents or other form of agents which do not need a web browser to operate: a user speaks a command, the agent triggers CIBA, the user confirms by voice, and the agent gets a scoped access token — all without a browser.

> **Resources:** [Documentation](https://authsec.ai/docs) · Demo Video *(coming soon)* · [Blog](https://authsec.ai/blogs)

---

### 2. Agent-to-Agent Authorization (Machine-to-Machine) via SPIFFE

When one agent needs to call another service or agent, you need a zero-trust, standards-based way to prove identity without managing static credentials.

AuthSec implements **SPIFFE (Secure Production Identity Framework for Everyone)** — each agent or workload gets a cryptographically verifiable **SVID (SPIFFE Verifiable Identity Document)**. Authorization between agents uses these short-lived, automatically rotated identities.

- No API keys or shared secrets between services
- Works across clouds: native federation with AWS IAM, Azure AD, and GCP IAM
- Fine-grained RBAC/ABAC policies on top of workload identity

---

### 3. Autonomous Execution with User Auth Delegation

Agents that act on behalf of users need scoped, time-limited delegated credentials — not the user's full session. AuthSec provides token exchange and delegation flows that let you issue an agent a narrowly scoped JWT derived from the user's authenticated session, so agents can act autonomously without over-privileged access.

---

### 4. MCP Authentication & Authorization

AuthSec provides built-in authentication and authorization for **Model Context Protocol (MCP)** servers. Agents connecting to MCP tools go through AuthSec to get verified, scoped tokens — ensuring every tool call is authenticated, audited, and permission-checked.

---

### 5. SSO — SAML 2.0 & OIDC Federation

Full enterprise SSO for your users and workforce:

- **SAML 2.0** identity provider and service provider flows (via Ory Hydra)
- **OIDC** federation — connect Google, GitHub, Microsoft, Okta, and any standards-compliant IdP
- **AD / Microsoft Entra ID** sync (LDAP + SCIM 2.0)
- Multi-tenant: each tenant gets isolated identity databases and configurable IdPs

---

## Run Individual Components instead of a single script

If you want to develop or run only specific parts of the stack:

**Backend only** (Go 1.25+, PostgreSQL 14+):

```bash
cd authsec
cp .env.example .env
# Edit .env — set DB_* credentials and required secrets
go build -o authsec ./cmd/
./authsec
# API at http://localhost:7468
```

**Frontend only** (Node.js >= 20, npm >= 10) — the UI is in a [separate repository](https://github.com/authsec-ai/Authsec-ui):

```bash
git clone https://github.com/authsec-ai/Authsec-ui.git
cd Authsec-ui
npm install
npm run build
export VITE_API_URL=http://localhost:7468
node server.js
# UI at http://localhost:3000
```

Detailed setup instructions, environment variable references, and API docs:

- [authsec/README.md](authsec/README.md) — Backend
- [Authsec-ui README](https://github.com/authsec-ai/Authsec-ui#readme) — Frontend (separate repo)
- [scripts/README.md](scripts/README.md) — Deployment
- [scripts/TROUBLESHOOTING.md](scripts/TROUBLESHOOTING.md) — Common issues

---

## How Do I

### Enable Voice Auth for an Agent

Voice auth uses the CIBA flow. Once AuthSec is running:

1. Register your agent as an OAuth2 client with `urn:openid:params:grant-type:ciba` grant type enabled
2. Configure your Twilio credentials in `.env` (`TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_NUMBER`)
3. The agent calls `POST /authsec/uflow/ciba/initiate` with the user's identifier
4. AuthSec places a voice call to the user for approval
5. Poll `POST /oauth2/token` with `grant_type=urn:openid:params:grant-type:ciba` until approved
6. Use the returned access token for subsequent API calls

> See the [CIBA API reference](authsec/README.md) for full request/response examples.

---

### Set Up Agent-to-Agent (M2M) Auth

AuthSec issues SPIFFE SVIDs to workloads so agents can call each other without static secrets.

1. Enable the SPIRE module — set `ICP_SERVICE_URL` in `.env`
2. Register each agent/service as a workload entry via `POST /authsec/spire/workloads`
3. Each workload fetches its SVID from the SPIFFE Workload API
4. Services validate incoming requests by verifying the SVID against the trust domain
5. Apply RBAC/ABAC policies on workload identity via `POST /authsec/spire/policies`

For cloud-native workloads, configure AWS/Azure/GCP federation so cloud-issued identities map to SPIFFE SVIDs automatically.

> See the [SPIRE module docs](authsec/README.md) for workload registration and policy setup.

---

### Configure SSO (SAML 2.0 / OIDC)

1. Open the AuthSec admin UI at `http://localhost`
2. Navigate to **Identity Providers** → **Add Provider**
3. Choose SAML 2.0 or OIDC and fill in the provider metadata
4. For OIDC providers: set `GOOGLE_CLIENT_SECRET`, `GITHUB_CLIENT_SECRET`, or `MICROSOFT_CLIENT_SECRET` in `.env` for social login
5. For SAML: upload the IdP metadata XML and download the AuthSec SP metadata to register with your IdP
6. For AD / Entra ID sync: configure LDAP host or SCIM endpoint and set `SYNC_CONFIG_ENCRYPTION_KEY`

> See [authsec/README.md](authsec/README.md) for OIDC config manager and SCIM 2.0 API details.

---

### Enable MCP Authentication

1. Register your MCP server as an OAuth2 client: `POST /authsec/clientms/clients` with appropriate scopes
2. Set `ICP_SERVICE_URL` in `.env` to point to your MCP coordinator
3. Use the SDK manager endpoints (`/authsec/sdkmgr/`) to create agent entries and issue scoped tokens
4. Agents authenticate to MCP via `POST /authsec/sdkmgr/agents/token` and present the JWT to your MCP server
5. MCP server validates tokens via `GET /authsec/authmgr/token/verify` or the JWKS endpoint at `/.well-known/jwks.json`

> See the [SDK Manager docs](authsec/README.md) for agent registration and token flows.

---

## Community

Join the conversation, ask questions, and share what you're building:

**Discord** *(invite link coming soon)*

---

## Contributing

We welcome contributions of all kinds — bug fixes, features, docs, and examples.

1. Fork the relevant repository and create a branch from `main`
2. For backend changes: fork [authsec-ai/authsec-ai](https://github.com/authsec-ai/authsec-ai) and work in [authsec/](authsec/) — follow existing Go conventions
3. For frontend changes: fork [authsec-ai/Authsec-ui](https://github.com/authsec-ai/Authsec-ui) — run `npm run lint` and `npm run type-check` before submitting
4. For deployment changes: work in [scripts/](scripts/)
5. Add or update tests for your changes
6. Open a pull request with a clear description of what and why

Bug reports and feature requests:

- Backend / deployment: [authsec-ai/authsec-ai Issues](https://github.com/authsec-ai/authsec-ai/issues)
- Frontend: [authsec-ai/Authsec-ui Issues](https://github.com/authsec-ai/Authsec-ui/issues)

---

## License

AuthSec is licensed under the [Apache 2.0 License](authsec/LICENSE).

#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# AuthSec — Interactive Setup Script
#
# Usage:  bash setup.sh
#
# What this does:
#   1. Checks prerequisites (Docker, openssl, curl)
#   2. Asks a handful of questions — all secrets are auto-generated for you
#   3. Writes .env
#   4. Pulls images and starts the full stack
#   5. Waits for services to be healthy
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

# ── Colours ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# ── Helpers ───────────────────────────────────────────────────────────────────
info()    { echo -e "  ${CYAN}→${NC} $*"; }
ok()      { echo -e "  ${GREEN}✓${NC} $*"; }
warn()    { echo -e "  ${YELLOW}⚠${NC}  $*"; }
die()     { echo -e "\n  ${RED}✗  ERROR:${NC} $*\n" >&2; exit 1; }
section() { echo -e "\n${BOLD}${CYAN}$*${NC}\n$(printf '%.s─' $(seq 1 60))"; }
gen()     { openssl rand -hex 32; }

ask() {
  # ask <VAR> <prompt> [default] [secret]
  local var="$1" prompt="$2" default="${3:-}" secret="${4:-}"
  local display_default=""
  [[ -n "$default" ]] && display_default=" ${DIM}[${default}]${NC}"

  if [[ "$secret" == "secret" ]]; then
    echo -ne "  ${BOLD}${prompt}${NC}${display_default}: "
    read -rs value; echo
  else
    echo -ne "  ${BOLD}${prompt}${NC}${display_default}: "
    read -r value
  fi

  [[ -z "$value" && -n "$default" ]] && value="$default"
  printf -v "$var" '%s' "$value"
}

ask_yn() {
  # ask_yn <prompt> [default y|n] → returns 0 for yes, 1 for no
  local prompt="$1" default="${2:-y}"
  local choices="[Y/n]"; [[ "$default" == "n" ]] && choices="[y/N]"
  echo -ne "  ${BOLD}${prompt}${NC} ${DIM}${choices}${NC} "
  read -r answer
  answer="${answer:-$default}"
  [[ "${answer,,}" == "y" ]]
}

# ── Banner ────────────────────────────────────────────────────────────────────
clear
echo -e "${BOLD}${CYAN}"
echo "  █████╗ ██╗   ██╗████████╗██╗  ██╗███████╗███████╗ ██████╗"
echo " ██╔══██╗██║   ██║╚══██╔══╝██║  ██║██╔════╝██╔════╝██╔════╝"
echo " ███████║██║   ██║   ██║   ███████║███████╗█████╗  ██║     "
echo " ██╔══██║██║   ██║   ██║   ██╔══██║╚════██║██╔══╝  ██║     "
echo " ██║  ██║╚██████╔╝   ██║   ██║  ██║███████║███████╗╚██████╗"
echo " ╚═╝  ╚═╝ ╚═════╝    ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝ ╚═════╝"
echo -e "${NC}"
echo -e "  ${DIM}On-Premise Setup  ·  https://github.com/authsec-ai/onprem${NC}"
echo -e "$(printf '%.s─' $(seq 1 60))\n"

# ── Prerequisites ─────────────────────────────────────────────────────────────
section "Checking prerequisites"

command -v docker   >/dev/null 2>&1 || die "Docker is not installed. See https://docs.docker.com/engine/install/"
docker compose version >/dev/null 2>&1 || die "Docker Compose v2 plugin is required. Run: sudo apt install docker-compose-plugin"
command -v openssl  >/dev/null 2>&1 || die "openssl is required (sudo apt install openssl)"
command -v curl     >/dev/null 2>&1 || die "curl is required (sudo apt install curl)"

ok "Docker        $(docker --version | awk '{print $3}' | tr -d ',')"
ok "Docker Compose $(docker compose version --short)"
ok "openssl and curl found"

# Guard: existing .env
WIPE_VOLUMES=false
if [[ -f .env ]]; then
  echo
  warn ".env already exists."
  if ! ask_yn "Overwrite it with new values?" "n"; then
    echo -e "\n  Keeping existing .env. To re-run setup: rm .env && bash setup.sh\n"
    exit 0
  fi
  echo
  warn "New secrets will be generated, including a new DB_PASSWORD."
  warn "The existing postgres volume must be wiped so postgres can"
  warn "reinitialize with the new password. All stored data will be lost."
  if ask_yn "Wipe existing volumes and start fresh?" "y"; then
    WIPE_VOLUMES=true
  else
    echo
    warn "Keeping volumes. If the DB password changed, hydra-migrate will fail."
    warn "Run 'docker compose down -v' manually before starting if needed."
  fi
fi

# ── Deployment mode ───────────────────────────────────────────────────────────
section "Deployment target"
echo -e "  ${DIM}Choose where you are deploying:${NC}\n"
echo -e "  ${BOLD}1)${NC} Local development  (http://localhost)"
echo -e "  ${BOLD}2)${NC} Self-hosted VM / server  (your own domain + HTTPS)"
echo
echo -ne "  ${BOLD}Choice${NC} ${DIM}[1]${NC}: "
read -r mode_choice
mode_choice="${mode_choice:-1}"

BASE_URL="http://localhost"
APP_DOMAIN="localhost"
DOMAIN="localhost"
WEBAUTHN_ORIGIN="http://localhost"
HYDRA_PUBLIC_URL="http://localhost:4444"
ENVIRONMENT="development"
GIN_MODE="debug"
REQUIRE_SERVER_AUTH="false"
CORS_ALLOW_ORIGIN="http://localhost:3000,http://localhost:5173,http://localhost,http://*.localhost"

if [[ "$mode_choice" == "2" ]]; then
  echo
  ask DOMAIN "Your root domain (e.g. example.com)"
  [[ -z "$DOMAIN" ]] && die "Domain is required for production deployment."

  APP_DOMAIN="$DOMAIN"
  BASE_URL="https://${DOMAIN}"
  WEBAUTHN_ORIGIN="https://${DOMAIN}"
  HYDRA_PUBLIC_URL="https://${DOMAIN}/oauth2"
  ENVIRONMENT="production"
  GIN_MODE="release"
  REQUIRE_SERVER_AUTH="true"
  CORS_ALLOW_ORIGIN="https://${DOMAIN},https://*.${DOMAIN}"

  echo
  ok "Domain set to: ${BOLD}${DOMAIN}${NC}"
  warn "Remember to point ${DOMAIN}'s DNS A record to this server's IP before starting."
  warn "TLS: uncomment the HTTPS server block in nginx/nginx.conf after obtaining certificates."
  warn "     See README.md → Production Deployment for Let's Encrypt instructions."
fi

REACT_APP_URL="${BASE_URL}"
TENANT_DOMAIN_SUFFIX="${DOMAIN}"

# ── Database ──────────────────────────────────────────────────────────────────
section "Database"
echo -e "  ${DIM}Leave blank to auto-generate a secure password.${NC}\n"
ask DB_PASSWORD "PostgreSQL password" "" "secret"
[[ -z "$DB_PASSWORD" ]] && DB_PASSWORD=$(gen) && ok "Auto-generated DB password"

DB_USER="authsec"
DB_NAME="kloudone_db"
HYDRA_DSN="postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/hydra?sslmode=disable"

# ── Secrets (all auto-generated) ─────────────────────────────────────────────
section "Generating secrets"
info "All JWT and encryption keys are auto-generated using openssl..."

JWT_DEF_SECRET=$(gen)
JWT_SECRET=$(gen)
JWT_SDK_SECRET=$(gen)
TOTP_ENCRYPTION_KEY=$(gen)
SYNC_CONFIG_ENCRYPTION_KEY=$(gen)
SESSION_SECRET=$(gen)
HYDRA_SECRETS_SYSTEM=$(gen)
HYDRA_SECRETS_COOKIE=$(gen)
VAULT_TOKEN=$(gen)

ok "JWT_DEF_SECRET, JWT_SECRET, JWT_SDK_SECRET"
ok "TOTP_ENCRYPTION_KEY, SYNC_CONFIG_ENCRYPTION_KEY"
ok "SESSION_SECRET"
ok "HYDRA_SECRETS_SYSTEM, HYDRA_SECRETS_COOKIE"
ok "VAULT_TOKEN"

# ── SMTP (optional) ───────────────────────────────────────────────────────────
section "Email / SMTP  ${DIM}(optional — needed for OTP emails and forgot-password)${NC}"
SMTP_HOST=""
SMTP_PORT="587"
SMTP_USER=""
SMTP_PASSWORD=""

if ask_yn "Configure SMTP now?" "n"; then
  ask SMTP_HOST     "SMTP host"              "smtp.example.com"
  ask SMTP_PORT     "SMTP port"              "587"
  ask SMTP_USER     "SMTP username / email"  ""
  ask SMTP_PASSWORD "SMTP password"          "" "secret"
fi

# ── Social OAuth (optional) ───────────────────────────────────────────────────
section "Social login providers  ${DIM}(optional)${NC}"
GOOGLE_CLIENT_SECRET=""
GITHUB_CLIENT_SECRET=""
MICROSOFT_CLIENT_SECRET=""

if ask_yn "Configure any social OAuth providers now?" "n"; then
  echo -e "  ${DIM}Leave blank to skip a provider.${NC}\n"
  ask GOOGLE_CLIENT_SECRET    "Google client secret"    "" "secret"
  ask GITHUB_CLIENT_SECRET    "GitHub client secret"    "" "secret"
  ask MICROSOFT_CLIENT_SECRET "Microsoft client secret" "" "secret"
fi

# ── Write .env ────────────────────────────────────────────────────────────────
section "Writing .env"

cat > .env <<EOF
# ─────────────────────────────────────────────────────────────────────────────
# AuthSec — generated by setup.sh
# DO NOT commit this file to version control.
# ─────────────────────────────────────────────────────────────────────────────

# ── Deployment ────────────────────────────────────────────────────────────────
ENVIRONMENT=${ENVIRONMENT}
GIN_MODE=${GIN_MODE}
LOG_LEVEL=info
DOMAIN=${DOMAIN}
APP_DOMAIN=${APP_DOMAIN}

# ── URLs ──────────────────────────────────────────────────────────────────────
BASE_URL=${BASE_URL}
REACT_APP_URL=${REACT_APP_URL}
HYDRA_PUBLIC_URL=${HYDRA_PUBLIC_URL}
IDENTITY_PROVIDER_URL=${REACT_APP_URL}
WEBAUTHN_ORIGIN=${WEBAUTHN_ORIGIN}
TENANT_DOMAIN_SUFFIX=${TENANT_DOMAIN_SUFFIX}

# ── CORS ──────────────────────────────────────────────────────────────────────
CORS_ALLOW_ORIGIN=${CORS_ALLOW_ORIGIN}

# ── Auth gate ─────────────────────────────────────────────────────────────────
REQUIRE_SERVER_AUTH=${REQUIRE_SERVER_AUTH}
AUTH_EXPECT_ISS=authsec-ai/auth-manager
AUTH_EXPECT_AUD=authsec

# ── Database ──────────────────────────────────────────────────────────────────
DB_USER=${DB_USER}
DB_NAME=${DB_NAME}
DB_PASSWORD=${DB_PASSWORD}
DB_SCHEMA=public

# Hydra uses a separate database on the same postgres instance
HYDRA_DSN=${HYDRA_DSN}

# ── JWT secrets ───────────────────────────────────────────────────────────────
JWT_DEF_SECRET=${JWT_DEF_SECRET}
JWT_SECRET=${JWT_SECRET}
JWT_SDK_SECRET=${JWT_SDK_SECRET}

# ── Encryption keys ───────────────────────────────────────────────────────────
TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY}
SYNC_CONFIG_ENCRYPTION_KEY=${SYNC_CONFIG_ENCRYPTION_KEY}

# ── Session ───────────────────────────────────────────────────────────────────
SESSION_SECRET=${SESSION_SECRET}

# ── Hydra secrets ─────────────────────────────────────────────────────────────
HYDRA_SECRETS_SYSTEM=${HYDRA_SECRETS_SYSTEM}
HYDRA_SECRETS_COOKIE=${HYDRA_SECRETS_COOKIE}

# ── SMTP ──────────────────────────────────────────────────────────────────────
SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USER=${SMTP_USER}
SMTP_PASSWORD=${SMTP_PASSWORD}

# ── Social OAuth providers ────────────────────────────────────────────────────
GOOGLE_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET}
GITHUB_CLIENT_SECRET=${GITHUB_CLIENT_SECRET}
MICROSOFT_CLIENT_SECRET=${MICROSOFT_CLIENT_SECRET}

# ── HashiCorp Vault ───────────────────────────────────────────────────────────
# Vault runs in dev mode alongside the stack. The token below is the root token.
# Social OAuth provider secrets (Google, Microsoft, GitHub) can be stored here
# via the Vault UI at http://localhost:8200 using this token.
VAULT_TOKEN=${VAULT_TOKEN}

# ── Optional: SPIFFE / AI agent identity delegation ──────────────────────────
SPIFFE_OIDC_ISSUER=
SPIFFE_JWKS_KEY_ID=
SPIFFE_RSA_PRIVATE_KEY_B64=
SPIFFE_TRUST_DOMAIN=

# ── Optional: Okta CIBA ───────────────────────────────────────────────────────
OKTA_DOMAIN=
OKTA_CLIENT_ID=
OKTA_CLIENT_SECRET=
OKTA_ISSUER=
OKTA_API_TOKEN=
EOF

ok ".env written"
warn "Keep .env safe — it contains all your secrets."

# ── Pull images ───────────────────────────────────────────────────────────────
section "Pulling Docker images"
if ask_yn "Pull latest images now? (skip if you already have them)" "y"; then
  docker compose pull
fi

# ── Start services ────────────────────────────────────────────────────────────
section "Starting services"
if [[ "$WIPE_VOLUMES" == "true" ]]; then
  info "Stopping and wiping existing volumes…"
  docker compose down -v 2>&1 | grep -E "Removed|Stopped|Network" || true
  ok "Volumes wiped"
else
  docker compose down 2>&1 | grep -E "Removed|Stopped|Network" || true
fi
docker compose up -d
ok "All containers started"

# ── Wait for AuthSec to be healthy ────────────────────────────────────────────
section "Waiting for AuthSec to be healthy"
echo -e "  ${DIM}This can take 60–90 seconds on first run (DB migrations, template DB build)…${NC}\n"

HEALTH_URL="http://localhost:7468/authsec/uflow/health"
MAX_WAIT=180   # seconds
ELAPSED=0
INTERVAL=5

while true; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$HEALTH_URL" 2>/dev/null || echo "000")
  if [[ "$STATUS" == "200" ]]; then
    ok "AuthSec is healthy"
    break
  fi

  if [[ $ELAPSED -ge $MAX_WAIT ]]; then
    echo
    warn "AuthSec did not become healthy within ${MAX_WAIT}s."
    warn "Check logs with:  docker compose logs -f authsec"
    warn "See README.md → Troubleshooting for next steps."
    exit 1
  fi

  echo -ne "  ${DIM}[${ELAPSED}s] waiting… (status ${STATUS})${NC}\r"
  sleep $INTERVAL
  ELAPSED=$((ELAPSED + INTERVAL))
done

# ── Done ──────────────────────────────────────────────────────────────────────
echo
echo -e "$(printf '%.s═' $(seq 1 60))"
echo -e "${BOLD}${GREEN}  AuthSec is ready!${NC}"
echo -e "$(printf '%.s═' $(seq 1 60))"
echo
echo -e "  ${BOLD}UI (frontend)${NC}           ${BASE_URL}"
echo -e "  ${BOLD}AuthSec API${NC}             ${BASE_URL}/authsec/"
echo -e "  ${BOLD}OIDC discovery${NC}          ${BASE_URL}/.well-known/openid-configuration"
echo -e "  ${BOLD}Hydra public API${NC}        http://localhost:4444"
echo -e "  ${BOLD}Hydra admin API${NC}         http://localhost:4445  ${DIM}(internal only)${NC}"
echo
echo -e "  ${DIM}Useful commands:${NC}"
echo -e "  ${DIM}  docker compose ps            — check service status${NC}"
echo -e "  ${DIM}  docker compose logs -f       — follow all logs${NC}"
echo -e "  ${DIM}  docker compose logs -f authsec — backend logs only${NC}"
echo -e "  ${DIM}  docker compose down          — stop everything${NC}"
echo -e "  ${DIM}  docker compose down -v       — stop + wipe all data${NC}"
echo
if [[ "$mode_choice" == "2" ]]; then
  echo -e "  ${YELLOW}Next steps for production:${NC}"
  echo -e "  ${DIM}  1. Obtain TLS cert:  sudo certbot certonly --standalone -d ${DOMAIN}${NC}"
  echo -e "  ${DIM}  2. Uncomment the HTTPS block in nginx/nginx.conf${NC}"
  echo -e "  ${DIM}  3. docker compose restart nginx${NC}"
  echo
fi

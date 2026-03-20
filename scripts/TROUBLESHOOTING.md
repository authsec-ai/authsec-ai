# AuthSec — Operations & Troubleshooting Guide

## Quick Reference

### Service management

```bash
docker compose ps                        # status of all services
docker compose up -d                     # start everything
docker compose down                      # stop everything (data preserved)
docker compose down -v                   # stop + wipe all volumes (destructive)
docker compose restart authsec           # restart the backend
docker compose restart nginx             # reload nginx
docker compose pull && docker compose up -d  # update to latest images
```

### Logs

```bash
docker compose logs -f                   # follow all services
docker compose logs -f authsec           # backend only
docker compose logs -f hydra             # Hydra OAuth2 server
docker compose logs -f postgres          # database
docker compose logs --tail=200 authsec   # last 200 lines, no follow
docker compose logs --timestamps authsec # include timestamps
```

### Health checks

```bash
# AuthSec backend
curl http://localhost:7468/authsec/uflow/health

# Via nginx (as the UI would call it)
curl http://localhost/authsec/uflow/health

# Hydra public API
curl http://localhost:4444/health/ready

# Hydra admin API
curl http://localhost:4445/health/ready

# OIDC discovery
curl http://localhost/.well-known/openid-configuration

# Postgres
docker compose exec postgres pg_isready -U authsec -d kloudone_db

# Redis
docker compose exec redis redis-cli ping
```

### Database access

```bash
# Master DB shell
docker compose exec postgres psql -U authsec -d kloudone_db

# List all databases (master + tenant DBs)
docker compose exec postgres psql -U authsec -c "\l"

# Check active connections
docker compose exec postgres psql -U authsec -d kloudone_db \
  -c "SELECT count(*) FROM pg_stat_activity;"
```

---

## Troubleshooting: Setup Script

### `setup.sh` fails at prerequisites

```text
✗  ERROR: Docker is not installed.
```

Install Docker following the official guide or the steps in [README.md](README.md#install-docker-ubuntu).

### `setup.sh` fails at "Waiting for AuthSec to be healthy"

The script polls `/authsec/uflow/health` for up to 3 minutes. If it times out:

```bash
# 1. Check which containers are not running
docker compose ps

# 2. Look at the failing service logs
docker compose logs --tail=50 authsec
docker compose logs --tail=50 hydra
docker compose logs --tail=50 hydra-migrate
docker compose logs --tail=50 postgres
```

Common causes and fixes are in the sections below. Once you've resolved the issue, complete setup manually:

```bash
# Register first admin
curl -X POST http://localhost:7468/authsec/uflow/auth/admin/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"YourPassword123!"}'
```

### Re-running `setup.sh`

```bash
# Remove existing .env to allow a clean re-run
rm .env
bash setup.sh
```

---

## Troubleshooting: Services Won't Start

### AuthSec container exits immediately

```bash
docker compose logs --tail=50 authsec
```

**Missing required env var:**

```text
FATAL: JWT_DEF_SECRET is required
```

Open `.env` and ensure none of the required vars are blank. Re-run `setup.sh` or set them manually and restart:

```bash
docker compose up -d authsec
```

**Cannot connect to postgres:**

```text
failed to connect to database: connection refused
```

Postgres may not have finished initialising. AuthSec has a `depends_on: service_healthy` on postgres, but on slow machines the first-run init (creating the `hydra` DB via `postgres/init.sql`) can take longer than the healthcheck allows. Fix:

```bash
# Wait until postgres is healthy
watch docker compose ps postgres

# Then start authsec
docker compose up -d authsec
```

**Cannot reach Hydra:**

```text
dial tcp: lookup hydra: no such host
```

Hydra may still be running its migration. Check:

```bash
docker compose logs hydra-migrate
docker compose ps hydra
```

### `hydra-migrate` fails

```bash
docker compose logs hydra-migrate
```

**Wrong DSN / password mismatch:**

```text
FATAL: password authentication failed for user "authsec"
```

The `HYDRA_DSN` in `.env` must use the same password as `DB_PASSWORD`:

```env
HYDRA_DSN=postgres://authsec:<DB_PASSWORD>@postgres:5432/hydra?sslmode=disable
```

**Hydra DB does not exist:**

```text
FATAL: database "hydra" does not exist
```

The `postgres/init.sql` creates this DB on first start, but only runs when the postgres data volume is empty. If you started postgres before `init.sql` existed, the hydra DB was never created.

```bash
# Option A: full reset (drops all data)
docker compose down -v
docker compose up -d

# Option B: create the DB manually without data loss
docker compose exec postgres psql -U authsec -c "CREATE DATABASE hydra;"
docker compose up -d hydra-migrate
docker compose up -d hydra
docker compose up -d authsec
```

### `hydra-migrate` keeps re-running

`hydra-migrate` has `restart: "no"`. If Docker shows it looping, you may have an older compose version. Upgrade Docker Compose to v2.20+.

### Postgres container is unhealthy

```bash
docker compose logs postgres
```

**Data directory permissions issue (common after a hard reset):**

```text
FATAL: data directory "/var/lib/postgresql/data" has wrong ownership
```

```bash
docker compose down -v    # wipe volumes
docker compose up -d postgres
```

---

## Troubleshooting: Database Issues

### Master migration errors

```bash
# Check what happened — master migrations run automatically on startup
docker compose logs authsec | grep -i migrat
```

### Tenant DB creation fails

AuthSec creates a new database per tenant at registration time. The `authsec` postgres user must have `CREATEDB`. The official postgres Docker image grants this automatically when `POSTGRES_USER` is set — but if you connected an external postgres, you must grant it:

```sql
ALTER ROLE authsec CREATEDB;
```

### Run tenant migrations after an upgrade

```bash
curl -X POST http://localhost/authsec/migration/tenants/migrate-all \
  -H "Authorization: Bearer <admin-jwt>"
```

### Inspect a tenant database

```bash
# List tenant databases (named tenant_<uuid>)
docker compose exec postgres psql -U authsec \
  -c "SELECT datname FROM pg_database WHERE datname LIKE 'tenant_%';"

# Connect to a specific tenant DB
docker compose exec postgres psql -U authsec -d tenant_<uuid>
```

### Backup all databases

```bash
# Master DB only
docker compose exec postgres pg_dump -U authsec kloudone_db \
  > backup_master_$(date +%Y%m%d).sql

# All databases including tenant DBs
docker compose exec postgres pg_dumpall -U authsec \
  > backup_all_$(date +%Y%m%d).sql
```

### Restore a database backup

```bash
# Stop authsec and hydra while restoring
docker compose stop authsec hydra

# Restore master DB
docker compose exec -T postgres psql -U authsec -d kloudone_db \
  < backup_master_20260101.sql

# Restart
docker compose start authsec hydra
```

---

## Troubleshooting: Nginx / Network

### Nginx 502 Bad Gateway

```bash
# Check what nginx is trying to reach
docker compose logs nginx | tail -30

# Verify authsec is healthy
docker compose ps authsec
curl http://localhost:7468/authsec/uflow/health

# Verify UI is running
docker compose ps ui

# Test nginx config validity
docker compose exec nginx nginx -t
```

If authsec is healthy but nginx still 502s, the `depends_on` may have raced. Restart nginx:

```bash
docker compose restart nginx
```

### CORS errors in the browser

`Access to fetch at … has been blocked by CORS policy`

Ensure `CORS_ALLOW_ORIGIN` in `.env` exactly matches the frontend origin (no trailing slash, correct scheme):

```bash
# Check current value
grep CORS_ALLOW_ORIGIN .env

# Example for local dev
CORS_ALLOW_ORIGIN=http://localhost:3000,http://localhost:5173,http://localhost
```

Then restart authsec:

```bash
docker compose restart authsec
```

### Port 80 / 443 already in use

```bash
sudo lsof -i :80
sudo lsof -i :443

# Stop whatever is using the port, then
docker compose up -d nginx
```

### Containers cannot reach each other

```bash
# Verify they are all on the same network
docker network inspect onprem_authsec-net

# Test connectivity between containers
docker compose exec authsec wget -qO- http://postgres:5432 2>&1 | head -1
docker compose exec authsec wget -qO- http://hydra:4444/health/ready
docker compose exec authsec wget -qO- http://redis:6379
```

If a container is missing from the network, bring it down and back up:

```bash
docker compose down <service>
docker compose up -d <service>
```

---

## Troubleshooting: Hydra OAuth2

### Login / consent redirect loops

AuthSec's `hmgr` module handles Hydra's login and consent callbacks. Verify the URLs are reachable from inside the Hydra container:

```bash
docker compose exec hydra wget -qO- http://authsec:7468/authsec/uflow/health
```

Check the Hydra config in docker-compose.yaml:

```yaml
URLS_LOGIN:   http://localhost/authsec/hmgr/login
URLS_CONSENT: http://localhost/authsec/hmgr/consent
```

These must use the **nginx** hostname so all requests go through the reverse proxy.

### `invalid_client` or `client not found`

The OAuth client hasn't been registered in Hydra. Use the Hydra admin API:

```bash
curl http://localhost:4445/admin/clients
```

### Hydra `secret is too short`

`HYDRA_SECRETS_SYSTEM` must be at least 16 characters. If you set it manually, regenerate:

```bash
openssl rand -hex 32
# paste the result into HYDRA_SECRETS_SYSTEM in .env
docker compose restart hydra
```

### Re-run Hydra migration manually

```bash
docker compose up hydra-migrate
```

---

## Troubleshooting: WebAuthn / Passkeys

### `InvalidStateError` or `NotAllowedError` in the browser

- `WEBAUTHN_RP_ID` must be the **hostname only** — no `https://`, no port, no trailing slash.
  - Local dev: `localhost`
  - Production: `example.com`
- `WEBAUTHN_ORIGIN` must be the **full origin** matching what's in the browser's address bar.
  - Local dev: `http://localhost`
  - Production: `https://example.com`

```bash
grep WEBAUTHN .env
```

Restart authsec after changing these:

```bash
docker compose restart authsec
```

### WebAuthn only works on localhost, not on a custom domain without HTTPS

WebAuthn is a Secure Context API. On any hostname other than `localhost`, the browser requires HTTPS. Obtain a TLS certificate before enabling WebAuthn on a custom domain.

---

## Troubleshooting: SSL / TLS

### Obtain Let's Encrypt certificate

```bash
# Stop nginx first to free port 80
docker compose stop nginx

sudo certbot certonly --standalone -d your.domain.com

# Fix permissions for the nginx container
sudo chmod -R 755 /etc/letsencrypt/live/ /etc/letsencrypt/archive/

# Uncomment the HTTPS server block in nginx/nginx.conf, then:
docker compose start nginx
```

### Test certificate renewal

```bash
sudo certbot renew --dry-run
```

### Auto-renew via cron

```bash
echo "0 3 * * * certbot renew --quiet && docker compose -f $(pwd)/docker-compose.yaml restart nginx" \
  | sudo tee /etc/cron.d/authsec-certbot
```

### Certificate not found / nginx won't start after enabling TLS

```bash
# Verify the cert paths match nginx.conf
sudo certbot certificates
docker compose exec nginx ls /etc/letsencrypt/live/
```

---

## Performance

### Container resource usage

```bash
docker stats --no-stream
```

### Add memory limits

Edit `docker-compose.yaml` under any service:

```yaml
    deploy:
      resources:
        limits:
          memory: 1G
        reservations:
          memory: 512M
```

Then: `docker compose up -d`

### Slow queries

```bash
docker compose exec postgres psql -U authsec -d kloudone_db \
  -c "SELECT pid, now() - query_start AS duration, query
      FROM pg_stat_activity
      WHERE state = 'active' AND now() - query_start > interval '5s'
      ORDER BY duration DESC;"
```

### Disk space full

```bash
# Check Docker's disk usage
docker system df

# Remove unused images and stopped containers
docker system prune -a -f

# Rotate container logs
find /var/lib/docker/containers -name '*-json.log' -exec truncate -s 0 {} \;
```

Configure log rotation to prevent this recurring. Create `/etc/logrotate.d/docker-authsec`:

```logrotate
/var/lib/docker/containers/*/*.log {
  rotate 7
  daily
  compress
  size 10M
  missingok
  delaycompress
  copytruncate
}
```

### Enable swap (if RAM is limited)

```bash
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

---

## Monitoring

### Status script

Save as `/usr/local/bin/authsec-status.sh` and `chmod +x`:

```bash
#!/bin/bash
COMPOSE_DIR="$(cd "$(dirname "$0")/../$(dirname "$(realpath "$0")")" && pwd)"
# Adjust COMPOSE_DIR to wherever your docker-compose.yaml lives
COMPOSE_DIR="${AUTHSEC_DIR:-/opt/authsec}"

echo "=== AuthSec Status — $(date) ==="
echo

echo "── Containers ──────────────────────────────────────────"
cd "$COMPOSE_DIR" && docker compose ps
echo

echo "── Resource usage ──────────────────────────────────────"
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}"
echo

echo "── Disk ─────────────────────────────────────────────────"
df -h / /var/lib/docker 2>/dev/null | tail -n +1
echo

echo "── Health endpoints ─────────────────────────────────────"
check() {
  local label="$1" url="$2"
  code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null)
  [[ "$code" =~ ^2 ]] && echo "  ✓ $label ($code)" || echo "  ✗ $label ($code)"
}
check "AuthSec"    "http://localhost:7468/authsec/uflow/health"
check "Hydra"      "http://localhost:4444/health/ready"
check "Nginx"      "http://localhost/authsec/uflow/health"
check "Postgres"   "$(cd "$COMPOSE_DIR" && docker compose exec -T postgres pg_isready -U authsec >/dev/null 2>&1 && echo ok || echo fail)"
```

### Prometheus metrics

AuthSec exposes `/metrics` in Prometheus format. Add a scrape job to your Prometheus config:

```yaml
scrape_configs:
  - job_name: authsec
    static_configs:
      - targets: ['<server-ip>:80']
    metrics_path: /metrics
```

---

## Backup & Recovery

### Automated backup script

Save as `/usr/local/bin/authsec-backup.sh` and `chmod +x`:

```bash
#!/bin/bash
set -euo pipefail
BACKUP_DIR="${BACKUP_DIR:-/backup/authsec}"
DATE=$(date +%Y%m%d_%H%M%S)
KEEP_DAYS=7
COMPOSE_DIR="${AUTHSEC_DIR:-/opt/authsec}"

mkdir -p "$BACKUP_DIR"

echo "[$(date)] Starting backup…"

# Dump all databases (master + all tenant DBs)
cd "$COMPOSE_DIR"
docker compose exec -T postgres pg_dumpall -U authsec \
  | gzip > "${BACKUP_DIR}/db_${DATE}.sql.gz"

echo "[$(date)] Database dump: ${BACKUP_DIR}/db_${DATE}.sql.gz"

# Remove backups older than KEEP_DAYS
find "$BACKUP_DIR" -name "*.sql.gz" -mtime "+${KEEP_DAYS}" -delete

echo "[$(date)] Backup complete."
```

Schedule it:

```bash
echo "0 2 * * * AUTHSEC_DIR=/opt/authsec /usr/local/bin/authsec-backup.sh >> /var/log/authsec-backup.log 2>&1" \
  | sudo tee /etc/cron.d/authsec-backup
```

### Restore from backup

```bash
# Stop authsec and hydra to prevent writes during restore
docker compose stop authsec hydra

# Decompress and restore
gunzip -c /backup/authsec/db_20260101_020000.sql.gz \
  | docker compose exec -T postgres psql -U authsec

# Restart
docker compose start authsec hydra
```

### Full reset (wipes everything)

```bash
docker compose down -v          # removes containers and volumes
docker compose up -d            # fresh start with empty databases
# Then re-run setup steps from README → First-Time Setup
```

---

## Security Hardening

- **Rotate secrets regularly.** Update `.env`, then `docker compose up -d` to restart containers.
- **Bind internal ports to 127.0.0.1.** Postgres (5432), Redis (6379), Hydra admin (4445), and AuthSec (7468) are already bound to `127.0.0.1` in `docker-compose.yaml`. Never expose them on `0.0.0.0` in production.
- **Use TLS.** All production traffic should go through nginx over HTTPS. See [README.md — Production Deployment](README.md#production-deployment).
- **Set `REQUIRE_SERVER_AUTH=true`** in production.
- **Set `GIN_MODE=release`** and `ENVIRONMENT=production` to suppress debug output.
- **Fail2ban** for SSH brute-force protection: `sudo apt install fail2ban`
- **Unattended security upgrades:** `sudo apt install unattended-upgrades`

---

## Gathering Diagnostic Info

When reporting a bug or asking for help, include the output of:

```bash
# System and Docker versions
uname -a
docker version
docker compose version

# Container status
docker compose ps

# Last 100 lines from the failing service
docker compose logs --tail=100 <service-name>

# Resource usage
docker stats --no-stream
df -h
free -h
```

Open an issue at: **[github.com/authsec-ai/onprem/issues](https://github.com/authsec-ai/onprem/issues)**

---

**Last updated:** March 2026
**Version:** 2.0.0 (monolith)

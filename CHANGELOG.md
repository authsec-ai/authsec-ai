# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Graceful shutdown with 30-second drain period
- Docker HEALTHCHECK instruction
- HSTS support behind reverse proxies (`X-Forwarded-Proto`)
- JWT audience validation enforcement in production
- Secret masking in config logging
- Startup validation for required secrets
- Real system metrics (goroutines, memory) in Prometheus endpoint
- Lint, vet, and unit test stages in CI pipeline
- Rolling deployment (replaces pod deletion)
- Community files: CONTRIBUTING.md, CODE_OF_CONDUCT.md, issue/PR templates
- CHANGELOG.md
- golangci-lint configuration
- Redis-backed distributed rate limiting
- Circuit breakers for external service calls (Hydra, Vault)
- Middleware unit tests (security headers, auth, token blacklist, tenant resolution)
- Test coverage gates in CI

### Changed
- Password minimum length standardised to 10 characters
- Redis `KEYS` command replaced with cursor-based `SCAN`
- Migrated from `golang-jwt/jwt/v4` to `jwt/v5` (single version)
- Upgraded `go-redis` from v8 to v9

### Removed
- Debug JWT secret endpoint (`POST /debug/jwt-secret`)
- Hardcoded default credentials in config
- `postgresql-client` from production Docker image

### Security
- Auth error responses no longer leak internal error details
- Sensitive environment variable values masked in logs
- Service refuses to start without required secrets configured

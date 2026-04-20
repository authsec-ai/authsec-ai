# AuthSec API — cURL Reference

**Base URL:** `http://localhost:7468`

```bash
BASE="http://localhost:7468"
TOKEN="<your-jwt-token>"
TENANT_ID="<tenant-uuid>"
USER_ID="<user-uuid>"
```

---

## Well-Known / OIDC Discovery

```bash
# OpenID Configuration
curl "$BASE/authsec/.well-known/openid-configuration"

# JWKS (public keys)
curl "$BASE/authsec/.well-known/jwks.json"
```

---

## Debug

```bash
# Reveal JWT secret (development only)
curl -X POST "$BASE/authsec/uflow/debug/jwt-secret"
```

---

## Health

```bash
# Global health check
curl "$BASE/authsec/uflow/health"

# Tenant DB health
curl "$BASE/authsec/uflow/health/tenant/$TENANT_ID"

# All tenant DBs health
curl "$BASE/authsec/uflow/health/tenants"
```

---

## Admin Authentication  `/authsec/uflow/auth/admin`

```bash
# Get auth challenge
curl "$BASE/authsec/uflow/auth/admin/challenge"

# Pre-check before login (check if user exists, get MFA hint)
curl -X POST "$BASE/authsec/uflow/auth/admin/login/precheck" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Bootstrap first admin (initial setup)
curl -X POST "$BASE/authsec/uflow/auth/admin/login/bootstrap" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"changeme","tenant_id":"'"$TENANT_ID"'"}'

# Login
curl -X POST "$BASE/authsec/uflow/auth/admin/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"changeme","tenant_id":"'"$TENANT_ID"'"}'

# Login hybrid (password + MFA in one call)
curl -X POST "$BASE/authsec/uflow/auth/admin/login-hybrid" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"changeme","tenant_id":"'"$TENANT_ID"'","totp_code":"123456"}'

# Register admin
curl -X POST "$BASE/authsec/uflow/auth/admin/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"changeme","tenant_id":"'"$TENANT_ID"'","first_name":"Alice","last_name":"Admin"}'

# Complete registration (from invite link)
curl -X POST "$BASE/authsec/uflow/auth/admin/complete-registration" \
  -H "Content-Type: application/json" \
  -d '{"token":"<registration-token>","password":"changeme"}'

# Forgot password — send OTP
curl -X POST "$BASE/authsec/uflow/auth/admin/forgot-password" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Forgot password — verify OTP
curl -X POST "$BASE/authsec/uflow/auth/admin/forgot-password/verify-otp" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","tenant_id":"'"$TENANT_ID"'","otp":"123456"}'

# Forgot password — reset
curl -X POST "$BASE/authsec/uflow/auth/admin/forgot-password/reset" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","tenant_id":"'"$TENANT_ID"'","token":"<reset-token>","new_password":"newpass123"}'
```

---

## End-User Authentication  `/authsec/uflow/auth/enduser`

```bash
# Get auth challenge
curl "$BASE/authsec/uflow/auth/enduser/challenge"

# Initiate end-user registration (sends OTP)
curl -X POST "$BASE/authsec/uflow/auth/enduser/initiate-registration" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'"}'

# Verify OTP and complete registration
curl -X POST "$BASE/authsec/uflow/auth/enduser/verify-otp" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","otp":"123456"}'

# Login pre-check
curl -X POST "$BASE/authsec/uflow/auth/enduser/login/precheck" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# WebAuthn callback (after passkey assertion)
curl -X POST "$BASE/authsec/uflow/auth/enduser/webauthn-callback" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'

# SPIFFE SVID delegation
curl -X POST "$BASE/authsec/uflow/auth/enduser/delegate-svid" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"workload_id":"spiffe://example.org/workload"}'
```

---

## End-User Self-Service  `/authsec/user`

```bash
# Login (custom password login)
curl -X POST "$BASE/authsec/uflow/user/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'"}'

# Login status (poll for async logins)
curl -X POST "$BASE/authsec/uflow/user/login/status" \
  -H "Content-Type: application/json" \
  -d '{"request_id":"<request-id>"}'

# SAML login
curl -X POST "$BASE/authsec/uflow/user/saml/login" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","relay_state":"<state>"}'

# OIDC login
curl -X POST "$BASE/authsec/uflow/user/oidc/login" \
  -H "Content-Type: application/json" \
  -d '{"provider":"google","tenant_id":"'"$TENANT_ID"'"}'

# Initiate registration
curl -X POST "$BASE/authsec/uflow/user/register/initiate" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'"}'

# Complete registration
curl -X POST "$BASE/authsec/uflow/user/register/complete" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","otp":"123456","password":"pass"}'

# Register (combined)
curl -X POST "$BASE/authsec/uflow/user/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'"}'

# Forgot password — send OTP
curl -X POST "$BASE/authsec/uflow/user/forgot-password" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Forgot password — verify OTP
curl -X POST "$BASE/authsec/uflow/user/forgot-password/verify-otp" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","otp":"123456"}'

# Forgot password — reset
curl -X POST "$BASE/authsec/uflow/user/forgot-password/reset" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","token":"<reset-token>","new_password":"newpass"}'

# ── Authenticated below ──────────────────────────────────────────────────────

# Register client (device/app)
curl -X POST "$BASE/authsec/uflow/user/clients/register" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"client_name":"My App","tenant_id":"'"$TENANT_ID"'"}'

# Get clients
curl "$BASE/authsec/uflow/user/clients" \
  -H "Authorization: Bearer $TOKEN"

# Get specific end-user
curl "$BASE/authsec/uflow/user/enduser/$TENANT_ID/$USER_ID" \
  -H "Authorization: Bearer $TOKEN"

# List end-users
curl -X POST "$BASE/authsec/uflow/user/enduser/list" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","page":1,"limit":20}'

# Update end-user
curl -X PUT "$BASE/authsec/uflow/user/enduser/$TENANT_ID/$USER_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"first_name":"Bob","last_name":"Smith"}'

# Update end-user status
curl -X PUT "$BASE/authsec/uflow/user/enduser/$TENANT_ID/$USER_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"active":true}'

# Delete end-user
curl -X DELETE "$BASE/authsec/uflow/user/enduser/$TENANT_ID/$USER_ID" \
  -H "Authorization: Bearer $TOKEN"

# Admin change user password
curl -X POST "$BASE/authsec/uflow/user/admin/change-password" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"'"$USER_ID"'","new_password":"newpass","tenant_id":"'"$TENANT_ID"'"}'

# Admin reset user password (sends reset email)
curl -X POST "$BASE/authsec/uflow/user/admin/reset-password" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"'"$USER_ID"'","tenant_id":"'"$TENANT_ID"'"}'
```

---

## TOTP (User-flow)  `/authsec/uflow/auth/totp`

```bash
# Login with TOTP code
curl -X POST "$BASE/authsec/uflow/auth/totp/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","code":"123456"}'

# Approve device code with TOTP
curl -X POST "$BASE/authsec/uflow/auth/totp/device-approve" \
  -H "Content-Type: application/json" \
  -d '{"device_code":"<code>","totp_code":"123456"}'

# Register TOTP device (returns QR code)
curl -X POST "$BASE/authsec/uflow/auth/totp/register" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_name":"My Authenticator","tenant_id":"'"$TENANT_ID"'"}'

# Confirm TOTP registration
curl -X POST "$BASE/authsec/uflow/auth/totp/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"secret":"<base32-secret>","code":"123456","tenant_id":"'"$TENANT_ID"'"}'

# Verify TOTP (for actions requiring step-up)
curl -X POST "$BASE/authsec/uflow/auth/totp/verify" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code":"123456","tenant_id":"'"$TENANT_ID"'"}'

# List TOTP devices
curl "$BASE/authsec/uflow/auth/totp/devices" \
  -H "Authorization: Bearer $TOKEN"

# Delete TOTP device
curl -X POST "$BASE/authsec/uflow/auth/totp/device/delete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_id":"<device-uuid>","tenant_id":"'"$TENANT_ID"'"}'

# Set primary TOTP device
curl -X POST "$BASE/authsec/uflow/auth/totp/device/primary" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_id":"<device-uuid>","tenant_id":"'"$TENANT_ID"'"}'

# Regenerate backup codes
curl -X POST "$BASE/authsec/uflow/auth/totp/backup/regenerate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'"}'
```

---

## Tenant TOTP  `/authsec/uflow/auth/tenant/totp`

```bash
# Login with tenant TOTP
curl -X POST "$BASE/authsec/uflow/auth/tenant/totp/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","code":"123456"}'

# Register tenant TOTP device
curl -X POST "$BASE/authsec/uflow/auth/tenant/totp/register" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_name":"Work Authenticator","tenant_id":"'"$TENANT_ID"'"}'

# Confirm tenant TOTP device
curl -X POST "$BASE/authsec/uflow/auth/tenant/totp/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"secret":"<secret>","code":"123456","tenant_id":"'"$TENANT_ID"'"}'

# List tenant TOTP devices
curl "$BASE/authsec/uflow/auth/tenant/totp/devices" \
  -H "Authorization: Bearer $TOKEN"

# Delete tenant TOTP device
curl -X POST "$BASE/authsec/uflow/auth/tenant/totp/devices/delete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_id":"<device-uuid>","tenant_id":"'"$TENANT_ID"'"}'

# Set primary tenant TOTP device
curl -X POST "$BASE/authsec/uflow/auth/tenant/totp/devices/primary" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_id":"<device-uuid>","tenant_id":"'"$TENANT_ID"'"}'
```

---

## CIBA  `/authsec/uflow/auth/ciba`

```bash
# Initiate CIBA authentication
curl -X POST "$BASE/authsec/uflow/auth/ciba/initiate" \
  -H "Content-Type: application/json" \
  -d '{"client_id":"'"$CLIENT_ID"'","login_hint":"user@example.com","scope":"openid profile","binding_message":"Login to App"}'

# Poll for CIBA token
curl -X POST "$BASE/authsec/uflow/auth/ciba/token" \
  -H "Content-Type: application/json" \
  -d '{"auth_req_id":"<auth-req-id>","client_id":"'"$CLIENT_ID"'"}'

# Respond to CIBA request (approve/deny on device)
curl -X POST "$BASE/authsec/uflow/auth/ciba/respond" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"auth_req_id":"<auth-req-id>","approved":true}'

# Register push device for CIBA
curl -X POST "$BASE/authsec/uflow/auth/ciba/register-device" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_token":"<fcm-token>","device_name":"iPhone 15"}'

# List CIBA devices
curl "$BASE/authsec/uflow/auth/ciba/devices" \
  -H "Authorization: Bearer $TOKEN"

# Delete CIBA device
curl -X DELETE "$BASE/authsec/uflow/auth/ciba/devices/<device_id>" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Tenant CIBA  `/authsec/uflow/auth/tenant/ciba`

```bash
# Initiate tenant CIBA
curl -X POST "$BASE/authsec/uflow/auth/tenant/ciba/initiate" \
  -H "Content-Type: application/json" \
  -d '{"client_id":"'"$CLIENT_ID"'","login_hint":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Poll tenant CIBA token
curl -X POST "$BASE/authsec/uflow/auth/tenant/ciba/token" \
  -H "Content-Type: application/json" \
  -d '{"auth_req_id":"<auth-req-id>","client_id":"'"$CLIENT_ID"'"}'

# Respond to tenant CIBA
curl -X POST "$BASE/authsec/uflow/auth/tenant/ciba/respond" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"auth_req_id":"<auth-req-id>","approved":true}'

# Register tenant CIBA device
curl -X POST "$BASE/authsec/uflow/auth/tenant/ciba/register-device" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_token":"<fcm-token>","device_name":"My Phone"}'

# List tenant CIBA requests
curl "$BASE/authsec/uflow/auth/tenant/ciba/requests" \
  -H "Authorization: Bearer $TOKEN"

# List tenant CIBA devices
curl "$BASE/authsec/uflow/auth/tenant/ciba/devices" \
  -H "Authorization: Bearer $TOKEN"

# Delete tenant CIBA device
curl -X DELETE "$BASE/authsec/uflow/auth/tenant/ciba/devices/<device_id>" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Device Authorization  `/authsec/uflow/auth/device`

```bash
# Request device code (for TV / CLI flows)
curl -X POST "$BASE/authsec/uflow/auth/device/code" \
  -H "Content-Type: application/json" \
  -d '{"client_id":"'"$CLIENT_ID"'","scope":"openid profile"}'

# Poll for device token
curl -X POST "$BASE/authsec/uflow/auth/device/token" \
  -H "Content-Type: application/json" \
  -d '{"device_code":"<device-code>","client_id":"'"$CLIENT_ID"'"}'

# Get activation info (from device browser page)
curl "$BASE/authsec/uflow/auth/device/activate/info?user_code=ABCD-1234"

# Device activation page (HTML)
curl "$BASE/authsec/uflow/activate?user_code=ABCD-1234"

# Verify device code (user approves on phone)
curl -X POST "$BASE/authsec/uflow/auth/device/verify" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_code":"ABCD-1234"}'
```

---

## Voice Authentication  `/authsec/uflow/auth/voice`

```bash
# Initiate voice auth
curl -X POST "$BASE/authsec/uflow/auth/voice/initiate" \
  -H "Content-Type: application/json" \
  -d '{"phone_number":"+15551234567","tenant_id":"'"$TENANT_ID"'"}'

# Verify voice OTP
curl -X POST "$BASE/authsec/uflow/auth/voice/verify" \
  -H "Content-Type: application/json" \
  -d '{"phone_number":"+15551234567","code":"123456","tenant_id":"'"$TENANT_ID"'"}'

# Get token with voice credentials
curl -X POST "$BASE/authsec/uflow/auth/voice/token" \
  -H "Content-Type: application/json" \
  -d '{"phone_number":"+15551234567","code":"123456","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'"}'

# Link voice assistant
curl -X POST "$BASE/authsec/uflow/auth/voice/link" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"assistant_type":"alexa","device_id":"<device-id>"}'

# Unlink voice assistant
curl -X POST "$BASE/authsec/uflow/auth/voice/unlink" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"assistant_type":"alexa"}'

# List voice links
curl "$BASE/authsec/uflow/auth/voice/links" \
  -H "Authorization: Bearer $TOKEN"

# Get pending device codes
curl "$BASE/authsec/uflow/auth/voice/device-pending" \
  -H "Authorization: Bearer $TOKEN"

# Approve device code via voice
curl -X POST "$BASE/authsec/uflow/auth/voice/device-approve" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_code":"<device-code>"}'
```

---

## OIDC  `/authsec/oidc`

```bash
# List OIDC providers
curl "$BASE/authsec/uflow/oidc/providers"

# Initiate OIDC flow
curl -X POST "$BASE/authsec/uflow/oidc/initiate" \
  -H "Content-Type: application/json" \
  -d '{"provider":"google","tenant_id":"'"$TENANT_ID"'","redirect_uri":"https://app.example.com/callback"}'

# Initiate OIDC registration
curl -X POST "$BASE/authsec/uflow/oidc/register/initiate" \
  -H "Content-Type: application/json" \
  -d '{"provider":"google","tenant_id":"'"$TENANT_ID"'"}'

# Initiate OIDC login
curl -X POST "$BASE/authsec/uflow/oidc/login/initiate" \
  -H "Content-Type: application/json" \
  -d '{"provider":"google","tenant_id":"'"$TENANT_ID"'"}'

# OIDC callback (GET, called by provider)
curl "$BASE/authsec/uflow/oidc/callback?code=<code>&state=<state>"

# Exchange authorization code for tokens
curl -X POST "$BASE/authsec/uflow/oidc/exchange-code" \
  -H "Content-Type: application/json" \
  -d '{"code":"<auth-code>","state":"<state>","tenant_id":"'"$TENANT_ID"'"}'

# Complete OIDC registration
curl -X POST "$BASE/authsec/uflow/oidc/complete-registration" \
  -H "Content-Type: application/json" \
  -d '{"token":"<oidc-token>","tenant_id":"'"$TENANT_ID"'"}'

# Check if tenant exists for domain
curl "$BASE/authsec/uflow/oidc/check-tenant?domain=example.com"

# Get auth URL for provider
curl -X POST "$BASE/authsec/uflow/oidc/auth-url" \
  -H "Content-Type: application/json" \
  -d '{"provider":"google","tenant_id":"'"$TENANT_ID"'","redirect_uri":"https://app.example.com/callback"}'

# Link OIDC identity (authenticated)
curl -X POST "$BASE/authsec/uflow/oidc/link" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider":"github","code":"<auth-code>"}'

# Get linked identities
curl "$BASE/authsec/uflow/oidc/identities" \
  -H "Authorization: Bearer $TOKEN"

# Unlink OIDC provider
curl -X DELETE "$BASE/authsec/uflow/oidc/unlink/google" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Admin Management  `/authsec/admin`

### Tenants

```bash
# List tenants
curl "$BASE/authsec/uflow/admin/tenants" \
  -H "Authorization: Bearer $TOKEN"

# Create tenant
curl -X POST "$BASE/authsec/uflow/admin/tenants" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Corp","slug":"acme","domain":"acme.app.authsec.dev"}'

# Update tenant
curl -X PUT "$BASE/authsec/uflow/admin/tenants/$TENANT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Corp Updated"}'

# Delete tenant
curl -X DELETE "$BASE/authsec/uflow/admin/tenants/$TENANT_ID" \
  -H "Authorization: Bearer $TOKEN"

# List users in tenant
curl "$BASE/authsec/uflow/admin/tenants/$TENANT_ID/users" \
  -H "Authorization: Bearer $TOKEN"
```

### Tenant Domains

```bash
# Add domain to tenant
curl -X POST "$BASE/authsec/uflow/admin/tenants/$TENANT_ID/domains" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"domain":"custom.example.com"}'

# List tenant domains
curl "$BASE/authsec/uflow/admin/tenants/$TENANT_ID/domains" \
  -H "Authorization: Bearer $TOKEN"

# Verify domain ownership
curl -X POST "$BASE/authsec/uflow/admin/tenants/$TENANT_ID/domains/<domain_id>/verify" \
  -H "Authorization: Bearer $TOKEN"

# Set primary domain
curl -X POST "$BASE/authsec/uflow/admin/tenants/$TENANT_ID/domains/<domain_id>/set-primary" \
  -H "Authorization: Bearer $TOKEN"

# Get domain by ID
curl "$BASE/authsec/uflow/admin/tenants/$TENANT_ID/domains/<domain_id>" \
  -H "Authorization: Bearer $TOKEN"

# Delete domain
curl -X DELETE "$BASE/authsec/uflow/admin/tenants/$TENANT_ID/domains/<domain_id>" \
  -H "Authorization: Bearer $TOKEN"
```

### Admin Users

```bash
# List admin users
curl -X POST "$BASE/authsec/uflow/admin/users/list" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"page":1,"limit":20}'

# Delete admin user
curl -X DELETE "$BASE/authsec/uflow/admin/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN"

# Toggle admin user active/inactive
curl -X POST "$BASE/authsec/uflow/admin/users/active" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"'"$USER_ID"'","active":true}'

# List end-users by tenant
curl -X POST "$BASE/authsec/uflow/admin/enduser/list" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","page":1,"limit":20}'

# Toggle end-user active/inactive
curl -X POST "$BASE/authsec/uflow/admin/enduser/active" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"'"$USER_ID"'","tenant_id":"'"$TENANT_ID"'","active":false}'
```

### Admin Invites

```bash
# Invite admin
curl -X POST "$BASE/authsec/uflow/admin/invite" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"newadmin@example.com","tenant_id":"'"$TENANT_ID"'","role":"admin"}'

# Cancel invite
curl -X POST "$BASE/authsec/uflow/admin/invite/cancel" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"invite_id":"<invite-uuid>"}'

# Resend invite
curl -X POST "$BASE/authsec/uflow/admin/invite/resend" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"invite_id":"<invite-uuid>"}'

# List pending invites
curl "$BASE/authsec/uflow/admin/invite/pending" \
  -H "Authorization: Bearer $TOKEN"
```

### Projects

```bash
# Create project
curl -X POST "$BASE/authsec/uflow/admin/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"My Project","tenant_id":"'"$TENANT_ID"'"}'

# List projects
curl "$BASE/authsec/uflow/admin/projects" \
  -H "Authorization: Bearer $TOKEN"
```

### Groups

```bash
# Create user-defined group
curl -X POST "$BASE/authsec/uflow/admin/groups" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Engineering","tenant_id":"'"$TENANT_ID"'"}'

# List groups for tenant (admin)
curl -X POST "$BASE/authsec/uflow/admin/groups/list" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'"}'

# Get groups for tenant
curl "$BASE/authsec/uflow/admin/groups/$TENANT_ID" \
  -H "Authorization: Bearer $TOKEN"

# Update group
curl -X PUT "$BASE/authsec/uflow/admin/groups/<group_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Platform Engineering"}'

# Delete group
curl -X DELETE "$BASE/authsec/uflow/admin/groups" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"group_id":"<group-uuid>"}'

# Bulk add users to group
curl -X POST "$BASE/authsec/uflow/admin/groups/$TENANT_ID/users/bulk" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"group_id":"<group-uuid>","user_ids":["<uid1>","<uid2>"]}'

# Bulk remove users from group
curl -X DELETE "$BASE/authsec/uflow/admin/groups/$TENANT_ID/users/bulk" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"group_id":"<group-uuid>","user_ids":["<uid1>"]}'

# Map groups to client
curl -X POST "$BASE/authsec/uflow/admin/groups/map" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"client_id":"'"$CLIENT_ID"'","group_ids":["<gid1>"]}'

# Remove groups from client
curl -X DELETE "$BASE/authsec/uflow/admin/groups/map" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"client_id":"'"$CLIENT_ID"'","group_ids":["<gid1>"]}'
```

---

## RBAC  `/authsec/admin` and `/authsec/user`

### Roles

```bash
# Create role (admin)
curl -X POST "$BASE/authsec/uflow/admin/roles" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"viewer","permissions":["read:users"],"tenant_id":"'"$TENANT_ID"'"}'

# List roles (admin)
curl "$BASE/authsec/uflow/admin/roles" \
  -H "Authorization: Bearer $TOKEN"

# Update role (admin)
curl -X PUT "$BASE/authsec/uflow/admin/roles/<role_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"viewer-updated"}'

# Delete role (admin)
curl -X DELETE "$BASE/authsec/uflow/admin/roles/<role_id>" \
  -H "Authorization: Bearer $TOKEN"

# Create role (end-user context)
curl -X POST "$BASE/authsec/uflow/user/rbac/roles" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"app-admin","tenant_id":"'"$TENANT_ID"'"}'

# List roles (end-user context)
curl "$BASE/authsec/uflow/user/rbac/roles" \
  -H "Authorization: Bearer $TOKEN"
```

### Role Bindings

```bash
# Assign role (admin)
curl -X POST "$BASE/authsec/uflow/admin/bindings" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"'"$USER_ID"'","role_id":"<role-uuid>","tenant_id":"'"$TENANT_ID"'"}'

# List role bindings (admin)
curl "$BASE/authsec/uflow/admin/bindings" \
  -H "Authorization: Bearer $TOKEN"

# Assign role (end-user context)
curl -X POST "$BASE/authsec/uflow/user/rbac/bindings" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"'"$USER_ID"'","role_id":"<role-uuid>","tenant_id":"'"$TENANT_ID"'"}'
```

### Permissions

```bash
# Register permission (admin)
curl -X POST "$BASE/authsec/uflow/admin/permissions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"resource":"documents","action":"read","tenant_id":"'"$TENANT_ID"'"}'

# List permissions (admin)
curl "$BASE/authsec/uflow/admin/permissions" \
  -H "Authorization: Bearer $TOKEN"

# Delete permission
curl -X DELETE "$BASE/authsec/uflow/admin/permissions/<permission_id>" \
  -H "Authorization: Bearer $TOKEN"

# Policy check (admin)
curl -X POST "$BASE/authsec/uflow/admin/policy/check" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"'"$USER_ID"'","resource":"documents","action":"read","tenant_id":"'"$TENANT_ID"'"}'

# Get my permissions
curl "$BASE/authsec/uflow/user/permissions" \
  -H "Authorization: Bearer $TOKEN"

# Get my effective permissions
curl "$BASE/authsec/uflow/user/permissions/effective" \
  -H "Authorization: Bearer $TOKEN"

# Check single permission
curl "$BASE/authsec/uflow/user/permissions/check?resource=documents&action=read" \
  -H "Authorization: Bearer $TOKEN"

# Policy check (end-user)
curl -X POST "$BASE/authsec/uflow/user/rbac/policy/check" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"resource":"documents","action":"write"}'
```

### Scopes & API Scopes

```bash
# List scopes (admin)
curl "$BASE/authsec/uflow/admin/scopes" \
  -H "Authorization: Bearer $TOKEN"

# Add scope (admin)
curl -X POST "$BASE/authsec/uflow/admin/scopes" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"read:reports","description":"Read access to reports","tenant_id":"'"$TENANT_ID"'"}'

# Create API scope (admin)
curl -X POST "$BASE/authsec/uflow/admin/api_scopes" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"api:read","resource":"GET /api/data","tenant_id":"'"$TENANT_ID"'"}'

# List API scopes (admin)
curl "$BASE/authsec/uflow/admin/api_scopes" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Active Directory / Entra ID Sync  `/authsec/admin`

```bash
# Sync AD users
curl -X POST "$BASE/authsec/uflow/admin/ad/sync" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","ldap_url":"ldap://dc.corp.local","base_dn":"DC=corp,DC=local","username":"svc@corp.local","password":"pass"}'

# Test AD connection
curl -X POST "$BASE/authsec/uflow/admin/ad/test-connection" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ldap_url":"ldap://dc.corp.local","base_dn":"DC=corp,DC=local","username":"svc@corp.local","password":"pass"}'

# Test network connectivity
curl -X POST "$BASE/authsec/uflow/admin/ad/test-network" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"host":"dc.corp.local","port":389}'

# Agent-based AD sync
curl -X POST "$BASE/authsec/uflow/admin/ad/agent-sync" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","agent_id":"<agent-uuid>"}'

# Sync Entra ID (Azure AD) users
curl -X POST "$BASE/authsec/uflow/admin/entra/sync" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","azure_tenant_id":"<azure-tid>","client_id":"<app-id>","client_secret":"<secret>"}'

# Test Entra ID connection
curl -X POST "$BASE/authsec/uflow/admin/entra/test-connection" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"azure_tenant_id":"<azure-tid>","client_id":"<app-id>","client_secret":"<secret>"}'

# Sync AD admin users
curl -X POST "$BASE/authsec/uflow/admin/admin-users/ad/sync" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'"}'

# Sync Entra admin users
curl -X POST "$BASE/authsec/uflow/admin/admin-users/entra/sync" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'"}'
```

---

## SCIM 2.0  `/authsec/uflow/scim/v2`

```bash
# Discovery (public)
curl "$BASE/authsec/uflow/scim/v2/ServiceProviderConfig"
curl "$BASE/authsec/uflow/scim/v2/Schemas"
curl "$BASE/authsec/uflow/scim/v2/ResourceTypes"

# End-user provisioning (Bearer = SCIM token)
SCIM_TOKEN="<scim-token>"
CLIENT_ID="<client-uuid>"
PROJECT_ID="<project-uuid>"

# List users
curl "$BASE/authsec/uflow/scim/v2/$CLIENT_ID/$PROJECT_ID/Users" \
  -H "Authorization: Bearer $SCIM_TOKEN"

# Get user
curl "$BASE/authsec/uflow/scim/v2/$CLIENT_ID/$PROJECT_ID/Users/<scim-user-id>" \
  -H "Authorization: Bearer $SCIM_TOKEN"

# Create user
curl -X POST "$BASE/authsec/uflow/scim/v2/$CLIENT_ID/$PROJECT_ID/Users" \
  -H "Authorization: Bearer $SCIM_TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":"user@example.com","emails":[{"value":"user@example.com","primary":true}]}'

# Replace user
curl -X PUT "$BASE/authsec/uflow/scim/v2/$CLIENT_ID/$PROJECT_ID/Users/<scim-user-id>" \
  -H "Authorization: Bearer $SCIM_TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":"user@example.com","active":true}'

# Patch user (activate/deactivate)
curl -X PATCH "$BASE/authsec/uflow/scim/v2/$CLIENT_ID/$PROJECT_ID/Users/<scim-user-id>" \
  -H "Authorization: Bearer $SCIM_TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"replace","path":"active","value":false}]}'

# Delete user
curl -X DELETE "$BASE/authsec/uflow/scim/v2/$CLIENT_ID/$PROJECT_ID/Users/<scim-user-id>" \
  -H "Authorization: Bearer $SCIM_TOKEN"

# List groups
curl "$BASE/authsec/uflow/scim/v2/$CLIENT_ID/$PROJECT_ID/Groups" \
  -H "Authorization: Bearer $SCIM_TOKEN"

# Generate SCIM token
curl -X POST "$BASE/authsec/uflow/admin/scim/generate-token" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'"}'
```

---

## Legacy Login  `/authsec`

```bash
# Login (legacy path)
curl -X POST "$BASE/authsec/uflow/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass","tenant_id":"'"$TENANT_ID"'"}'

# Verify OTP and complete registration (legacy)
curl -X POST "$BASE/authsec/uflow/register/verify" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","otp":"123456","tenant_id":"'"$TENANT_ID"'"}'

# WebAuthn callback (legacy)
curl -X POST "$BASE/authsec/uflow/login/webauthn-callback" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'
```

---

## WebAuthn / Passkeys  `/webauthn`

### WebAuthn Health

```bash
curl "$BASE/authsec/uflow/health"
```

### MFA Status

```bash
# WebAuthn MFA login status (root-level)
curl -X POST "$BASE/authsec/webauthn/mfa/loginStatus" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Legacy flat endpoints
curl -X POST "$BASE/authsec/webauthn/mfa/status" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

curl -X POST "$BASE/authsec/webauthn/mfa/loginStatus" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

curl "$BASE/authsec/webauthn/mfa/loginStatus?email=user@example.com&tenant_id=$TENANT_ID"
```

### Legacy WebAuthn Registration & Authentication

```bash
# Begin registration
curl -X POST "$BASE/authsec/webauthn/beginRegistration" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Begin WebAuthn registration (alternate path)
curl -X POST "$BASE/authsec/webauthn/beginAuthRegistration" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Finish registration
curl -X POST "$BASE/authsec/webauthn/finishRegistration" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'

# Begin authentication
curl -X POST "$BASE/authsec/webauthn/beginAuthentication" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Finish authentication
curl -X POST "$BASE/authsec/webauthn/finishAuthentication" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'
```

### Biometric (Passkey Setup)

```bash
# Begin biometric setup (new passkey for MFA)
curl -X POST "$BASE/authsec/webauthn/biometric/beginSetup" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Confirm biometric setup
curl -X POST "$BASE/authsec/webauthn/biometric/confirmSetup" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'

# Begin biometric login setup
curl -X POST "$BASE/authsec/webauthn/biometric/beginLoginSetup" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Confirm biometric login setup
curl -X POST "$BASE/authsec/webauthn/biometric/confirmLoginSetup" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'

# Begin biometric verify (MFA challenge)
curl -X POST "$BASE/authsec/webauthn/biometric/verifyBegin" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Finish biometric verify
curl -X POST "$BASE/authsec/webauthn/biometric/verifyFinish" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'

# Begin biometric login verify
curl -X POST "$BASE/authsec/webauthn/biometric/verifyLoginBegin" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Finish biometric login verify
curl -X POST "$BASE/authsec/webauthn/biometric/verifyLoginFinish" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'
```

### Admin WebAuthn  `/webauthn/admin`

```bash
# MFA status (admin user)
curl -X POST "$BASE/authsec/webauthn/admin/mfa/status" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Begin registration (admin)
curl -X POST "$BASE/authsec/webauthn/admin/beginRegistration" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"admin@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Finish registration (admin)
curl -X POST "$BASE/authsec/webauthn/admin/finishRegistration" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"admin@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'

# Begin authentication (admin)
curl -X POST "$BASE/authsec/webauthn/admin/beginAuthentication" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"admin@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Finish authentication (admin)
curl -X POST "$BASE/authsec/webauthn/admin/finishAuthentication" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"admin@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'
```

### End-User WebAuthn  `/webauthn/enduser`

```bash
# MFA status (end-user, uses tenant DB)
curl -X POST "$BASE/authsec/webauthn/enduser/mfa/status" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Begin registration (end-user)
curl -X POST "$BASE/authsec/webauthn/enduser/beginRegistration" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Finish registration (end-user)
curl -X POST "$BASE/authsec/webauthn/enduser/finishRegistration" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'

# Begin authentication (end-user)
curl -X POST "$BASE/authsec/webauthn/enduser/beginAuthentication" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Finish authentication (end-user)
curl -X POST "$BASE/authsec/webauthn/enduser/finishAuthentication" \
  -H "Content-Type: application/json" \
  -H "Origin: https://app.authsec.dev" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","credential":{...}}'
```

### TOTP (WebAuthn service)  `/webauthn/totp`

```bash
# Begin login TOTP setup
curl -X POST "$BASE/authsec/webauthn/totp/beginLoginSetup" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Begin TOTP setup
curl -X POST "$BASE/authsec/webauthn/totp/beginSetup" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'"}'

# Confirm login TOTP setup
curl -X POST "$BASE/authsec/webauthn/totp/confirmLoginSetup" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","secret":"<base32>","code":"123456"}'

# Confirm TOTP setup
curl -X POST "$BASE/authsec/webauthn/totp/confirmSetup" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'","secret":"<base32>","code":"123456"}'

# Verify login TOTP
curl -X POST "$BASE/authsec/webauthn/totp/verifyLogin" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","code":"123456"}'

# Verify TOTP
curl -X POST "$BASE/authsec/webauthn/totp/verify" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'","code":"123456"}'
```

### SMS MFA  `/webauthn/sms`

```bash
# Begin SMS setup (send code to phone)
curl -X POST "$BASE/authsec/webauthn/sms/beginSetup" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'","phone_number":"+15551234567"}'

# Confirm SMS setup
curl -X POST "$BASE/authsec/webauthn/sms/confirmSetup" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'","phone_number":"+15551234567","code":"123456"}'

# Request SMS code (for login)
curl -X POST "$BASE/authsec/webauthn/sms/requestCode" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'"}'

# Verify SMS code
curl -X POST "$BASE/authsec/webauthn/sms/verify" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'","code":"123456"}'
```

---

## HubSpot Integration

```bash
curl -X POST "$BASE/authsec/uflow/hubspot/contacts/sync" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'
```

---

## API Documentation

```bash
# Swagger UI
open "$BASE/authsec/uflow/swagger/index.html"

# ReDoc UI
open "$BASE/authsec/uflow/docs"

# API info
curl "$BASE/authsec/uflow/apidocs"
```

---

## External Services  `/authsec/exsvc/services`

Manages external service registrations with credentials stored in HashiCorp Vault.
Requires `external-service` RBAC permissions seeded per tenant on first access.

```bash
# Create a service (stores secret_data in Vault)
curl -X POST "$BASE/authsec/exsvc/services" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GitHub API",
    "type": "api",
    "url": "https://api.github.com",
    "description": "GitHub REST API integration",
    "tags": ["git", "ci"],
    "resource_id": "'"$RESOURCE_UUID"'",
    "auth_type": "api_key",
    "agent_accessible": true,
    "secret_data": {
      "api_key": "ghp_xxxxxxxxxxxx"
    }
  }'

# List services (for authenticated client)
curl "$BASE/authsec/exsvc/services" \
  -H "Authorization: Bearer $TOKEN"

# Get service by ID
curl "$BASE/authsec/exsvc/services/$SERVICE_ID" \
  -H "Authorization: Bearer $TOKEN"

# Update a service
curl -X PUT "$BASE/authsec/exsvc/services/$SERVICE_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GitHub API v2",
    "url": "https://api.github.com/v2",
    "secret_data": {
      "api_key": "ghp_yyyyyyyyyyyyyy"
    }
  }'

# Delete a service (also removes Vault secret)
curl -X DELETE "$BASE/authsec/exsvc/services/$SERVICE_ID" \
  -H "Authorization: Bearer $TOKEN"

# Get service credentials (reads from Vault)
curl "$BASE/authsec/exsvc/services/$SERVICE_ID/credentials" \
  -H "Authorization: Bearer $TOKEN"

# Debug: dump JWT claims
curl "$BASE/authsec/exsvc/debug/auth" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Client Management  `/authsec/clientms`

Manages multi-tenant client registrations. Formerly the standalone `clients-microservice`, now merged into authsec under `/authsec/clientms`.
Requires `clients` RBAC permissions (seeded automatically per-tenant on first client creation).

```bash
CLIENT_ID="<client-uuid>"
```

### Client Management Health

```bash
curl "$BASE/authsec/clientms/health"
```

### List Clients  `GET /authsec/clientms/tenants/:tenantId/clients/getClients`

```bash
# List all clients for a tenant (paginated)
curl "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/getClients" \
  -H "Authorization: Bearer $TOKEN"

# With filters
curl "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/getClients?status=Active&page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN"

# Filter by active only
curl "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/getClients?active_only=true" \
  -H "Authorization: Bearer $TOKEN"

# Include soft-deleted clients
curl "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/getClients?deleted=true" \
  -H "Authorization: Bearer $TOKEN"

# Legacy POST route (body-based tenant filter)
curl -X POST "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/getClients" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","active_only":false}'
```

### Get Client  `GET /authsec/clientms/tenants/:tenantId/clients/:id`

```bash
curl "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/$CLIENT_ID" \
  -H "Authorization: Bearer $TOKEN"
```

### Create Client  `POST /authsec/clientms/tenants/:tenantId/clients/create`

```bash
curl -X POST "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My App",
    "email": "myapp@example.com",
    "active": true,
    "status": "Active",
    "tags": ["web", "production"],
    "oidc_enabled": false
  }'
```

### Register Client (full registration with Hydra + Vault)  `POST /authsec/clientms/tenants/:tenantId/clients/register`

> Note: Uses the legacy RegisterClient route which also creates a Vault secret and registers with Hydra.

```bash
# The RegisterClient endpoint is wired via the route group;
# use CreateClient above for standard creation without Hydra/Vault.
```

### Update Client  `PUT /authsec/clientms/tenants/:tenantId/clients/:id`

```bash
curl -X PUT "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/$CLIENT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated App Name",
    "status": "Active",
    "tags": ["web", "v2"]
  }'
```

### Edit Client (partial update)  `PATCH /authsec/clientms/tenants/:tenantId/clients/:id`

```bash
curl -X PATCH "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/$CLIENT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Patched Name"}'
```

### Soft Delete Client  `PATCH /authsec/clientms/tenants/:tenantId/clients/:id/soft-delete`

```bash
curl -X PATCH "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/$CLIENT_ID/soft-delete" \
  -H "Authorization: Bearer $TOKEN"
```

### Delete Client (soft delete via DELETE)  `DELETE /authsec/clientms/tenants/:tenantId/clients/:id`

```bash
curl -X DELETE "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/$CLIENT_ID" \
  -H "Authorization: Bearer $TOKEN"
```

### Hard Delete Client  `POST /authsec/clientms/tenants/:tenantId/clients/delete-complete`

Permanently removes the client from both tenant DB and main DB, and cleans up Hydra via OOC Manager.

```bash
curl -X POST "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/delete-complete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'"}'
```

### Activate Client  `PATCH /authsec/clientms/tenants/:tenantId/clients/:id/activate`

```bash
curl -X PATCH "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/$CLIENT_ID/activate" \
  -H "Authorization: Bearer $TOKEN"
```

### Deactivate Client  `PATCH /authsec/clientms/tenants/:tenantId/clients/:id/deactivate`

```bash
curl -X PATCH "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/$CLIENT_ID/deactivate" \
  -H "Authorization: Bearer $TOKEN"
```

### Set Client Status  `POST /authsec/clientms/tenants/:tenantId/clients/set-status`

```bash
curl -X POST "$BASE/authsec/clientms/tenants/$TENANT_ID/clients/set-status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'","active":true}'
```

### Admin — List All Clients  `GET /authsec/clientms/admin/clients/`

Requires `clients:admin` permission.

```bash
curl "$BASE/authsec/clientms/admin/clients/" \
  -H "Authorization: Bearer $TOKEN"
```

### OOC Manager Integration  `POST /authsec/clientms/oocmgr/tenant/delete-complete`

Internal service-to-service route for OOC Manager callbacks.

```bash
curl -X POST "$BASE/authsec/clientms/oocmgr/tenant/delete-complete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'"}'
```

### Client Management API Docs

```bash
# Swagger/Redoc docs (no auth required)
curl "$BASE/authsec/clientms/swagger"
```

---

## Hydra Manager  `/authsec/hmgr`

Handles OAuth2/OIDC login flows and SAML SP-initiated authentication. Formerly the standalone `hydra-service`.

```bash
PROVIDER="github"   # oidc provider name
```

### Hydra Manager Health

```bash
curl "$BASE/authsec/hmgr/health"
```

### Login Page Data  `GET /authsec/hmgr/login/page-data`

```bash
curl "$BASE/authsec/hmgr/login/page-data?tenant_id=$TENANT_ID&client_id=$CLIENT_ID"
```

### Initiate OIDC Auth  `POST /authsec/hmgr/auth/initiate/:provider`

```bash
curl -X POST "$BASE/authsec/hmgr/auth/initiate/$PROVIDER" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "client_id": "'"$CLIENT_ID"'",
    "login_challenge": "<hydra-challenge>",
    "redirect_uri": "https://app.example.com/callback"
  }'
```

### OIDC Callback  `POST /authsec/hmgr/auth/callback`

```bash
curl -X POST "$BASE/authsec/hmgr/auth/callback" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "<auth-code>",
    "state": "<state>",
    "provider": "'"$PROVIDER"'"
  }'
```

### Exchange Token  `POST /authsec/hmgr/auth/exchange-token`

```bash
curl -X POST "$BASE/authsec/hmgr/auth/exchange-token" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "<auth-code>",
    "redirect_uri": "https://app.example.com/callback",
    "tenant_id": "'"$TENANT_ID"'"
  }'
```

### SAML — Initiate  `POST /authsec/hmgr/saml/initiate/:provider`

```bash
curl -X POST "$BASE/authsec/hmgr/saml/initiate/$PROVIDER" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "client_id": "'"$CLIENT_ID"'",
    "login_challenge": "<hydra-challenge>"
  }'
```

### SAML — ACS (Assertion Consumer Service)  `POST /authsec/hmgr/saml/acs`

```bash
curl -X POST "$BASE/authsec/hmgr/saml/acs" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "SAMLResponse=<base64-response>&RelayState=<state>"
```

### SAML — ACS per client  `POST /authsec/hmgr/saml/acs/:tenant_id/:client_id`

```bash
curl -X POST "$BASE/authsec/hmgr/saml/acs/$TENANT_ID/$CLIENT_ID" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "SAMLResponse=<base64-response>&RelayState=<state>"
```

### SAML — SP Metadata  `GET /authsec/hmgr/saml/metadata/:tenant_id/:client_id`

```bash
curl "$BASE/authsec/hmgr/saml/metadata/$TENANT_ID/$CLIENT_ID"
```

### SAML — Test Provider  `POST /authsec/hmgr/saml/test-provider`

```bash
curl -X POST "$BASE/authsec/hmgr/saml/test-provider" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "client_id": "'"$CLIENT_ID"'",
    "provider_name": "'"$PROVIDER"'"
  }'
```

### Hydra Login Redirect  `GET /authsec/hmgr/login`

```bash
curl "$BASE/authsec/hmgr/login?login_challenge=<challenge>"
```

### Hydra Consent  `GET /authsec/hmgr/consent`

```bash
curl "$BASE/authsec/hmgr/consent?consent_challenge=<challenge>"
```

### Hydra Login Challenge Info  `GET /authsec/hmgr/challenge`

```bash
curl "$BASE/authsec/hmgr/challenge?login_challenge=<challenge>"
```

### Admin — SAML Providers  (require `admin` + `manage` permissions)

```bash
PROVIDER_ID="<provider-uuid>"

# List SAML providers
curl "$BASE/authsec/hmgr/admin/saml-providers" \
  -H "Authorization: Bearer $TOKEN"

# Create SAML provider
curl -X POST "$BASE/authsec/hmgr/admin/saml-providers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "client_id": "'"$CLIENT_ID"'",
    "provider_name": "okta",
    "display_name": "Okta SAML",
    "entity_id": "https://idp.example.com",
    "sso_url": "https://idp.example.com/sso",
    "certificate": "<pem-cert>",
    "is_active": true
  }'

# Update SAML provider
curl -X PUT "$BASE/authsec/hmgr/admin/saml-providers/$PROVIDER_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"display_name": "Updated Name", "is_active": true}'

# Delete SAML provider
curl -X DELETE "$BASE/authsec/hmgr/admin/saml-providers/$PROVIDER_ID" \
  -H "Authorization: Bearer $TOKEN"
```

### Admin — Users

```bash
# List users
curl "$BASE/authsec/hmgr/admin/users" \
  -H "Authorization: Bearer $TOKEN"

# Create user
curl -X POST "$BASE/authsec/hmgr/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","tenant_id":"'"$TENANT_ID"'"}'

# Update user
curl -X PUT "$BASE/authsec/hmgr/admin/users/<user-id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"updated@example.com"}'

# Delete user
curl -X DELETE "$BASE/authsec/hmgr/admin/users/<user-id>" \
  -H "Authorization: Bearer $TOKEN"
```

### Admin — Tenants

```bash
# List tenants
curl "$BASE/authsec/hmgr/admin/tenants" \
  -H "Authorization: Bearer $TOKEN"

# Create tenant
curl -X POST "$BASE/authsec/hmgr/admin/tenants" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Corp","domain":"acme.example.com"}'

# Update tenant
curl -X PUT "$BASE/authsec/hmgr/admin/tenants/<tenant-id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Corp Updated"}'

# Delete tenant
curl -X DELETE "$BASE/authsec/hmgr/admin/tenants/<tenant-id>" \
  -H "Authorization: Bearer $TOKEN"
```

### Admin — Roles & Permissions

```bash
# List roles
curl "$BASE/authsec/hmgr/admin/roles" \
  -H "Authorization: Bearer $TOKEN"

# Create role
curl -X POST "$BASE/authsec/hmgr/admin/roles" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"viewer","description":"Read only"}'

# List permissions
curl "$BASE/authsec/hmgr/admin/permissions" \
  -H "Authorization: Bearer $TOKEN"

# Assign role to user
curl -X POST "$BASE/authsec/hmgr/admin/users/<user-id>/roles" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role_id":"<role-id>"}'
```

### User — Profile

```bash
# Get own profile
curl "$BASE/authsec/hmgr/admin/profile" \
  -H "Authorization: Bearer $TOKEN"

# Update own profile
curl -X PUT "$BASE/authsec/hmgr/admin/profile" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"display_name":"Jane Doe"}'
```

---

## OIDC Configuration Manager  `/authsec/oocmgr`

Manages Hydra OAuth2 client registrations, OIDC provider configurations, and SAML providers per tenant.
Formerly the standalone `oath_oidc_configuration_manager` microservice.

```bash
HYDRA_CLIENT_ID="<hydra-client-id>"
PROVIDER="github"
PROVIDER_ID="<saml-provider-uuid>"
REACT_APP_URL="https://app.example.com"
```

### OIDC Config Manager Health

```bash
curl "$BASE/authsec/oocmgr/health"
```

---

### Complete OIDC Setup (single call)  `POST /authsec/oocmgr/configure-complete-oidc`

Creates the tenant's main Hydra client + all OIDC provider configs in one request.

```bash
curl -X POST "$BASE/authsec/oocmgr/configure-complete-oidc" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "org_id": "org-123",
    "tenant_name": "Acme Corp",
    "tenant_client": {
      "client_name": "Acme Main Client",
      "redirect_uris": ["https://app.acme.com/callback"],
      "scopes": ["openid","profile","email","offline_access"]
    },
    "oidc_providers": [
      {
        "provider_name": "github",
        "display_name": "GitHub",
        "client_id": "gh-client-id",
        "client_secret": "gh-secret",
        "auth_url": "https://github.com/login/oauth/authorize",
        "token_url": "https://github.com/login/oauth/access_token",
        "user_info_url": "https://api.github.com/user",
        "scopes": ["user:email"],
        "is_active": true,
        "sort_order": 1
      }
    ],
    "created_by": "admin"
  }'
```

---

### Tenant Management

#### Create Base Hydra Client  `POST /authsec/oocmgr/tenant/create-base-client`

```bash
curl -X POST "$BASE/authsec/oocmgr/tenant/create-base-client" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "tenant_name": "Acme Corp",
    "client_id": "acme-main-client",
    "client_secret": "super-secret",
    "redirect_uris": ["https://app.acme.com/callback"],
    "scopes": ["openid","profile","email","offline_access"],
    "created_by": "admin"
  }'
```

#### Check Tenant Exists  `POST /authsec/oocmgr/tenant/check-exists`

```bash
curl -X POST "$BASE/authsec/oocmgr/tenant/check-exists" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","org_id":"org-123"}'
```

#### List All Tenants  `POST /authsec/oocmgr/tenant/list-all`

```bash
curl -X POST "$BASE/authsec/oocmgr/tenant/list-all" \
  -H "Content-Type: application/json" \
  -d '{}'
```

#### Delete Complete Tenant Config  `POST /authsec/oocmgr/tenant/delete-complete`

```bash
curl -X POST "$BASE/authsec/oocmgr/tenant/delete-complete" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "client_id": "acme-main-client",
    "force": false
  }'
```

#### Update Complete Tenant Config  `POST /authsec/oocmgr/tenant/update-complete`

```bash
curl -X POST "$BASE/authsec/oocmgr/tenant/update-complete" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "org_id": "org-123",
    "tenant_name": "Acme Corp v2",
    "tenant_client": {
      "client_name": "Acme Client v2",
      "redirect_uris": ["https://app.acme.com/callback","https://app.acme.com/silent-callback"]
    },
    "updated_by": "admin"
  }'
```

#### Get Tenant Login Page Data  `POST /authsec/oocmgr/tenant/login-page-data`

```bash
curl -X POST "$BASE/authsec/oocmgr/tenant/login-page-data" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","org_id":"org-123"}'
```

---

### Config Management

#### Edit Config  `POST /authsec/oocmgr/config/edit`

```bash
curl -X POST "$BASE/authsec/oocmgr/config/edit" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "<config-uuid>",
    "org_id": "org-123",
    "tenant_id": "'"$TENANT_ID"'",
    "name": "My OIDC Config",
    "config_type": "oidc",
    "config_files": {"issuer":"https://idp.example.com"},
    "is_active": true,
    "updated_by": "admin"
  }'
```

---

### OIDC Provider Management

#### Add OIDC Provider  `POST /authsec/oocmgr/oidc/add-provider`

```bash
curl -X POST "$BASE/authsec/oocmgr/oidc/add-provider" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "client_id": "acme-main-client",
    "react_app_url": "'"$REACT_APP_URL"'",
    "provider": {
      "provider_name": "google",
      "display_name": "Google",
      "client_id": "google-oauth-client-id",
      "client_secret": "google-secret",
      "auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
      "token_url": "https://oauth2.googleapis.com/token",
      "user_info_url": "https://www.googleapis.com/oauth2/v2/userinfo",
      "scopes": ["openid","profile","email"],
      "is_active": true,
      "sort_order": 1
    },
    "created_by": "admin"
  }'
```

#### Get Tenant OIDC Config  `POST /authsec/oocmgr/oidc/get-config`

```bash
curl -X POST "$BASE/authsec/oocmgr/oidc/get-config" \
  -H "Content-Type: application/json" \
  -d '{"client_id":"'"$TENANT_ID"'"}'
```

#### Get Specific Provider  `POST /authsec/oocmgr/oidc/get-provider`

```bash
curl -X POST "$BASE/authsec/oocmgr/oidc/get-provider" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","org_id":"org-123","provider_name":"github"}'
```

#### Get Provider Secret  `POST /authsec/oocmgr/oidc/get-provider-secret`

```bash
curl -X POST "$BASE/authsec/oocmgr/oidc/get-provider-secret" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","provider_name":"github"}'
```

#### Update Provider  `POST /authsec/oocmgr/oidc/update-provider`

```bash
curl -X POST "$BASE/authsec/oocmgr/oidc/update-provider" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "provider_name": "github",
    "display_name": "GitHub (updated)",
    "is_active": true,
    "updated_by": "admin"
  }'
```

#### Delete Provider  `POST /authsec/oocmgr/oidc/delete-provider`

```bash
curl -X POST "$BASE/authsec/oocmgr/oidc/delete-provider" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","client_id":"acme-github-oidc","provider_name":"github"}'
```

#### Get Provider Templates  `POST /authsec/oocmgr/oidc/templates`

```bash
curl -X POST "$BASE/authsec/oocmgr/oidc/templates" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","org_id":"org-123"}'
```

#### Validate OIDC Config  `POST /authsec/oocmgr/oidc/validate`

```bash
curl -X POST "$BASE/authsec/oocmgr/oidc/validate" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","org_id":"org-123","provider_name":"github"}'
```

#### Show Auth Providers  `POST /authsec/oocmgr/oidc/show-auth-providers`

```bash
# Aggregated view (all providers for tenant)
curl -X POST "$BASE/authsec/oocmgr/oidc/show-auth-providers" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'"}'

# Filtered by client_id
curl -X POST "$BASE/authsec/oocmgr/oidc/show-auth-providers" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","client_id":"some-service-client-id"}'
```

#### Dump Raw Hydra Data  `POST /authsec/oocmgr/oidc/raw-hydra-dump`  *(auth required)*

```bash
curl -X POST "$BASE/authsec/oocmgr/oidc/raw-hydra-dump" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","client_type":"oidc_provider"}'
```

#### Edit Client Auth Provider (upsert)  `POST /authsec/oocmgr/oidc/edit-client-auth-provider`

```bash
# Activate / create provider for a specific client
curl -X POST "$BASE/authsec/oocmgr/oidc/edit-client-auth-provider" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "client_id": "service-client-id",
    "provider_name": "github",
    "is_active": true,
    "updated_by": "admin"
  }'

# Deactivate / delete provider for a specific client
curl -X POST "$BASE/authsec/oocmgr/oidc/edit-client-auth-provider" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "client_id": "service-client-id",
    "provider_name": "github",
    "is_active": false,
    "updated_by": "admin"
  }'
```

---

### SAML Provider Management

#### Add SAML Provider  `POST /authsec/oocmgr/saml/add-provider`

```bash
curl -X POST "$BASE/authsec/oocmgr/saml/add-provider" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "client_id": "'"$CLIENT_ID"'",
    "provider_name": "okta",
    "display_name": "Okta SAML",
    "entity_id": "https://idp.okta.com/exk1234",
    "sso_url": "https://acme.okta.com/app/saml2/sso",
    "certificate": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
    "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
    "attribute_mapping": {"email":"email","first_name":"firstName","last_name":"lastName"},
    "is_active": true,
    "sort_order": 1
  }'
```

#### List SAML Providers  `POST /authsec/oocmgr/saml/list-providers`

```bash
# All providers for a tenant
curl -X POST "$BASE/authsec/oocmgr/saml/list-providers" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'"}'

# Filter by client
curl -X POST "$BASE/authsec/oocmgr/saml/list-providers" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","client_id":"'"$CLIENT_ID"'"}'
```

#### Get SAML Provider  `POST /authsec/oocmgr/saml/get-provider`

```bash
curl -X POST "$BASE/authsec/oocmgr/saml/get-provider" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","provider_id":"'"$PROVIDER_ID"'"}'
```

#### Update SAML Provider  `POST /authsec/oocmgr/saml/update-provider`

```bash
curl -X POST "$BASE/authsec/oocmgr/saml/update-provider" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "provider_id": "'"$PROVIDER_ID"'",
    "display_name": "Okta SAML v2",
    "is_active": true,
    "sort_order": 2
  }'
```

#### Delete SAML Provider  `POST /authsec/oocmgr/saml/delete-provider`

```bash
curl -X POST "$BASE/authsec/oocmgr/saml/delete-provider" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","provider_id":"'"$PROVIDER_ID"'"}'
```

#### SAML Provider Templates  `POST /authsec/oocmgr/saml/templates`

```bash
curl -X POST "$BASE/authsec/oocmgr/saml/templates" \
  -H "Content-Type: application/json" \
  -d '{}'
```

---

### Hydra Client Mappings

#### List All Hydra Client Mappings  `POST /authsec/oocmgr/hydra-clients/list`

```bash
curl -X POST "$BASE/authsec/oocmgr/hydra-clients/list" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","org_id":"org-123"}'
```

#### Get Tenant Hydra Clients  `POST /authsec/oocmgr/hydra-clients/get-by-tenant`

```bash
curl -X POST "$BASE/authsec/oocmgr/hydra-clients/get-by-tenant" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","org_id":"org-123"}'
```

#### Sync Hydra Clients  `POST /authsec/oocmgr/hydra-clients/sync`

```bash
curl -X POST "$BASE/authsec/oocmgr/hydra-clients/sync" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'"}'
```

---

### Testing & Stats

#### Test OIDC Flow  `POST /authsec/oocmgr/test/oidc-flow`

```bash
curl -X POST "$BASE/authsec/oocmgr/test/oidc-flow" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","org_id":"org-123","provider_name":"github"}'
```

#### Tenant Stats  `POST /authsec/oocmgr/stats/tenant`

```bash
curl -X POST "$BASE/authsec/oocmgr/stats/tenant" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","org_id":"org-123"}'
```

#### Get Clients by Tenant  `POST /authsec/oocmgr/clients/getClients`

```bash
curl -X POST "$BASE/authsec/oocmgr/clients/getClients" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","active_only":true}'
```

---

## Auth Manager  `/authsec/authmgr`

> Formerly the standalone `auth-manager` microservice.  
> Token endpoints are public; admin/user endpoints require a valid Bearer token.

```bash
BASE=http://localhost:8080
TOKEN="<your-jwt>"
TENANT_ID="<tenant-uuid>"
USER_ID="<user-uuid>"
```

### Health  `GET /authsec/authmgr/health`

```bash
curl "$BASE/authsec/authmgr/health"
```

### Verify Token  `POST /authsec/authmgr/token/verify`

```bash
curl -X POST "$BASE/authsec/authmgr/token/verify" \
  -H "Content-Type: application/json" \
  -d '{"token":"'"$TOKEN"'"}'
```

### Generate Token  `POST /authsec/authmgr/token/generate`

```bash
curl -X POST "$BASE/authsec/authmgr/token/generate" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","email_id":"user@example.com","client_id":"<client-uuid>","project_id":"<project-uuid>"}'
```

### OIDC Token Exchange  `POST /authsec/authmgr/token/oidc`

```bash
curl -X POST "$BASE/authsec/authmgr/token/oidc" \
  -H "Content-Type: application/json" \
  -d '{"oidc_token":"<hydra-access-token>"}'
```

### Get Profile  `GET /authsec/authmgr/admin/profile`

```bash
curl "$BASE/authsec/authmgr/admin/profile" \
  -H "Authorization: Bearer $TOKEN"
```

### Auth Status  `GET /authsec/authmgr/admin/auth-status`

```bash
curl "$BASE/authsec/authmgr/admin/auth-status?tenant_id=$TENANT_ID&email=user@example.com" \
  -H "Authorization: Bearer $TOKEN"
```

### Validate Token  `GET /authsec/authmgr/admin/validate/token`

```bash
curl "$BASE/authsec/authmgr/admin/validate/token" \
  -H "Authorization: Bearer $TOKEN"
```

### Validate Scope  `GET /authsec/authmgr/admin/validate/scope`

```bash
curl "$BASE/authsec/authmgr/admin/validate/scope" \
  -H "Authorization: Bearer $TOKEN"
```

### Validate Resource  `GET /authsec/authmgr/admin/validate/resource`

```bash
curl "$BASE/authsec/authmgr/admin/validate/resource" \
  -H "Authorization: Bearer $TOKEN"
```

### Validate Permissions  `GET /authsec/authmgr/admin/validate/permissions`

```bash
curl "$BASE/authsec/authmgr/admin/validate/permissions" \
  -H "Authorization: Bearer $TOKEN"
```

### Check Permission  `GET /authsec/authmgr/admin/check/permission`

```bash
curl "$BASE/authsec/authmgr/admin/check/permission?resource=invoice&scope=read" \
  -H "Authorization: Bearer $TOKEN"
```

### Check Role  `GET /authsec/authmgr/admin/check/role`

```bash
curl "$BASE/authsec/authmgr/admin/check/role?role=admin" \
  -H "Authorization: Bearer $TOKEN"
```

### Check Role Resource  `GET /authsec/authmgr/admin/check/role-resource`

```bash
curl "$BASE/authsec/authmgr/admin/check/role-resource?role=editor&scope_type=project&resource_id=<resource-uuid>" \
  -H "Authorization: Bearer $TOKEN"
```

### Check Permission Scoped  `GET /authsec/authmgr/admin/check/permission-scoped`

```bash
curl "$BASE/authsec/authmgr/admin/check/permission-scoped?resource=invoice&scope=write&scope_type=project&scope_id=<scope-uuid>" \
  -H "Authorization: Bearer $TOKEN"
```

### Check OAuth Scope Permission  `GET /authsec/authmgr/admin/check/oauth-scope`

```bash
curl "$BASE/authsec/authmgr/admin/check/oauth-scope?scope_name=invoice.read&resource=invoice&action=read" \
  -H "Authorization: Bearer $TOKEN"
```

### List User Permissions  `GET /authsec/authmgr/admin/permissions`

```bash
curl "$BASE/authsec/authmgr/admin/permissions" \
  -H "Authorization: Bearer $TOKEN"
```

### Create Groups  `POST /authsec/authmgr/admin/groups`

```bash
curl -X POST "$BASE/authsec/authmgr/admin/groups" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","groups":["engineers","admins"]}'
```

### List Groups  `GET /authsec/authmgr/admin/groups`

```bash
curl "$BASE/authsec/authmgr/admin/groups?tenant_id=$TENANT_ID" \
  -H "Authorization: Bearer $TOKEN"
```

### Get Group  `GET /authsec/authmgr/admin/groups/:id`

```bash
curl "$BASE/authsec/authmgr/admin/groups/1?tenant_id=$TENANT_ID" \
  -H "Authorization: Bearer $TOKEN"
```

### Update Group  `PUT /authsec/authmgr/admin/groups/:id`

```bash
curl -X PUT "$BASE/authsec/authmgr/admin/groups/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","name":"senior-engineers"}'
```

### Delete Group  `DELETE /authsec/authmgr/admin/groups/:id`

```bash
curl -X DELETE "$BASE/authsec/authmgr/admin/groups/1?tenant_id=$TENANT_ID" \
  -H "Authorization: Bearer $TOKEN"
```

### Add Users to Group  `POST /authsec/authmgr/admin/groups/:id/users`

```bash
curl -X POST "$BASE/authsec/authmgr/admin/groups/1/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","user_ids":["'"$USER_ID"'"]}'
```

### Remove Users from Group  `DELETE /authsec/authmgr/admin/groups/:id/users`

```bash
curl -X DELETE "$BASE/authsec/authmgr/admin/groups/1/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","user_ids":["'"$USER_ID"'"]}'
```

### List Group Users  `GET /authsec/authmgr/admin/groups/:id/users`

```bash
curl "$BASE/authsec/authmgr/admin/groups/1/users?tenant_id=$TENANT_ID" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Migration Management  `/authsec/migration`

All migration endpoints require JWT authentication.

### Run Master Migrations  `POST /authsec/migration/migrations/master/run`

```bash
curl -X POST "$BASE/authsec/migration/migrations/master/run" \
  -H "Authorization: Bearer $TOKEN"
```

### Master Migration Status  `GET /authsec/migration/migrations/master/status`

```bash
curl "$BASE/authsec/migration/migrations/master/status" \
  -H "Authorization: Bearer $TOKEN"
```

### List Tenants  `GET /authsec/migration/tenants`

```bash
curl "$BASE/authsec/migration/tenants" \
  -H "Authorization: Bearer $TOKEN"
```

### Create Tenant Database  `POST /authsec/migration/tenants/create-db`

Creates the tenant database (if it doesn't exist) and runs migrations asynchronously.

```bash
curl -X POST "$BASE/authsec/migration/tenants/create-db" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'"}'

# Optionally specify a database name (defaults to tenant_<uuid>)
curl -X POST "$BASE/authsec/migration/tenants/create-db" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"'"$TENANT_ID"'","database_name":"my_tenant_db"}'
```

### Run Tenant Migrations  `POST /authsec/migration/tenants/:tenant_id/migrations/run`

Synchronously creates the database (if needed) and runs all pending tenant migrations.

```bash
curl -X POST "$BASE/authsec/migration/tenants/$TENANT_ID/migrations/run" \
  -H "Authorization: Bearer $TOKEN"
```

### Tenant Migration Status  `GET /authsec/migration/tenants/:tenant_id/migrations/status`

```bash
curl "$BASE/authsec/migration/tenants/$TENANT_ID/migrations/status" \
  -H "Authorization: Bearer $TOKEN"
```

### Migrate All Tenants  `POST /authsec/migration/tenants/migrate-all`

Runs migrations for every tenant that is not yet in `completed` status. Tenants already completed are skipped.

```bash
curl -X POST "$BASE/authsec/migration/tenants/migrate-all" \
  -H "Authorization: Bearer $TOKEN"
```

Example response:

```json
{
  "total": 3,
  "succeeded": 2,
  "failed": 0,
  "skipped": 1,
  "results": [
    {"tenant_id": "...", "database_name": "tenant_abc123", "status": "completed"},
    {"tenant_id": "...", "database_name": "tenant_def456", "status": "completed"},
    {"tenant_id": "...", "database_name": "tenant_ghi789", "status": "skipped"}
  ]
}
```

---

## SPIRE Headless  `/authsec/spire`

Formerly **spire-headless**. Provides SPIFFE workload identity, OIDC token issuance with cloud federation (AWS/Azure/GCP), and a built-in RBAC/ABAC policy engine.

```bash
WORKLOAD_ID="<workload-uuid>"
POLICY_ID="<policy-uuid>"
```

### Health  `GET /authsec/spire/health`

```bash
curl "$BASE/authsec/spire/health"
```

### OIDC Discovery  `GET /authsec/.well-known/openid-configuration`

```bash
curl "$BASE/authsec/.well-known/openid-configuration"
```

### JWK Set  `GET /authsec/.well-known/jwks.json`

```bash
curl "$BASE/authsec/.well-known/jwks.json"
```

---

### Workload Registry

#### Register Workload  `POST /authsec/spire/registry/workloads`

```bash
curl -X POST "$BASE/authsec/spire/registry/workloads" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "spiffe_id": "spiffe://example.org/service/my-svc",
    "selectors": [{"type": "k8s", "value": "ns:default/sa:my-svc"}],
    "ttl": 3600
  }'
```

#### Update Workload  `PUT /authsec/spire/registry/workloads/:id`

```bash
curl -X PUT "$BASE/authsec/spire/registry/workloads/$WORKLOAD_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "ttl": 7200
  }'
```

#### Delete Workload  `DELETE /authsec/spire/registry/workloads/:id`

```bash
curl -X DELETE "$BASE/authsec/spire/registry/workloads/$WORKLOAD_ID" \
  -H "Authorization: Bearer $TOKEN"
```

#### List Workloads  `GET /authsec/spire/registry/workloads`

```bash
curl "$BASE/authsec/spire/registry/workloads" \
  -H "Authorization: Bearer $TOKEN"
```

---

### OIDC / Token Exchange

#### Exchange Credentials for OIDC Token  `POST /authsec/spire/oidc/token`

```bash
curl -X POST "$BASE/authsec/spire/oidc/token" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "spiffe://example.org/service/my-svc",
    "audience": "https://target.example.com",
    "ttl": 3600
  }'
```

#### Introspect Token  `POST /authsec/spire/oidc/introspect`

```bash
curl -X POST "$BASE/authsec/spire/oidc/introspect" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"token": "<oidc-token>"}'
```

#### Revoke Token  `POST /authsec/spire/oidc/revoke`

```bash
curl -X POST "$BASE/authsec/spire/oidc/revoke" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"token": "<oidc-token>"}'
```

#### Exchange SPIFFE SVID for OIDC Token  `POST /authsec/spire/oidc/exchange/spiffe`

```bash
curl -X POST "$BASE/authsec/spire/oidc/exchange/spiffe" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "svid": "<jwt-svid>",
    "audience": "https://target.example.com"
  }'
```

#### Issue JWT-SVID  `POST /authsec/spire/oidc/issue/jwt-svid`

```bash
curl -X POST "$BASE/authsec/spire/oidc/issue/jwt-svid" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "spiffe_id": "spiffe://example.org/service/my-svc",
    "audience": ["https://target.example.com"],
    "ttl": 3600
  }'
```

#### Generic Cloud Token Exchange  `POST /authsec/spire/oidc/exchange/cloud`

```bash
curl -X POST "$BASE/authsec/spire/oidc/exchange/cloud" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aws",
    "svid": "<jwt-svid>",
    "role_arn": "arn:aws:iam::123456789012:role/my-role"
  }'
```

#### Exchange for AWS STS Credentials  `POST /authsec/spire/oidc/exchange/aws`

```bash
curl -X POST "$BASE/authsec/spire/oidc/exchange/aws" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "svid": "<jwt-svid>",
    "role_arn": "arn:aws:iam::123456789012:role/my-role",
    "session_name": "my-session",
    "duration": 3600
  }'
```

#### Exchange for Azure AD Token  `POST /authsec/spire/oidc/exchange/azure`

```bash
curl -X POST "$BASE/authsec/spire/oidc/exchange/azure" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "svid": "<jwt-svid>",
    "tenant_id": "<azure-tenant-id>",
    "client_id": "<azure-client-id>",
    "scope": "https://management.azure.com/.default"
  }'
```

#### Exchange for GCP Access Token  `POST /authsec/spire/oidc/exchange/gcp`

```bash
curl -X POST "$BASE/authsec/spire/oidc/exchange/gcp" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "svid": "<jwt-svid>",
    "service_account": "my-sa@my-project.iam.gserviceaccount.com",
    "scopes": ["https://www.googleapis.com/auth/cloud-platform"]
  }'
```

---

### Policy Engine

#### Create Policy  `POST /authsec/spire/policy`

```bash
curl -X POST "$BASE/authsec/spire/policy" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "allow-svc-read",
    "rules": [
      {"effect": "allow", "subject": "spiffe://example.org/service/my-svc", "action": "read", "resource": "data/*"}
    ]
  }'
```

#### List Policies  `GET /authsec/spire/policy`

```bash
curl "$BASE/authsec/spire/policy" \
  -H "Authorization: Bearer $TOKEN"
```

#### Get Policy  `GET /authsec/spire/policy/:id`

```bash
curl "$BASE/authsec/spire/policy/$POLICY_ID" \
  -H "Authorization: Bearer $TOKEN"
```

#### Update Policy  `PUT /authsec/spire/policy/:id`

```bash
curl -X PUT "$BASE/authsec/spire/policy/$POLICY_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "rules": [
      {"effect": "allow", "subject": "spiffe://example.org/service/my-svc", "action": "read", "resource": "data/*"},
      {"effect": "deny",  "subject": "spiffe://example.org/service/my-svc", "action": "delete", "resource": "data/*"}
    ]
  }'
```

#### Delete Policy  `DELETE /authsec/spire/policy/:id`

```bash
curl -X DELETE "$BASE/authsec/spire/policy/$POLICY_ID" \
  -H "Authorization: Bearer $TOKEN"
```

#### Evaluate Policy  `POST /authsec/spire/policy/evaluate`

```bash
curl -X POST "$BASE/authsec/spire/policy/evaluate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "policy_id": "'"$POLICY_ID"'",
    "subject": "spiffe://example.org/service/my-svc",
    "action": "read",
    "resource": "data/config.yaml"
  }'
```

#### Batch Evaluate Policies  `POST /authsec/spire/policy/batch-evaluate`

```bash
curl -X POST "$BASE/authsec/spire/policy/batch-evaluate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {"policy_id": "'"$POLICY_ID"'", "subject": "spiffe://example.org/service/svc-a", "action": "read",  "resource": "data/config.yaml"},
      {"policy_id": "'"$POLICY_ID"'", "subject": "spiffe://example.org/service/svc-b", "action": "write", "resource": "data/secret.yaml"}
    ]
  }'
```

#### Test Policy (dry-run)  `POST /authsec/spire/policy/test`

Evaluates a policy definition without persisting it.

```bash
curl -X POST "$BASE/authsec/spire/policy/test" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "rules": [
      {"effect": "allow", "subject": "spiffe://example.org/service/my-svc", "action": "read", "resource": "data/*"}
    ],
    "subject": "spiffe://example.org/service/my-svc",
    "action": "read",
    "resource": "data/config.yaml"
  }'
```

---

### SPIRE Role Bindings

#### Bind Role  `POST /authsec/spire/roles/bind`

```bash
curl -X POST "$BASE/authsec/spire/roles/bind" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "spiffe://example.org/service/my-svc",
    "role": "reader",
    "resource": "data/*"
  }'
```

#### Unbind Role  `POST /authsec/spire/roles/unbind`

```bash
curl -X POST "$BASE/authsec/spire/roles/unbind" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "spiffe://example.org/service/my-svc",
    "role": "reader",
    "resource": "data/*"
  }'
```

#### List Role Bindings  `GET /authsec/spire/roles/bindings`

```bash
curl "$BASE/authsec/spire/roles/bindings" \
  -H "Authorization: Bearer $TOKEN"

# Filter by subject
curl "$BASE/authsec/spire/roles/bindings?subject=spiffe://example.org/service/my-svc" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Audit Logs

#### Query Audit Logs  `GET /authsec/spire/audit/logs`

```bash
curl "$BASE/authsec/spire/audit/logs" \
  -H "Authorization: Bearer $TOKEN"

# Filter by date range and actor
curl "$BASE/authsec/spire/audit/logs?from=2026-01-01T00:00:00Z&to=2026-03-13T23:59:59Z&actor=spiffe://example.org/service/my-svc" \
  -H "Authorization: Bearer $TOKEN"
```

#### Export Audit Logs  `GET /authsec/spire/audit/logs/export`

```bash
curl "$BASE/authsec/spire/audit/logs/export" \
  -H "Authorization: Bearer $TOKEN" \
  -o audit-logs.json

# Export with filters
curl "$BASE/authsec/spire/audit/logs/export?from=2026-01-01T00:00:00Z&format=csv" \
  -H "Authorization: Bearer $TOKEN" \
  -o audit-logs.csv
```

import type { SDKHelpItem } from "../types";

export const AUTHENTICATION_SDK_HELP: SDKHelpItem[] = [
  {
    id: "generate-token",
    question: "How do I generate a user token?",
    description:
      "Learn how to generate JWT access tokens for users with tenant/project credentials.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install requests PyJWT`,
        },
        {
          label: "Step 2: Generate User Token",
          code: `from minimal import AuthSecClient

# Initialize client with your AuthSec server URL
client = AuthSecClient(base_url="https://your-authsec-server.com")

# Generate token for a user
token = client.generate_token(
    tenant_id="your-tenant-id",
    project_id="your-project-id",
    client_id="your-client-id",
    email_id="user@example.com",
    secret_id="optional-secret-id"  # Optional
)

print(f"Generated token: {token}")

# Token is automatically set in the client
# Use it for subsequent authorization checks
if client.authorize("documents", "read"):
    print("User can read documents")`,
        },
        {
          label: "Step 3: Use Token in Your Application",
          code: `# The generated token is a JWT that you can send to users
# Users include it in requests: Authorization: Bearer <token>

from flask import Flask, request, jsonify

app = Flask(__name__)
client = AuthSecClient(base_url="https://your-authsec-server.com")

@app.route("/protected")
def protected_route():
    # Extract token from Authorization header
    auth_header = request.headers.get("Authorization", "")
    if not auth_header.startswith("Bearer "):
        return jsonify({"error": "No token provided"}), 401

    token = auth_header[7:]  # Remove "Bearer " prefix
    client.set_token(token)

    # Now you can authorize requests
    if not client.authorize("api", "access"):
        return jsonify({"error": "Insufficient permissions"}), 403

    return jsonify({"message": "Access granted"})`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "verify-token",
    question: "How do I verify a user token?",
    description:
      "Validate user tokens against the AuthSec server for strict security.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install requests PyJWT`,
        },
        {
          label: "Step 2: Verify Token Server-Side",
          code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")

# Verify token against AuthSec server (strict validation)
try:
    claims = client.verify_token(token="user-jwt-token-here")

    print("Token is valid!")
    print(f"User: {claims.get('email')}")
    print(f"User ID: {claims.get('sub')}")
    print(f"Tenant: {claims.get('tenant_id')}")
    print(f"Roles: {claims.get('roles', [])}")
    print(f"Permissions: {claims.get('perms', [])}")
    print(f"Expires: {claims.get('exp')}")

except Exception as e:
    print(f"Token verification failed: {e}")
    # Token is invalid or expired`,
        },
        {
          label: "Step 3: Verify Token in Middleware",
          code: `from flask import request, jsonify
from functools import wraps

def require_auth(f):
    """Decorator to verify tokens on protected endpoints"""
    @wraps(f)
    def decorated_function(*args, **kwargs):
        auth_header = request.headers.get("Authorization", "")

        if not auth_header.startswith("Bearer "):
            return jsonify({"error": "Missing token"}), 401

        token = auth_header[7:]

        try:
            # Verify token server-side
            claims = client.verify_token(token)

            # Attach claims to request for use in endpoint
            request.user_claims = claims

            return f(*args, **kwargs)
        except Exception:
            return jsonify({"error": "Invalid token"}), 401

    return decorated_function

# Use the decorator
@app.route("/user/profile")
@require_auth
def get_profile():
    user_email = request.user_claims.get("email")
    return jsonify({"email": user_email})`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "exchange-oidc",
    question: "How do I exchange OIDC tokens for application tokens?",
    description:
      "Exchange third-party OIDC tokens (Google, GitHub, etc.) for AuthSec application tokens.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install requests PyJWT`,
        },
        {
          label: "Step 2: Exchange OIDC Token",
          code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")

# Exchange OIDC token from external provider (Google, GitHub, etc.)
oidc_token = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."  # From OAuth provider

try:
    # Exchange OIDC token for application token
    app_token = client.exchange_oidc(oidc_token)

    print(f"Application token: {app_token}")

    # Token is now set in client and ready to use
    if client.authorize("documents", "read"):
        print("User authorized via OIDC has document access")

except Exception as e:
    print(f"OIDC token exchange failed: {e}")`,
        },
        {
          label: "Step 3: OAuth Callback Handler",
          code: `from flask import Flask, request, redirect, session
import requests

app = Flask(__name__)
client = AuthSecClient(base_url="https://your-authsec-server.com")

@app.route("/auth/callback")
def oauth_callback():
    # Get authorization code from OAuth provider
    code = request.args.get("code")

    # Exchange code for OIDC token (example with Google)
    token_response = requests.post("https://oauth2.googleapis.com/token", data={
        "code": code,
        "client_id": "google-client-id",
        "client_secret": "google-client-secret",
        "redirect_uri": "https://your-app.com/auth/callback",
        "grant_type": "authorization_code"
    })

    oidc_token = token_response.json()["id_token"]

    # Exchange OIDC token for AuthSec application token
    app_token = client.exchange_oidc(oidc_token)

    # Store token in session
    session["access_token"] = app_token

    return redirect("/dashboard")`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "make-authenticated-request",
    question: "How do I make authenticated API requests?",
    description:
      "Use the SDK to make authenticated requests to your application backend.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install requests PyJWT`,
        },
        {
          label: "Step 2: Make Authenticated Requests",
          code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")

# Generate or set token first
token = client.generate_token(
    tenant_id="tenant-id",
    project_id="project-id",
    client_id="client-id",
    email_id="user@example.com"
)

# Make authenticated GET request
response = client.request("GET", "https://your-api.com/api/documents")
documents = response.json()

# Make authenticated POST request
response = client.request(
    "POST",
    "https://your-api.com/api/documents",
    json={"title": "New Document", "content": "..."}
)

# Make authenticated PUT request
response = client.request(
    "PUT",
    "https://your-api.com/api/documents/123",
    json={"title": "Updated Title"}
)

# Make authenticated DELETE request
response = client.request("DELETE", "https://your-api.com/api/documents/123")

print(f"Status: {response.status_code}")`,
        },
        {
          label: "Step 3: Custom Headers in Requests",
          code: `# Add custom headers to authenticated requests
response = client.request(
    "GET",
    "https://your-api.com/api/data",
    headers={
        "X-Custom-Header": "value",
        "Content-Type": "application/json"
    }
)

# The SDK automatically adds: Authorization: Bearer <token>
# Your custom headers are merged with auth headers`,
        },
      ],
      typescript: [],
    },
  },
];

// Generate dynamic SDK code for a specific auth method
export function generateAuthMethodSDKCode(authMethod: {
  id: string;
  name?: string;
  displayName?: string;
  providerType?: string;
  provider?: string;
  methodKey?: string;
  type?: string;
}) {
  const methodName = authMethod.displayName || authMethod.name || "Auth Method";
  const providerType = (authMethod.providerType || authMethod.type || "oauth").toLowerCase();

  return {
    python: [
      {
        label: "Generate Token for User",
        code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")

# Generate token for user authenticated via ${methodName}
token = client.generate_token(
    tenant_id="your-tenant-id",
    project_id="your-project-id",
    client_id="your-client-id",
    email_id="user@example.com"
)

print(f"User token: {token}")

# Token includes authentication from ${methodName}
claims = client._claims()
print(f"User: {claims.get('email')}")
print(f"Roles: {claims.get('roles', [])}")`,
        description: `Generate token for ${methodName} authenticated user`,
      },
      {
        label: "Verify ${methodName} Token",
        code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")

# Verify token server-side
try:
    claims = client.verify_token(token="user-jwt-token")

    print("Token valid!")
    print(f"Auth method: ${methodName}")
    print(f"User: {claims.get('email')}")
    print(f"User ID: {claims.get('sub')}")

except Exception as e:
    print(f"Token verification failed: {e}")`,
        description: `Verify tokens from ${methodName} authentication`,
      },
      {
        label: "Make Authenticated Request",
        code: `# Set token and make authenticated API calls
client.set_token("user-jwt-token")

# Make authenticated request
response = client.request("GET", "https://your-api.com/api/user/profile")

user_data = response.json()
print(f"User profile: {user_data}")

# Authorization header is automatically included
# Authorization: Bearer <token>`,
        description: "Make authenticated requests with this token",
      },
    ],
    typescript: [],
  };
}

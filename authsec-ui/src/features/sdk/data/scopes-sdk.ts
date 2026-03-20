import type { SDKHelpItem } from "../types";

export const SCOPES_SDK_HELP: SDKHelpItem[] = [
  {
    id: "check-scope",
    question: "Checking whether a given resource is within scope",
    description:
      "Verify scope access using JWT token claims for OAuth-style authorization.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK",
          code: `pip install git+https://github.com/authsec-ai/authz-sdk.git`,
        },
        {
          label: "Step 2: Check Token Scopes",
          code: `# Import
from authsec import AuthSecClient
import os

# Initialize
client = AuthSecClient(
    base_url="https://dev.api.authsec.dev",
    token=os.getenv('AUTHSEC_TOKEN')
)

# Check scoped permission (resource:action within specific scope)
can_write = client.check_permission_scoped(
    resource="document",
    action="write",
    scope_type="project",          # e.g., "project", "organization", "billing_account"
    scope_id="project-uuid-123"    # UUID of the scope entity
)

if can_write:
    print("✓ User can write documents in this project")
else:
    print("✗ No permission in this scope")`,
        },
      ],
      typescript: [],
    },
  },
  // Admin function
  {
    id: "create-scope",
    question: "Managing a scope for user.",
    description:
      "Learn how to create OAuth/API scopes programmatically using the AuthSec SDK.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK",
          code: `pip install git+https://github.com/authsec-ai/authz-sdk.git`,
        },
        {
          label: "Step 2: Initialize Admin Helper & Create Scope",
          code: `# Import
from authsec import AdminHelper
import os

# Initialize
admin = AdminHelper(
    token=os.getenv('AUTHSEC_ADMIN_TOKEN'),
    base_url="https://dev.api.authsec.dev"
)

# --- CREATE: Create a new scope ---
scope = admin.create_scope(
    name="api.documents.write",
    description="Write access to documents API",
    resources=["document"]
)
print(f"✓ Created scope: {scope}")

# Create multiple scopes
scopes_to_create = [
    {"name": "api.users.read", "description": "Read users", "resources": ["user"]},
    {"name": "api.invoices.manage", "description": "Manage invoices", "resources": ["invoice"]},
    {"name": "api.admin.full", "description": "Full admin access", "resources": ["user", "document", "invoice"]}
]

for s in scopes_to_create:
    created = admin.create_scope(
        name=s["name"],
        description=s["description"],
        resources=s["resources"]
    )
    print(f"✓ Created: {s['name']}")

# --- LIST: List all scopes ---
all_scopes = admin.list_scopes()
print(f"\nTotal scopes: {len(all_scopes)}")
for scope in all_scopes:
    print(f"  - {scope.get('name')}: {scope.get('description', 'No description')}")`,
        },
      ],
      typescript: [],
    },
  },
];

// Generate dynamic SDK code for a specific scope
export function generateScopeSDKCode(scope: {
  id: string;
  name: string;
  description?: string;
  resources?: string[];
}) {
  // Parse scope name to extract resource and action (format: "resource:action")
  const scopeParts = scope.name.split(":");
  const resource = scopeParts[0] || "resource";
  const action = scopeParts[1] || "action";

  return {
    python: [
      {
        label: "Check if Token Has This Scope",
        code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")
client.set_token("user-jwt-token")

# Check if token has "${scope.name}" scope
has_scope = client.authorize("${resource}", "${action}")

if has_scope:
    print("Token has ${scope.name} scope - access granted")
else:
    print("Token missing ${scope.name} scope - access denied")`,
        description: `Verify if a token has the ${scope.name} scope`,
      },
      {
        label: "Protect Endpoint with This Scope",
        code: `from minimal import AuthSecClient
from flask import request, jsonify

client = AuthSecClient(base_url="https://your-authsec-server.com")

@app.route("/protected")
def protected_endpoint():
    # Extract token
    token = request.headers.get("Authorization", "")[7:]
    client.set_token(token)

    # Require ${scope.name} scope
    if not client.authorize("${resource}", "${action}"):
        return jsonify({"error": "Scope '${scope.name}' required"}), 403

    # ${scope.description || "User has required scope"}
    return jsonify({"status": "success"})`,
        description: `Protect an endpoint requiring ${scope.name} scope`,
      },
      {
        label: "Extract Scope from Token Claims",
        code: `# Get token claims to inspect scopes
claims = client._claims()

if claims:
    # Check scopes claim (string)
    scope_string = claims.get("scope", "")
    has_in_string = "${scope.name}" in scope_string.split()

    # Check scopes claim (array)
    scopes_array = claims.get("scopes", [])
    has_in_array = "${scope.name}" in scopes_array

    print(f"Scope string: {scope_string}")
    print(f"Scopes array: {scopes_array}")
    print(f"Has ${scope.name}: {has_in_string or has_in_array}")`,
        description: "Extract and inspect scope from JWT claims",
      },
    ],
    typescript: [],
  };
}

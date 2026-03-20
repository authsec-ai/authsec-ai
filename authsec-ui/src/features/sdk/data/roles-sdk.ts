import type { SDKHelpItem } from "../types";

export const ROLES_SDK_HELP: SDKHelpItem[] = [
  {
    id: "check-role",
    question: "Checking a user token for specific roles",
    description:
      "Verify role membership using JWT token claims for fast authorization checks.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK",
          code: `pip install git+https://github.com/authsec-ai/authz-sdk.git`,
        },
        {
          label: "Step 2: Check User Role",
          code: `# Import
from authsec import AuthSecClient
import os

# Initialize
client = AuthSecClient(
    base_url="https://dev.api.authsec.dev",
    token=os.getenv('AUTHSEC_TOKEN')
)

# List role bindings for current user (extract user_id from token)
user_id = "user-uuid-here"  # Your user UUID
bindings = client.list_role_bindings(user_id=user_id)

# Check if user has specific role
target_role_id = "role-uuid-to-check"
has_role = any(b['role_id'] == target_role_id for b in bindings)

if has_role:
    print("✓ User has the role")
else:
    print("✗ User does not have the role")

# Print all roles for user
for binding in bindings:
    print(f"Role: {binding.get('role_name')} | Scope: {binding.get('scope_type', 'tenant-wide')}")`,
        },
      ],
      typescript: [],
    },
  },
  // Admin function
  {
    id: "get-user-roles",
    question: "Assign Roles to Users.",
    description:
      "Learn how to assign roles to users programmatically using the AuthSec SDK.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install git+https://github.com/authsec-ai/authz-sdk.git`,
        },
        {
          label: "Step 2: Assign Roles to Users",
          code: `# Import
from authsec import AuthSecClient
import os

# Initialize
client = AuthSecClient(
    base_url="https://dev.api.authsec.dev",
    token=os.getenv('AUTHSEC_TOKEN')
)

# Assign role (tenant-wide)
binding = client.assign_role(
    user_id="user-uuid-here",
    role_id="role-uuid-here"
)
print(f"✓ Role assigned: {binding}")

# Assign role with scope (project-specific)
scoped_binding = client.assign_role(
    user_id="user-uuid-here",
    role_id="role-uuid-here",
    scope_type="project",
    scope_id="project-uuid-123"
)

# Assign role with conditions
conditional_binding = client.assign_role(
    user_id="user-uuid-here",
    role_id="admin-role-uuid",
    conditions={"mfa_required": True}
)`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "role-based-access",
    question: "Managing Roles for a User (Show/Add/Remove).",
    description:
      "Learn how to manage user roles programmatically using the AuthSec SDK.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install git+https://github.com/authsec-ai/authz-sdk.git`,
        },
        {
          label: "Step 2: Role-Based Authorization Decorator",
          code: `# Import
from authsec import AdminHelper
import os

# Initialize
admin = AdminHelper(
    token=os.getenv('AUTHSEC_ADMIN_TOKEN'),
    base_url="https://dev.api.authsec.dev"
)

user_id = "user-uuid-here"

# --- SHOW: List all roles for a user ---
user_bindings = admin.list_role_bindings(user_id=user_id)
print(f"User has {len(user_bindings)} role(s):")
for binding in user_bindings:
    print(f"  - {binding.get('role_name')} (scope: {binding.get('scope_type', 'tenant-wide')})")

# --- ADD: Assign a new role ---
new_binding = admin.create_role_binding(
    user_id=user_id,
    role_id="new-role-uuid"
)
print(f"✓ Added role: {new_binding.get('id')}")

# --- REMOVE: Remove a specific role binding ---
binding_to_remove = user_bindings[0]['id']  # Get binding ID from list
admin.remove_role_binding(binding_id=binding_to_remove)
print(f"✓ Removed binding: {binding_to_remove}")

# --- FULL WORKFLOW: Replace all user roles ---
# 1. Remove all existing bindings
for binding in user_bindings:
    admin.remove_role_binding(binding['id'])
    print(f"  Removed: {binding.get('role_name')}")

# 2. Add new roles
new_roles = ["viewer-role-uuid", "editor-role-uuid"]
for role_id in new_roles:
    admin.create_role_binding(user_id=user_id, role_id=role_id)
    print(f"  Added role: {role_id}")`,
        },
      ],
      typescript: [],
    },
  },
];

// Generate dynamic SDK code for a specific role
export function generateRoleSDKCode(role: {
  id: string;
  name: string;
  description?: string;
  permissions?: any[];
  grants?: any[];
}) {
  return {
    python: [
      {
        label: "Check if User Has This Role",
        code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")

# Set user token
client.set_token("user-jwt-token")

# Check if user has the "${role.name}" role
if client.has_role("${role.id}"):
    print("User has ${role.name} role")
else:
    print("User does not have ${role.name} role")`,
        description: `Verify if a user has the ${role.name} role`,
      },
      {
        label: "Protect Endpoint Requiring This Role",
        code: `from minimal import AuthSecClient
from flask import request, jsonify

client = AuthSecClient(base_url="https://your-authsec-server.com")

@app.route("/protected-endpoint")
def protected_endpoint():
    # Extract token from Authorization header
    token = request.headers.get("Authorization", "")[7:]
    client.set_token(token)

    # Check for ${role.name} role
    if not client.has_role("${role.id}"):
        return jsonify({"error": "Requires ${role.name} role"}), 403

    # User has ${role.name} role - proceed
    return jsonify({"message": "Access granted"})`,
        description: `Protect an endpoint requiring ${role.name} role`,
      },
      {
        label: "Get User Roles from Token",
        code: `# Extract all roles from user's token
claims = client._claims()
if claims:
    user_roles = claims.get("roles", [])

    # Check if "${role.id}" is in the list
    has_role = "${role.id}" in user_roles

    print(f"User roles: {user_roles}")
    print(f"Has ${role.name}: {has_role}")`,
        description: "Extract roles from JWT claims",
      },
    ],
    typescript: [],
  };
}

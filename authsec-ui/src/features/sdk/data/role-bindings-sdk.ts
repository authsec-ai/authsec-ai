import type { SDKHelpItem } from "../types";

export const ROLE_BINDINGS_SDK_HELP: SDKHelpItem[] = [
  // Admin function
  {
    id: "create-binding",
    question: "Create Role Bindings for a user.",
    description:
      "Learn how to create role bindings programmatically using the AuthSec SDK.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install git+https://github.com/authsec-ai/authz-sdk.git`,
        },
        {
          label: "Step 2: Verify Token Server-Side",
          code: `# Import
from authsec import AdminHelper
import os

# Initialize
admin = AdminHelper(
    token=os.getenv('AUTHSEC_ADMIN_TOKEN'),
    base_url="https://dev.api.authsec.dev"
)

# Create tenant-wide role binding
binding = admin.create_role_binding(
    user_id="user-uuid-here",
    role_id="role-uuid-here"
)
print(f"✓ Created binding: {binding}")

# Create scoped role binding (project-specific)
scoped_binding = admin.create_role_binding(
    user_id="user-uuid-here",
    role_id="editor-role-uuid",
    scope_type="project",
    scope_id="project-uuid-123"
)

# Create conditional role binding
conditional_binding = admin.create_role_binding(
    user_id="user-uuid-here",
    role_id="admin-role-uuid",
    conditions={"mfa_required": True, "ip_whitelist": ["192.168.1.0/24"]}
)`,
        },
      ],
      typescript: [],
    },
  },
];

// Generate dynamic SDK code for a specific role binding
export function generateRoleBindingSDKCode(binding: {
  id: string;
  role_id?: string;
  role_name?: string;
  subject_type?: string;
  subject_id?: string;
  user_id?: string;
  group_id?: string;
  expires_at?: string;
}) {
  const roleId = binding.role_id || binding.role_name || "role-id";
  const subjectType =
    binding.subject_type || (binding.user_id ? "user" : "group");
  const subjectId =
    binding.subject_id || binding.user_id || binding.group_id || "subject-id";

  return {
    python: [
      {
        label: "Check if User Has This Role Binding",
        code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")
client.set_token("user-jwt-token")

# Check if user has the "${roleId}" role binding
if client.has_role("${roleId}"):
    print("User has ${roleId} role binding")
else:
    print("User does not have ${roleId} role binding")`,
        description: `Verify if user has the ${roleId} role binding`,
      },
      {
        label: "Validate This Role Binding for Access",
        code: `from minimal import AuthSecClient
from flask import request, jsonify

client = AuthSecClient(base_url="https://your-authsec-server.com")

@app.route("/protected")
def protected_endpoint():
    token = request.headers.get("Authorization", "")[7:]
    client.set_token(token)

    # Require ${roleId} role binding
    if not client.has_role("${roleId}"):
        return jsonify({
            "error": "Role binding '${roleId}' required"
        }), 403

    return jsonify({"status": "success"})`,
        description: "Protect endpoint with this role binding",
      },
      {
        label: "Extract Role Binding Details from Token",
        code: `# Get token claims
claims = client._claims()

if claims:
    roles = claims.get("roles", [])

    # Check if this binding exists
    has_binding = "${roleId}" in roles

    print(f"All role bindings: {roles}")
    print(f"Has ${roleId} binding: {has_binding}")${
      binding.expires_at
        ? `

    # Check expiration
    exp = claims.get("exp")
    if exp:
        from datetime import datetime
        expires = datetime.fromtimestamp(exp)
        print(f"Binding expires: {expires}")`
        : ""
    }`,
        description: "Extract binding details from JWT claims",
      },
    ],
    typescript: [],
  };
}

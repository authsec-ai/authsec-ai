import type { SDKHelpItem } from "../types";

export const PERMISSIONS_SDK_HELP: SDKHelpItem[] = [
  {
    id: "check-permission",
    question: "Checking a user token for permissions.",
    description:
      "Learn how to verify if a user has the required permission to perform an action on a resource using local JWT checks.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK",
          code: `pip install git+https://github.com/authsec-ai/authz-sdk.git`,
        },
        {
          label: "Step 2: Initialize Client & Check Permission",
          code: `# Import
from authsec import AuthSecClient
import os

# Initialize
client = AuthSecClient(
    base_url="https://dev.api.authsec.dev",
    token=os.getenv('AUTHSEC_TOKEN')
)

# Check single permission (resource:action)
if client.check_permission("document", "read"):
    print("✓ User can read documents")
else:
    print("✗ Access denied")

# List all user permissions
permissions = client.list_permissions()
for perm in permissions:
    print(f"{perm['resource']}: {perm['actions']}")`,
        },
        {
          label: "Step 3: Check scoped permission",
          code: `# Check permission with scope (tenant/project level)
can_access = client.check_permission_scoped("resource", "action", scope_type="project", scope_id="project-id")`,
        },
      ],
      typescript: [],
    },
  },
  // Admin function
  {
    id: "create-permission",
    question:
      "Create a permission i.e. a resource + method definition via SDK.",
    description:
      "Learn how to create new permissions programmatically using the AuthSec SDK.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK",
          code: `pip install git+https://github.com/authsec-ai/authz-sdk.git`,
        },
        {
          label: "Step 2: Create Permission",
          code: `# Import
from authsec import AdminHelper
import os

# Initialize
admin = AdminHelper(
    token=os.getenv('AUTHSEC_ADMIN_TOKEN'),
    base_url="https://dev.api.authsec.dev"
)

# Create single permission
perm = admin.create_permission(
    resource="document",
    action="read",
    description="Read documents"
)
print(f"✓ Created permission: {perm}")

# Batch create permissions
resources = {
    "document": ["read", "write", "delete"],
    "invoice": ["read", "create", "approve"],
    "user": ["read", "invite", "delete"]
}

for resource, actions in resources.items():
    for action in actions:
        admin.create_permission(resource, action, f"{action.capitalize()} {resource}")
        print(f"✓ Created: {resource}:{action}")

# List all permissions
all_perms = admin.list_permissions()
print(f"Total permissions: {len(all_perms)}")`,
        },
      ],
      typescript: [],
    },
  },
];

// Generate dynamic SDK code for a specific permission
export function generatePermissionSDKCode(permission: {
  action: string;
  resource: string;
  full_permission_string: string;
  description?: string;
}) {
  return {
    python: [
      {
        label: "Check This Permission (Local JWT Check)",
        code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")

# Set user token
client.set_token("user-jwt-token")

# Check if user has "${permission.full_permission_string}" permission
has_permission = client.authorize(
    resource="${permission.resource}",
    action="${permission.action}"
)

if has_permission:
    print("Access granted for ${permission.full_permission_string}")
else:
    print("Access denied for ${permission.full_permission_string}")`,
        description: `Verify if a user has the ${permission.full_permission_string} permission`,
      },
      {
        label: "Protect Endpoint with This Permission",
        code: `from minimal import AuthSecClient
from flask import request, jsonify

client = AuthSecClient(base_url="https://your-authsec-server.com")

def protected_endpoint():
    # Extract token from Authorization header
    token = request.headers.get("Authorization", "")[7:]
    client.set_token(token)

    # Check for ${permission.full_permission_string} permission
    if not client.authorize("${permission.resource}", "${permission.action}"):
        return jsonify({"error": "Requires ${
          permission.full_permission_string
        }"}), 403

    # ${permission.description || "Your protected logic here"}
    return jsonify({"status": "success"})`,
        description: `Protect an endpoint requiring ${permission.full_permission_string}`,
      },
      {
        label: "Check with Resource List Validation",
        code: `# Strict check: require resource in 'resources' claim AND permission
has_permission = client.authorize(
    resource="${permission.resource}",
    action="${permission.action}",
    require_resource_list=True  # User must have "${permission.resource}" in token
)`,
        description: "Strict permission check with resource list validation",
      },
    ],
    typescript: [],
  };
}

import type { SDKHelpItem } from "../types";

export const RESOURCES_SDK_HELP: SDKHelpItem[] = [
  {
    id: "protect-resource",
    question: "How do I protect a resource with authorization?",
    description:
      "Learn how to protect resources by checking user permissions using JWT token claims.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install requests PyJWT`,
        },
        {
          label: "Step 2: Protect Resource with Permission Check",
          code: `from minimal import AuthSecClient
from flask import request, jsonify

client = AuthSecClient(base_url="https://your-authsec-server.com")

@app.route("/documents/<doc_id>", methods=["GET"])
def get_document(doc_id):
    # Extract and set user token
    token = request.headers.get("Authorization", "")[7:]
    client.set_token(token)

    # Check if user can read documents resource
    if not client.authorize("documents", "read"):
        return jsonify({"error": "Access denied"}), 403

    # User has permission - return document
    return jsonify({"document": {...}})

@app.route("/documents", methods=["POST"])
def create_document():
    token = request.headers.get("Authorization", "")[7:]
    client.set_token(token)

    # Check write permission for documents resource
    if not client.authorize("documents", "write"):
        return jsonify({"error": "Cannot create documents"}), 403

    return jsonify({"status": "created"}), 201`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "check-resource-access",
    question: "How do I check user access to a resource?",
    description:
      "Verify user permissions for specific resources and actions.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install requests PyJWT`,
        },
        {
          label: "Step 2: Check Resource Access",
          code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")
client.set_token("user-jwt-token")

# Check read access to documents resource
can_read = client.authorize("documents", "read")

# Check write access to documents resource
can_write = client.authorize("documents", "write")

# Check delete access to documents resource
can_delete = client.authorize("documents", "delete")

print(f"Read access: {can_read}")
print(f"Write access: {can_write}")
print(f"Delete access: {can_delete}")`,
        },
        {
          label: "Step 3: Check with Resource List Validation",
          code: `# Strict check: require resource in token's 'resources' claim
can_access = client.authorize(
    "documents",
    "read",
    require_resource_list=True  # User must have "documents" in resources claim
)

if can_access:
    print("User has explicit access to documents resource")
else:
    print("User does not have documents in resource list")`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "multiple-resources",
    question: "How do I check access to multiple resources?",
    description:
      "Validate user permissions across multiple resources efficiently.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install requests PyJWT`,
        },
        {
          label: "Step 2: Check Multiple Resource Permissions",
          code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")
client.set_token("user-jwt-token")

# Check if user has ALL required permissions (AND logic)
has_all = client.authorize_all([
    ("documents", "read"),
    ("documents", "write"),
    ("reports", "read"),
])

if has_all:
    print("User can read/write documents AND read reports")

# Check if user has ANY of the permissions (OR logic)
has_any = client.authorize_any([
    ("documents", "admin"),
    ("documents", "owner"),
])

if has_any:
    print("User has elevated access to documents")`,
        },
        {
          label: "Step 3: Resource Access Matrix",
          code: `# Build access matrix for multiple resources
resources = ["documents", "reports", "settings", "users"]
actions = ["read", "write", "delete"]

access_matrix = {}
for resource in resources:
    access_matrix[resource] = {}
    for action in actions:
        access_matrix[resource][action] = client.authorize(resource, action)

# Display access matrix
for resource, perms in access_matrix.items():
    allowed = [action for action, has_perm in perms.items() if has_perm]
    print(f"{resource}: {', '.join(allowed) if allowed else 'No access'}")`,
        },
      ],
      typescript: [],
    },
  },
];

// Generate dynamic SDK code for a specific resource
export function generateResourceSDKCode(resource: {
  id: string;
  name: string;
  description?: string;
}) {
  return {
    python: [
      {
        label: "Check Access to This Resource",
        code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")
client.set_token("user-jwt-token")

# Check read access to "${resource.name}" resource
can_read = client.authorize("${resource.name}", "read")

# Check write access to "${resource.name}" resource
can_write = client.authorize("${resource.name}", "write")

# Check delete access to "${resource.name}" resource
can_delete = client.authorize("${resource.name}", "delete")

print(f"Read: {can_read}, Write: {can_write}, Delete: {can_delete}")`,
        description: `Check user access to the ${resource.name} resource`,
      },
      {
        label: "Protect Endpoint with This Resource",
        code: `from minimal import AuthSecClient
from flask import request, jsonify

client = AuthSecClient(base_url="https://your-authsec-server.com")

@app.route("/${resource.name.replace(/-/g, "_")}", methods=["GET"])
def access_${resource.name.replace(/-/g, "_")}():
    # Extract token
    token = request.headers.get("Authorization", "")[7:]
    client.set_token(token)

    # Check read permission for ${resource.name}
    if not client.authorize("${resource.name}", "read"):
        return jsonify({"error": "Cannot access ${resource.name}"}), 403

    # ${resource.description || "User has access"}
    return jsonify({"${resource.name}": [...]})`,
        description: `Protect an endpoint requiring ${resource.name} access`,
      },
      {
        label: "Strict Resource Check with Resource List",
        code: `# Strict check: user must have "${resource.name}" in resources claim
has_access = client.authorize(
    "${resource.name}",
    "read",
    require_resource_list=True
)

if has_access:
    print("User has ${resource.name} in resource list AND read permission")
else:
    print("User missing ${resource.name} resource or permission")`,
        description: "Validate resource with strict resource list check",
      },
    ],
    typescript: [],
  };
}

import type { SDKHelpItem } from "../types";

export const OAUTH_API_SDK_HELP: SDKHelpItem[] = [
  {
    id: "create-scope",
    question: "Create an OAuth/API scope via SDK.",
    description:
      "Learn how to create OAuth/API scopes programmatically using the AuthSec SDK.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK Dependencies",
          code: `pip install requests PyJWT`,
        },
        {
          label: "Step 2: Initialize Admin Helper & Create Scope",
          code: `from admin_helper import AdminHelper

# Initialize with token
admin = AdminHelper(
    token="your-admin-token",
    base_url="https://test.api.authsec.dev"
)

# Create an OAuth/API scope
scope = admin.create_scope(
    name="api.documents.write",
    description="Write access to documents API",
    permission=["document"]
)`,
        },
        {
          label: "Returns",
          code: `Scope object with details of the created scope.`,
        },
        {
          label: "Endpoint",
          code: `POST /uflow/admin/api_scopes`,
        },
      ],
      typescript: [],
    },
  },
  //   {
  //     id: "list-scopes",
  //     question: "List all OAuth/API scopes via SDK.",
  //     description:
  //       "Learn how to list all OAuth/API scopes programmatically using the AuthSec SDK.",
  //     code: {
  //       python: [
  //         {
  //           label: "Step 1: Install SDK Dependencies",
  //           code: `pip install requests PyJWT`,
  //         },
  //         {
  //           label: "Step 2: Initialize Admin Helper & List Scopes",
  //           code: `from admin_helper import AdminHelper

  // # Initialize with token
  // admin = AdminHelper(
  //     token="your-admin-token",
  //     base_url="https://test.api.authsec.dev"
  // )

  // # List all OAuth/API scopes
  // scopes = admin.list_scopes()

  // for scope in scopes:
  //     print(f"{scope['name']}: {scope['description']}")`,
  //         },
  //         {
  //           label: "Returns",
  //           code: `List of scope objects.`,
  //         },
  //         {
  //           label: "Endpoint",
  //           code: `GET /uflow/enduser/scopes`,
  //         },
  //       ],
  //       typescript: [],
  //     },
  //   },
];

// Generate dynamic SDK code for a specific OAuth/API scope
export function generateOAuthApiScopeSDKCode(scope: {
  id: string;
  name: string;
  description?: string;
  permissions_linked?: number;
  permission_strings?: string[];
}) {
  return {
    python: [
      {
        label: "Scope Details",
        code: `# Scope: ${scope.name}
# ID: ${scope.id}
# Description: ${scope.description || "No description"}
# Permissions linked: ${scope.permissions_linked || 0}`,
        description: `Details for the ${scope.name} scope`,
      },
      {
        label: "Check if Token Has This Scope",
        code: `from minimal import AuthSecClient

client = AuthSecClient(base_url="https://your-authsec-server.com")
client.set_token("user-jwt-token")

# Get token claims and check for "${scope.name}" scope
claims = client._claims()

if claims:
    scope_string = claims.get("scope", "")
    scopes_list = scope_string.split()

    if "${scope.name}" in scopes_list:
        print("Token has ${scope.name} scope")
    else:
        print("Token missing ${scope.name} scope")`,
        description: `Check if a token has the ${scope.name} scope`,
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

    # Check for "${scope.name}" scope
    claims = client._claims()
    scope_string = claims.get("scope", "") if claims else ""

    if "${scope.name}" not in scope_string.split():
        return jsonify({"error": "Scope '${scope.name}' required"}), 403

    # ${scope.description || "User has required scope"}
    return jsonify({"status": "success"})`,
        description: `Protect an endpoint requiring ${scope.name} scope`,
      },
    ],
    typescript: [],
  };
}

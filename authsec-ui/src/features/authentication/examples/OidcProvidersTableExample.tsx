import React, { useState } from "react";
import {
  EnhancedOidcProvidersTable,
  type ApiOidcProvider,
  type OidcProviderTableActions,
} from "../components";

// Example data based on your API response
const exampleOidcProviders: ApiOidcProvider[] = [
  {
    callback_url: "https://dev.app.authsec.dev/oidc/auth/callback/github",
    client_id: "e2eafae5-06a0-43e8-832a-713ea0d28cc2-github-oidc",
    created_at: "2025-09-10T08:36:45Z",
    display_name: "GitHub",
    is_active: true,
    provider_config: {
      additional_params: null,
      auth_url: "https://github.com/login/oauth/authorize",
      client_id: "ritom.k.7@gmail.com",
      client_secret: "Snapchat7@#",
      issuer_url: "",
      jwks_url: "",
      scopes: ["user:email"],
      token_url: "https://github.com/login/oauth/access_token",
      user_info_url: "https://api.github.com/user",
    },
    provider_name: "github",
    sort_order: 0,
  },
  {
    callback_url: "https://dev.app.authsec.dev/oidc/auth/callback/google",
    client_id: "e2eafae5-06a0-43e8-832a-713ea0d28cc2-google-oidc",
    created_at: "2025-09-08T17:33:43Z",
    display_name: "Google",
    is_active: true,
    provider_config: {
      additional_params: null,
      auth_url: "https://accounts.google.com/o/oauth2/v2/auth",
      client_id: "YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com",
      client_secret: "YOUR_GOOGLE_CLIENT_SECRET",
      issuer_url: "",
      jwks_url: "",
      scopes: ["openid", "profile", "email"],
      token_url: "https://oauth2.googleapis.com/token",
      user_info_url: "https://www.googleapis.com/oauth2/v2/userinfo",
    },
    provider_name: "google",
    sort_order: 1,
  },
];

export function OidcProvidersTableExample() {
  const [selectedProviders, setSelectedProviders] = useState<string[]>([]);
  const [providers, setProviders] = useState<ApiOidcProvider[]>(exampleOidcProviders);

  // Table action handlers
  const handleDuplicateProvider = (providerId: string) => {
    console.log("Duplicate provider:", providerId);
    // Implement duplication logic
    const provider = providers.find((p) => p.client_id === providerId);
    if (provider) {
      const duplicated = {
        ...provider,
        client_id: `${provider.client_id}-copy`,
        display_name: `${provider.display_name} (Copy)`,
        created_at: new Date().toISOString(),
        is_active: false, // Start as inactive
      };
      setProviders([...providers, duplicated]);
    }
  };

  const handleDeleteProvider = (providerId: string) => {
    console.log("Delete provider:", providerId);
    // Implement delete logic with confirmation
    if (window.confirm("Are you sure you want to delete this provider?")) {
      setProviders(providers.filter((p) => p.client_id !== providerId));
      setSelectedProviders(selectedProviders.filter((id) => id !== providerId));
    }
  };

  const handleToggleActive = (providerId: string, isActive: boolean) => {
    console.log("Toggle provider status:", providerId, isActive);
    // Implement status toggle logic
    setProviders(
      providers.map((p) => (p.client_id === providerId ? { ...p, is_active: isActive } : p))
    );
  };

  const handleViewConfiguration = (providerId: string) => {
    console.log("View configuration:", providerId);
    // Implement configuration viewing logic
    const provider = providers.find((p) => p.client_id === providerId);
    if (provider) {
      alert(
        `Configuration for ${provider.display_name}:\n${JSON.stringify(
          provider.provider_config,
          null,
          2
        )}`
      );
    }
  };

  const handleTestConnection = (providerId: string) => {
    console.log("Test connection:", providerId);
    // Implement connection testing logic
    const provider = providers.find((p) => p.client_id === providerId);
    if (provider) {
      alert(`Testing connection for ${provider.display_name}...`);
    }
  };

  const handleCreateProvider = () => {
    console.log("Create new provider");
    // Implement create provider logic
    alert("Create new OIDC provider dialog would open here");
  };

  const handleSelectProvider = (providerId: string) => {
    setSelectedProviders((prev) =>
      prev.includes(providerId) ? prev.filter((id) => id !== providerId) : [...prev, providerId]
    );
  };

  const handleSelectAll = () => {
    setSelectedProviders(
      selectedProviders.length === providers.length ? [] : providers.map((p) => p.client_id)
    );
  };

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">OIDC Providers</h1>
          <p className="text-foreground">
            Manage OAuth and OpenID Connect authentication providers
          </p>
        </div>
        <button
          onClick={handleCreateProvider}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          Add Provider
        </button>
      </div>

      <EnhancedOidcProvidersTable
        data={providers}
        selectedProviders={selectedProviders}
        onSelectAll={handleSelectAll}
        onSelectProvider={handleSelectProvider}
        onDuplicateProvider={handleDuplicateProvider}
        onDeleteProvider={handleDeleteProvider}
        onToggleActive={handleToggleActive}
        onViewConfiguration={handleViewConfiguration}
        onTestConnection={handleTestConnection}
        onCreateProvider={handleCreateProvider}
        enableDynamicColumns={true}
      />

      {/* Selected providers info */}
      {selectedProviders.length > 0 && (
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <h3 className="font-semibold text-blue-900 mb-2">
            {selectedProviders.length} provider{selectedProviders.length !== 1 ? "s" : ""} selected
          </h3>
          <div className="flex gap-2">
            <button
              onClick={() => {
                selectedProviders.forEach((id) => handleToggleActive(id, false));
                setSelectedProviders([]);
              }}
              className="px-3 py-1 bg-orange-600 text-white text-sm rounded hover:bg-orange-700"
            >
              Deactivate Selected
            </button>
            <button
              onClick={() => {
                selectedProviders.forEach((id) => handleToggleActive(id, true));
                setSelectedProviders([]);
              }}
              className="px-3 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700"
            >
              Activate Selected
            </button>
            <button
              onClick={() => {
                if (window.confirm(`Delete ${selectedProviders.length} selected providers?`)) {
                  setProviders(providers.filter((p) => !selectedProviders.includes(p.client_id)));
                  setSelectedProviders([]);
                }
              }}
              className="px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700"
            >
              Delete Selected
            </button>
          </div>
        </div>
      )}

      {/* Provider statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white border rounded-lg p-4">
          <div className="text-2xl font-bold text-blue-600">{providers.length}</div>
          <div className="text-sm text-foreground">Total Providers</div>
        </div>
        <div className="bg-white border rounded-lg p-4">
          <div className="text-2xl font-bold text-green-600">
            {providers.filter((p) => p.is_active).length}
          </div>
          <div className="text-sm text-foreground">Active Providers</div>
        </div>
        <div className="bg-white border rounded-lg p-4">
          <div className="text-2xl font-bold text-orange-600">
            {providers.filter((p) => !p.is_active).length}
          </div>
          <div className="text-sm text-foreground">Inactive Providers</div>
        </div>
        <div className="bg-white border rounded-lg p-4">
          <div className="text-2xl font-bold text-blue-600">
            {new Set(providers.map((p) => p.provider_name)).size}
          </div>
          <div className="text-sm text-foreground">Provider Types</div>
        </div>
      </div>
    </div>
  );
}

export default OidcProvidersTableExample;

import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
  useParams,
  useNavigate,
  useLocation,
} from "react-router-dom";
import { Provider } from "react-redux";
import { Toaster } from "react-hot-toast";
import { store } from "./app/store";
import { AppLayout } from "./components/layout/AppLayout";
import { AuthProvider } from "./auth/context/AuthContext";
import { ProtectedRoute } from "./components/auth/ProtectedRoute";
import { useSessionInit } from "./hooks/useSessionInit";
import {
  RbacAudienceProvider,
  useRbacAudience,
  type RbacAudience,
} from "./contexts/RbacAudienceContext";
import { GuidedTourProvider, GuidedTourOverlay } from "./features/guided-tour";
import { WizardProvider } from "./contexts/WizardContext";
import React, { useEffect, useRef } from "react";

import { DashboardPage } from "./features/dashboard/DashboardPage";

import { UsersPage } from "./features/users/UsersPage";
// import { GroupsPage } from "./features/groups/GroupsPage";
// import ResourcesPage from "./features/resources/ResourcesPage";

import { ClientsPage } from "./features/clients/ClientsPage";
import { WorkloadIdentitiesPage } from "./features/workloads/WorkloadIdentitiesPage";
import { WorkloadCertificatePage } from "./features/workloads/WorkloadCertificatePage";
import { AgentsPage } from "./features/workloads/components/AgentsPage";

import VoiceAgentWizardPage from "./features/clients/VoiceAgentWizardPage";
import { AdminVoiceAgentPage } from "./features/voice-auth/AdminVoiceAgentPage";
import { LogsConfigurationPage } from "./features/logging/LogsConfigurationPage";
import { AuthLogsPage } from "./features/logging/AuthLogsPage";
import { AuditLogsPage } from "./features/logging/AuditLogsPage";
import { M2MLogsPage } from "./features/logging/M2MLogsPage";
import { VaultPage } from "./features/vault/VaultPage";

import { ImportSecretsPage } from "./features/vault/ImportSecretsPage";
import { RolesPage } from "./features/roles/RolesPage";
import { RoleTemplatesPage } from "./features/roles/RoleTemplatesPage";
import { AuthenticationPage } from "./features/authentication/AuthenticationPage";
import { CreateAuthMethodPage } from "./features/authentication/CreateAuthMethodPage";
import { CreateSamlMethodPage } from "./features/authentication/CreateSamlMethodPage";
import { EditSamlMethodPage } from "./features/authentication/EditSamlMethodPage";

// External services and secrets management
import { ExternalServicesPage } from "./features/external-services/ExternalServicesPage";
import { AddExternalServicePage } from "./features/external-services/AddExternalServicePage";

// Custom Domains
import { CustomDomainsPage } from "./features/custom-domains";

// Trust Delegation
import {
  TrustDelegationPoliciesPage,
  TrustDelegationPolicyDetailPage,
  TrustDelegationPolicyFormPage,
} from "./features/trust-delegation";

// LEGACY/OBSOLETE: SDK Manager has been deprecated
// import { SDKManagerPage } from "./features/_LEGACY_sdk-manager/SDKManagerPage";

// New pages
// import CreateGroupPage from "./features/groups/CreateGroupPage";
// import { AddResourcePage } from "./features/resources/AddResourcePage";

// RBAC pages
import { ScopesPage } from "./features/scopes/ScopesPage";
import { ApiOAuthScopesPage } from "./features/api-oauth-scopes/ApiOAuthScopesPage";
import { PermissionsPage } from "./features/permissions/PermissionsPage";
import { RoleBindingsPage } from "./features/role-bindings/RoleBindingsPage";
import { PermissionResourcesPage } from "./features/resources/PermissionResourcesPage";
import SDKHubPage from "./features/sdk/SDKHubPage";

import { UnifiedAuthFlowPage } from "./auth/app/UnifiedAuthFlowPage";

// Other pages
import { LandingPage } from "./pages/LandingPage";

/**
 * Context Route Sync Component
 * Syncs URL context (admin/enduser) with RbacAudienceContext
 */
function ContextRouteSync() {
  const { audience, setAudience } = useRbacAudience();
  const navigate = useNavigate();
  const location = useLocation();
  const previousAudienceRef = useRef<RbacAudience>(audience);
  const previousPathContextRef = useRef<RbacAudience | null>(null);

  useEffect(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    const pathContextRaw = pathParts[0] ?? null;
    const pathContext: RbacAudience | null =
      pathContextRaw === "admin"
        ? "admin"
        : pathContextRaw === "enduser"
          ? "endUser"
          : null;

    const prevAudience = previousAudienceRef.current;
    const prevPathContext = previousPathContextRef.current;

    // If the URL context changed (navigation), update the audience state
    if (pathContext && pathContext !== audience) {
      const pathContextChanged = pathContext !== prevPathContext;
      if (pathContextChanged) {
        console.log("🔄 Syncing URL context to state:", pathContext);
        setAudience(pathContext);
        previousAudienceRef.current = audience;
        previousPathContextRef.current = pathContext;
        return;
      }
    }

    const currentContextSegment =
      pathContext === "admin"
        ? "admin"
        : pathContext === "endUser"
          ? "enduser"
          : null;
    const expectedContextSegment = audience === "admin" ? "admin" : "enduser";

    // If the audience state changed (toggle) while on a context route, update the URL
    if (
      (pathContextRaw === "admin" || pathContextRaw === "enduser") &&
      currentContextSegment !== expectedContextSegment &&
      audience !== prevAudience
    ) {
      const remainder = pathParts.slice(1).join("/");
      const newPath = remainder
        ? `/${expectedContextSegment}/${remainder}`
        : `/${expectedContextSegment}`;

      console.log(
        `🔄 Syncing state change to URL: ${currentContextSegment} → ${expectedContextSegment}`,
      );
      navigate(newPath, { replace: true });
      previousPathContextRef.current = audience;
      previousAudienceRef.current = audience;
      return;
    }

    previousAudienceRef.current = audience;
    previousPathContextRef.current = pathContext;
  }, [audience, location.pathname, navigate, setAudience]);

  return null;
}

function LegacyTrustDelegationPolicyDetailRedirect() {
  const { policyId = "" } = useParams();
  return <Navigate to={`/trust-delegation/${policyId}`} replace />;
}

function LegacyTrustDelegationPolicyEditRedirect() {
  const { policyId = "" } = useParams();
  return <Navigate to={`/trust-delegation/${policyId}/edit`} replace />;
}

function LegacyClientOnboardRedirect() {
  const { clientId } = useParams<{ clientId?: string }>();
  const target = clientId
    ? `/sdk/clients/${encodeURIComponent(clientId)}`
    : "/sdk/clients";

  return <Navigate to={target} replace />;
}

function LegacyExternalServiceSdkRedirect() {
  const { serviceId } = useParams<{ serviceId?: string }>();
  const target = serviceId
    ? `/sdk/external-services/${encodeURIComponent(serviceId)}`
    : "/sdk/external-services";

  return <Navigate to={target} replace />;
}

/**
 * App content component that uses session initialization
 */
function AppContent() {
  useSessionInit();

  return (
    <AuthProvider>
      <RbacAudienceProvider>
        <Router>
          <WizardProvider>
            <GuidedTourProvider>
              <ContextRouteSync />
              <div className="min-h-screen bg-background text-foreground">
                <Routes>
                  {/* Auth routes - accessible without authentication */}
                  <Route path="/admin/login" element={<UnifiedAuthFlowPage />} />
                  <Route
                    path="/admin/signin"
                    element={<Navigate to="/admin/login" replace />}
                  />
                  <Route
                    path="/admin/signup"
                    element={<Navigate to="/admin/login" replace />}
                  />
                  <Route
                    path="/admin/verify-otp"
                    element={<UnifiedAuthFlowPage />}
                  />
                  <Route path="/admin/webauthn" element={<UnifiedAuthFlowPage />} />
                  <Route
                    path="/authsec/uflow/oidc/callback"
                    element={<UnifiedAuthFlowPage />}
                  />
                  {/* Backward compat: backend renderOAuthCallbackHTML redirects here */}
                  <Route
                    path="/auth/callback"
                    element={<UnifiedAuthFlowPage />}
                  />
                  <Route
                    path="/admin/create-workspace"
                    element={
                      <ProtectedRoute requireProject={false}>
                        <UnifiedAuthFlowPage />
                      </ProtectedRoute>
                    }
                  />

                  {/* Redirect incorrect hyphenated URLs to correct routes */}
                  <Route
                    path="/admin/sign-up"
                    element={<Navigate to="/admin/login" replace />}
                  />
                  <Route
                    path="/admin/sign-in"
                    element={<Navigate to="/admin/login" replace />}
                  />

                  {/* Backwards compatibility redirects */}
                  <Route
                    path="/auth/signin"
                    element={<Navigate to="/admin/login" replace />}
                  />
                  <Route
                    path="/auth/signup"
                    element={<Navigate to="/admin/login" replace />}
                  />
                  <Route
                    path="/auth/sign-in"
                    element={<Navigate to="/admin/login" replace />}
                  />
                  <Route
                    path="/auth/sign-up"
                    element={<Navigate to="/admin/login" replace />}
                  />
                  <Route
                    path="/auth/verify-otp"
                    element={<Navigate to="/admin/verify-otp" replace />}
                  />
                  <Route
                    path="/auth/webauthn"
                    element={<Navigate to="/admin/webauthn" replace />}
                  />
                  <Route
                    path="/auth/create-workspace"
                    element={<Navigate to="/admin/create-workspace" replace />}
                  />
                  <Route
                    path="/obsolete/login"
                    element={<Navigate to="/admin/login" replace />}
                  />

                  {/* OIDC login page matching backend template design */}
                  <Route path="/oidc/login" element={<UnifiedAuthFlowPage />} />
                  <Route
                    path="/oidc/auth/callback"
                    element={<UnifiedAuthFlowPage />}
                  />

                  {/* Root route - handles authentication redirect */}
                  <Route path="/" element={<LandingPage />} />

                  {/* Protected routes - require authentication and workspace */}

                  <Route
                    path="/dashboard"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <DashboardPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  {/* Redirect /clients to /clients/mcp */}
                  <Route
                    path="/clients"
                    element={<Navigate to="/clients/mcp" replace />}
                  />

                  <Route
                    path="/clients/mcp"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <ClientsPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/clients/onboard"
                    element={<Navigate to="/sdk/clients" replace />}
                  />

                  <Route
                    path="/clients/onboard/:clientId"
                    element={<LegacyClientOnboardRedirect />}
                  />

                  <Route
                    path="/clients/voice-agent"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <VoiceAgentWizardPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/admin/voice-agent"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <AdminVoiceAgentPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  {/* <Route
                path="/clients/workloads"
                element={
                  <ProtectedRoute requireProject>
                    <AppLayout>
                      <WorkloadsPage />
                    </AppLayout>
                  </ProtectedRoute>
                }
              /> */}
                  <Route
                    path="/clients/agents"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <AgentsPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  {/* <Route
                path="/clients/workloads/create"
                element={
                  <ProtectedRoute requireProject>
                    <AppLayout>
                      <CreateWorkloadPage />
                    </AppLayout>
                  </ProtectedRoute>
                }
              /> */}

                  {/* <Route
                path="/clients/workloads/edit/:id"
                element={
                  <ProtectedRoute requireProject>
                    <AppLayout>
                      <CreateWorkloadPage />
                    </AppLayout>
                  </ProtectedRoute>
                }
              /> */}

                  <Route
                    path="/clients/workloads/create"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <WorkloadIdentitiesPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/clients/workloads/edit/:id"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <WorkloadIdentitiesPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/clients/workloads"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <WorkloadCertificatePage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  {/* Redirects for legacy non-context routes */}
                  <Route
                    path="/users"
                    element={<Navigate to="/admin/users" replace />}
                  />

                  {/* Context-aware RBAC routes */}
                  <Route path="/:context">
                    <Route
                      path="users"
                      element={
                        <ProtectedRoute requireProject>
                          <AppLayout>
                            <UsersPage />
                          </AppLayout>
                        </ProtectedRoute>
                      }
                    />

                    {/* <Route
                  path="groups"
                  element={
                    <ProtectedRoute requireProject>
                      <AppLayout>
                        <GroupsPage />
                      </AppLayout>
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="groups/create"
                  element={
                    <ProtectedRoute requireProject>
                      <AppLayout>
                        <CreateGroupPage />
                      </AppLayout>
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="groups/edit/:id"
                  element={
                    <ProtectedRoute requireProject>
                      <AppLayout>
                        <CreateGroupPage />
                      </AppLayout>
                    </ProtectedRoute>
                  }
                /> */}

                    <Route
                      path="roles"
                      element={
                        <ProtectedRoute requireProject>
                          <AppLayout>
                            <RolesPage />
                          </AppLayout>
                        </ProtectedRoute>
                      }
                    />

                    <Route
                      path="scopes"
                      element={
                        <ProtectedRoute requireProject>
                          <AppLayout>
                            <ScopesPage />
                          </AppLayout>
                        </ProtectedRoute>
                      }
                    />

                    <Route
                      path="api-oauth-scopes"
                      element={
                        <ProtectedRoute requireProject>
                          <AppLayout>
                            <ApiOAuthScopesPage />
                          </AppLayout>
                        </ProtectedRoute>
                      }
                    />

                    <Route
                      path="permissions"
                      element={
                        <ProtectedRoute requireProject>
                          <AppLayout>
                            <PermissionsPage />
                          </AppLayout>
                        </ProtectedRoute>
                      }
                    />

                    <Route
                      path="resources"
                      element={
                        <ProtectedRoute requireProject>
                          <AppLayout>
                            <PermissionResourcesPage />
                          </AppLayout>
                        </ProtectedRoute>
                      }
                    />

                    <Route
                      path="role-bindings"
                      element={
                        <ProtectedRoute requireProject>
                          <AppLayout>
                            <RoleBindingsPage />
                          </AppLayout>
                        </ProtectedRoute>
                      }
                    />
                  </Route>

                  {/* Legacy redirects - redirect old paths to admin context */}
                  <Route
                    path="/roles"
                    element={<Navigate to="/admin/roles" replace />}
                  />
                  <Route
                    path="/scopes"
                    element={<Navigate to="/admin/scopes" replace />}
                  />
                  <Route
                    path="/api-oauth-scopes"
                    element={<Navigate to="/admin/api-oauth-scopes" replace />}
                  />
                  <Route
                    path="/permissions"
                    element={<Navigate to="/admin/permissions" replace />}
                  />
                  <Route
                    path="/resources"
                    element={<Navigate to="/admin/resources" replace />}
                  />
                  <Route
                    path="/mappings"
                    element={<Navigate to="/admin/role-bindings" replace />}
                  />
                  <Route
                    path="/role-bindings"
                    element={<Navigate to="/admin/role-bindings" replace />}
                  />

                  <Route
                    path="/authentication"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <AuthenticationPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/authentication/create"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <CreateAuthMethodPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/authentication/saml/create"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <CreateSamlMethodPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/authentication/saml/edit/:id"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <EditSamlMethodPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/vault"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <VaultPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/vault/import"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <ImportSecretsPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/logs"
                    element={<Navigate to="/logs/auth" replace />}
                  />

                  <Route
                    path="/logs/auth"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <AuthLogsPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/logs/audit"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <AuditLogsPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/logs/m2m"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <M2MLogsPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/logs/configure"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <LogsConfigurationPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/roles"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <RolesPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/roles/templates"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <RoleTemplatesPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  {/* RBAC Routes */}
                  <Route
                    path="/scopes"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <ScopesPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/permissions"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <PermissionsPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/external-services"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <ExternalServicesPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/external-services/add"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <AddExternalServicePage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/external-services/:serviceId/sdk"
                    element={<LegacyExternalServiceSdkRedirect />}
                  />

                  <Route
                    path="/sdk"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <SDKHubPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/sdk/:surface"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <SDKHubPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/sdk/:surface/:entityId"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <SDKHubPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  {/* Custom Domains */}
                  <Route
                    path="/custom-domains"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <CustomDomainsPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  {/* LEGACY/OBSOLETE: SDK Manager route has been deprecated */}
                  {/* <Route
                path="/sdk/manager"
                element={
                  <ProtectedRoute requireProject>
                    <AppLayout>
                      <SDKManagerPage />
                    </AppLayout>
                  </ProtectedRoute>
                }
              /> */}

                  {/* Trust Delegation */}
                  <Route
                    path="/trust-delegation"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <TrustDelegationPoliciesPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/trust-delegation/active"
                    element={
                      <ProtectedRoute requireProject>
                        <Navigate to="/trust-delegation" replace />
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/trust-delegation/new"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <TrustDelegationPolicyFormPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/trust-delegation/:policyId"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <TrustDelegationPolicyDetailPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/trust-delegation/:policyId/edit"
                    element={
                      <ProtectedRoute requireProject>
                        <AppLayout>
                          <TrustDelegationPolicyFormPage />
                        </AppLayout>
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/trust-delegation/policies"
                    element={
                      <ProtectedRoute requireProject>
                        <Navigate to="/trust-delegation" replace />
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/trust-delegation/policies/new"
                    element={
                      <ProtectedRoute requireProject>
                        <Navigate to="/trust-delegation/new" replace />
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/trust-delegation/policies/:policyId"
                    element={
                      <ProtectedRoute requireProject>
                        <LegacyTrustDelegationPolicyDetailRedirect />
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/trust-delegation/policies/:policyId/edit"
                    element={
                      <ProtectedRoute requireProject>
                        <LegacyTrustDelegationPolicyEditRedirect />
                      </ProtectedRoute>
                    }
                  />

                  <Route
                    path="/trust-delegation/logs"
                    element={
                      <ProtectedRoute requireProject>
                        <Navigate to="/trust-delegation" replace />
                      </ProtectedRoute>
                    }
                  />
                </Routes>

                {/* Professional toast notification system */}
                <Toaster
                  position="bottom-right"
                  toastOptions={{
                    duration: 4000,
                    style: {
                      fontSize: "14px",
                      fontWeight: "500",
                      maxWidth: "500px",
                    },
                  }}
                />

                {/* Guided tour overlay */}
                <GuidedTourOverlay />
              </div>
            </GuidedTourProvider>
          </WizardProvider>
        </Router>
      </RbacAudienceProvider>
    </AuthProvider>
  );
}

/**
 * Main App component with routing and state management
 *
 * Provides:
 * - Redux store provider
 * - Session initialization
 * - React Router for navigation
 * - Main application layout
 * - Route definitions for all pages
 * - Toast notifications system
 */
function App() {
  return (
    <Provider store={store}>
      <AppContent />
    </Provider>
  );
}

export default App;

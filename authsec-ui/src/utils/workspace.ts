import { SessionManager } from "./sessionManager";
import { getTenantFromUrl } from "./subdomainUtils";

/**
 * Get the current workspace ID
 * @returns The current workspace ID or null if no workspace is selected
 */
export function getWorkspaceId(): string | null {
  const session = SessionManager.getSession();
  if (session?.tenant_id) {
    return session.tenant_id;
  }

  if (session?.jwtPayload?.tenant_id) {
    return session.jwtPayload.tenant_id;
  }

  const slug = getTenantFromUrl();
  if (slug && isLikelyTenantId(slug)) {
    return slug;
  }

  return null;
}

/**
 * Get the current workspace
 * @returns The current workspace or null if no workspace is selected
 */
export function getCurrentWorkspace() {
  const session = SessionManager.getSession();
  const tenantId = getWorkspaceId();
  if (!tenantId) return null;
  
  return {
    id: tenantId,
    name: `Tenant ${tenantId}`,
    slug: tenantId
  };
}

/**
 * Resolve the current tenant ID using session data or URL context
 */
export function resolveTenantId(): string | null {
  return getWorkspaceId();
}

function isLikelyTenantId(value: string): boolean {
  // Accept canonical UUIDs only
  const uuidRegex = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$/;
  return uuidRegex.test(value);
}

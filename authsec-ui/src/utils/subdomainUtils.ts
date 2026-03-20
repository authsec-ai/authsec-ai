/**
 * Subdomain Detection Utility
 *
 * Provides utilities to detect subdomain context and suggest appropriate auth pages
 */

export interface SubdomainInfo {
  isMainDomain: boolean;
  isTenantSubdomain: boolean;
  tenantSlug: string | null;
  fullHostname: string;
}

/**
 * Detects the current subdomain and determines the context
 *
 * Examples:
 * - app.authsec.dev → Main domain
 * - brcm.app.authsec.dev → Tenant subdomain (signin to brcm)
 * - localhost → Main domain (development)
 */
export function detectSubdomain(): SubdomainInfo {
  const hostname = window.location.hostname;

  // For local development
  if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname.startsWith('192.168')) {
    return {
      isMainDomain: true,
      isTenantSubdomain: false,
      tenantSlug: null,
      fullHostname: hostname
    };
  }

  // Parse subdomain from hostname
  const parts = hostname.split('.');

  if (parts.length >= 3) {
    const tenantSlug = parts[0];

    // Check if subdomain is NOT 'app' or 'www' (main domain indicators)
    if (tenantSlug !== 'app' && tenantSlug !== 'www') {
      return {
        isMainDomain: false,
        isTenantSubdomain: true,
        tenantSlug,
        fullHostname: hostname
      };
    }
  }

  // Default to main domain
  return {
    isMainDomain: true,
    isTenantSubdomain: false,
    tenantSlug: null,
    fullHostname: hostname
  };
}

/**
 * Gets the tenant slug from the current URL
 */
export function getTenantFromUrl(): string | null {
  const info = detectSubdomain();
  return info.tenantSlug;
}

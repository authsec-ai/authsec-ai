import type { ExternalService, Resource, Scope } from "@/types/entities";
import { convertExternalServicesToResources } from "./external-resource-utils";

/**
 * Generates a list of mock resources for external services to be used in the role permission matrix.
 * This allows roles to be assigned permissions for external services.
 *
 * @param services Array of external services to convert to resources
 * @returns Array of resources with scopes that can be used in the role permission matrix
 */
export function generateExternalServiceResources(services: ExternalService[]): Resource[] {
  // Convert external services to resources
  return convertExternalServicesToResources(services);
}

/**
 * Updates the mockResources array to include external service resources
 * so they appear in the role permission matrix.
 *
 * @param mockResources The original mock resources array
 * @param externalServices The external services to include
 * @returns Updated resources array with external services
 */
export function injectExternalServicesIntoResources(
  mockResources: Resource[],
  externalServices: ExternalService[]
): Resource[] {
  const externalResources = generateExternalServiceResources(externalServices);

  // Filter out any existing external resources to avoid duplicates
  const internalResources = mockResources.filter((r) => !r.isExternal);

  return [...internalResources, ...externalResources];
}

/**
 * Checks if a resource ID belongs to an external service
 *
 * @param resourceId The resource ID to check
 * @returns Boolean indicating if the resource is from an external service
 */
export function isExternalServiceResource(resourceId: string): boolean {
  return resourceId.startsWith("ext-");
}

/**
 * Extracts the external service ID from a resource ID
 *
 * @param resourceId The resource ID to extract from
 * @returns The external service ID or null if not an external resource
 */
export function getExternalServiceIdFromResource(resourceId: string): string | null {
  if (!isExternalServiceResource(resourceId)) {
    return null;
  }

  return resourceId.substring(4); // Remove 'ext-' prefix
}

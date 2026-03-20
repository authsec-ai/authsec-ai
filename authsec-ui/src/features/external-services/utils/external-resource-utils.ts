import type { ExternalService, Resource, Scope } from "@/types/entities";

/**
 * Converts external services to resource objects that can be displayed
 * in the Resources page alongside internal resources.
 *
 * @param services Array of external services to convert
 * @returns Array of Resource objects with isExternal flag set to true
 */
export function convertExternalServicesToResources(services: ExternalService[]): Resource[] {
  return services
    .filter((service) => service.status === "connected") // Only include connected services
    .map((service) => {
      // Create default scopes based on service provider
      const scopes: Scope[] = generateDefaultScopes(service);

      return {
        id: `ext-${service.id}`,
        name: service.name,
        description: `External ${service.provider} service`,
        clientId: "external",
        clientName: "External services and secrets management",
        type: "external",
        scopes,
        linkedRoles: [],
        isActive: service.status === "connected",
        createdAt: service.createdAt || service.lastSync,
        updatedAt: service.lastSync,
        isExternal: true,
        externalServiceId: service.id,
        externalServiceName: service.name,
      };
    });
}

/**
 * Generates default scopes for an external service based on its provider.
 * These are placeholder scopes that would typically be discovered from the
 * external service's API documentation.
 */
function generateDefaultScopes(service: ExternalService): Scope[] {
  const baseScopes: Scope[] = [];
  const now = new Date().toISOString();

  // Add read scope for all services
  baseScopes.push({
    id: `${service.id}-read`,
    name: `${service.provider}:read`,
    description: `Read access to ${service.name}`,
    resourceId: `ext-${service.id}`,
    createdAt: now,
    updatedAt: now,
  });

  // Add provider-specific scopes
  switch (service.provider) {
    case "google":
      baseScopes.push(
        {
          id: `${service.id}-drive`,
          name: `${service.provider}:drive`,
          description: "Access to Google Drive files and folders",
          resourceId: `ext-${service.id}`,
          createdAt: now,
          updatedAt: now,
        },
        {
          id: `${service.id}-sheets`,
          name: `${service.provider}:sheets`,
          description: "Access to Google Sheets",
          resourceId: `ext-${service.id}`,
          createdAt: now,
          updatedAt: now,
        }
      );
      break;

    case "microsoft":
      baseScopes.push(
        {
          id: `${service.id}-files`,
          name: `${service.provider}:files`,
          description: "Access to OneDrive files",
          resourceId: `ext-${service.id}`,
          createdAt: now,
          updatedAt: now,
        },
        {
          id: `${service.id}-mail`,
          name: `${service.provider}:mail`,
          description: "Access to Outlook mail",
          resourceId: `ext-${service.id}`,
          createdAt: now,
          updatedAt: now,
        }
      );
      break;

    case "salesforce":
      baseScopes.push(
        {
          id: `${service.id}-contacts`,
          name: `${service.provider}:contacts`,
          description: "Access to Salesforce contacts",
          resourceId: `ext-${service.id}`,
          createdAt: now,
          updatedAt: now,
        },
        {
          id: `${service.id}-opportunities`,
          name: `${service.provider}:opportunities`,
          description: "Access to Salesforce opportunities",
          resourceId: `ext-${service.id}`,
          createdAt: now,
          updatedAt: now,
        }
      );
      break;

    case "slack":
      baseScopes.push(
        {
          id: `${service.id}-channels`,
          name: `${service.provider}:channels`,
          description: "Access to Slack channels",
          resourceId: `ext-${service.id}`,
          createdAt: now,
          updatedAt: now,
        },
        {
          id: `${service.id}-messages`,
          name: `${service.provider}:messages`,
          description: "Access to Slack messages",
          resourceId: `ext-${service.id}`,
          createdAt: now,
          updatedAt: now,
        }
      );
      break;

    case "github":
      baseScopes.push(
        {
          id: `${service.id}-repos`,
          name: `${service.provider}:repos`,
          description: "Access to GitHub repositories",
          resourceId: `ext-${service.id}`,
          createdAt: now,
          updatedAt: now,
        },
        {
          id: `${service.id}-issues`,
          name: `${service.provider}:issues`,
          description: "Access to GitHub issues",
          resourceId: `ext-${service.id}`,
          createdAt: now,
          updatedAt: now,
        }
      );
      break;

    default:
      // Add generic scopes for other providers
      baseScopes.push({
        id: `${service.id}-data`,
        name: `${service.provider}:data`,
        description: `Access to ${service.name} data`,
        resourceId: `ext-${service.id}`,
        createdAt: now,
        updatedAt: now,
      });
  }

  // Add write scope for all services
  baseScopes.push({
    id: `${service.id}-write`,
    name: `${service.provider}:write`,
    description: `Write access to ${service.name}`,
    resourceId: `ext-${service.id}`,
    createdAt: now,
    updatedAt: now,
  });

  return baseScopes;
}

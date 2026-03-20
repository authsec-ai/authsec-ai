import type { TourConfig } from "../types";

/**
 * Central registry for all guided tours
 * Add new tours here to make them available across the application
 */
export const TOUR_REGISTRY: Record<string, TourConfig> = {
  "clients-onboarding": {
    tourId: "clients-onboarding",
    pageId: "clients",
    autoStart: true,
    steps: [
      {
        id: "add-voice-agent-button",
        target: '[data-tour-id="add-voice-agent-button"]',
        heading: "Add Voice Agent",
        description:
          "Add voice authentication capabilities to your application. Click here to configure a voice agent for biometric authentication.",
        position: "bottom",
        spotlightPadding: 12,
      },
      {
        id: "onboard-button",
        target: '[data-tour-id="onboard-button"]',
        heading: "Onboard Your First Client",
        description:
          "Start by adding an MCP server or AI agent. Click here to create a new client with authentication methods and get your SDK integration code.",
        position: "bottom",
        spotlightPadding: 12,
      },
      // {
      //   id: "clients-table",
      //   target: '[data-tour-id="clients-table"]',
      //   heading: "Manage Your Clients",
      //   description:
      //     "View all your onboarded clients, their status, and authentication methods. Click on any client to view details, manage auth providers, or get SDK integration code.",
      //   position: "top",
      //   spotlightPadding: 16,
      // },
      {
        id: "default-client",
        target: '[data-tour-id="clients-table"] tbody tr:first-child',
        heading: "Your Default Client",
        description:
          "This is your default client, automatically created when you first logged in. Every user gets a default client to get started quickly with authentication and SDK integration.",
        position: "top",
        spotlightPadding: 12,
      },
    ],
  },

  "external-services-intro": {
    tourId: "external-services-intro",
    pageId: "external-services",
    autoStart: true,
    steps: [
      {
        id: "add-service-button",
        target: '[data-tour-id="add-service-button"]',
        heading: "Connect External Services",
        description:
          "Add OAuth connections to third-party APIs like GitHub, Google, or custom providers. Click here to set up your first external service integration.",
        position: "bottom",
        spotlightPadding: 12,
      },
    ],
  },

  "users-management": {
    tourId: "users-management",
    pageId: "users",
    autoStart: true,
    steps: [
      {
        id: "invite-user-button",
        target: '[data-tour-id="invite-user-button"]',
        heading: "Invite Team Members",
        description:
          "Add users to your organization by sending invite links. Manage their roles and permissions to control access to your resources.",
        position: "bottom",
        spotlightPadding: 12,
      },
    ],
  },

  "authentication-setup": {
    tourId: "authentication-setup",
    pageId: "authentication",
    autoStart: true,
    steps: [
      {
        id: "create-auth-method-button",
        target: '[data-tour-id="create-auth-method-button"]',
        heading: "Configure Authentication Methods",
        description:
          "Set up how users will authenticate with your application. Choose from OAuth, OIDC, SAML, API keys, and more.",
        position: "bottom",
        spotlightPadding: 12,
      },
    ],
  },

  "roles-management": {
    tourId: "roles-management",
    pageId: "roles",
    autoStart: true,
    steps: [
      {
        id: "create-role-button",
        target: '[data-tour-id="create-role-button"]',
        heading: "Define User Roles",
        description:
          "Create roles to group permissions together. Assign roles to users to control what they can do in your application.",
        position: "bottom",
        spotlightPadding: 12,
      },
      // {
      //   id: "roles-table",
      //   target: '[data-tour-id="roles-table"]',
      //   heading: "Manage Your Roles",
      //   description:
      //     "View all roles, their permissions, and assigned users. Click on any role to edit, delete, or assign it to users and groups.",
      //   position: "top",
      //   spotlightPadding: 16,
      // },
    ],
  },

  "permissions-management": {
    tourId: "permissions-management",
    pageId: "permissions",
    autoStart: true,
    steps: [
      {
        id: "create-permission-button",
        target: '[data-tour-id="create-permission-button"]',
        heading: "Create Permissions",
        description:
          "Define granular permissions to control access to specific features and resources. Link roles, scopes, and resources to create fine-grained access control.",
        position: "bottom",
        spotlightPadding: 12,
      },
      // {
      //   id: "permissions-table",
      //   target: '[data-tour-id="permissions-table"]',
      //   heading: "View All Permissions",
      //   description:
      //     "See all defined permissions with their roles, scopes, and resources. Click on any permission to edit or delete it.",
      //   position: "top",
      //   spotlightPadding: 16,
      // },
    ],
  },

  "scopes-management": {
    tourId: "scopes-management",
    pageId: "scopes",
    autoStart: true,
    steps: [
      {
        id: "create-scope-button",
        target: '[data-tour-id="create-scope-button"]',
        heading: "Define Permission Scopes",
        description:
          "Scopes define the boundaries of permissions within your projects. Create scopes to organize and limit access to specific areas of your application.",
        position: "bottom",
        spotlightPadding: 12,
      },
      // {
      //   id: "scopes-table",
      //   target: '[data-tour-id="scopes-table"]',
      //   heading: "Manage Scopes",
      //   description:
      //     "View all scopes and their associated resources. Click on any scope to edit its configuration or delete it.",
      //   position: "top",
      //   spotlightPadding: 16,
      // },
    ],
  },

  "role-bindings-management": {
    tourId: "role-bindings-management",
    pageId: "role-bindings",
    autoStart: true,
    steps: [
      {
        id: "create-binding-button",
        target: '[data-tour-id="create-binding-button"]',
        heading: "Create Role Bindings",
        description:
          "Bind roles to users or groups to grant them specific permissions. This is where you assign roles to control who can access what in your application.",
        position: "bottom",
        spotlightPadding: 12,
      },
      // {
      //   id: "bindings-table",
      //   target: '[data-tour-id="bindings-table"]',
      //   heading: "View Role Bindings",
      //   description:
      //     "See all role bindings showing which users have which roles and scopes. Click on any binding to view details or remove it.",
      //   position: "top",
      //   spotlightPadding: 16,
      // },
    ],
  },

  "mappings-management": {
    tourId: "mappings-management",
    pageId: "mappings",
    autoStart: true,
    steps: [
      {
        id: "create-mapping-button",
        target: '[data-tour-id="create-mapping-button"]',
        heading: "Manage Role Mappings",
        description:
          "View and manage all role bindings across your organization. See which users have which roles and scopes at a glance.",
        position: "bottom",
        spotlightPadding: 12,
      },
      // {
      //   id: "mappings-table",
      //   target: '[data-tour-id="mappings-table"]',
      //   heading: "Complete Mapping View",
      //   description:
      //     "View the comprehensive mapping of users to roles and scopes. This gives you a complete overview of all access permissions.",
      //   position: "top",
      //   spotlightPadding: 16,
      // },
    ],
  },

  "spire-agents-intro": {
    tourId: "spire-agents-intro",
    pageId: "spire-agents",
    autoStart: true,
    steps: [
      {
        id: "agents-refresh",
        target: '[data-tour-id="agents-refresh"]',
        heading: "Refresh Agent Status",
        description:
          "Click here to refresh the agent list and get the latest status information from all SPIRE agents in your infrastructure.",
        position: "bottom",
        spotlightPadding: 12,
      },
      {
        id: "agents-table",
        target: '[data-tour-id="agents-table"]',
        heading: "SPIRE Agent Status",
        description:
          "Monitor all SPIRE agents running in your infrastructure. See agent health, attestation status, and connected workloads. Agents are responsible for issuing and rotating SPIFFE certificates.",
        position: "top",
        spotlightPadding: 16,
      },
    ],
  },

  "workload-certificates-intro": {
    tourId: "workload-certificates-intro",
    pageId: "workload-certificates",
    autoStart: true,
    steps: [
      {
        id: "register-workload-button",
        target: '[data-tour-id="register-workload-button"]',
        heading: "Register Workload Identities",
        description:
          "Create and manage SPIFFE workload identities for your services. Click here to register a new workload and get mTLS certificates for secure service-to-service communication.",
        position: "bottom",
        spotlightPadding: 12,
      },
      // {
      //   id: "certificates-table",
      //   target: '[data-tour-id="certificates-table"]',
      //   heading: "Manage Certificates",
      //   description:
      //     "View all SPIFFE certificates issued to your workloads. Monitor expiration dates, rotation status, and certificate health. Click on any certificate to view full details.",
      //   position: "top",
      //   spotlightPadding: 16,
      // },
    ],
  },

  "api-oauth-scopes-intro": {
    tourId: "api-oauth-scopes-intro",
    pageId: "api-oauth-scopes",
    autoStart: true,
    steps: [
      {
        id: "create-scope-button",
        target: '[data-tour-id="create-api-scope-button"]',
        heading: "Define API Scopes",
        description:
          "Create OAuth 2.0 scopes to define what permissions external applications can request when accessing your API.",
        position: "bottom",
        spotlightPadding: 12,
      },
      // {
      //   id: "scopes-table",
      //   target: '[data-tour-id="api-scopes-table"]',
      //   heading: "Manage OAuth Scopes",
      //   description:
      //     "View all OAuth scopes defined for your API. Each scope represents a specific permission that can be granted to client applications.",
      //   position: "top",
      //   spotlightPadding: 16,
      // },
    ],
  },

  "auth-logs-intro": {
    tourId: "auth-logs-intro",
    pageId: "auth-logs",
    autoStart: true,
    steps: [
      {
        id: "logs-configure",
        target: '[data-tour-id="logs-configure"]',
        heading: "Configure Log Settings",
        description:
          "Adjust log retention, filtering rules, and export settings to customize how authentication events are tracked.",
        position: "bottom",
        spotlightPadding: 12,
      },
      // {
      //   id: "logs-table",
      //   target: '[data-tour-id="auth-logs-table"]',
      //   heading: "Track Authentication Events",
      //   description:
      //     "Monitor all authentication attempts including successful logins, failed attempts, and suspicious activities. Use this to detect security threats and troubleshoot access issues.",
      //   position: "top",
      //   spotlightPadding: 16,
      // },
    ],
  },
};

/**
 * Get a tour configuration by ID
 */
export function getTourConfig(tourId: string): TourConfig | null {
  return TOUR_REGISTRY[tourId] || null;
}

/**
 * Get all tour IDs
 */
export function getAllTourIds(): string[] {
  return Object.keys(TOUR_REGISTRY);
}

/**
 * Get tours for a specific page
 */
export function getToursForPage(pageId: string): TourConfig[] {
  return Object.values(TOUR_REGISTRY).filter((tour) => tour.pageId === pageId);
}

/**
 * Check if a tour exists in registry
 */
export function tourExists(tourId: string): boolean {
  return tourId in TOUR_REGISTRY;
}

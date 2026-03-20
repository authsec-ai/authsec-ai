import type {
  Client,
  ClientInsert,
  ClientUpdate,
  ClientWithAuthMethods,
  PaginatedClientsResponse,
  ClientsFilters,
  ClientsPagination,
} from "@/types/entities";
import { mockClients } from "@/data/clients";

export class ClientsService {
  private static readonly DEFAULT_PAGE_SIZE = 10;
  private static readonly MAX_PAGE_SIZE = 100;

  /**
   * Get paginated clients with optional filtering and sorting
   */
  static async getClients(
    filters: ClientsFilters = {},
    pagination: ClientsPagination = {}
  ): Promise<PaginatedClientsResponse> {
    const {
      page = 1,
      pageSize = this.DEFAULT_PAGE_SIZE,
      sortBy = "created_at",
      sortOrder = "desc",
    } = pagination;

    const validatedPageSize = Math.min(pageSize, this.MAX_PAGE_SIZE);
    
    // Simulate async operation
    await new Promise(resolve => setTimeout(resolve, 200));

    // Filter mock data
    let filteredClients = [...mockClients];

    // Apply filters
    if (filters.workspace_id) {
      filteredClients = filteredClients.filter(client => client.workspace_id === filters.workspace_id);
    }
    if (filters.type) {
      filteredClients = filteredClients.filter(client => client.type === filters.type);
    }
    if (filters.access_status) {
      filteredClients = filteredClients.filter(client => client.access_status === filters.access_status);
    }
    if (filters.access_level) {
      filteredClients = filteredClients.filter(client => client.access_level === filters.access_level);
    }
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      filteredClients = filteredClients.filter(client => 
        client.name.toLowerCase().includes(searchLower) ||
        client.description?.toLowerCase().includes(searchLower) ||
        client.endpoint.toLowerCase().includes(searchLower) ||
        client.tags.toLowerCase().includes(searchLower)
      );
    }
    if (filters.authentication_type) {
      filteredClients = filteredClients.filter(client => client.authentication_type === filters.authentication_type);
    }
    if (filters.mfa_enabled !== undefined) {
      filteredClients = filteredClients.filter(client => 
        (client.mfa_config?.enabled || false) === filters.mfa_enabled
      );
    }

    // Sort clients
    filteredClients.sort((a, b) => {
      let aValue: any, bValue: any;
      
      switch (sortBy) {
        case "name":
          aValue = a.name;
          bValue = b.name;
          break;
        case "created_at":
          aValue = new Date(a.created_at);
          bValue = new Date(b.created_at);
          break;
        case "last_accessed":
          aValue = a.last_accessed ? new Date(a.last_accessed) : new Date(0);
          bValue = b.last_accessed ? new Date(b.last_accessed) : new Date(0);
          break;
        case "total_requests":
          aValue = a.total_requests;
          bValue = b.total_requests;
          break;
        default:
          aValue = a.created_at;
          bValue = b.created_at;
      }

      if (sortOrder === "asc") {
        return aValue > bValue ? 1 : -1;
      } else {
        return aValue < bValue ? 1 : -1;
      }
    });

    // Paginate
    const offset = (page - 1) * validatedPageSize;
    const paginatedData = filteredClients.slice(offset, offset + validatedPageSize);
    const totalPages = Math.ceil(filteredClients.length / validatedPageSize);

    return {
      data: paginatedData,
      count: filteredClients.length,
      totalPages,
      currentPage: page,
      hasMore: page < totalPages,
    };
  }

  /**
   * Get a single client by ID with auth methods
   */
  static async getClient(id: string): Promise<ClientWithAuthMethods | null> {
    // Simulate async operation
    await new Promise(resolve => setTimeout(resolve, 100));

    const client = mockClients.find(c => c.id === id);
    return client || null;
  }

  /**
   * Create a new client
   */
  static async createClient(clientData: ClientInsert): Promise<Client> {
    // Simulate async operation
    await new Promise(resolve => setTimeout(resolve, 300));

    const newClient: Client = {
      id: `client_${Date.now()}`,
      workspace_id: clientData.workspace_id,
      secret_id: clientData.secret_id || null,
      name: clientData.name,
      description: clientData.description || null,
      type: clientData.type || "other",
      tags: clientData.tags || "",
      authentication_type: clientData.authentication_type || "custom",
      metadata: clientData.metadata || {},
      roles: clientData.roles || [],
      mfa_config: clientData.mfa_config || null,
      successful_authentications: clientData.successful_authentications || 0,
      denied_authentications: clientData.denied_authentications || 0,
      view_policies_applicable: clientData.view_policies_applicable || [],
      endpoint: clientData.endpoint || "",
      access_status: clientData.access_status || "active",
      access_level: clientData.access_level || "internal",
      total_requests: clientData.total_requests || 0,
      last_accessed: clientData.last_accessed || null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      created_by: clientData.created_by || "system",
    };

    // In a real app, this would be persisted to the database
    console.log("Mock: Created client", newClient);
    
    return newClient;
  }

  /**
   * Update an existing client
   */
  static async updateClient(id: string, updates: ClientUpdate): Promise<Client> {
    // Simulate async operation
    await new Promise(resolve => setTimeout(resolve, 200));

    const client = mockClients.find(c => c.id === id);
    if (!client) {
      throw new Error("Client not found");
    }

    const updatedClient = {
      ...client,
      ...updates,
      updated_at: new Date().toISOString(),
    };

    console.log("Mock: Updated client", updatedClient);
    return updatedClient;
  }

  /**
   * Delete a client
   */
  static async deleteClient(id: string): Promise<void> {
    // Simulate async operation
    await new Promise(resolve => setTimeout(resolve, 200));

    const client = mockClients.find(c => c.id === id);
    if (!client) {
      throw new Error("Client not found");
    }

    console.log("Mock: Deleted client", id);
  }

  /**
   * Toggle client status (active/disabled)
   */
  static async toggleClientStatus(id: string): Promise<Client> {
    // First get current status
    const client = await this.getClient(id);

    if (!client) {
      throw new Error("Client not found");
    }

    const newStatus = client.access_status === "active" ? "disabled" : "active";

    return this.updateClient(id, { access_status: newStatus });
  }

  /**
   * Update last accessed timestamp for a client
   */
  static async updateLastAccessed(id: string): Promise<void> {
    // Mock implementation - would update via API
    console.log(`Mock: Updated last accessed for client ${id}`);
  }

  /**
   * Increment total requests counter for a client
   */
  static async incrementRequests(id: string, increment: number = 1): Promise<void> {
    // Mock implementation - would update via API
    console.log(`Mock: Incremented requests for client ${id} by ${increment}`);
  }

  /**
   * Get clients by workspace with pagination
   */
  static async getClientsByWorkspace(
    workspaceId: string,
    pagination: ClientsPagination = {}
  ): Promise<PaginatedClientsResponse> {
    return this.getClients({ workspace_id: workspaceId }, pagination);
  }

  /**
   * Search clients across multiple fields
   */
  static async searchClients(
    searchTerm: string,
    workspaceId?: string,
    pagination: ClientsPagination = {}
  ): Promise<PaginatedClientsResponse> {
    const filters: ClientsFilters = { search: searchTerm };
    if (workspaceId) {
      filters.workspace_id = workspaceId;
    }
    return this.getClients(filters, pagination);
  }

  /**
   * Get clients statistics for a workspace
   */
  static async getClientsStats(workspaceId: string) {
    // Mock implementation - return empty stats
    const stats = {
      total: 0,
      byStatus: {
        active: 0,
        disabled: 0,
        restricted: 0,
      },
      byType: {
        mcp_server: 0,
        app: 0,
        api: 0,
        other: 0,
      },
      byAuthType: {
        sso: 0,
        custom: 0,
        saml2: 0,
      },
      security: {
        mfaEnabled: 0,
        highAuthSuccess: 0,
      },
    };

    return stats;
  }

  // ========== BULK OPERATIONS ==========

  /**
   * Bulk update client status
   */
  static async bulkUpdateStatus(
    clientIds: string[],
    status: "active" | "restricted" | "disabled"
  ): Promise<Client[]> {
    // Mock implementation
    return [];
  }

  /**
   * Bulk assign roles to clients
   */
  static async bulkAssignRoles(
    clientIds: string[],
    roles: string[],
    mode: "add" | "replace" = "add"
  ): Promise<Client[]> {
    // Mock implementation
    return [];
  }

  /**
   * Bulk configure MFA for clients
   */
  static async bulkConfigureMFA(
    clientIds: string[],
    mfaConfig: {
      enabled: boolean;
      methods: string[];
      backup_codes?: boolean;
      grace_period?: number;
    }
  ): Promise<Client[]> {
    // Mock implementation
    return [];
  }

  /**
   * Bulk apply policies to clients
   */
  static async bulkApplyPolicies(
    clientIds: string[],
    policyIds: string[],
    mode: "add" | "replace" = "add"
  ): Promise<Client[]> {
    // Mock implementation
    return [];
  }

  /**
   * Bulk delete clients
   */
  static async bulkDeleteClients(clientIds: string[]): Promise<void> {
    // Mock implementation
    console.log(`Mock: Bulk deleted clients ${clientIds.join(", ")}`);
  }

  /**
   * Bulk update authentication type
   */
  static async bulkUpdateAuthType(
    clientIds: string[],
    authType: "sso" | "custom" | "saml2"
  ): Promise<Client[]> {
    // Mock implementation
    return [];
  }

  /**
   * Reset authentication statistics for clients
   */
  static async bulkResetAuthStats(clientIds: string[]): Promise<Client[]> {
    // Mock implementation
    return [];
  }
}

// React hooks for easier integration
export function useClientsPagination() {
  const defaultPagination: ClientsPagination = {
    page: 1,
    pageSize: 10,
    sortBy: "created_at",
    sortOrder: "desc",
  };

  return {
    defaultPagination,
    createPaginationParams: (
      page: number,
      pageSize: number = 10,
      sortBy: string = "created_at",
      sortOrder: "asc" | "desc" = "desc"
    ): ClientsPagination => ({
      page,
      pageSize,
      sortBy: sortBy as any,
      sortOrder,
    }),
  };
}

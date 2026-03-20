export interface DelegationPolicyUI {
  id: string;
  roleName: string;
  agentType: string;
  allowedPermissions: string[];
  maxTtlSeconds: number;
  maxTtlLabel: string;
  enabled: boolean;
  clientId: string;
  clientLabel: string;
  tenantId?: string;
  createdBy?: string;
}

export interface PermissionOption {
  key: string;
  label: string;
  group: string;
  description?: string;
  sensitive: boolean;
}

export interface DurationParts {
  value: number;
  unit: "minutes" | "hours" | "days";
}

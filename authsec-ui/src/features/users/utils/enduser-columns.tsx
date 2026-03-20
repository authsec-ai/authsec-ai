import React from "react";
import type { ApiUser } from "./user-table-utils";
import type { ColumnConfig } from "../components/ColumnSelector";
import { UserTableUtils } from "./user-table-utils";
import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { CopyButton } from "../../../components/ui/copy-button";
import { 
  Avatar, 
  AvatarFallback, 
  AvatarImage 
} from "../../../components/ui/avatar";
import { 
  Calendar, 
  MapPin, 
  Phone, 
  Building, 
  User, 
  Shield,
  Clock,
  AlertTriangle,
  CheckCircle
} from "lucide-react";

const formatIdentifier = (value?: string | null) => {
  if (!value) return "";
  if (value.length <= 16) return value;
  return `${value.slice(0, 8)}…${value.slice(-4)}`;
};

// Define all possible column configurations based on API response structure (for EndUser view)
export const END_USER_COLUMN_CONFIGS: ColumnConfig[] = [
  // Basic Information
  {
    id: "user",
    label: "User",
    description: "Name, email, and avatar",
    isVisible: true,
    isRequired: true,
    category: "basic"
  },
  {
    id: "status",
    label: "Status",
    description: "Active/Inactive status",
    isVisible: true,
    isRequired: false,
    category: "basic"
  },
  {
    id: "roleTeam",
    label: "Roles",
    description: "Assigned roles",
    isVisible: true,
    isRequired: false,
    category: "roles"
  },
  {
    id: "provider",
    label: "Auth Provider",
    description: "Authentication provider (Entra ID, Google, etc.)",
    isVisible: true,
    isRequired: false,
    category: "basic"
  },
  {
    id: "mfaEnabled",
    label: "MFA Status",
    description: "Multi-factor authentication status",
    isVisible: true,
    isRequired: false,
    category: "security"
  },
  
  
  // Activity & Dates
  {
    id: "lastLogin",
    label: "Last Login",
    description: "Last login date and time",
    isVisible: true,
    isRequired: false,
    category: "activity"
  },
  {
    id: "updatedAt",
    label: "Last Updated",
    description: "Last modification date",
    isVisible: false,
    isRequired: false,
    category: "activity"
  },


  // Security and MFA
  {
    id: "mfaVerified",
    label: "MFA Verified",
    description: "Whether MFA is verified",
    isVisible: false,
    isRequired: false,
    category: "security"
  },
  {
    id: "syncSource",
    label: "Sync Source",
    description: "Directory sync source",
    isVisible: false,
    isRequired: false,
    category: "source"
  },
  {
    id: "providerId",
    label: "Provider ID",
    description: "External provider identifier",
    isVisible: false,
    isRequired: false,
    category: "source"
  },


  // Technical identifiers
  {
    id: "clientId",
    label: "Client ID",
    description: "Associated client identifier",
    isVisible: false,
    isRequired: false,
    category: "technical"
  },
  {
    id: "projectId",
    label: "Project ID",
    description: "Associated project identifier",
    isVisible: false,
    isRequired: false,
    category: "technical"
  },
  {
    id: "tenantDomain",
    label: "Tenant Domain",
    description: "Tenant domain information",
    isVisible: false,
    isRequired: false,
    category: "technical"
  },
  {
    id: "scopesCount",
    label: "Scopes",
    description: "Number of scopes assigned",
    isVisible: false,
    isRequired: false,
    category: "permissions"
  },
  {
    id: "resourcesCount",
    label: "Resources",
    description: "Number of resources accessible",
    isVisible: false,
    isRequired: false,
    category: "permissions"
  },

  // Actions (always visible)
  {
    id: "permissions",
    label: "Permissions",
    description: "View user permissions matrix",
    isVisible: false,
    isRequired: false,
    category: "actions"
  },
  {
    id: "actions",
    label: "Actions",
    description: "User management actions",
    isVisible: true,
    isRequired: true,
    category: "actions"
  }
];

// Cell components for dynamic columns
export const DynamicCellComponents = {
  user: ({ user, onUserClick }: { user: ApiUser; onUserClick?: (user: ApiUser) => void }) => (
    <div className="flex items-center gap-2 sm:gap-3 min-w-0">
      <Avatar className="h-8 w-8 flex-shrink-0">
        <AvatarImage src={user.avatar || (user as any).avatar_url || ""} alt={user.name} />
        <AvatarFallback>
          {UserTableUtils.getUserInitials(user.name, user.email)}
        </AvatarFallback>
      </Avatar>
      <div className="flex-1 min-w-0 overflow-hidden">
        <div className="font-medium truncate" title={user.name}>
          {onUserClick ? (
            <Button 
              variant="link" 
              className="h-auto p-0 font-medium text-left justify-start"
              onClick={() => onUserClick(user)}
            >
              {user.name}
            </Button>
          ) : (
            user.name
          )}
        </div>
        <div className="space-y-1">
          <div className="text-sm text-foreground truncate" title={user.email}>
            {user.email}
          </div>
          {user.id && (
            <div className="flex items-center gap-2 text-xs text-foreground">
              <span
                className="font-mono truncate max-w-[140px]"
                title={user.id}
              >
                {formatIdentifier(user.id)}
              </span>
              <CopyButton
                text={user.id}
                label="User ID"
                variant="ghost"
                size="sm"
                className="h-6 px-2"
              />
            </div>
          )}
        </div>
      </div>
    </div>
  ),

  status: ({ user }: { user: ApiUser }) => (
    <div className="space-y-1">
      <Badge variant={UserTableUtils.getStatusVariant(user.active)}>
        {user.active ? "Active" : "Inactive"}
      </Badge>
      {user.MFAEnabled && (
        <div className="flex items-center gap-1">
          <Shield className="h-3 w-3 text-green-600" />
          <span className="text-xs text-green-600">MFA</span>
        </div>
      )}
    </div>
  ),

  roleTeam: ({ user }: { user: ApiUser }) => (
    <div className="space-y-1">
      <div className="flex flex-wrap gap-1">
        {user.roles && user.roles.length > 0 ? (
          user.roles.slice(0, 2).map((role: any, index) => (
            <Badge 
              key={index} 
              className={UserTableUtils.getRoleColor(user.roles)} 
              variant="outline"
            >
              {typeof role === 'string' ? role : role.name || 'Role'}
            </Badge>
          ))
        ) : (
          <Badge variant="outline" className="text-xs text-gray-500">
            No roles
          </Badge>
        )}
        {user.roles && user.roles.length > 2 && (
          <Badge variant="outline" className="text-xs">
            +{user.roles.length - 2}
          </Badge>
        )}
      </div>
    </div>
  ),

  lastLogin: ({ user }: { user: ApiUser }) => (
    <div className="space-y-1">
      <div className="flex items-center gap-1 text-sm">
        <Calendar className="h-3 w-3 text-foreground" />
        <span className="text-foreground">
          {UserTableUtils.formatLastLogin(user.last_login)}
        </span>
      </div>
      <div className="text-sm text-foreground">
        Provider: {UserTableUtils.formatProvider(user.provider)}
      </div>
    </div>
  ),

  provider: ({ user }: { user: ApiUser }) => (
    <Badge variant="outline" className="text-xs">
      {UserTableUtils.formatProvider(user.provider)}
    </Badge>
  ),

  updatedAt: ({ user }: { user: ApiUser }) => (
    <div className="text-sm text-foreground">
      {new Date(user.updated_at).toLocaleDateString()}
    </div>
  ),

  mfaEnabled: ({ user }: { user: ApiUser }) => (
    <Badge variant={UserTableUtils.getMfaStatusVariant(user.MFAEnabled)} className="text-xs">
      {user.MFAEnabled ? "Enabled" : "Disabled"}
    </Badge>
  ),

  mfaVerified: ({ user }: { user: ApiUser }) => (
    <Badge variant={user.mfa_verified ? "default" : "outline"} className="text-xs">
      {user.mfa_verified ? "Verified" : "Not Verified"}
    </Badge>
  ),

  syncSource: ({ user }: { user: ApiUser }) => (
    <Badge variant="outline" className="text-xs">
      {(user as any).sync_source || "manual"}
    </Badge>
  ),

  providerId: ({ user }: { user: ApiUser }) => (
    <span className="text-sm font-mono">
      {user.provider_id ? 
        (user.provider_id.length > 20 ? 
          `${user.provider_id.substring(0, 20)}...` : 
          user.provider_id) : 
        "—"
      }
    </span>
  ),

  clientId: ({ user }: { user: ApiUser }) => (
    <span className="text-sm font-mono">
      {user.client_id ? 
        `${user.client_id.substring(0, 8)}...` : 
        "—"
      }
    </span>
  ),

  projectId: ({ user }: { user: ApiUser }) => (
    <span className="text-sm font-mono">
      {user.project_id ? 
        `${user.project_id.substring(0, 8)}...` : 
        "—"
      }
    </span>
  ),

  tenantDomain: ({ user }: { user: ApiUser }) => (
    <span className="text-sm font-mono">
      {user.tenant_domain || "—"}
    </span>
  ),

  scopesCount: ({ user }: { user: ApiUser }) => (
    <div className="flex items-center gap-1">
      <Shield className="h-3 w-3 text-foreground" />
      <span className="text-sm">{user.scopes?.length || 0}</span>
    </div>
  ),

  resourcesCount: ({ user }: { user: ApiUser }) => (
    <div className="flex items-center gap-1">
      <Building className="h-3 w-3 text-foreground" />
      <span className="text-sm">{user.resources?.length || 0}</span>
    </div>
  )
};

// Get column header display name
export function getColumnHeader(columnId: string): string {
  const config = END_USER_COLUMN_CONFIGS.find(col => col.id === columnId);
  return config?.label || columnId;
}

// Get column accessor key for sorting
export function getColumnAccessorKey(columnId: string): string {
  const accessorMap: Record<string, string> = {
    user: "name",
    status: "active",
    roleTeam: "roles",
    lastLogin: "last_login",
    updatedAt: "updated_at",
    provider: "provider",
    mfaEnabled: "MFAEnabled",
    mfaVerified: "mfa_verified",
    syncSource: "sync_source",
    providerId: "provider_id",
    clientId: "client_id",
    projectId: "project_id",
    tenantDomain: "tenant_domain",
    scopesCount: "scopes",
    resourcesCount: "resources"
  };
  return accessorMap[columnId] || columnId;
}

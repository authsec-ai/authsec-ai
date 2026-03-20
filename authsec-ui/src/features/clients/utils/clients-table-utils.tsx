import * as React from "react";
import { Link } from "react-router-dom";
import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { CopyButton } from "../../../components/ui/copy-button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../../components/ui/dropdown-menu";
import {
  Popover,
  PopoverAnchor,
  PopoverContent,
} from "../../../components/ui/popover";
import { cn } from "../../../lib/utils";
import {
  Trash2,
  Key,
  Globe,
  Shield,
  ShieldCheck,
  ShieldAlert,
  Tag,
  FileKey,
  Power,
  PowerOff,
  Code,
  ShieldOff,
  MoreHorizontal,
  Copy,
  Eye,
  Mic,
} from "lucide-react";
import type { ClientWithAuthMethods } from "@/types/entities";
import type { ClientData } from "@/app/api/clientApi";
import type { ResponsiveColumnDef } from "../../../components/ui/responsive-data-table";
import type { AdaptiveColumn } from "@/components/ui/adaptive-table";
import { ClientsTableUtils } from "./clients-table-constants";

// Helper functions for getting icons
const getAuthTypeIcon = (authType: ClientWithAuthMethods["authentication_type"]) => {
  const iconMap = {
    sso: Shield,
    custom: Key,
    saml2: FileKey,
  };
  return iconMap[authType as keyof typeof iconMap] || Key;
};

const getAuthMethodVisuals = (methodName: string) => {
  const normalized = methodName.toLowerCase();
  const palette = [
    {
      keywords: ["sso", "saml", "oidc"],
      className: "bg-sky-500/10 text-sky-100 border border-sky-400/30",
      iconWrap: "bg-sky-500/15 border border-sky-400/40 text-sky-100",
      Icon: Shield,
    },
    {
      keywords: ["password", "pwd", "credential", "login"],
      className: "bg-slate-100/5 text-slate-100 border border-slate-200/25",
      iconWrap: "bg-slate-100/10 border border-slate-200/30 text-slate-50",
      Icon: Key,
    },
    {
      keywords: ["mfa", "otp", "totp", "factor"],
      className: "bg-emerald-500/10 text-emerald-50 border border-emerald-300/30",
      iconWrap: "bg-emerald-500/15 border border-emerald-300/40 text-emerald-50",
      Icon: ShieldCheck,
    },
  ];

  const match = palette.find((entry) =>
    entry.keywords.some((keyword) => normalized.includes(keyword)),
  );

  return (
    match || {
      className: "bg-slate-200/10 text-slate-50 border border-slate-200/30",
      iconWrap: "bg-slate-200/15 border border-slate-200/30 text-slate-50",
      Icon: Shield,
    }
  );
};

// Table action handlers interface
export interface ClientsTableActions {
  onDelete: (clientId: string) => void;
  onToggleStatus?: (clientId: string) => void;
  onViewSDK?: (clientId: string) => void;
  onAddAuthMethod?: (clientId: string) => void;
  onShowAuthMethods?: (client: ClientWithAuthMethods) => void;
  onPreviewLogin?: (clientId: string) => void;
  onConfigureVoiceAgent?: (clientId: string) => void;
  newClientId?: string;
  newClientStep?: number;
  onNextNewClientStep?: () => void;
  onDismissNewClient?: () => void;
}

// Cell: Name - Proper hierarchy (14-15px semibold)
export function ClientNameCell({ client }: { client: ClientWithAuthMethods }) {
  const rawClient =
    (client.metadata?.raw_client as ClientData | undefined) ?? (client as unknown as ClientData);
  const oidcEnabled = rawClient?.oidc_enabled ?? false;
  const attachedMethods = Array.isArray(client.attachedMethods) ? client.attachedMethods : [];
  const rawAuthMethods = Array.isArray((rawClient as any)?.auth_methods)
    ? (rawClient as any).auth_methods
    : [];
  const enhancedAuthMethods = Array.isArray(rawClient?.authentication_methods)
    ? rawClient.authentication_methods
    : [];
  const hasAuthMethods =
    oidcEnabled ||
    attachedMethods.length > 0 ||
    rawAuthMethods.length > 0 ||
    enhancedAuthMethods.length > 0;
  const authMethodClientId = rawClient?.client_id || client.id;
  const clientName = client.name || "Unnamed client";

  return (
    <div className="flex items-start min-w-0 w-full">
      <div className="min-w-0 flex-1 space-y-1 overflow-hidden">
        <div
          className="text-[14px] font-semibold text-foreground truncate leading-5"
          title={clientName}
        >
          {clientName}
        </div>
        {!hasAuthMethods && authMethodClientId && (
          <Link
            to="/authentication/create"
            state={{ prefillClientId: authMethodClientId }}
            className="text-[12px] font-medium text-amber-600 dark:text-amber-400 underline underline-offset-2 hover:text-amber-500 dark:hover:text-amber-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500/50 focus-visible:ring-offset-1 block truncate transition-colors"
          >
            Auth methods not added
          </Link>
        )}
      </div>
    </div>
  );
}

// Cell: Authentication Type
export function AuthTypeCell({ client }: { client: ClientWithAuthMethods }) {
  const authType = client.authentication_type || "custom"; // Default to custom if undefined
  const AuthIcon = getAuthTypeIcon(authType);

  return (
    <div className="flex items-center gap-2">
      <AuthIcon className="w-4 h-4 text-foreground" />
      <Badge className={ClientsTableUtils.getAuthTypeBadge(authType)} variant="outline">
        {authType.toUpperCase()}
      </Badge>
    </div>
  );
}

// Cell: Authentication Methods - True pill chips with soft fill
export function AuthMethodsCell({ client }: { client: ClientWithAuthMethods }) {
  const rawClient =
    (client.metadata?.raw_client as ClientData | undefined) ?? (client as unknown as ClientData);
  const enhancedAuthMethods = Array.isArray(rawClient?.authentication_methods)
    ? rawClient.authentication_methods
    : [];
  const attachedMethods = Array.isArray(client.attachedMethods) ? client.attachedMethods : [];

  // Prioritize enhanced authentication methods from API response, fall back to attached methods
  const authMethods = enhancedAuthMethods.length > 0 ? enhancedAuthMethods : attachedMethods;

  // Memoize the auth methods to prevent infinite re-renders
  // Use JSON.stringify to create a stable dependency
  const authMethodsKey = JSON.stringify(authMethods);
  const memoizedAuthMethods = React.useMemo(() => {
    return authMethods.slice(0, 3).map((method) => {
      // Handle both old string format and new object format
      const methodName = typeof method === "string" ? method : method.name;
      const displayName = methodName || "Auth method";
      const isDefault = typeof method === "object" ? method.isDefault : false;
      const methodId = typeof method === "string" ? method : method.id;
      const { Icon, className } = getAuthMethodVisuals(displayName);

      return {
        key: methodId,
        displayName,
        isDefault,
        Icon,
        className,
      };
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authMethodsKey]);

  if (authMethods.length === 0) {
    return <span className="text-muted-foreground text-[14px]">None</span>;
  }

  return (
    <div className="flex flex-wrap items-center gap-1.5 min-w-0">
      {memoizedAuthMethods.map(({ key, displayName, isDefault, Icon }) => (
        <span
          key={key}
          className={`inline-flex items-center gap-1.5 h-7 px-2.5 rounded-full text-[12.5px] font-medium transition-colors ${
            isDefault
              ? "border border-[color:color-mix(in_srgb,var(--editorial-accent)_22%,var(--editorial-border-soft))] bg-[color:color-mix(in_srgb,var(--editorial-accent)_8%,var(--editorial-panel))] text-[var(--editorial-text-1)]"
              : "border border-[color:color-mix(in_srgb,var(--editorial-border-soft)_86%,transparent)] bg-[var(--editorial-panel-soft)] text-[var(--editorial-text-1)]"
          }`}
        >
          <Icon className="h-3 w-3 shrink-0" />
          <span className="truncate capitalize" title={displayName}>
            {displayName}
          </span>
          {isDefault && (
            <span className="text-[11px] font-semibold uppercase tracking-wide opacity-80">
              Default
            </span>
          )}
        </span>
      ))}
      {authMethods.length > 3 && (
        <span className="inline-flex items-center gap-1 h-7 px-2.5 rounded-full border border-dashed border-[color:color-mix(in_srgb,var(--editorial-border-soft)_90%,transparent)] bg-[color:color-mix(in_srgb,var(--editorial-panel-soft)_75%,var(--editorial-panel))] text-[12.5px] font-medium text-[var(--editorial-text-2)]">
          +{authMethods.length - 3} more
        </span>
      )}
    </div>
  );
}

// Cell: Tags
export function TagsCell({ client }: { client: ClientWithAuthMethods }) {
  const tags = ClientsTableUtils.parseTags(client.tags || "");

  if (tags.length === 0) {
    return <span className="text-foreground text-sm">No tags</span>;
  }

  const visibleTags = tags.slice(0, 2);
  const remainingCount = tags.length - visibleTags.length;

  return (
    <div className="flex items-center gap-1 min-w-0">
      {visibleTags.map((tag, index) => (
        <Badge
          key={index}
          variant="secondary"
          className="text-xs bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300"
        >
          <Tag className="w-3 h-3 mr-1" />
          {tag}
        </Badge>
      ))}
      {remainingCount > 0 && (
        <Badge variant="outline" className="text-xs">
          +{remainingCount}
        </Badge>
      )}
    </div>
  );
}

// Cell: Active Status
export function ActiveStatusCell({ client }: { client: ClientWithAuthMethods }) {
  const rawClient =
    (client.metadata?.raw_client as ClientData | undefined) ?? (client as unknown as ClientData);
  const isActive =
    rawClient?.active !== undefined ? rawClient.active : client.access_status === "active";

  return (
    <div className="flex items-center gap-2">
      <div
        className={`h-2.5 w-2.5 rounded-full ring-2 ring-background ${
          isActive ? "bg-emerald-500 dark:bg-emerald-400" : "bg-gray-300 dark:bg-gray-600"
        }`}
      />
      <span
        className={`text-[13.5px] font-medium leading-5 ${
          isActive ? "text-foreground" : "text-muted-foreground"
        }`}
      >
        {isActive ? "Active" : "Inactive"}
      </span>
    </div>
  );
}

// Cell: MFA Enabled (simplified)
export function MfaStatusCell({ client }: { client: ClientWithAuthMethods }) {
  const rawClient =
    (client.metadata?.raw_client as ClientData | undefined) ?? (client as unknown as ClientData);
  const mfaEnabled = rawClient?.mfa_enabled ?? client.mfa_config?.enabled ?? true; // Default to ON

  return (
    <div className="flex items-center gap-2">
      {mfaEnabled ? (
        <ShieldCheck className="w-4 h-4 text-green-600 dark:text-green-400" />
      ) : (
        <ShieldAlert className="w-4 h-4 text-red-600 dark:text-red-400" />
      )}
      <Badge
        variant={mfaEnabled ? "default" : "destructive"}
        className={
          mfaEnabled ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300" : ""
        }
      >
        {mfaEnabled ? "ON" : "OFF"}
      </Badge>
    </div>
  );
}

// Cell: Security Status (MFA + Policies)
export function SecurityStatusCell({ client }: { client: ClientWithAuthMethods }) {
  const rawClient =
    (client.metadata?.raw_client as ClientData | undefined) ?? (client as unknown as ClientData);
  const mfaEnabled = rawClient?.mfa_enabled ?? client.mfa_config?.enabled ?? false;
  const oidcEnabled = rawClient?.oidc_enabled ?? false;
  const policyCount = client.view_policies_applicable?.length ?? 0;

  return (
    <div className="flex flex-col gap-1 text-xs">
      <div className="flex items-center gap-2">
        {mfaEnabled ? (
          <ShieldCheck className="w-4 h-4 text-green-600 dark:text-green-400" />
        ) : (
          <ShieldAlert className="w-4 h-4 text-orange-600 dark:text-orange-400" />
        )}
        <span className="text-foreground">{mfaEnabled ? "MFA enabled" : "MFA disabled"}</span>
        <span className="text-foreground">• {policyCount} policies</span>
      </div>
      {!oidcEnabled && (
        <div className="flex items-center gap-2 text-red-600 dark:text-red-300">
          <ShieldOff className="w-3 h-3" />
          <span>OIDC not configured</span>
        </div>
      )}
    </div>
  );
}

// Cell: Status - Updated to handle actual API response
export function AccessLevelCell({ client }: { client: ClientWithAuthMethods }) {
  return <Badge variant="outline">{client.access_level}</Badge>;
}

// Cell: Total Users - Placeholder for future implementation
export function TotalUsersCell({ client }: { client: ClientWithAuthMethods }) {
  const rawClient =
    (client.metadata?.raw_client as ClientData | undefined) ?? (client as unknown as ClientData);
  const userCount =
    rawClient?.user_count ??
    (typeof client.metadata?.user_count === "number" ? client.metadata.user_count : undefined);

  if (typeof userCount !== "number") {
    return <span className="text-sm text-foreground">—</span>;
  }

  return <span className="text-sm font-medium text-foreground">{userCount.toLocaleString()}</span>;
}

function IdentifierLink({ value, label }: { value?: string | null; label: string }) {
  if (!value) {
    return <span className="text-[12px] text-muted-foreground">Not available</span>;
  }

  return (
    <div className="flex items-center gap-2 max-w-full min-w-0 group">
      <span className="truncate font-mono text-[12.5px] text-muted-foreground group-hover:text-foreground transition-colors leading-5">
        {value}
      </span>
      <CopyButton
        text={value}
        label={label}
        variant="ghost"
        size="sm"
        className="admin-icon-btn-subtle h-6 w-6 p-0 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
      />
    </div>
  );
}

export function ClientIdCell({ client }: { client: ClientWithAuthMethods }) {
  const rawClient =
    (client.metadata?.raw_client as ClientData | undefined) ?? (client as unknown as ClientData);
  const clientId = rawClient?.client_id || client.id;
  return <IdentifierLink value={clientId} label="Client ID" />;
}

export function TenantIdCell({ client }: { client: ClientWithAuthMethods }) {
  const rawClient =
    (client.metadata?.raw_client as ClientData | undefined) ?? (client as unknown as ClientData);
  const tenantId =
    rawClient?.tenant_id ||
    client.metadata?.tenant_id ||
    client.workspace_id ||
    client.workspace_id;
  return <IdentifierLink value={tenantId} label="Tenant ID" />;
}

// Shared footer for step popovers
function StepPopoverFooter({
  step,
  totalSteps,
  onNext,
}: {
  step: number;
  totalSteps: number;
  onNext: () => void;
}) {
  const isLast = step === totalSteps - 1;
  return (
    <div className="flex items-center justify-between pt-1">
      <span className="text-[10px] text-muted-foreground">{step + 1} of {totalSteps}</span>
      <Button
        size="sm"
        className="h-6 px-3 text-xs bg-foreground text-background hover:bg-foreground/85 border-0 shadow-none"
        onClick={onNext}
      >
        {isLast ? "Got it" : "Next →"}
      </Button>
    </div>
  );
}

// Cell: Actions - Ghost icon buttons + overflow menu
export function ActionsCell({
  client,
  actions,
}: {
  client: ClientWithAuthMethods;
  actions: ClientsTableActions;
}) {
  const rawClient =
    (client.metadata?.raw_client as ClientData | undefined) ?? (client as unknown as ClientData);
  const isActive =
    rawClient?.active !== undefined ? rawClient.active : client.access_status === "active";
  const clientId = rawClient?.client_id || client.id;
  const isNewClient = clientId === actions.newClientId;
  const step = isNewClient ? (actions.newClientStep ?? 0) : -1;

  const dismiss = () => actions.onDismissNewClient?.();
  const next = () => actions.onNextNewClientStep?.();

  const ringCls = [
    "ring-2 ring-blue-500",
    "ring-offset-2 ring-offset-background",
    "bg-blue-500/10",
    "shadow-[0_0_0_4px_rgba(59,130,246,0.18),0_0_14px_rgba(59,130,246,0.35)]",
    "rounded-md",
  ].join(" ");

  return (
    <div
      data-new-client={isNewClient ? "true" : undefined}
      className="flex items-center justify-end gap-1"
    >
      {/* Inline Action: View SDK — step 0 */}
      {actions.onViewSDK && (
        <Popover open={step === 0} onOpenChange={(open) => { if (!open) dismiss(); }}>
          <PopoverAnchor asChild>
            <span
              className={cn("inline-flex rounded-md", step === 0 && ringCls)}
              {...(isNewClient ? { "data-new-client-step": "0" } : {})}
            >
              <Button
                variant="ghost"
                size="sm"
                className="admin-row-icon-btn h-8 w-8 p-0"
                data-tone="sdk"
                onClick={() => { dismiss(); actions.onViewSDK?.(clientId); }}
                title="View SDK Hub"
                aria-label="View SDK Hub"
              >
                <Code className="h-4 w-4" />
              </Button>
            </span>
          </PopoverAnchor>
          <PopoverContent side="bottom" align="center" sideOffset={8} className="w-56 p-3 space-y-1.5 z-[56]" data-new-client-popover="true">
            <p className="text-xs font-semibold text-foreground flex items-center gap-1.5">
              <Code className="h-3.5 w-3.5" /> SDK Hub
            </p>
            <p className="text-xs text-muted-foreground">
              Connect your app by following the integration guide and copying code samples.
            </p>
            <StepPopoverFooter step={0} totalSteps={3} onNext={next} />
          </PopoverContent>
        </Popover>
      )}

      {/* Inline Action: Hosted Login — step 1 */}
      {actions.onPreviewLogin && (
        <Popover open={step === 1} onOpenChange={(open) => { if (!open) dismiss(); }}>
          <PopoverAnchor asChild>
            <span
              className={cn("inline-flex rounded-md", step === 1 && ringCls)}
              {...(isNewClient ? { "data-new-client-step": "1" } : {})}
            >
              <Button
                variant="ghost"
                size="sm"
                className="admin-row-icon-btn h-8 w-8 p-0"
                onClick={() => { dismiss(); actions.onPreviewLogin?.(clientId); }}
                title="Preview Login Page"
                aria-label="Preview login page"
              >
                <Eye className="h-4 w-4" />
              </Button>
            </span>
          </PopoverAnchor>
          <PopoverContent side="bottom" align="center" sideOffset={8} className="w-56 p-3 space-y-1.5 z-[56]" data-new-client-popover="true">
            <p className="text-xs font-semibold text-foreground flex items-center gap-1.5">
              <Eye className="h-3.5 w-3.5" /> Preview Login
            </p>
            <p className="text-xs text-muted-foreground">
              Open the hosted login page to see exactly what your end-users will experience.
            </p>
            <StepPopoverFooter step={1} totalSteps={3} onNext={next} />
          </PopoverContent>
        </Popover>
      )}

      {/* Overflow Menu — step 2 */}
      <Popover open={step === 2} onOpenChange={(open) => { if (!open) dismiss(); }}>
        <DropdownMenu>
          <PopoverAnchor asChild>
            <span
              className={cn("inline-flex rounded-md", step === 2 && ringCls)}
              {...(isNewClient ? { "data-new-client-step": "2" } : {})}
            >
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="admin-row-icon-btn h-8 w-8 p-0"
                  onClick={step === 2 ? dismiss : undefined}
                  title="More actions"
                  aria-label="More actions"
                >
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
            </span>
          </PopoverAnchor>
          <DropdownMenuContent
            align="end"
            visualVariant="row-actions" className="w-56 overflow-hidden p-1"
          >
            {/* Primary Actions Group */}
            {actions.onShowAuthMethods && (
              <DropdownMenuItem
                onClick={() => actions.onShowAuthMethods?.(client)}
                className="admin-menu-item-accent px-3 py-2 text-[14px] cursor-pointer transition-colors rounded-sm"
              >
                <Shield className="mr-2.5 h-4 w-4 text-muted-foreground" />
                Edit Auth Methods
              </DropdownMenuItem>
            )}

            {actions.onAddAuthMethod && (
              <DropdownMenuItem
                onClick={() => actions.onAddAuthMethod?.(clientId)}
                className="admin-menu-item-accent px-3 py-2 text-[14px] cursor-pointer transition-colors rounded-sm"
              >
                <Key className="mr-2.5 h-4 w-4 text-muted-foreground" />
                Create Auth Method
              </DropdownMenuItem>
            )}

            {actions.onConfigureVoiceAgent && (
              <DropdownMenuItem
                onClick={() => actions.onConfigureVoiceAgent?.(clientId)}
                className="admin-menu-item-voice px-3 py-2 text-[14px] cursor-pointer transition-colors rounded-sm"
              >
                <Mic className="mr-2.5 h-4 w-4" />
                Configure Voice Agent
              </DropdownMenuItem>
            )}

            <DropdownMenuSeparator className="bg-border my-1" />

            {/* Status Toggle (Risky Action) */}
            {actions.onToggleStatus &&
              (isActive ? (
                <DropdownMenuItem
                  onClick={() => actions.onToggleStatus?.(client.id)}
                  className="px-3 py-2 text-amber-600 dark:text-amber-400 text-[14px] hover:bg-amber-500/10 focus:bg-amber-500/10 cursor-pointer transition-colors rounded-sm"
                >
                  <PowerOff className="mr-2.5 h-4 w-4" />
                  Deactivate Client
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem
                  onClick={() => actions.onToggleStatus?.(client.id)}
                  className="px-3 py-2 text-emerald-600 dark:text-emerald-400 text-[14px] hover:bg-emerald-500/10 focus:bg-emerald-500/10 cursor-pointer transition-colors rounded-sm"
                >
                  <Power className="mr-2.5 h-4 w-4" />
                  Activate Client
                </DropdownMenuItem>
              ))}

            <DropdownMenuSeparator className="bg-border my-1" />

            {/* Destructive Action - Isolated */}
            <DropdownMenuItem
              onClick={() => actions.onDelete(client.id)}
              className="px-3 py-2 text-red-600 dark:text-red-400 text-[14px] hover:bg-red-500/10 focus:bg-red-500/10 cursor-pointer transition-colors rounded-sm"
            >
              <Trash2 className="mr-2.5 h-4 w-4" />
              Delete Client
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <PopoverContent side="bottom" align="end" sideOffset={8} className="w-56 p-3 space-y-1.5 z-[56]" data-new-client-popover="true">
          <p className="text-xs font-semibold text-foreground flex items-center gap-1.5">
            <MoreHorizontal className="h-3.5 w-3.5" /> More Actions
          </p>
          <p className="text-xs text-muted-foreground">
            Edit auth methods, configure voice agent, or manage client status from here.
          </p>
          <StepPopoverFooter step={2} totalSteps={3} onNext={dismiss} />
        </PopoverContent>
      </Popover>
    </div>
  );
}

// Expanded row with real client details and copy buttons
export function ClientExpandedRow({ client }: { client: ClientWithAuthMethods }) {
  const rawClient =
    (client.metadata?.raw_client as ClientData | undefined) ?? (client as unknown as ClientData);
  const clientId = rawClient.client_id || client.id;
  const orgId = rawClient.org_id || client.metadata?.org_id;
  const hydraClientId = rawClient.hydra_client_id || client.metadata?.hydra_client_id;
  const userCount =
    rawClient.user_count ??
    (typeof client.metadata?.user_count === "number" ? client.metadata.user_count : undefined);

  const normalizedAuthMethods = (() => {
    if (
      Array.isArray(rawClient?.authentication_methods) &&
      rawClient.authentication_methods.length
    ) {
      return rawClient.authentication_methods.map((method, index) => {
        if (typeof method === "string") {
          return { id: `auth-${index}`, name: method };
        }
        return {
          id: method.id || `auth-${index}`,
          name: method.name || method.type || "Auth Method",
        };
      });
    }

    if (Array.isArray(client.attachedMethods) && client.attachedMethods.length) {
      return client.attachedMethods.map((method, index) => {
        if (typeof method === "string") {
          return { id: `auth-${index}`, name: method };
        }
        return { id: method.id || `auth-${index}`, name: method.name };
      });
    }

    return [];
  })();

  const InfoLine = ({
    label,
    value,
    copyable = false,
  }: {
    label: string;
    value?: string | number | null;
    copyable?: boolean;
  }) => {
    if (value === undefined || value === null || value === "") {
      return null;
    }

    const handleCopy = () => {
      navigator.clipboard.writeText(String(value));
    };

    return (
      <div className="flex items-center justify-between gap-3 group">
        <span className="text-[12px] font-medium text-muted-foreground">{label}</span>
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-mono text-[12px] truncate text-foreground" title={String(value)}>
            {String(value)}
          </span>
          {copyable && (
            <Copy
              className="h-3.5 w-3.5 cursor-pointer text-muted-foreground hover:text-primary transition-colors flex-shrink-0 opacity-0 group-hover:opacity-100"
              onClick={handleCopy}
            />
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="min-w-0">
      <div className="grid gap-5 md:grid-cols-[minmax(0,1fr)_1px_minmax(0,1fr)]">
        <div className="min-w-0 space-y-3.5">
          <h4 className="flex items-center gap-2 text-[14px] font-semibold text-foreground">
            <Key className="h-4 w-4 text-primary" />
            Client Details
          </h4>
          <div className="space-y-2.5 text-sm">
            <InfoLine label="Client Name" value={client.name || "Unnamed client"} />
            <InfoLine label="Client ID" value={clientId} copyable />
            <InfoLine label="Org ID" value={orgId} copyable />
            <InfoLine label="Hydra Client ID" value={hydraClientId} copyable />
          </div>
        </div>

        {/* Vertical Separator */}
        <div data-slot="table-expanded-divider" className="hidden md:block w-px" />

        <div className="min-w-0 space-y-3.5">
          <h4 className="flex items-center gap-2 text-[14px] font-semibold text-foreground">
            <Globe className="h-4 w-4 text-primary" />
            Usage
          </h4>
          <div className="space-y-3.5 text-sm">
            <div>
              <span className="text-[12px] font-medium uppercase tracking-wide text-muted-foreground">
                Auth Methods
              </span>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {normalizedAuthMethods.length ? (
                  normalizedAuthMethods.map((method) => (
                    <span
                      key={method.id}
                      data-slot="table-expanded-pill"
                      className="inline-flex items-center h-6 px-2.5 rounded-sm text-[12px] font-medium text-foreground"
                    >
                      {method.name}
                    </span>
                  ))
                ) : (
                  <span className="text-[13px] text-muted-foreground">No methods attached</span>
                )}
              </div>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-[12px] font-medium uppercase tracking-wide text-muted-foreground">
                Total Users
              </span>
              <span className="text-[16px] font-semibold text-foreground">
                {typeof userCount === "number" ? userCount.toLocaleString() : "—"}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
// Column factory
// Create columns for AdaptiveTable (new pattern matching users table)
export function createAdaptiveClientsTableColumns(
  actions: ClientsTableActions,
): AdaptiveColumn<ClientWithAuthMethods>[] {
  return [
    {
      id: "name",
      header: "Client",
      accessorKey: "name",
      alwaysVisible: true,
      enableSorting: true,
      resizable: true,
      approxWidth: 280,
      cell: ({ row }) => <ClientNameCell client={row.original} />,
    },
    {
      id: "clientId",
      header: "Client ID",
      accessorKey: "id",
      priority: 1,
      enableSorting: true,
      resizable: true,
      approxWidth: 260,
      cell: ({ row }) => <ClientIdCell client={row.original} />,
    },
    {
      id: "status",
      header: "Status",
      accessorKey: "access_status",
      priority: 2,
      enableSorting: true,
      resizable: true,
      approxWidth: 140,
      cell: ({ row }) => <ActiveStatusCell client={row.original} />,
    },
    {
      id: "authMethods",
      header: "Auth Methods",
      accessorKey: "attachedMethods",
      priority: 3,
      enableSorting: false,
      resizable: true,
      approxWidth: 200,
      cell: ({ row }) => <AuthMethodsCell client={row.original} />,
    },
    {
      id: "totalUsers",
      header: "Total Users",
      accessorKey: "total_requests",
      priority: 4,
      enableSorting: true,
      resizable: true,
      approxWidth: 130,
      cell: ({ row }) => <TotalUsersCell client={row.original} />,
    },
    {
      id: "actions",
      header: "Actions",
      alwaysVisible: true,
      enableSorting: false,
      resizable: false,
      size: 80,
      className: "w-[80px] text-right",
      cellClassName: "text-right",
      approxWidth: 100,
      cell: ({ row }) => <ActionsCell client={row.original} actions={actions} />,
    },
  ];
}

// Legacy columns for ResponsiveDataTable (kept for backwards compatibility)
export function createClientsTableColumns(
  actions: ClientsTableActions,
): ResponsiveColumnDef<ClientWithAuthMethods, unknown>[] {
  return [
    {
      id: "client",
      accessorKey: "name",
      header: "Client",
      resizable: true,
      responsive: true,
      cell: ({ row }) => <ClientNameCell client={row.original} />,
      size: 300,
      minSize: 200,
      maxSize: 400,
      cellClassName: "max-w-0",
    },
    {
      id: "clientId",
      accessorKey: "id", // Use id instead of client_id since that's what exists in ClientWithAuthMethods
      header: "Client ID",
      resizable: true,
      responsive: true,
      cell: ({ row }) => <ClientIdCell client={row.original} />,
      minSize: 200,
      maxSize: 420,
    },
    {
      id: "status",
      accessorKey: "access_status",
      header: "Status",
      resizable: true,
      responsive: true,
      cell: ({ row }) => <ActiveStatusCell client={row.original} />,
      minSize: 120,
      maxSize: 150,
    },
    {
      id: "authMethods",
      accessorKey: "attachedMethods",
      header: "Auth Methods",
      resizable: true,
      responsive: true,
      cell: ({ row }) => <AuthMethodsCell client={row.original} />,
      minSize: 150,
      maxSize: 300,
    },
    {
      id: "totalUsers",
      accessorKey: "total_requests",
      header: "Total Users",
      resizable: true,
      responsive: true,
      cell: ({ row }) => <TotalUsersCell client={row.original} />,
      minSize: 110,
      maxSize: 130,
    },
    {
      id: "actions",
      header: "Actions",
      resizable: false,
      responsive: false,
      enableSorting: false,
      cell: ({ row }) => <ActionsCell client={row.original} actions={actions} />,
      minSize: 80,
      maxSize: 100,
      cellClassName: "text-center",
    },
  ];
}

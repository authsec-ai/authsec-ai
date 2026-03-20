import React from "react";
import { CopyButton } from "@/components/ui/copy-button";
import { Badge } from "@/components/ui/badge";
import type { ApiOAuthScope, ApiOAuthScopeDetails } from "../types";

// Cell component for ID column
export const IdCell = ({ scope }: { scope: ApiOAuthScope }) => (
  <div className="flex items-center gap-2">
    <code
      className="text-xs font-mono text-foreground bg-muted px-2 py-1 rounded truncate max-w-[150px]"
      title={scope.id}
    >
      {scope.id}
    </code>
    <CopyButton text={scope.id} label="ID" size="sm" />
  </div>
);

// Cell component for Name column
export const NameCell = ({ scope }: { scope: ApiOAuthScope }) => (
  <p
    className="truncate text-sm font-medium text-foreground"
    title={scope.name}
  >
    {scope.name}
  </p>
);

// Cell component for Description column
export const DescriptionCell = ({ scope }: { scope: ApiOAuthScope }) => (
  <p
    className="text-sm text-foreground truncate"
    title={scope.description}
  >
    {scope.description || "—"}
  </p>
);

// Cell component for Permissions Linked column
export const PermissionsLinkedCell = ({ scope }: { scope: ApiOAuthScope }) => (
  <Badge variant="secondary" className="font-medium">
    {scope.permissions_linked} {scope.permissions_linked === 1 ? 'permission' : 'permissions'}
  </Badge>
);

// Cell component for Created At column
export const CreatedAtCell = ({ scope }: { scope: ApiOAuthScope }) => (
  <span className="text-sm text-foreground">
    {scope.created_at ? new Date(scope.created_at).toLocaleDateString() : "—"}
  </span>
);

// Expanded row component for mobile/detail view
export const ApiOAuthScopeExpandedRow = ({
  scope,
  scopeDetails,
}: {
  scope: ApiOAuthScope;
  scopeDetails?: ApiOAuthScopeDetails;
}) => {
  const InfoLine = ({
    label,
    value,
    copyable = false,
  }: {
    label: string;
    value?: string | null | number;
    copyable?: boolean;
  }) => {
    if (value === undefined || value === null || value === "") return null;
    return (
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs font-medium text-foreground">
          {label}
        </span>
        <div className="flex items-center gap-2 min-w-0">
          <span
            className="font-mono text-xs truncate text-foreground"
            title={String(value)}
          >
            {String(value)}
          </span>
          {copyable && <CopyButton text={String(value)} label={label} size="sm" />}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-5">
      <div className="grid gap-6 md:grid-cols-2">
        {/* Left Column: Identifiers */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            Identifiers
          </h4>
          <div className="space-y-3 text-sm">
            <InfoLine label="Scope ID" value={scope.id} copyable />
            <InfoLine label="Scope Name" value={scope.name} copyable />
            <InfoLine label="Permissions Linked" value={scope.permissions_linked} />
          </div>
        </div>

        {/* Right Column: Details */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            Details
          </h4>
          <div className="space-y-3 text-sm">
            <InfoLine label="Description" value={scope.description} />
            {scope.created_at && (
              <InfoLine
                label="Created"
                value={new Date(scope.created_at).toLocaleString()}
              />
            )}
          </div>
        </div>
      </div>

      {/* Permission Strings Section */}
      {scopeDetails?.permission_strings && scopeDetails.permission_strings.length > 0 && (
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            Mapped Permissions
          </h4>
          <div className="flex flex-wrap gap-2">
            {scopeDetails.permission_strings.map((permString, index) => (
              <Badge key={index} variant="outline" className="font-mono text-xs">
                {permString}
              </Badge>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

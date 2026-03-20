import type {
  DelegationPolicyUI,
  DurationParts,
  PermissionOption,
} from "./types";
import { getErrorMessage } from "@/lib/error-utils";

const SENSITIVE_PERMISSION_MARKERS = [
  "credentials",
  "payments",
  "vault",
  "admin",
];

export function humanizePermissionKey(permission: string) {
  if (!permission) return "Unknown permission";

  const [resourcePart, actionPart = "access"] = permission.split(":");
  const resourceLabel = resourcePart
    .replace(/[-_]/g, " ")
    .replace(/\b\w/g, (char) => char.toUpperCase());
  const actionLabel = actionPart
    .replace(/[-_]/g, " ")
    .replace(/\b\w/g, (char) => char.toUpperCase());

  return `${resourceLabel} ${actionLabel}`.trim();
}

export function groupPermissionOptions(permissions: string[]): PermissionOption[] {
  return permissions
    .filter(Boolean)
    .map((permission) => {
      const [resourcePart] = permission.split(":");
      const group = resourcePart
        .replace(/[-_]/g, " ")
        .replace(/\b\w/g, (char) => char.toUpperCase());

      const sensitive = SENSITIVE_PERMISSION_MARKERS.some((marker) =>
        permission.toLowerCase().includes(marker),
      );

      return {
        key: permission,
        label: humanizePermissionKey(permission),
        group,
        description: permission,
        sensitive,
      };
    })
    .sort((left, right) => left.label.localeCompare(right.label));
}

export function formatDurationLabel(totalSeconds: number) {
  if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) return "0 minutes";
  if (totalSeconds % 86400 === 0) {
    const days = totalSeconds / 86400;
    return `${days} day${days === 1 ? "" : "s"}`;
  }
  if (totalSeconds % 3600 === 0) {
    const hours = totalSeconds / 3600;
    return `${hours} hour${hours === 1 ? "" : "s"}`;
  }
  const minutes = Math.max(1, Math.round(totalSeconds / 60));
  return `${minutes} minute${minutes === 1 ? "" : "s"}`;
}

export function secondsToDurationParts(totalSeconds: number): DurationParts {
  if (totalSeconds % 86400 === 0) {
    return {
      value: totalSeconds / 86400,
      unit: "days",
    };
  }

  if (totalSeconds % 3600 === 0) {
    return {
      value: totalSeconds / 3600,
      unit: "hours",
    };
  }

  return {
    value: Math.max(1, Math.round(totalSeconds / 60)),
    unit: "minutes",
  };
}

export function durationPartsToSeconds(parts: DurationParts) {
  switch (parts.unit) {
    case "days":
      return parts.value * 86400;
    case "hours":
      return parts.value * 3600;
    case "minutes":
    default:
      return parts.value * 60;
  }
}

export function getPolicyStatus(policy: DelegationPolicyUI) {
  return policy.enabled ? "enabled" : "disabled";
}

export function buildTrustDelegationPath(
  pathname: string,
  params?: Record<string, string | undefined>,
) {
  const url = new URL(pathname, "https://authsec.local");
  Object.entries(params || {}).forEach(([key, value]) => {
    if (value) {
      url.searchParams.set(key, value);
    }
  });
  return `${url.pathname}${url.search}`;
}

export function getTrustDelegationErrorMessage(
  error: unknown,
  fallback: string,
) {
  const errorRecord =
    error && typeof error === "object"
      ? (error as {
          status?: unknown;
          originalStatus?: unknown;
          data?: unknown;
          error?: unknown;
        })
      : null;

  const rawBody =
    typeof errorRecord?.data === "string" ? errorRecord.data.toLowerCase() : "";
  const isParsingError = errorRecord?.status === "PARSING_ERROR";
  const isMissingEndpoint =
    errorRecord?.originalStatus === 404 ||
    rawBody.includes("404 page not found");

  if (isMissingEndpoint) {
    return "Trust delegation data is unavailable because the backend endpoint is not responding.";
  }

  if (isParsingError) {
    return "Trust delegation data could not be read from the backend response.";
  }

  return getErrorMessage(error, fallback);
}

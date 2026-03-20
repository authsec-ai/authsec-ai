export const SDK_HUB_MODULES = [
  "overview",
  "mcp-oauth",
  "rbac",
  "service-access",
  "ciba",
  "spiffe",
  "env",
] as const;

export type SDKHubModule = (typeof SDK_HUB_MODULES)[number];

const LEGACY_DOC_MODULE_MAP: Record<string, SDKHubModule> = {
  authentication: "mcp-oauth",
  permissions: "rbac",
  resources: "rbac",
  roles: "rbac",
  scopes: "rbac",
  "role-bindings": "rbac",
  "oauth-api-scopes": "rbac",
  "voice-agent": "ciba",
  "external-services": "service-access",
};

export function isSDKHubModule(value: string | null | undefined): value is SDKHubModule {
  if (!value) return false;
  return (SDK_HUB_MODULES as readonly string[]).includes(value);
}

export function inferHubModuleFromTitle(title: string | undefined): SDKHubModule {
  const normalized = (title ?? "").toLowerCase();

  if (
    normalized.includes("permission") ||
    normalized.includes("resource") ||
    normalized.includes("role") ||
    normalized.includes("scope") ||
    normalized.includes("rbac") ||
    normalized.includes("binding")
  ) {
    return "rbac";
  }

  if (
    normalized.includes("service") ||
    normalized.includes("secret") ||
    normalized.includes("credential")
  ) {
    return "service-access";
  }

  if (
    normalized.includes("voice") ||
    normalized.includes("ciba") ||
    normalized.includes("totp")
  ) {
    return "ciba";
  }

  if (
    normalized.includes("workload") ||
    normalized.includes("spiffe") ||
    normalized.includes("spire")
  ) {
    return "spiffe";
  }

  if (
    normalized.includes("auth") ||
    normalized.includes("oauth") ||
    normalized.includes("client")
  ) {
    return "mcp-oauth";
  }

  return "overview";
}

export function inferHubModuleFromSurface(surface: string | undefined): SDKHubModule {
  if (!surface) return "overview";

  const normalized = surface.toLowerCase();
  if (normalized === "clients") return "mcp-oauth";
  if (normalized === "external-services") return "service-access";
  if (normalized === "rbac") return "rbac";
  if (normalized === "voice-agent") return "ciba";
  if (normalized === "workloads") return "spiffe";

  return "overview";
}

export function getModuleFromLegacyDocsLink(
  docsLink: string | undefined,
): SDKHubModule | null {
  if (!docsLink?.startsWith("/docs/sdk/")) {
    return null;
  }

  const slug = docsLink
    .replace("/docs/sdk/", "")
    .split(/[?#]/)[0]
    .split("/")
    .filter(Boolean)[0];

  if (!slug) {
    return "overview";
  }

  return LEGACY_DOC_MODULE_MAP[slug] ?? "overview";
}

interface BuildSDKHubLinkOptions {
  module?: SDKHubModule;
  surface?: string;
  entityId?: string;
}

export function buildSDKHubLink({
  module = "overview",
  surface,
  entityId,
}: BuildSDKHubLinkOptions): string {
  const basePath = surface
    ? `/sdk/${surface}${entityId ? `/${encodeURIComponent(entityId)}` : ""}`
    : "/sdk";

  if (module === "overview") {
    return basePath;
  }

  return `${basePath}?module=${module}`;
}

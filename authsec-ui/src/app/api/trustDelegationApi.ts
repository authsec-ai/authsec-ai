import { baseApi } from "./baseApi";
import type { DelegationPolicyUI } from "@/features/trust-delegation/types";
import {
  formatDurationLabel,
} from "@/features/trust-delegation/utils";

type UnknownRecord = Record<string, unknown>;

export interface DelegationPolicyRecord {
  id: string;
  tenant_id?: string;
  role_name: string;
  agent_type: string;
  allowed_permissions: string[];
  max_ttl_seconds: number;
  enabled: boolean;
  client_id: string;
  created_by?: string;
  client_label?: string;
}

export interface CreateDelegationPolicyRequest {
  role_name: string;
  agent_type: string;
  allowed_permissions: string[];
  max_ttl_seconds: number;
  enabled: boolean;
  client_id: string;
}

export interface ListDelegationPoliciesParams {
  role_name?: string;
  agent_type?: string;
  enabled?: boolean;
  client_id?: string;
}

function asArray<T>(value: unknown, key?: string) {
  if (Array.isArray(value)) return value as T[];
  if (value && typeof value === "object" && key && Array.isArray((value as UnknownRecord)[key])) {
    return (value as UnknownRecord)[key] as T[];
  }
  return [] as T[];
}

function isRecord(value: unknown): value is UnknownRecord {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function parseFirstValidDelegationJSON<T>(response: unknown): T {
  if (isRecord(response) || Array.isArray(response)) {
    return response as T;
  }

  if (typeof response !== "string") {
    return response as T;
  }

  const normalized = response.trim();
  if (!normalized) {
    throw new Error("Server returned an unreadable response.");
  }

  const jsonBoundaryPattern = /\}\s*\{/g;
  const matches = normalized.match(jsonBoundaryPattern);

  if (matches && matches.length > 0) {
    const firstJsonEnd = normalized.indexOf(matches[0]) + 1;
    const firstJsonStr = normalized.slice(0, firstJsonEnd);

    try {
      return JSON.parse(firstJsonStr) as T;
    } catch {
      throw new Error("Server returned an unreadable response.");
    }
  }

  try {
    return JSON.parse(normalized) as T;
  } catch {
    throw new Error("Server returned an unreadable response.");
  }
}

function extractList<T>(response: unknown, keys: string[]) {
  if (Array.isArray(response)) {
    return response as T[];
  }

  for (const key of keys) {
    const items = asArray<T>(response, key);
    if (items.length > 0) {
      return items;
    }
  }

  return [] as T[];
}

function buildQueryString(params: Record<string, unknown>) {
  const searchParams = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") return;
    searchParams.set(key, String(value));
  });
  const serialized = searchParams.toString();
  return serialized ? `?${serialized}` : "";
}

function toClientLabel(record: DelegationPolicyRecord) {
  if (typeof record.client_label === "string" && record.client_label) {
    return record.client_label;
  }
  return record.client_id || "Unknown client";
}

export function normalizeDelegationPolicy(record: DelegationPolicyRecord): DelegationPolicyUI {
  const clientLabel = toClientLabel(record);

  return {
    id: record.id,
    roleName: record.role_name,
    agentType: record.agent_type,
    allowedPermissions: Array.isArray(record.allowed_permissions)
      ? record.allowed_permissions
      : [],
    maxTtlSeconds: record.max_ttl_seconds,
    maxTtlLabel: formatDurationLabel(record.max_ttl_seconds),
    enabled: Boolean(record.enabled),
    clientId: record.client_id,
    clientLabel,
    tenantId: record.tenant_id,
    createdBy: record.created_by,
  };
}

function addPermissionKey(accumulator: Set<string>, candidate: unknown) {
  if (typeof candidate !== "string") return;
  const permission = candidate.trim();
  if (!permission || !permission.includes(":")) return;
  accumulator.add(permission);
}

function collectPermissionKeys(input: unknown, accumulator: Set<string>) {
  if (Array.isArray(input)) {
    input.forEach((item) => collectPermissionKeys(item, accumulator));
    return;
  }

  if (!input || typeof input !== "object") return;

  const record = input as UnknownRecord;

  addPermissionKey(accumulator, record.full_permission_string);
  addPermissionKey(accumulator, record.permission_string);
  addPermissionKey(accumulator, record.permission);
  addPermissionKey(accumulator, record.key);
  addPermissionKey(accumulator, record.name);

  const nestedKeys = [
    "permissions",
    "allowed_permissions",
    "granted_permissions",
    "effective_permissions",
    "roles_permissions",
    "roles",
    "data",
    "items",
  ] as const;

  nestedKeys.forEach((key) => {
    if (key in record) {
      collectPermissionKeys(record[key], accumulator);
    }
  });
}

export function normalizeDelegationPermissionCatalog(response: unknown): string[] {
  const permissions = new Set<string>();
  collectPermissionKeys(response, permissions);
  return Array.from(permissions).sort((left, right) => left.localeCompare(right));
}

export const trustDelegationApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getDelegationPermissionCatalog: builder.query<string[], void>({
      query: () => "authsec/uflow/admin/me/roles-permissions",
      transformResponse: (response: unknown) =>
        normalizeDelegationPermissionCatalog(response),
    }),

    listDelegationPolicies: builder.query<DelegationPolicyUI[], ListDelegationPoliciesParams | void>({
      query: (params) => ({
        url: `authsec/uflow/delegation-policies${buildQueryString(params || {})}`,
        responseHandler: "text",
      }),
      transformResponse: (response: unknown) => {
        const parsed = parseFirstValidDelegationJSON<
          DelegationPolicyRecord[] | { data?: DelegationPolicyRecord[]; policies?: DelegationPolicyRecord[] }
        >(response);
        return extractList<DelegationPolicyRecord>(parsed, ["policies", "data"]).map(
          normalizeDelegationPolicy,
        );
      },
      providesTags: (result) =>
        result?.length
          ? [
              ...result.map((policy) => ({ type: "DelegationPolicy" as const, id: policy.id })),
              { type: "DelegationPolicy" as const, id: "LIST" },
            ]
          : [{ type: "DelegationPolicy" as const, id: "LIST" }],
    }),

    getDelegationPolicy: builder.query<DelegationPolicyUI, string>({
      query: (id) => ({
        url: `authsec/uflow/delegation-policies/${id}`,
        responseHandler: "text",
      }),
      transformResponse: (response: unknown) => {
        const parsed = parseFirstValidDelegationJSON<
          DelegationPolicyRecord | { data?: DelegationPolicyRecord; policy?: DelegationPolicyRecord }
        >(response);
        return normalizeDelegationPolicy(
          (parsed as { policy?: DelegationPolicyRecord }).policy ||
            (parsed as { data?: DelegationPolicyRecord }).data ||
            (parsed as DelegationPolicyRecord),
        );
      },
      providesTags: (_result, _error, id) => [{ type: "DelegationPolicy", id }],
    }),

    createDelegationPolicy: builder.mutation<DelegationPolicyUI, CreateDelegationPolicyRequest>({
      query: (body) => ({
        url: "authsec/uflow/delegation-policies",
        method: "POST",
        body,
      }),
      transformResponse: (response: DelegationPolicyRecord) =>
        normalizeDelegationPolicy(response),
      invalidatesTags: [{ type: "DelegationPolicy", id: "LIST" }],
    }),

    updateDelegationPolicy: builder.mutation<
      DelegationPolicyUI,
      { id: string; body: CreateDelegationPolicyRequest }
    >({
      query: ({ id, body }) => ({
        url: `authsec/uflow/delegation-policies/${id}`,
        method: "PUT",
        body,
      }),
      transformResponse: (response: DelegationPolicyRecord) =>
        normalizeDelegationPolicy(response),
      invalidatesTags: (_result, _error, { id }) => [
        { type: "DelegationPolicy", id },
        { type: "DelegationPolicy", id: "LIST" },
      ],
    }),

    deleteDelegationPolicy: builder.mutation<{ message?: string }, string>({
      query: (id) => ({
        url: `authsec/uflow/delegation-policies/${id}`,
        method: "DELETE",
      }),
      invalidatesTags: (_result, _error, id) => [
        { type: "DelegationPolicy", id },
        { type: "DelegationPolicy", id: "LIST" },
      ],
    }),
  }),
  overrideExisting: false,
});

export const {
  useGetDelegationPermissionCatalogQuery,
  useListDelegationPoliciesQuery,
  useGetDelegationPolicyQuery,
  useCreateDelegationPolicyMutation,
  useUpdateDelegationPolicyMutation,
  useDeleteDelegationPolicyMutation,
} = trustDelegationApi;

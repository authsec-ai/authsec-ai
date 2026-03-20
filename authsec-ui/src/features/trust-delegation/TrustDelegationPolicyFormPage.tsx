import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  CheckCircle,
  ChevronRight,
  Clock3,
  Settings,
  ShieldCheck,
} from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SearchableSelect } from "@/components/ui/searchable-select";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { getErrorMessage } from "@/lib/error-utils";
import { toast } from "@/lib/toast";
import { resolveTenantId } from "@/utils/workspace";
import { useGetAdminUsersQuery } from "@/app/api/admin/usersApi";
import { useGetAllClientsQuery } from "@/app/api/clientApi";
import { useGetAuthSecRolesQuery } from "@/app/api/rolesApi";
import {
  useCreateDelegationPolicyMutation,
  useGetDelegationPermissionCatalogQuery,
  useGetDelegationPolicyQuery,
  useListDelegationPoliciesQuery,
  useUpdateDelegationPolicyMutation,
} from "@/app/api/trustDelegationApi";
import {
  TrustDelegationFieldMeta,
} from "./components/TrustDelegationFieldMeta";
import { TrustDelegationWizardShell } from "./components/TrustDelegationWizardShell";
import { renderTrustDelegationSingleSelectOption } from "./components/trustDelegationSelectRenderers";
import {
  durationPartsToSeconds,
  formatDurationLabel,
  groupPermissionOptions,
  humanizePermissionKey,
} from "./utils";

const AGENT_TYPE_OPTIONS = [
  {
    value: "mcp-agent",
    label: "AI agent / MCP server",
    description: "Trust delegation for MCP servers and agent-driven client actions.",
  },
  {
    value: "autonomous-workload",
    label: "Autonomous workload",
    description: "Trust delegation for workload identities acting without direct human interaction.",
  },
  {
    value: "user-service-account",
    label: "User / service account",
    description: "Trust delegation for another human-backed or service-backed identity.",
  },
];

const WIZARD_STEPS = [
  {
    id: "context",
    label: "Context",
    icon: Settings,
    headerSubtitle:
      "Define the role, target type, application boundary, and planned users for this trust delegation.",
    sectionTitle: "Trust delegation context",
    sectionDescription:
      "Each trust delegation is matched by role, target type, and client before it can be used.",
  },
  {
    id: "actions",
    label: "Allowed Actions",
    icon: ShieldCheck,
    headerSubtitle:
      "Choose the maximum action set that this trust delegation is allowed to grant.",
    sectionTitle: "Allowed actions",
    sectionDescription:
      "Permissions come from the current admin roles-and-permissions catalog and are stored as raw permission keys.",
  },
  {
    id: "duration",
    label: "Duration & Lifecycle",
    icon: Clock3,
    headerSubtitle:
      "Define the maximum duration and whether this trust delegation can be used for new issuance.",
    sectionTitle: "Duration and lifecycle",
    sectionDescription:
      "Keep maximum durations short for higher-risk delegated access.",
  },
  {
    id: "review",
    label: "Review & Save",
    icon: CheckCircle,
    headerSubtitle:
      "Review the final trust delegation configuration before saving it.",
    sectionTitle: "Review and save",
    sectionDescription:
      "Confirm the scope, maximum duration, and status before you save this trust delegation.",
  },
] as const;

const policySchema = z.object({
  roleName: z.string().min(1, "Role is required"),
  agentType: z.string().min(1, "Target type is required"),
  clientId: z.string().min(1, "Client is required"),
  userIds: z.array(z.string()),
  allowedPermissions: z.array(z.string()).min(1, "Select at least one action"),
  durationValue: z.coerce.number().int().positive("Duration must be greater than zero"),
  durationUnit: z.enum(["minutes", "hours", "days"]),
  enabled: z.boolean(),
});

type PolicyFormValues = z.infer<typeof policySchema>;

const STEP_FIELDS: Record<number, Array<keyof PolicyFormValues>> = {
  0: ["roleName", "agentType", "clientId", "userIds"],
  1: ["allowedPermissions"],
  2: ["durationValue", "durationUnit"],
  3: [],
};

function mapAdminUsers(usersResponse: unknown) {
  if (!usersResponse || typeof usersResponse !== "object") return [];

  const response = usersResponse as {
    users?: Array<Record<string, unknown>>;
  };

  return (response.users || []).map((user) => {
    const normalizedName =
      typeof user.name === "string" && user.name.trim()
        ? user.name.trim()
        : undefined;
    const normalizedUsername =
      typeof user.username === "string" && user.username.trim()
        ? user.username.trim()
        : undefined;
    const normalizedEmail =
      typeof user.email === "string" && user.email.trim()
        ? user.email.trim()
        : undefined;
    const label =
      normalizedName || normalizedEmail || normalizedUsername || "Unknown user";
    const roleLabel = Array.isArray(user.roles)
      ? (user.roles as Array<Record<string, unknown>>)
          .map((role) =>
            typeof role.name === "string" && role.name.trim()
              ? role.name.trim()
              : undefined,
          )
          .filter((roleName): roleName is string => Boolean(roleName))
          .join(", ")
      : "";
    const description = [
      normalizedEmail && normalizedEmail !== label ? normalizedEmail : undefined,
      roleLabel,
    ]
      .filter(Boolean)
      .join(" · ");

    return {
      value: String(user.id || user.user_id || user.email || ""),
      label,
      description: description || undefined,
    };
  });
}

export function TrustDelegationPolicyFormPage() {
  const navigate = useNavigate();
  const tenantId = resolveTenantId();
  const { policyId } = useParams();
  const isEditMode = Boolean(policyId);
  const [searchParams] = useSearchParams();
  const [currentStepIndex, setCurrentStepIndex] = useState(0);

  const { data: roles = [] } = useGetAuthSecRolesQuery(
    { tenant_id: tenantId || "", audience: "admin" },
    { skip: !tenantId },
  );
  const { data: adminUsersResponse } = useGetAdminUsersQuery(
    { page: 1, limit: 100, tenant_id: tenantId || "" },
    { skip: !tenantId },
  );
  const { data: clientsData } = useGetAllClientsQuery(
    { tenant_id: tenantId || "", active_only: false },
    { skip: !tenantId },
  );
  const {
    data: permissionCatalog = [],
    error: permissionCatalogError,
  } = useGetDelegationPermissionCatalogQuery();
  const {
    data: policy,
    isLoading: isLoadingPolicy,
  } = useGetDelegationPolicyQuery(policyId || "", {
    skip: !policyId,
  });
  const { data: existingPolicies = [] } = useListDelegationPoliciesQuery();
  const [createPolicy, { isLoading: isCreating }] =
    useCreateDelegationPolicyMutation();
  const [updatePolicy, { isLoading: isUpdating }] =
    useUpdateDelegationPolicyMutation();

  const form = useForm<PolicyFormValues>({
    resolver: zodResolver(policySchema),
    defaultValues: {
      roleName: searchParams.get("roleName") || "",
      agentType: searchParams.get("agentType") || "",
      clientId: searchParams.get("clientId") || "",
      userIds: (searchParams.get("userIds") || "")
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean),
      allowedPermissions: [],
      durationValue: 1,
      durationUnit: "hours",
      enabled: true,
    },
  });

  useEffect(() => {
    if (!policy) return;

    form.reset({
      roleName: policy.roleName,
      agentType: policy.agentType,
      clientId: policy.clientId,
      userIds: [],
      allowedPermissions: policy.allowedPermissions,
      durationValue:
        policy.maxTtlSeconds % 86400 === 0
          ? policy.maxTtlSeconds / 86400
          : policy.maxTtlSeconds % 3600 === 0
            ? policy.maxTtlSeconds / 3600
            : Math.max(1, Math.round(policy.maxTtlSeconds / 60)),
      durationUnit:
        policy.maxTtlSeconds % 86400 === 0
          ? "days"
          : policy.maxTtlSeconds % 3600 === 0
            ? "hours"
            : "minutes",
      enabled: policy.enabled,
    });
  }, [form, policy]);

  const values = form.watch();
  const currentStep = WIZARD_STEPS[currentStepIndex];
  const showLoadingState = isEditMode && isLoadingPolicy && !policy;
  const totalSeconds = durationPartsToSeconds({
    value: values.durationValue || 0,
    unit: values.durationUnit || "hours",
  });

  const roleOptions = useMemo(
    () =>
      roles.map((role) => ({
        value: role.name,
        label: role.name,
        description: role.description || undefined,
      })),
    [roles],
  );

  const clientOptions = useMemo(
    () =>
      (clientsData?.clients || []).map((client) => ({
        value: client.client_id,
        label: client.name || client.client_name || client.client_id,
        description: client.description || undefined,
      })),
    [clientsData],
  );
  const userOptions = useMemo(
    () => mapAdminUsers(adminUsersResponse),
    [adminUsersResponse],
  );

  const permissionOptions = useMemo(
    () =>
      groupPermissionOptions(permissionCatalog).map((permission) => ({
        value: permission.key,
        label: permission.label,
        description: permission.description,
        group: permission.group,
      })),
    [permissionCatalog],
  );

  const duplicatePolicy = useMemo(
    () =>
      existingPolicies.find(
        (existingPolicy) =>
          existingPolicy.id !== policyId &&
          existingPolicy.roleName === values.roleName &&
          existingPolicy.agentType === values.agentType &&
          existingPolicy.clientId === values.clientId,
      ),
    [existingPolicies, policyId, values.agentType, values.clientId, values.roleName],
  );

  const selectedClient = clientOptions.find(
    (client) => client.value === values.clientId,
  );
  const selectedUsers = userOptions.filter((user) =>
    values.userIds.includes(user.value),
  );
  const selectedAgentType = AGENT_TYPE_OPTIONS.find(
    (option) => option.value === values.agentType,
  );

  const canProceed = useMemo(() => {
    if (showLoadingState) {
      return false;
    }

    if (currentStepIndex === 0) {
      return Boolean(values.roleName && values.agentType && values.clientId);
    }

    if (currentStepIndex === 1) {
      return values.allowedPermissions.length > 0;
    }

    if (currentStepIndex === 2) {
      return totalSeconds > 0;
    }

    return true;
  }, [
    currentStepIndex,
    showLoadingState,
    totalSeconds,
    values.agentType,
    values.allowedPermissions.length,
    values.clientId,
    values.roleName,
  ]);

  const handleClose = () => {
    navigate("/trust-delegation");
  };

  const handleBack = () => {
    if (currentStepIndex > 0) {
      setCurrentStepIndex((step) => step - 1);
      return;
    }
    handleClose();
  };

  const handleNext = async () => {
    const valid = await form.trigger(STEP_FIELDS[currentStepIndex]);
    if (!valid) return;

    if (currentStepIndex < WIZARD_STEPS.length - 1) {
      setCurrentStepIndex((step) => step + 1);
    }
  };

  const handleSubmit = form.handleSubmit(async (submittedValues) => {
    const body = {
      role_name: submittedValues.roleName,
      agent_type: submittedValues.agentType,
      allowed_permissions: submittedValues.allowedPermissions,
      max_ttl_seconds: durationPartsToSeconds({
        value: submittedValues.durationValue,
        unit: submittedValues.durationUnit,
      }),
      enabled: submittedValues.enabled,
      client_id: submittedValues.clientId,
    };

    try {
      const result = isEditMode
        ? await updatePolicy({ id: policyId || "", body }).unwrap()
        : await createPolicy(body).unwrap();

      toast.success(
        isEditMode ? "Trust delegation updated" : "Trust delegation created",
      );
      navigate(`/trust-delegation/${result.id}`);
    } catch (error) {
      toast.error(getErrorMessage(error, "Failed to save trust delegation"));
    }
  });

  return (
    <TrustDelegationWizardShell
      title={isEditMode ? "Edit Trust Delegation" : "Create Trust Delegation"}
      description={currentStep.headerSubtitle}
      steps={WIZARD_STEPS}
      currentStepIndex={currentStepIndex}
      onClose={handleClose}
      onBack={handleBack}
      onPrimaryAction={
        currentStepIndex < WIZARD_STEPS.length - 1 ? handleNext : handleSubmit
      }
      primaryActionLabel={
        currentStepIndex < WIZARD_STEPS.length - 1
          ? "Next"
          : isEditMode
            ? "Save Trust Delegation"
            : "Create Trust Delegation"
      }
      primaryActionIcon={
        currentStepIndex < WIZARD_STEPS.length - 1
          ? ChevronRight
          : CheckCircle
      }
      primaryActionIconPosition={
        currentStepIndex < WIZARD_STEPS.length - 1 ? "right" : "left"
      }
      primaryActionDisabled={!canProceed}
      primaryActionLoading={currentStepIndex === WIZARD_STEPS.length - 1 && (isCreating || isUpdating)}
      primaryActionLoadingLabel="Saving..."
    >
      {showLoadingState ? (
        <div className="py-20 text-center text-muted-foreground">
          Loading trust delegation...
        </div>
      ) : null}

      {!showLoadingState && currentStepIndex === 0 && (
        <div className="space-y-4">
          <div className="mb-4">
            <h3 className="text-base font-semibold">{currentStep.sectionTitle}</h3>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {currentStep.sectionDescription}
            </p>
          </div>

          {duplicatePolicy && (
            <Alert>
              <AlertTitle>Duplicate trust delegation combination</AlertTitle>
              <AlertDescription>
                A trust delegation already exists for the same role, target type, and client.
                Review the existing record before saving another one.
              </AlertDescription>
            </Alert>
          )}

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <div className="space-y-1.5">
              <Label className="text-sm">
                Role <span className="text-destructive">*</span>
              </Label>
              <Controller
                control={form.control}
                name="roleName"
                render={({ field }) => (
                  <SearchableSelect
                    className="w-full"
                    options={roleOptions}
                    value={field.value}
                    onChange={(value) => field.onChange(value || "")}
                    placeholder="Select role"
                    clearable={false}
                    renderOption={renderTrustDelegationSingleSelectOption}
                  />
                )}
              />
              <TrustDelegationFieldMeta
                helper="Trust delegation matching starts with the selected role."
                error={form.formState.errors.roleName?.message}
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-sm">
                Target Type <span className="text-destructive">*</span>
              </Label>
              <Controller
                control={form.control}
                name="agentType"
                render={({ field }) => (
                  <SearchableSelect
                    className="w-full"
                    options={AGENT_TYPE_OPTIONS}
                    value={field.value}
                    onChange={(value) => field.onChange(value || "")}
                    placeholder="Select target type"
                    clearable={false}
                    renderOption={renderTrustDelegationSingleSelectOption}
                  />
                )}
              />
              <TrustDelegationFieldMeta
                helper="Each trust delegation is constrained to a single target type."
                error={form.formState.errors.agentType?.message}
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-sm">
                Client / Application <span className="text-destructive">*</span>
              </Label>
              <Controller
                control={form.control}
                name="clientId"
                render={({ field }) => (
                  <SearchableSelect
                    className="w-full"
                    options={clientOptions}
                    value={field.value}
                    onChange={(value) => field.onChange(value || "")}
                    placeholder="Select client"
                    clearable={false}
                    renderOption={renderTrustDelegationSingleSelectOption}
                  />
                )}
              />
              <TrustDelegationFieldMeta
                helper="Application boundary this trust delegation applies to."
                error={form.formState.errors.clientId?.message}
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-sm">Users</Label>
              <Controller
                control={form.control}
                name="userIds"
                render={({ field }) => (
                  <SearchableSelect
                    multiple
                    className="w-full"
                    value={field.value}
                    onChange={field.onChange}
                    options={userOptions}
                    placeholder="Select users"
                    maxBadges={4}
                  />
                )}
              />
              <TrustDelegationFieldMeta
                helper="Planning-only assignment list for now. Selected users are not yet stored by the backend."
              />
            </div>
          </div>
        </div>
      )}

      {!showLoadingState && currentStepIndex === 1 && (
        <div className="space-y-4">
          <div className="mb-4">
            <h3 className="text-base font-semibold">{currentStep.sectionTitle}</h3>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {currentStep.sectionDescription}
            </p>
          </div>

          {permissionCatalogError && (
            <Alert variant="destructive">
              <AlertTitle>Unable to load permission catalog</AlertTitle>
              <AlertDescription>
                {getErrorMessage(
                  permissionCatalogError,
                  "The trust delegation action catalog could not be loaded from the current admin permission set.",
                )}
              </AlertDescription>
            </Alert>
          )}

          <div className="space-y-1.5">
            <Label className="text-sm">
              Allowed Actions <span className="text-destructive">*</span>
            </Label>
            <Controller
              control={form.control}
              name="allowedPermissions"
              render={({ field }) => (
                <SearchableSelect
                  multiple
                  className="w-full"
                  value={field.value}
                  onChange={field.onChange}
                  options={permissionOptions}
                  placeholder="Choose allowed actions"
                  maxBadges={6}
                />
              )}
            />
            <TrustDelegationFieldMeta
              helper="Allowed actions are sourced from /uflow/admin/me/roles-permissions and saved as raw permission keys."
              error={form.formState.errors.allowedPermissions?.message}
            />
          </div>
        </div>
      )}

      {!showLoadingState && currentStepIndex === 2 && (
        <div className="space-y-4">
          <div className="mb-4">
            <h3 className="text-base font-semibold">{currentStep.sectionTitle}</h3>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {currentStep.sectionDescription}
            </p>
          </div>

          {policy?.enabled && values.enabled === false && (
            <Alert>
              <AlertTitle>Disabling impact</AlertTitle>
              <AlertDescription>
                Disabling this trust delegation stops future issuance. Existing
                audit records remain intact.
              </AlertDescription>
            </Alert>
          )}

          <div className="grid grid-cols-1 gap-4 xl:grid-cols-[180px_180px_1fr]">
            <div className="space-y-1.5">
              <Label className="text-sm">
                Maximum Duration <span className="text-destructive">*</span>
              </Label>
              <Input
                type="number"
                min={1}
                {...form.register("durationValue", {
                  valueAsNumber: true,
                })}
              />
              <TrustDelegationFieldMeta
                helper="Numeric maximum duration allowed for this trust delegation."
                error={form.formState.errors.durationValue?.message}
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-sm">
                Unit <span className="text-destructive">*</span>
              </Label>
              <Controller
                control={form.control}
                name="durationUnit"
                render={({ field }) => (
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="minutes">Minutes</SelectItem>
                      <SelectItem value="hours">Hours</SelectItem>
                      <SelectItem value="days">Days</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              />
              <TrustDelegationFieldMeta
                helper={`${formatDurationLabel(totalSeconds)} (${totalSeconds} seconds)`}
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-sm">Lifecycle</Label>
              <Controller
                control={form.control}
                name="enabled"
                render={({ field }) => (
                  <div className="flex items-center gap-3 rounded-lg border px-3 py-2">
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                    <span className="text-sm">
                      {field.value
                        ? "Trust delegation can be used for new issuance"
                        : "Trust delegation is disabled for new issuance"}
                    </span>
                  </div>
                )}
              />
              <TrustDelegationFieldMeta helper="Disabled trust delegations remain visible in history but cannot be used for new issuance." />
            </div>
          </div>
        </div>
      )}

      {!showLoadingState && currentStepIndex === 3 && (
        <div className="space-y-4">
          <div className="mb-4">
            <h3 className="text-base font-semibold">{currentStep.sectionTitle}</h3>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {currentStep.sectionDescription}
            </p>
          </div>

          {duplicatePolicy && (
            <Alert>
              <AlertTitle>Duplicate trust delegation combination</AlertTitle>
              <AlertDescription>
                Another trust delegation already exists for this role, target type, and client combination.
              </AlertDescription>
            </Alert>
          )}

          <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
            <div className="rounded-lg border bg-muted/30 p-4">
              <div className="mb-3 text-sm font-semibold">Trust delegation context</div>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div className="flex items-start justify-between gap-4">
                  <span>Role</span>
                  <span className="text-right text-foreground">{values.roleName}</span>
                </div>
                <div className="flex items-start justify-between gap-4">
                  <span>Target type</span>
                  <span className="text-right text-foreground">
                    {selectedAgentType?.label || values.agentType}
                  </span>
                </div>
                <div className="flex items-start justify-between gap-4">
                  <span>Client</span>
                  <span className="text-right text-foreground">
                    {selectedClient?.label || values.clientId}
                  </span>
                </div>
              </div>
            </div>

            <div className="rounded-lg border bg-muted/30 p-4">
              <div className="mb-3 text-sm font-semibold">Duration and lifecycle</div>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div className="flex items-start justify-between gap-4">
                  <span>Maximum duration</span>
                  <span className="text-right text-foreground">
                    {formatDurationLabel(totalSeconds)}
                  </span>
                </div>
                <div className="flex items-start justify-between gap-4">
                  <span>Status</span>
                  <span className="text-right text-foreground">
                    {values.enabled ? "Enabled" : "Disabled"}
                  </span>
                </div>
              </div>
            </div>
          </div>

          {selectedUsers.length > 0 && (
            <div className="rounded-lg border bg-muted/30 p-4">
              <div className="mb-2 text-sm font-semibold">Planning-only users</div>
              <div className="mb-3 flex flex-wrap gap-2">
                {selectedUsers.map((user) => (
                  <Badge key={user.value} variant="outline">
                    {user.label}
                  </Badge>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">
                These selected users help with assignment planning only. They are
                not included in the current backend save payload.
              </p>
            </div>
          )}

          <div className="rounded-lg border bg-muted/30 p-4">
            <div className="mb-3 text-sm font-semibold">Allowed actions</div>
            <div className="flex flex-wrap gap-2">
              {values.allowedPermissions.length > 0 ? (
                values.allowedPermissions.map((permission) => (
                  <Badge key={permission} variant="outline">
                    {humanizePermissionKey(permission)}
                  </Badge>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">
                  No actions selected yet.
                </p>
              )}
            </div>
          </div>
        </div>
      )}
    </TrustDelegationWizardShell>
  );
}

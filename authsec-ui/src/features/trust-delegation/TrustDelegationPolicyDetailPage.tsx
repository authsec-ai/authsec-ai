import { useNavigate, useParams } from "react-router-dom";
import { CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { TableCard } from "@/theme/components/cards";
import {
  useDeleteDelegationPolicyMutation,
  useGetDelegationPolicyQuery,
  useUpdateDelegationPolicyMutation,
} from "@/app/api/trustDelegationApi";
import { getErrorMessage } from "@/lib/error-utils";
import { toast } from "@/lib/toast";
import { TrustDelegationPageFrame } from "./components/TrustDelegationPageFrame";
import { humanizePermissionKey } from "./utils";

export function TrustDelegationPolicyDetailPage() {
  const navigate = useNavigate();
  const { policyId = "" } = useParams();
  const { data: policy, isLoading, error } = useGetDelegationPolicyQuery(policyId, {
    skip: !policyId,
  });
  const [updatePolicy] = useUpdateDelegationPolicyMutation();
  const [deletePolicy, { isLoading: isDeleting }] =
    useDeleteDelegationPolicyMutation();

  const handleToggleEnabled = async () => {
    if (!policy) return;

    try {
      await updatePolicy({
        id: policy.id,
        body: {
          role_name: policy.roleName,
          agent_type: policy.agentType,
          allowed_permissions: policy.allowedPermissions,
          max_ttl_seconds: policy.maxTtlSeconds,
          enabled: !policy.enabled,
          client_id: policy.clientId,
        },
      }).unwrap();
      toast.success(
        policy.enabled
          ? "Trust delegation disabled"
          : "Trust delegation enabled",
      );
    } catch (requestError) {
      toast.error(
        getErrorMessage(requestError, "Failed to update trust delegation"),
      );
    }
  };

  const handleDelete = async () => {
    if (!policy) return;

    try {
      await deletePolicy(policy.id).unwrap();
      toast.success("Trust delegation deleted");
      navigate("/trust-delegation");
    } catch (requestError) {
      toast.error(
        getErrorMessage(requestError, "Failed to delete trust delegation"),
      );
    }
  };

  return (
    <TrustDelegationPageFrame
      title="Trust Delegation Detail"
      description="Inspect one saved trust delegation configuration and its current guardrails."
      actions={
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            onClick={() => navigate(`/trust-delegation/${policyId}/edit`)}
          >
            Edit trust delegation
          </Button>
          <Button variant="outline" onClick={handleToggleEnabled} disabled={!policy}>
            {policy?.enabled ? "Disable" : "Enable"}
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={!policy || isDeleting}
          >
            Delete trust delegation
          </Button>
        </div>
      }
    >
      <TableCard>
        <CardContent className="space-y-4 p-6">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>
                Failed to load this trust delegation. Please go back to the trust delegation list and try again.
              </AlertDescription>
            </Alert>
          )}

          {isLoading && (
            <div className="py-20 text-center text-muted-foreground">
              Loading trust delegation…
            </div>
          )}

          {policy && (
            <>
              <div className="grid gap-4 lg:grid-cols-2">
                <div className="rounded-lg border p-4">
                  <div className="text-xs uppercase tracking-[0.14em] text-muted-foreground">
                    Trust delegation label
                  </div>
                  <div className="mt-2 text-lg font-semibold">
                    {policy.roleName} · {policy.agentType} · {policy.clientLabel}
                  </div>
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Badge variant={policy.enabled ? "default" : "outline"}>
                      {policy.enabled ? "Enabled" : "Disabled"}
                    </Badge>
                    <Badge variant="outline">{policy.maxTtlLabel}</Badge>
                  </div>
                </div>

                <div className="rounded-lg border p-4">
                  <div className="text-xs uppercase tracking-[0.14em] text-muted-foreground">
                    Scope
                  </div>
                  <div className="mt-2 space-y-1 text-sm">
                    <div>Role: {policy.roleName}</div>
                    <div>Target type: {policy.agentType}</div>
                    <div>Client: {policy.clientLabel}</div>
                    <div>Created by: {policy.createdBy || "Unknown actor"}</div>
                  </div>
                </div>
              </div>

              <div className="rounded-lg border p-4">
                <div className="text-sm font-semibold">Allowed actions</div>
                <div className="mt-3 flex flex-wrap gap-2">
                  {policy.allowedPermissions.map((permission) => (
                    <Badge key={permission} variant="outline">
                      {humanizePermissionKey(permission)}
                    </Badge>
                  ))}
                </div>
              </div>
            </>
          )}
        </CardContent>
      </TableCard>
    </TrustDelegationPageFrame>
  );
}

import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Loader2, Shield } from "lucide-react";
import type { ClientWithAuthMethods } from "@/types/entities";
import {
  useShowAuthProvidersQuery,
  useEditClientAuthProviderMutation,
  type ShowAuthProvidersResponse,
  type EditClientAuthProviderRequest,
} from "@/app/api/authMethodApi";
import { SessionManager } from "@/utils/sessionManager";
import { toast } from "react-hot-toast";

type ClientAuthMethodsModalProps = {
  open: boolean;
  client: ClientWithAuthMethods | null;
  onClose: () => void;
};

type Provider = ShowAuthProvidersResponse["data"]["providers"][0];

export function ClientAuthMethodsModal({
  open,
  client,
  onClose,
}: ClientAuthMethodsModalProps) {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [togglingProvider, setTogglingProvider] = useState<string | null>(null);

  const session = SessionManager.getSession();
  const rawClient = (client?.metadata?.raw_client as any);

  // Extract tenant_id and client_id dynamically
  const tenantId = client?.workspace_id ||
    rawClient?.tenant_id ||
    session?.tenant_id ||
    "";

  const clientId = client?.id ||
    rawClient?.client_id ||
    "";

  // Call 1: Get active providers for this specific client (WITH client_id)
  const {
    data: activeProvidersData,
    isLoading: isLoadingActive,
    error: activeError,
    refetch: refetchActive,
  } = useShowAuthProvidersQuery(
    { tenant_id: tenantId, client_id: clientId },
    { skip: !open || !tenantId || !clientId }
  );

  // Call 2: Get all available providers (WITHOUT client_id to get tenant-wide list)
  const {
    data: allProvidersData,
    isLoading: isLoadingAll,
    error: allError,
    refetch: refetchAll,
  } = useShowAuthProvidersQuery(
    { tenant_id: tenantId, client_id: "" }, // Empty client_id to get all
    { skip: !open || !tenantId }
  );

  const [editClientAuthProvider] = useEditClientAuthProviderMutation();

  const isLoading = isLoadingActive || isLoadingAll;
  const error = activeError || allError;

  // Debug logging - only log once when modal opens or on errors
  useEffect(() => {
    if (!open) return;

    console.log('[ClientAuthMethodsModal] Query state:', {
      tenantId,
      clientId,
      isLoadingActive,
      isLoadingAll,
      hasActiveError: !!activeError,
      hasAllError: !!allError,
      hasActiveData: !!activeProvidersData,
      hasAllData: !!allProvidersData,
    });
  }, [open]); // Only run when modal opens

  // Separate effect for error logging to avoid re-renders
  useEffect(() => {
    if (activeError) {
      console.error('[ClientAuthMethodsModal] Error loading active providers:', activeError);
    }
  }, [activeError]);

  useEffect(() => {
    if (allError) {
      console.error('[ClientAuthMethodsModal] Error loading all providers:', allError);
    }
  }, [allError]);

  // Merge active and all providers into a single list
  useEffect(() => {
    if (!allProvidersData?.data?.providers) {
      return;
    }

    const allProviders = allProvidersData.data.providers;
    const activeProviders = activeProvidersData?.data?.providers || [];

    // Create a map of active providers by provider_name
    const activeProviderMap = new Map(
      activeProviders.map(p => [p.provider_name, p])
    );

    // Merge: Use all providers as base, mark which ones are active for this client
    const mergedProviders = allProviders.map(provider => {
      const isActiveForClient = activeProviderMap.has(provider.provider_name);
      const activeProvider = activeProviderMap.get(provider.provider_name);
      
      return {
        ...provider,
        is_active: isActiveForClient && activeProvider ? activeProvider.is_active : false,
        // Keep other properties from active provider if it exists
        ...(activeProvider && {
          callback_url: activeProvider.callback_url,
          endpoints: activeProvider.endpoints,
        })
      };
    });

    console.log('[ClientAuthMethodsModal] Merged providers:', {
      allCount: allProviders.length,
      activeCount: activeProviders.length,
      mergedCount: mergedProviders.length,
      merged: mergedProviders
    });

    setProviders(mergedProviders);
  }, [activeProvidersData, allProvidersData]);

  const handleToggleStatus = async (provider: Provider, nextStatus: boolean) => {
    if (!tenantId || !clientId) {
      toast.error("Missing tenant or client information.");
      return;
    }

    if (togglingProvider) {
      return;
    }

    const payload: EditClientAuthProviderRequest = {
      tenant_id: tenantId,
      client_id: clientId,
      provider_name: provider.provider_name,
      display_name: provider.display_name,
      is_active: nextStatus,
      callback_url: provider.callback_url,
      provider_config: {
        auth_url: provider?.endpoints?.auth_url ?? "",
        token_url: provider?.endpoints?.token_url ?? "",
        user_info_url: provider?.endpoints?.user_info_url,
      },
      updated_by: session?.user?.email || "system",
    };

    const previousStatus = provider.is_active;
    setTogglingProvider(provider.provider_name);
    setProviders((prev) =>
      prev.map((current) =>
        current.provider_name === provider.provider_name
          ? { ...current, is_active: nextStatus }
          : current
      )
    );

    try {
      console.log("Updating provider for specific client:", clientId);
      console.log("Update payload:", JSON.stringify(payload, null, 2));
      
      const result = await editClientAuthProvider(payload).unwrap();
      console.log("Update response:", result);
      
      toast.success(`${provider.display_name} ${nextStatus ? "activated" : "deactivated"}`);
      // Refetch both queries to get updated state
      refetchActive();
      refetchAll();
    } catch (err: any) {
      setProviders((prev) =>
        prev.map((current) =>
          current.provider_name === provider.provider_name
            ? { ...current, is_active: previousStatus }
            : current
        )
      );
      const message =
        err?.data?.message ||
        `Failed to ${nextStatus ? "activate" : "deactivate"} ${provider.display_name}`;
      toast.error(message);
    } finally {
      setTogglingProvider(null);
    }
  };

  if (!open) return null;

  return (
    <>
      <Dialog open={open} onOpenChange={onClose}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Shield className="h-5 w-5" />
              Authentication Methods
            </DialogTitle>
            <DialogDescription>
              Enable or disable authentication providers for <strong>{client?.name}</strong>. 
              Toggle providers on/off to control which authentication methods are available for this client.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {isLoading && (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-5 w-5 animate-spin" />
              </div>
            )}

            {error && (
              <div className="space-y-2">
                <div className="text-center py-4 text-sm text-destructive">
                  Failed to load authentication providers
                </div>
                <div className="text-center text-xs text-foreground">
                  {error && 'data' in error && error.data 
                    ? JSON.stringify(error.data)
                    : error && 'message' in error 
                    ? error.message 
                    : 'Unknown error'}
                </div>
                <div className="text-center">
                  <Button 
                    variant="outline" 
                    size="sm"
                    onClick={() => {
                      refetchActive();
                      refetchAll();
                    }}
                  >
                    Retry
                  </Button>
                </div>
              </div>
            )}

            {!isLoading && !error && providers.length === 0 && (
              <div className="text-center py-8 text-sm text-foreground">
                <p className="font-medium mb-2">No authentication providers available</p>
                <p className="text-xs">Contact your administrator to configure authentication providers for this tenant.</p>
              </div>
            )}

            {!isLoading && !error && providers.length > 0 && (
              <div className="space-y-3 max-h-96 overflow-y-auto overflow-x-hidden pr-1">
                {providers.map((provider) => {
                  const isToggling = togglingProvider === provider.provider_name;
                  return (
                    <div
                      key={provider.provider_name}
                      className={`flex items-center justify-between p-3 border rounded-lg transition-colors ${
                        provider.is_active 
                          ? 'bg-green-50/50 dark:bg-green-950/20 border-green-200 dark:border-green-900' 
                          : 'hover:bg-muted/50'
                      }`}
                    >
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <p className="font-medium">{provider.display_name}</p>
                          {provider.is_active && (
                            <span className="inline-flex items-center rounded-full bg-green-100 dark:bg-green-900/30 px-2 py-0.5 text-xs font-medium text-green-700 dark:text-green-400">
                              Enabled
                            </span>
                          )}
                        </div>
                        <p className="text-sm text-foreground mt-0.5">
                          {provider.provider_name}
                        </p>
                        {provider.callback_url && (
                          <p className="text-xs text-foreground mt-1 truncate max-w-md">
                            Callback: {provider.callback_url}
                          </p>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        {isToggling && (
                          <Loader2 className="h-4 w-4 animate-spin text-foreground" />
                        )}
                        <Switch
                          id={`toggle-${provider.provider_name}`}
                          checked={provider.is_active}
                          onCheckedChange={(checked) => handleToggleStatus(provider, checked)}
                          disabled={isToggling}
                          aria-label={`Toggle ${provider.display_name} status`}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={onClose}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

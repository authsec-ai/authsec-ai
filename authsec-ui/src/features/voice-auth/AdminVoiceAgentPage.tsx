"use client";

import { useState } from "react";
import { toast } from "react-hot-toast";
import {
  AlertTriangle,
  Bell,
  CheckCircle2,
  RefreshCw,
  ShieldCheck,
  Smartphone,
  Trash2,
} from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  useDeleteAdminCibaDeviceMutation,
  useGetAdminCibaDevicesQuery,
  useGetAdminTotpDevicesQuery,
} from "@/app/api/adminVoiceAgentApi";

const formatTimestamp = (value?: number | null) => {
  if (!value) return "Never";
  const milliseconds = value > 1e12 ? value : value * 1000;
  return new Date(milliseconds).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
};

const formatLastUsed = (value?: number | null) => {
  if (!value) return "Not used yet";
  const milliseconds = value > 1e12 ? value : value * 1000;
  return new Date(milliseconds).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
};

const getPlatformLabel = (platform?: string) => {
  if (!platform) return "Unknown";
  const normalized = platform.toLowerCase();
  if (normalized.includes("android")) return "Android";
  if (normalized.includes("ios") || normalized.includes("iphone")) return "iOS";
  return platform;
};

export function AdminVoiceAgentPage() {
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const {
    data: cibaData,
    isLoading: isLoadingCiba,
    isFetching: isFetchingCiba,
    error: cibaError,
    refetch: refetchCiba,
  } = useGetAdminCibaDevicesQuery();

  const {
    data: totpData,
    isLoading: isLoadingTotp,
    isFetching: isFetchingTotp,
    error: totpError,
    refetch: refetchTotp,
  } = useGetAdminTotpDevicesQuery();

  const [deleteCibaDevice] = useDeleteAdminCibaDeviceMutation();

  const cibaDevices = cibaData?.devices ?? [];
  const totpDevices = totpData?.devices ?? [];

  const handleRefreshAll = () => {
    refetchCiba();
    refetchTotp();
  };

  const handleDeleteDevice = async (deviceId: string, deviceName: string) => {
    if (!confirm(`Deactivate "${deviceName}"?`)) return;
    setDeletingId(deviceId);
    try {
      await deleteCibaDevice({ deviceId }).unwrap();
      toast.success("Voice agent device deactivated");
    } catch (error) {
      console.error("Failed to delete CIBA device:", error);
      toast.error("Failed to deactivate device");
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="flex flex-col gap-6 p-6">
      <PageHeader
        title="Voice Agent Access"
        description="Register your admin account for voice approvals using the AuthSec mobile app."
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefreshAll}
            disabled={isFetchingCiba || isFetchingTotp}
          >
            <RefreshCw className={`mr-2 h-4 w-4 ${isFetchingCiba || isFetchingTotp ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        }
      />

      <div className="grid gap-6 lg:grid-cols-[2fr_3fr]">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ShieldCheck className="h-5 w-5 text-primary" />
              Register your device
            </CardTitle>
            <CardDescription>
              Voice agent approvals require a CIBA push device linked to your admin account.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {[
              {
                title: "Install the AuthSec mobile app",
                detail: "Sign in with the same admin email you use in the dashboard.",
              },
              {
                title: "Enable push approvals",
                detail: "Allow notifications so you can approve voice agent sign-ins.",
              },
              {
                title: "Confirm the device appears here",
                detail: "Refresh the list to verify your device is connected.",
              },
            ].map((step, index) => (
              <div key={step.title} className="flex items-start gap-3">
                <div className="mt-0.5 flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                  {index + 1}
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">{step.title}</p>
                  <p className="text-xs text-muted-foreground">{step.detail}</p>
                </div>
              </div>
            ))}
            <div className="rounded-lg border border-dashed border-border/70 bg-muted/20 p-3 text-xs text-muted-foreground">
              Tip: Use a dedicated admin device so approvals are fast and consistent across voice agents.
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-4">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Bell className="h-5 w-5 text-primary" />
                Voice devices
              </CardTitle>
              <CardDescription>Push-enabled devices for CIBA approvals.</CardDescription>
            </div>
            <Badge variant="secondary">{cibaDevices.length} devices</Badge>
          </CardHeader>
          <CardContent className="space-y-3">
            {isLoadingCiba && (
              <div className="flex items-center justify-center py-6">
                <div className="h-6 w-6 animate-spin rounded-full border-2 border-muted border-t-primary" />
              </div>
            )}

            {!isLoadingCiba && cibaError && (
              <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-xs text-red-700">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4" />
                  Failed to load voice devices.
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-2"
                  onClick={refetchCiba}
                >
                  Retry
                </Button>
              </div>
            )}

            {!isLoadingCiba && !cibaError && cibaDevices.length === 0 && (
              <div className="rounded-lg border border-dashed border-border/70 bg-muted/10 p-4 text-sm text-muted-foreground">
                No devices registered yet. Install the mobile app and sign in to add your first device.
              </div>
            )}

            {!isLoadingCiba && !cibaError && cibaDevices.length > 0 && (
              <div className="space-y-2">
                {cibaDevices.map((device) => (
                  <div
                    key={device.id}
                    className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border/60 bg-background px-4 py-3"
                  >
                    <div className="flex items-start gap-3">
                      <div className="mt-1 flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                        <Smartphone className="h-5 w-5" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="text-sm font-semibold text-foreground">
                            {device.device_name}
                          </p>
                          {device.is_active && (
                            <Badge variant="secondary" className="text-[10px] uppercase">
                              Active
                            </Badge>
                          )}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {getPlatformLabel(device.platform)}
                          {device.device_model ? ` • ${device.device_model}` : ""}
                          {device.app_version ? ` • App ${device.app_version}` : ""}
                        </p>
                        <p className="text-[11px] text-muted-foreground">
                          Added {formatTimestamp(device.created_at)} • Last used{" "}
                          {formatLastUsed(device.last_used)}
                        </p>
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDeleteDevice(device.id, device.device_name)}
                      disabled={deletingId === device.id}
                      className="text-muted-foreground hover:text-red-600"
                    >
                      {deletingId === device.id ? (
                        <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                      ) : (
                        <Trash2 className="mr-2 h-4 w-4" />
                      )}
                      Deactivate
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-4">
          <div>
            <CardTitle className="flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-primary" />
              TOTP fallback devices
            </CardTitle>
            <CardDescription>
              Optional backup for voice agent access when push approvals are unavailable.
            </CardDescription>
          </div>
          <Badge variant="outline">{totpDevices.length} devices</Badge>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoadingTotp && (
            <div className="flex items-center justify-center py-6">
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-muted border-t-primary" />
            </div>
          )}

          {!isLoadingTotp && totpError && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-xs text-red-700">
              <div className="flex items-center gap-2">
                <AlertTriangle className="h-4 w-4" />
                Failed to load TOTP devices.
              </div>
              <Button
                variant="outline"
                size="sm"
                className="mt-2"
                onClick={refetchTotp}
              >
                Retry
              </Button>
            </div>
          )}

          {!isLoadingTotp && !totpError && totpDevices.length === 0 && (
            <div className="rounded-lg border border-dashed border-border/70 bg-muted/10 p-4 text-sm text-muted-foreground">
              No TOTP devices found. Add a backup authenticator if you want a fallback for voice
              approvals.
            </div>
          )}

          {!isLoadingTotp && !totpError && totpDevices.length > 0 && (
            <div className="grid gap-2 md:grid-cols-2">
              {totpDevices.map((device) => (
                <div
                  key={device.id}
                  className="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-background px-4 py-3"
                >
                  <div>
                    <p className="text-sm font-semibold text-foreground">{device.device_name}</p>
                    <p className="text-xs text-muted-foreground">
                      {device.device_type} • Added {formatTimestamp(device.created_at)}
                    </p>
                  </div>
                  {device.is_primary && (
                    <Badge variant="secondary" className="text-[10px] uppercase">
                      Primary
                    </Badge>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

import { useState, useEffect } from "react";
import { Button } from "../../../../components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../../../components/ui/card";
import { Input } from "../../../../components/ui/input";
import { Label } from "../../../../components/ui/label";
import { Workflow } from "lucide-react";
import { SessionManager } from "../../../../utils/sessionManager";
import { isIPv4Address } from "../../../../utils/validation";

interface FluentbitConfigurationProps {
  config: {
    tenant_id: string;
    host: string;
    port: string;
  };
  onChange: (config: any) => void;
  onSave: () => void;
  onCancel: () => void;
  isLoading?: boolean;
}

export function FluentbitConfiguration({ config, onChange, onSave, onCancel, isLoading }: FluentbitConfigurationProps) {
  const [showPort, setShowPort] = useState(false);

  // Check if host is IP and show/hide port field
  useEffect(() => {
    setShowPort(isIPv4Address(config.host));
  }, [config.host]);

  // Auto-fill tenant_id on mount
  useEffect(() => {
    const tenantId = SessionManager.getSession()?.tenant_id || "";
    if (tenantId && !config.tenant_id) {
      onChange({ ...config, tenant_id: tenantId });
    }
  }, []);
  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <div className="lg:col-span-2">
        <Card className="border border-slate-200/50 dark:border-neutral-800/80 bg-white/80 dark:bg-neutral-950/70 backdrop-blur-sm shadow-lg dark:shadow-[0_18px_32px_rgba(8,8,12,0.38)]">
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-slate-200 dark:border-neutral-800/70 bg-slate-50 dark:bg-neutral-900/80 text-emerald-600 dark:text-emerald-200">
                <Workflow className="h-5 w-5" />
              </div>
              <div>
                <CardTitle className="text-lg text-slate-900 dark:text-zinc-100">Fluent Bit Configuration</CardTitle>
                <CardDescription className="text-sm text-slate-600 dark:text-zinc-500">
                  Configure Fluent Bit log forwarding settings
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-5">
            {/* Tenant ID (read-only, auto-filled) */}
            <div>
              <Label htmlFor="fluentbit-tenant-id" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                Tenant ID *
              </Label>
              <Input
                id="fluentbit-tenant-id"
                value={config.tenant_id}
                readOnly
                disabled
                className="bg-slate-50 dark:bg-neutral-900/50"
              />
            </div>

            {/* Host (domain or IP) */}
            <div>
              <Label htmlFor="fluentbit-host" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                Host *
              </Label>
              <Input
                id="fluentbit-host"
                placeholder="logs.example.com or 192.168.1.100"
                value={config.host}
                onChange={(e) => onChange({ ...config, host: e.target.value })}
              />
            </div>

            {/* Port (conditional - only for IPs) */}
            {showPort && (
              <div>
                <Label htmlFor="fluentbit-port" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                  Port *
                </Label>
                <Input
                  id="fluentbit-port"
                  placeholder="8080"
                  value={config.port}
                  onChange={(e) => onChange({ ...config, port: e.target.value })}
                />
              </div>
            )}

            <div className="flex justify-end gap-3 pt-4">
              <Button variant="outline" onClick={onCancel} disabled={isLoading}>
                Cancel
              </Button>
              <Button onClick={onSave} disabled={isLoading}>
                {isLoading ? "Saving..." : "Save Configuration"}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <div>
        <Card className="border border-slate-200/50 dark:border-neutral-800/80 bg-white/80 dark:bg-neutral-950/70 backdrop-blur-sm shadow-lg dark:shadow-[0_16px_28px_rgba(8,8,12,0.32)]">
          <CardHeader>
            <CardTitle className="text-lg text-slate-900 dark:text-zinc-100">About Fluent Bit</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-slate-600 dark:text-zinc-500 leading-relaxed">
              Fluent Bit is a lightweight and high-performance log processor and forwarder that allows you to collect logs from different sources.
            </p>
            <div className="space-y-3">
              {[
                "Tenant ID is automatically filled from your session",
                "Enter domain (logs.example.com) or IP address (192.168.1.100)",
                "Port field appears only for IP addresses",
              ].map((tip, index) => (
                <div key={tip} className="flex items-start gap-3">
                  <div className="flex h-6 w-6 items-center justify-center rounded-full border border-slate-200 dark:border-neutral-800/70 bg-slate-50 dark:bg-neutral-900/80 text-[11px] text-emerald-600 dark:text-emerald-200">
                    {index + 1}
                  </div>
                  <p className="text-sm text-slate-700 dark:text-zinc-400 leading-relaxed">{tip}</p>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

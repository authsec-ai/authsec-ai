import { Button } from "../../../../components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../../../components/ui/card";
import { Input } from "../../../../components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../../components/ui/select";
import { Label } from "../../../../components/ui/label";
import { Textarea } from "../../../../components/ui/textarea";
import { Globe } from "lucide-react";

interface HttpConfigurationProps {
  config: {
    host: string;
    port: string;
    uri: string;
    contentType: string;
    customHeaders: string;
    useTls: string;
    tlsVerify: string;
    jsonDateKey: string;
    jsonDateFormat: string;
    logType: string;
  };
  onChange: (config: any) => void;
  onSave: () => void;
  onCancel: () => void;
}

export function HttpConfiguration({ config, onChange, onSave, onCancel }: HttpConfigurationProps) {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <div className="lg:col-span-2">
        <Card className="border border-slate-200/50 dark:border-neutral-800/80 bg-white/80 dark:bg-neutral-950/70 backdrop-blur-sm shadow-lg dark:shadow-[0_18px_32px_rgba(8,8,12,0.38)]">
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-slate-200 dark:border-neutral-800/70 bg-slate-50 dark:bg-neutral-900/80 text-emerald-600 dark:text-emerald-200">
                <Globe className="h-5 w-5" />
              </div>
              <div>
                <CardTitle className="text-lg text-slate-900 dark:text-zinc-100">HTTP Output</CardTitle>
                <CardDescription className="text-sm text-slate-600 dark:text-zinc-500">
                  Send logs to any HTTP/HTTPS endpoint
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="http-host" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                  Host *
                </Label>
                <Input
                  id="http-host"
                  placeholder="Enter host"
                  value={config.host}
                  onChange={(e) => onChange({ ...config, host: e.target.value })}
                />
              </div>
              <div>
                <Label htmlFor="http-port" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                  Port *
                </Label>
                <Input
                  id="http-port"
                  placeholder="Enter port"
                  value={config.port}
                  onChange={(e) => onChange({ ...config, port: e.target.value })}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="http-uri" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                  URI
                </Label>
                <Input
                  id="http-uri"
                  placeholder="Enter uri"
                  value={config.uri}
                  onChange={(e) => onChange({ ...config, uri: e.target.value })}
                />
              </div>
              <div>
                <Label htmlFor="http-content-type" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                  Content Type
                </Label>
                <Input
                  id="http-content-type"
                  placeholder="Enter content type"
                  value={config.contentType}
                  onChange={(e) => onChange({ ...config, contentType: e.target.value })}
                />
              </div>
            </div>

            <div>
              <Label htmlFor="http-headers" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                Custom Headers
              </Label>
              <Textarea
                id="http-headers"
                placeholder="Enter custom headers"
                value={config.customHeaders}
                onChange={(e) => onChange({ ...config, customHeaders: e.target.value })}
                rows={3}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="http-use-tls" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                  Use TLS
                </Label>
                <Select
                  value={config.useTls}
                  onValueChange={(value) => onChange({ ...config, useTls: value })}
                >
                  <SelectTrigger className="bg-white dark:bg-neutral-950/80 border-slate-200 dark:border-neutral-800/70 text-slate-900 dark:text-zinc-200">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="false">false</SelectItem>
                    <SelectItem value="true">true</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="http-tls-verify" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                  TLS Verify
                </Label>
                <Select
                  value={config.tlsVerify}
                  onValueChange={(value) => onChange({ ...config, tlsVerify: value })}
                >
                  <SelectTrigger className="bg-white dark:bg-neutral-950/80 border-slate-200 dark:border-neutral-800/70 text-slate-900 dark:text-zinc-200">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="false">false</SelectItem>
                    <SelectItem value="true">true</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="http-json-date-key" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                  JSON Date Key
                </Label>
                <Input
                  id="http-json-date-key"
                  placeholder="Enter json date key"
                  value={config.jsonDateKey}
                  onChange={(e) => onChange({ ...config, jsonDateKey: e.target.value })}
                />
              </div>
              <div>
                <Label htmlFor="http-json-date-format" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                  JSON Date Format
                </Label>
                <Input
                  id="http-json-date-format"
                  placeholder="Enter json date format"
                  value={config.jsonDateFormat}
                  onChange={(e) => onChange({ ...config, jsonDateFormat: e.target.value })}
                />
              </div>
            </div>

            <div>
              <Label htmlFor="http-log-type" className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                Log Type *
              </Label>
              <Select
                value={config.logType}
                onValueChange={(value) => onChange({ ...config, logType: value })}
              >
                <SelectTrigger className="bg-white dark:bg-neutral-950/80 border-slate-200 dark:border-neutral-800/70 text-slate-900 dark:text-zinc-200">
                  <SelectValue placeholder="Select log type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="application">Application</SelectItem>
                  <SelectItem value="system">System</SelectItem>
                  <SelectItem value="security">Security</SelectItem>
                  <SelectItem value="network">Network</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex justify-end gap-3 pt-4">
              <Button variant="outline" onClick={onCancel}>
                Cancel
              </Button>
              <Button onClick={onSave}>
                Save Configuration
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <div>
        <Card className="border border-slate-200/50 dark:border-neutral-800/80 bg-white/80 dark:bg-neutral-950/70 backdrop-blur-sm shadow-lg dark:shadow-[0_16px_28px_rgba(8,8,12,0.32)]">
          <CardHeader>
            <CardTitle className="text-lg text-slate-900 dark:text-zinc-100">About HTTP</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-slate-600 dark:text-zinc-500 leading-relaxed">
              Forward logs to web services or custom endpoints with configurable formats,
              headers, and authentication methods.
            </p>
            <div className="space-y-3">
              {[
                "Confirm the host, port, and URI path expected by your receiving service.",
                "Include authentication and TLS options that match the destination requirements.",
                "Monitor response codes after saving to ensure the pipeline is healthy.",
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

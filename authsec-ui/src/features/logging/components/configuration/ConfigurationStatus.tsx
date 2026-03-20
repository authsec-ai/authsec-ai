import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../../../components/ui/card";

interface ConfigurationStatusProps {
  configStatus: {
    splunk: boolean;
    // http: boolean;  // Commented for future use
    fluentbit: boolean;
    syslog: boolean;
    elasticsearch: boolean;
  };
  isLoading?: boolean;
}

export function ConfigurationStatus({ configStatus, isLoading }: ConfigurationStatusProps) {
  const services = [
    { label: "Splunk", value: configStatus.splunk },
    // { label: "HTTP", value: configStatus.http },  // Commented for future use
    { label: "Fluent Bit", value: configStatus.fluentbit },
    { label: "Syslog", value: configStatus.syslog },
    { label: "Elasticsearch", value: configStatus.elasticsearch },
  ] as const;

  return (
    <>
      <Card className="border border-slate-200/50 dark:border-neutral-800/80 bg-white/80 dark:bg-neutral-950/70 backdrop-blur-sm shadow-lg dark:shadow-[0_16px_28px_rgba(8,8,12,0.32)]">
        <CardHeader>
          <CardTitle className="text-slate-900 dark:text-zinc-100">Configuration Status</CardTitle>
          <CardDescription className="text-sm text-slate-600 dark:text-zinc-500">
            We support multiple output destinations for your logs. Choose what best fits your
            infrastructure and monitoring needs.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading
            ? Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="flex items-center justify-between animate-pulse">
                  <div className="h-3 w-24 rounded bg-slate-200 dark:bg-neutral-800" />
                  <div className="h-3 w-20 rounded bg-slate-200 dark:bg-neutral-800" />
                </div>
              ))
            : services.map(({ label, value }) => (
                <div key={label} className="flex items-center justify-between">
                  <span className="text-xs uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500">
                    {label}
                  </span>
                  <span
                    className={`text-sm font-medium ${
                      value
                        ? "text-emerald-600 dark:text-emerald-300"
                        : "text-slate-400 dark:text-zinc-600"
                    }`}
                  >
                    {value ? "Configured" : "Not configured"}
                  </span>
                </div>
              ))}
        </CardContent>
      </Card>

      <Card className="mt-6 border border-slate-200/50 dark:border-neutral-800/80 bg-white/80 dark:bg-neutral-950/70 backdrop-blur-sm shadow-lg dark:shadow-[0_16px_28px_rgba(8,8,12,0.32)]">
        <CardHeader>
          <CardTitle className="text-lg text-slate-900 dark:text-zinc-100">
            About Logs Configuration
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-slate-600 dark:text-zinc-500 leading-relaxed">
            Send your telemetry to the destinations that match your observability stack.
            Configure each connector once and reuse it across environments.
          </p>
        </CardContent>
      </Card>
    </>
  );
}

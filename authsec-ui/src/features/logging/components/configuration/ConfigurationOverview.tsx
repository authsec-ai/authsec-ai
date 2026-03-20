import { Button } from "../../../../components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../../../components/ui/card";
import { Database, CheckCircle, Search, Workflow } from "lucide-react";

interface ConfigurationOverviewProps {
  onTabChange: (tab: string) => void;
  configuredServices: {
    splunk: boolean;
    // http: boolean;  // Commented for future use
    fluentbit: boolean;
    syslog: boolean;
    elasticsearch: boolean;
  };
}

export function ConfigurationOverview({ onTabChange, configuredServices }: ConfigurationOverviewProps) {
  const services = [
    {
      key: "splunk",
      icon: Database,
      title: "Splunk",
      description: "Connect to your Splunk instance via HTTP Event Collector",
      details: "Splunk allows you to search, analyze, and visualize machine data in real-time—turning logs, events, and metrics into security insights, operational intelligence, and actionable answers.",
    },
    /*
    // HTTP Configuration - Commented for future use
    {
      key: "http",
      icon: Globe,
      title: "HTTP",
      description: "Send logs to any HTTP/HTTPS endpoint",
      details: "The HTTP output plugin allows you to flush your records to an HTTP endpoint. It supports both HTTP and HTTPS and provides options for authentication, payload format, and retry mechanisms.",
    },
    */
    {
      key: "fluentbit",
      icon: Workflow,
      title: "Fluent Bit",
      description: "Stream logs to Fluent Bit for processing and forwarding",
      details: "Fluent Bit is a lightweight and high-performance log processor and forwarder. It allows you to collect logs from different sources, enrich them with filters, and send them to multiple destinations.",
    },
    {
      key: "syslog",
      icon: CheckCircle,
      title: "Syslog",
      description: "Stream logs to your Syslog server via UDP or TCP",
      details: "The Syslog output plugin allows you to deliver records to a Syslog server through TCP or UDP. It follows the RFC5424 protocol standard for message formatting.",
    },
    {
      key: "elasticsearch",
      icon: Search,
      title: "Elasticsearch",
      description: "Index logs directly into Elasticsearch",
      details: "The Elasticsearch output plugin allows you to ingest your records into an Elasticsearch database. It supports template management, HTTP authentication, and TLS/SSL encryption.",
    },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      {services.map((service) => {
        const Icon = service.icon;
        const isConfigured = configuredServices[service.key as keyof typeof configuredServices];

        return (
          <Card
            key={service.key}
            className="group border border-slate-200/50 dark:border-neutral-800/80 bg-white/80 dark:bg-neutral-950/70 backdrop-blur-sm shadow-lg dark:shadow-[0_18px_32px_rgba(8,8,12,0.38)] transition-all duration-200 hover:shadow-xl dark:hover:shadow-[0_24px_40px_rgba(8,8,12,0.48)] hover:border-slate-300 dark:hover:border-neutral-700"
          >
            <CardHeader className="pb-3">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-slate-200 dark:border-neutral-800/70 bg-slate-50 dark:bg-neutral-900/80 text-emerald-600 dark:text-emerald-200">
                  <Icon className="h-5 w-5" />
                </div>
                <div>
                  <CardTitle className="text-lg text-slate-900 dark:text-zinc-100">
                    {service.title}
                  </CardTitle>
                  <CardDescription className="text-sm text-slate-600 dark:text-zinc-500">
                    {service.description}
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-slate-600 dark:text-zinc-500 leading-relaxed mb-4">
                {service.details}
              </p>
              {isConfigured && (
                <div className="flex items-center gap-2 mb-4 text-emerald-600 dark:text-emerald-300/90">
                  <CheckCircle className="h-4 w-4" />
                  <span className="text-sm font-medium">Configured</span>
                </div>
              )}
              <Button
                className="w-full opacity-0 group-hover:opacity-100 transition-opacity duration-200"
                variant={isConfigured ? "outline" : "default"}
                onClick={() => onTabChange(service.key)}
              >
                {isConfigured ? "Edit" : "Configure"}
              </Button>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}

import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../../components/ui/button";
import { Alert, AlertTitle, AlertDescription } from "../../components/ui/alert";
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "../../components/ui/tabs";
import { ArrowLeft, CheckCircle, X } from "lucide-react";
import {
  useConfigureLogServiceMutation,
  useGetLogConfigurationStatusQuery,
} from "../../app/api/logsApi";
import { toast } from "../../lib/toast";
import { buildHostString } from "../../utils/validation";
import {
  SplunkConfiguration,
  // HttpConfiguration,  // Commented for future use
  SyslogConfiguration,
  ElasticsearchConfiguration,
  FluentbitConfiguration,
  ConfigurationOverview,
  ConfigurationStatus,
} from "./components/configuration";

export function LogsConfigurationPage() {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState("overview");
  const [banner, setBanner] = useState<null | {
    title: string;
    description?: string;
  }>(null);

  const tenantId = (() => {
    try {
      const session = JSON.parse(
        localStorage.getItem("authsec_session_v2") || "{}"
      );
      return session?.tenant_id || "";
    } catch {
      return "";
    }
  })();

  const { data: configStatus, isLoading: isStatusLoading } =
    useGetLogConfigurationStatusQuery(
      { tenant_id: tenantId },
      { skip: !tenantId }
    );

  const [configureLogService, { isLoading: isConfiguring }] =
    useConfigureLogServiceMutation();

  const [splunkConfig, setSplunkConfig] = useState({
    tenant_id: "",
    host: "",
    port: "",
  });

  /*
  // HTTP Configuration - Commented for future use
  const [httpConfig, setHttpConfig] = useState({
    tenant_id: "",
    host: "",
    port: "",
  });
  */

  const [fluentbitConfig, setFluentbitConfig] = useState({
    tenant_id: "",
    host: "",
    port: "",
  });

  const [syslogConfig, setSyslogConfig] = useState({
    tenant_id: "",
    host: "",
    port: "",
  });

  const [elasticsearchConfig, setElasticsearchConfig] = useState({
    tenant_id: "",
    host: "",
    port: "",
  });

  // Unified save handler factory
  const createSaveHandler = (
    name: "splunk" | "fluentbit" | "elasticsearch" | "syslog",
    config: { tenant_id: string; host: string; port: string }
  ) => {
    return async () => {
      try {
        // Build host string (adds :port for IPs)
        const hostString = buildHostString(config.host, config.port);

        await configureLogService({
          host: hostString,
          tenant_id: config.tenant_id,
          name: name,
        }).unwrap();

        setBanner({
          title: `${
            name.charAt(0).toUpperCase() + name.slice(1)
          } configured successfully!`,
          description: "Your log forwarding settings have been updated.",
        });

        toast.success(
          `${
            name.charAt(0).toUpperCase() + name.slice(1)
          } configured successfully.`
        );

        setActiveTab("overview");
      } catch (error: any) {
        toast.error(error?.data?.message || `Failed to configure ${name}.`);
      }
    };
  };

  // Create handlers for each service
  const handleSplunkSave = createSaveHandler("splunk", splunkConfig);
  const handleFluentbitSave = createSaveHandler("fluentbit", fluentbitConfig);
  const handleSyslogSave = createSaveHandler("syslog", syslogConfig);
  const handleElasticsearchSave = createSaveHandler(
    "elasticsearch",
    elasticsearchConfig
  );

  const handleCancel = () => {
    navigate("/logs");
  };

  return (
    <div className="min-h-full flex flex-col bg-gradient-to-b from-slate-50 via-white to-slate-100 dark:from-neutral-950 dark:via-neutral-950 dark:to-neutral-900">
      {/* Sticky header */}
      <div className="sticky top-0 z-30 w-full border-b border-slate-200 dark:border-neutral-900/70 bg-white/95 dark:bg-neutral-950/95 backdrop-blur">
        <div className="flex items-center justify-between px-6 md:px-10 py-5">
          <div className="flex items-center gap-4">
            {/* <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate("/logs")}
              className="shrink-0 text-slate-600 dark:text-zinc-400 hover:text-slate-900 dark:hover:text-zinc-100 hover:bg-slate-100 dark:hover:bg-neutral-900/70"
            >
              <ArrowLeft className="h-4 w-4 mr-1" />
            </Button> */}
            <div className="flex flex-col items-start gap-2 max-w-2xl">
              <h1 className="text-2xl md:text-2xl font-semibold text-slate-900 dark:text-zinc-100 tracking-tight">
                Logs Configuration
              </h1>
              <p className="text-sm md:text-base text-slate-600 dark:text-zinc-500 leading-relaxed">
                Configure output destinations for your logs: Splunk, HTTP
                endpoints, Syslog servers, or Elasticsearch.
              </p>
            </div>
          </div>
        </div>
      </div>

      {banner && (
        <div className="w-full   mx-auto px-6 md:px-8 pt-6">
          <Alert className="relative border border-emerald-500/30 bg-emerald-500/5 backdrop-blur-sm">
            <CheckCircle className="h-4 w-4 text-emerald-600 dark:text-emerald-300" />
            <AlertTitle className="text-slate-900 dark:text-zinc-100">
              {banner.title}
            </AlertTitle>
            {banner.description && (
              <AlertDescription className="text-sm text-slate-700 dark:text-zinc-400">
                {banner.description}
              </AlertDescription>
            )}
            <div className="mt-3 flex items-center gap-3">
              <Button
                size="sm"
                onClick={() => {
                  setBanner(null);
                  navigate("/logs");
                }}
              >
                View Logs
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setBanner(null)}>
                Dismiss
              </Button>
            </div>
            <button
              className="absolute right-3 top-3 text-slate-500 dark:text-zinc-500 hover:text-slate-900 dark:hover:text-zinc-200"
              onClick={() => setBanner(null)}
              aria-label="Dismiss"
            >
              <X className="h-4 w-4" />
            </button>
          </Alert>
        </div>
      )}

      <div className="w-full   mx-auto px-6 md:px-8 py-8 space-y-8">
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList className="flex w-full flex-wrap gap-2 rounded-full border border-slate-200 dark:border-neutral-800/70 bg-slate-50 dark:bg-neutral-950/70 p-1 mb-8">
            <TabsTrigger
              value="overview"
              className="flex-1 rounded-full px-4 py-2 text-sm font-medium text-slate-600 dark:text-zinc-500 transition data-[state=active]:bg-white dark:data-[state=active]:bg-neutral-900/80 data-[state=active]:text-emerald-600 dark:data-[state=active]:text-emerald-200"
            >
              Overview
            </TabsTrigger>
            <TabsTrigger
              value="splunk"
              className="flex-1 rounded-full px-4 py-2 text-sm font-medium text-slate-600 dark:text-zinc-500 transition data-[state=active]:bg-white dark:data-[state=active]:bg-neutral-900/80 data-[state=active]:text-emerald-600 dark:data-[state=active]:text-emerald-200"
            >
              Splunk
            </TabsTrigger>
            {/* HTTP Tab - Commented for future use
            <TabsTrigger
              value="http"
              className="flex-1 rounded-full px-4 py-2 text-sm font-medium text-slate-600 dark:text-zinc-500 transition data-[state=active]:bg-white dark:data-[state=active]:bg-neutral-900/80 data-[state=active]:text-emerald-600 dark:data-[state=active]:text-emerald-200"
            >
              HTTP
            </TabsTrigger>
            */}
            <TabsTrigger
              value="fluentbit"
              className="flex-1 rounded-full px-4 py-2 text-sm font-medium text-slate-600 dark:text-zinc-500 transition data-[state=active]:bg-white dark:data-[state=active]:bg-neutral-900/80 data-[state=active]:text-emerald-600 dark:data-[state=active]:text-emerald-200"
            >
              Fluent Bit
            </TabsTrigger>
            <TabsTrigger
              value="syslog"
              className="flex-1 rounded-full px-4 py-2 text-sm font-medium text-slate-600 dark:text-zinc-500 transition data-[state=active]:bg-white dark:data-[state=active]:bg-neutral-900/80 data-[state=active]:text-emerald-600 dark:data-[state=active]:text-emerald-200"
            >
              Syslog
            </TabsTrigger>
            <TabsTrigger
              value="elasticsearch"
              className="flex-1 rounded-full px-4 py-2 text-sm font-medium text-slate-600 dark:text-zinc-500 transition data-[state=active]:bg-white dark:data-[state=active]:bg-neutral-900/80 data-[state=active]:text-emerald-600 dark:data-[state=active]:text-emerald-200"
            >
              Elasticsearch
            </TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-8">
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              <div className="lg:col-span-2">
                <ConfigurationOverview
                  onTabChange={setActiveTab}
                  configuredServices={
                    configStatus ?? {
                      splunk: false,
                      fluentbit: false,
                      syslog: false,
                      elasticsearch: false,
                    }
                  }
                />
              </div>
              <div>
                <ConfigurationStatus
                  configStatus={
                    configStatus ?? {
                      splunk: false,
                      fluentbit: false,
                      syslog: false,
                      elasticsearch: false,
                    }
                  }
                  isLoading={isStatusLoading}
                />
              </div>
            </div>
          </TabsContent>

          <TabsContent value="splunk" className="space-y-8">
            <SplunkConfiguration
              config={splunkConfig}
              onChange={setSplunkConfig}
              onSave={handleSplunkSave}
              onCancel={handleCancel}
              isLoading={isConfiguring}
            />
          </TabsContent>

          {/* HTTP Tab Content - Commented for future use
          <TabsContent value="http" className="space-y-8">
            <HttpConfiguration
              config={httpConfig}
              onChange={setHttpConfig}
              onSave={handleSaveConfiguration}
              onCancel={handleCancel}
            />
          </TabsContent>
          */}

          <TabsContent value="fluentbit" className="space-y-8">
            <FluentbitConfiguration
              config={fluentbitConfig}
              onChange={setFluentbitConfig}
              onSave={handleFluentbitSave}
              onCancel={handleCancel}
              isLoading={isConfiguring}
            />
          </TabsContent>

          <TabsContent value="syslog" className="space-y-8">
            <SyslogConfiguration
              config={syslogConfig}
              onChange={setSyslogConfig}
              onSave={handleSyslogSave}
              onCancel={handleCancel}
              isLoading={isConfiguring}
            />
          </TabsContent>

          <TabsContent value="elasticsearch" className="space-y-8">
            <ElasticsearchConfiguration
              config={elasticsearchConfig}
              onChange={setElasticsearchConfig}
              onSave={handleElasticsearchSave}
              onCancel={handleCancel}
              isLoading={isConfiguring}
            />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}

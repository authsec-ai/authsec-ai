import React, { useState } from "react";
import { Button } from "../../../components/ui/button";
import { Copy, ExternalLink } from "lucide-react";
import { toast } from "react-hot-toast";
import { cn } from "../../../lib/utils";
import type { ExternalServiceFormData } from "../types";

interface SDKSnippetProps {
  serviceData: ExternalServiceFormData;
  className?: string;
}

export function SDKSnippet({ serviceData, className }: SDKSnippetProps) {
  const [copied, setCopied] = useState(false);

  // Generate Python code based on the form data
  const generatePythonCode = () => {
    const providerImport = serviceData.provider
      ? `from authsec.ext import ${getProviderClassName(serviceData.provider)}`
      : "from authsec.ext import ExternalService  # Select a provider";

    const providerClass = serviceData.provider
      ? getProviderClassName(serviceData.provider)
      : "ExternalService";

    const serviceIdComment = serviceData.serviceId ? "" : "  # auto-generated from service name";

    const clientIdValue = serviceData.clientId || "YOUR_CLIENT_ID";
    const clientIdComment = serviceData.clientId ? "" : "   # paste from provider console";

    const defaultClientComment = serviceData.defaultClientId
      ? `\n\n# Redirect URI for ${getClientName(serviceData.defaultClientId)}: ${
          serviceData.redirectUri
        }`
      : "";

    return `# pip install authsec
${providerImport}

${providerClass.toLowerCase()} = ${providerClass}(
    workspace_id = "acme-prod",
    service_id   = "${serviceData.serviceId || "service-id"}"${serviceIdComment},
    client_id    = "${clientIdValue}"${clientIdComment},
    client_secret= "${serviceData.clientSecret || "YOUR_CLIENT_SECRET"}"
)

@app.route("/connect/${serviceData.serviceId || "service"}")
def connect():
    return ${providerClass.toLowerCase()}.auth_url()${defaultClientComment}`;
  };

  const getProviderClassName = (providerId: string) => {
    const providerMap: Record<string, string> = {
      google_drive: "GoogleDrive",
      google_calendar: "GoogleCalendar",
      salesforce: "Salesforce",
      microsoft_graph: "MicrosoftGraph",
      slack: "Slack",
      github: "GitHub",
      dropbox: "Dropbox",
      box: "Box",
      custom_oauth2: "CustomOAuth2",
    };

    return providerMap[providerId] || "ExternalService";
  };

  const getClientName = (clientId: string) => {
    // This would normally fetch from a client list
    // For now, we'll just return a placeholder
    return clientId || "default client";
  };

  const handleCopyCode = () => {
    navigator.clipboard.writeText(generatePythonCode());
    setCopied(true);
    toast.success("SDK snippet copied to clipboard");

    setTimeout(() => {
      setCopied(false);
    }, 2000);
  };

  const openDocs = () => {
    window.open("https://docs.authsec.com/external-services", "_blank");
  };

  return (
    <div className={cn("relative bg-zinc-950 text-zinc-50 rounded-lg overflow-hidden", className)}>
      <div className="flex items-center justify-between p-3 bg-zinc-900 border-b border-zinc-800">
        <div className="text-sm font-medium flex items-center gap-2">
          <div className="h-3 w-3 rounded-full bg-emerald-500"></div>
          <span>Python SDK</span>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            className="h-8 text-zinc-400 hover:text-zinc-50 hover:bg-zinc-800"
            onClick={handleCopyCode}
          >
            <Copy className="h-4 w-4 mr-2" />
            {copied ? "Copied!" : "Copy"}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-8 text-zinc-400 hover:text-zinc-50 hover:bg-zinc-800"
            onClick={openDocs}
          >
            <ExternalLink className="h-4 w-4 mr-2" />
            Docs
          </Button>
        </div>
      </div>
      <div className="p-4 overflow-x-auto-hidden">
        <pre className="text-sm font-mono">
          <code>{generatePythonCode()}</code>
        </pre>
      </div>
      <div className="absolute top-0 left-0 w-full h-full pointer-events-none bg-gradient-to-br from-primary/5 to-transparent opacity-50"></div>
    </div>
  );
}

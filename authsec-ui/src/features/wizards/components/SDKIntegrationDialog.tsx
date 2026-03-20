import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Check, Code2, Copy } from "lucide-react";
import "@/features/sdk/sdk-editorial-theme.css";

interface SDKIntegrationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// SDK Integration content (from SPRIRE_FAQ_DATA)
const sdkIntegrationSteps = {
  python: [
    {
      label: "Step 1: Install AuthSec SDK",
      code: `pip install git+https://github.com/authsec-ai/sdk-authsec.git`,
    },
    {
      label: "Import Dependencies",
      code: `from authsec_sdk import QuickStartSVID`,
    },
    {
      label: "Example Usage",
      code: `from AuthSec_SDK import (
    mcp_tool, # unprotected tool decorator
    protected_by_AuthSec, # protected tool decorator
    run_mcp_server_with_oauth, # function to run MCP server with OAuth
    QuickStartSVID  # SPIRE workload identity
)

@mcp_tool(
    "get_spire_identity",
    description="Get current SPIRE workload identity (SPIFFE ID and certificate paths)",
    inputSchema={"type": "object", "properties": {}}
)
async def get_spire_identity(arguments: dict) -> list:
    """Get SPIRE workload identity information"""
    try:
        svid = await QuickStartSVID.initialize(socket_path="your/agent/path.sock")
        result = {
            "status": "success",
            "spiffe_id": svid.spiffe_id,
            "certificate": str(svid.cert_file_path),
            "private_key": str(svid.key_file_path),
            "ca_bundle": str(svid.ca_file_path),
            "auto_renewal": "enabled (30 min)"
        }
        return [{"type": "text", "text": json.dumps(result, indent=2)}]
    except RuntimeError as e:
        # SPIRE not enabled
        return [{"type": "text", "text": json.dumps({
            "status": "disabled",
            "message": str(e),
            "note": "To enable SPIRE, add 'spire_socket_path' parameter to run_mcp_server_with_oauth()"
        }, indent=2)}]
    except Exception as e:
        # SPIRE enabled but error occurred
        return [{"type": "text", "text": json.dumps({
            "status": "error",
            "error": str(e),
            "note": "SPIRE is enabled but agent connection failed"
        }, indent=2)}]`,
    },
    {
      label: "Main Server Entry Point",
      code: `if __name__ == "__main__":
    import sys

    run_mcp_server_with_oauth(
        user_module=sys.modules[__name__],
        client_id="your_client_id",
        app_name="Secure MCP Server with AuthSec",
        host="0.0.0.0",
        port=3008,
    )`,
    },
  ],
  typescript: [
    {
      label: "Define Types",
      code: `import axios from 'axios';

interface CreateClientPayload {
  name: string;
  description: string;
  type: 'mcp_server' | 'app' | 'api';
  active: boolean;
}

interface ClientResponse {
  client_id: string;
  client_secret: string;
  name: string;
  description: string;
}`,
    },
    {
      label: "Create Client Function",
      code: `async function createClient(
  token: string,
  clientName: string,
  description: string
): Promise<ClientResponse | null> {
  const url = 'https://api.authsec.dev/api/v1/clients';

  const headers = {
    'Authorization': \`Bearer \${token}\`,
    'Content-Type': 'application/json'
  };

  const payload: CreateClientPayload = {
    name: clientName,
    description: description,
    type: 'mcp_server',
    active: true
  };

  try {
    const response = await axios.post<ClientResponse>(url, payload, { headers });
    const client = response.data;

    console.log('Client created successfully!');
    console.log(\`Client ID: \${client.client_id}\`);
    console.log(\`Client Secret: \${client.client_secret}\`);

    return client;
  } catch (error) {
    console.error('Failed to create client:', error);
    return null;
  }
}`,
    },
    {
      label: "Usage",
      code: `const token = 'your-access-token';
const client = await createClient(token, 'My MCP Server', 'Production MCP server');`,
    },
  ],
};

const CodeBlock = ({
  code,
  label,
  onCopy,
  copied,
}: {
  code: string;
  label: string;
  onCopy: () => void;
  copied: boolean;
}) => {
  return (
    <div
      className="border border-neutral-700 rounded-lg overflow-hidden bg-neutral-800 backdrop-blur-sm"
      data-sdk-code-block="true"
    >
      <div className="flex items-center justify-between border-b border-neutral-700 bg-neutral-800 px-2.5 py-0.5">
        <span className="text-[11px] font-semibold text-neutral-400">
          {label}
        </span>
        <Button
          size="sm"
          variant="ghost"
          className="h-5 w-5 p-0 hover:bg-neutral-700 text-neutral-400 hover:text-neutral-300"
          onClick={onCopy}
        >
          {copied ? (
            <Check className="h-2.5 w-2.5 text-green-500" />
          ) : (
            <Copy className="h-2.5 w-2.5" />
          )}
        </Button>
      </div>
      <div className="p-3 overflow-x-auto bg-neutral-900">
        <pre className="text-sm font-mono text-neutral-300 whitespace-pre-wrap leading-relaxed">
          {code}
        </pre>
      </div>
    </div>
  );
};

export function SDKIntegrationDialog({
  open,
  onOpenChange,
}: SDKIntegrationDialogProps) {
  const [activeLanguage, setActiveLanguage] = useState("python");
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const handleCopy = (code: string, id: string) => {
    navigator.clipboard.writeText(code);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="!max-w-none w-[65vw] max-h-[90vh] overflow-y-auto"
        data-dashboard="overview"
        data-sdk-surface="sdk-integration-dialog"
      >
        <DialogHeader className="sdk-editorial-soft-panel p-4">
          <DialogTitle className="text-lg font-semibold">
            How do I attest my workload?
          </DialogTitle>
          <DialogDescription>
            Learn how to attest your workload by programmatically integrating
            with the AuthSec SDK
          </DialogDescription>
        </DialogHeader>

        <Tabs value={activeLanguage} onValueChange={setActiveLanguage} className="mt-4">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="python" className="gap-2">
              <Code2 className="h-4 w-4" />
              Python
            </TabsTrigger>
            <TabsTrigger value="typescript" className="gap-2">
              <Code2 className="h-4 w-4" />
              TypeScript
            </TabsTrigger>
          </TabsList>

          <TabsContent value="python" className="mt-4 space-y-4">
            {sdkIntegrationSteps.python.map((step, index) => (
              <CodeBlock
                key={index}
                label={step.label}
                code={step.code}
                onCopy={() => handleCopy(step.code, `python-${index}`)}
                copied={copiedId === `python-${index}`}
              />
            ))}
          </TabsContent>

          <TabsContent value="typescript" className="mt-4 space-y-4">
            {sdkIntegrationSteps.typescript.map((step, index) => (
              <CodeBlock
                key={index}
                label={step.label}
                code={step.code}
                onCopy={() => handleCopy(step.code, `typescript-${index}`)}
                copied={copiedId === `typescript-${index}`}
              />
            ))}
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}

"use client";

import { useEffect, useState } from "react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import {
  Shield,
  Server,
  Bot,
  Copy,
  Check,
  Eye,
  X,
  ArrowLeft,
  Maximize2,
  ChevronRight,
  Mic,
  Smartphone,
  KeyRound,
} from "lucide-react";
import { toast } from "react-hot-toast";
import { SessionManager } from "../../utils/sessionManager";
import { useRegisterClientMutation } from "../../app/api/clientApi";
import { useNavigate, useParams, useLocation } from "react-router-dom";

// Code Block Component with Tabs
const CodeBlock = ({
  code,
  language = "python",
  label,
  onCopy,
  copied,
  showTabs = false,
  onExpand,
}: {
  code: string;
  language?: string;
  label?: string;
  onCopy?: () => void;
  copied?: boolean;
  showTabs?: boolean;
  onExpand?: () => void;
}) => {
  const [activeTab, setActiveTab] = useState<"python" | "typescript">("python");

  return (
    <div className="border border-border/60 rounded-lg overflow-hidden bg-background/50 backdrop-blur-sm">
      {/* Header with tabs or label */}
      <div className="flex items-center justify-between border-b border-border/50 bg-muted/40 px-2.5 py-0.5">
        {showTabs ? (
          <div className="flex gap-1 -mb-[3px]">
            <button
              onClick={() => setActiveTab("python")}
              className={`px-2 py-0.5 text-[11px] font-semibold transition-all rounded-t ${
                activeTab === "python"
                  ? "text-foreground bg-background/80 border-b-2 border-primary"
                  : "text-foreground/60 hover:text-foreground/90 hover:bg-muted/30"
              }`}
            >
              Python
            </button>
            <button
              onClick={() => setActiveTab("typescript")}
              className={`px-2 py-0.5 text-[11px] font-semibold transition-all rounded-t ${
                activeTab === "typescript"
                  ? "text-foreground bg-background/80 border-b-2 border-primary"
                  : "text-foreground/60 hover:text-foreground/90 hover:bg-muted/30"
              }`}
            >
              TypeScript
            </button>
          </div>
        ) : (
          <span className="text-[11px] font-semibold text-foreground/75">{label}</span>
        )}

        <div className="flex items-center">
          {onExpand && (
            <Button
              size="sm"
              variant="ghost"
              className="h-5 w-5 p-0 hover:bg-background/60 text-foreground/60 hover:text-foreground"
              onClick={onExpand}
            >
              <Maximize2 className="h-2.5 w-2.5" />
            </Button>
          )}
          {onCopy && (
            <Button
              size="sm"
              variant="ghost"
              className="h-5 w-5 p-0 hover:bg-background/60 text-foreground/60 hover:text-foreground"
              onClick={onCopy}
            >
              {copied ? (
                <Check className="h-2.5 w-2.5 text-green-600 dark:text-green-500" />
              ) : (
                <Copy className="h-2.5 w-2.5" />
              )}
            </Button>
          )}
        </div>
      </div>

      {/* Code content */}
      <div className="p-3 overflow-x-auto-hidden bg-background/30">
        <pre className="text-sm font-mono text-foreground/90 whitespace-pre-wrap leading-relaxed">
          {code}
        </pre>
      </div>
    </div>
  );
};

export default function OnboardingPage() {
  const { clientId } = useParams<{ clientId?: string }>();
  const location = useLocation();
  const [clientType, setClientType] = useState<"MCP-Server" | "AI-Agent" | null>(null);
  const [clientName, setClientName] = useState("My Client");
  const [copiedSteps, setCopiedSteps] = useState<Set<string>>(new Set());
  const [showExample, setShowExample] = useState(false);
  // First step form state
  const [email, setEmail] = useState("");
  const [tenantId, setTenantId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [createdClientId, setCreatedClientId] = useState<string | null>(null);
  const [registerClient, { isLoading: isCreating }] = useRegisterClientMutation();
  const navigate = useNavigate();

  // Check if we're coming from wizard
  const locationState = location.state as any;
  const isFromWizard = locationState?.fromWizard === true;

  // Detect if we're coming from an existing client
  const isExistingClient = Boolean(clientId);
  const [existingClientName, setExistingClientName] = useState<string>("");

  useEffect(() => {
    try {
      // Prefer SessionManager but fall back to direct localStorage read
      const session =
        SessionManager.getSession() ||
        JSON.parse(localStorage.getItem("authsec_session_v2") || "null");
      if (session) {
        setTenantId(session.tenant_id || "");
        setProjectId(session.project_id || "");
        const sessionEmail = session.user?.email || session.user?.email_id || "";
        if (sessionEmail) setEmail(sessionEmail);
      }

      // If we have a clientId in the URL, we're viewing an existing client
      if (clientId) {
        setCreatedClientId(clientId);
        // You can set a default client name or fetch it from API
        setExistingClientName("Existing Client"); // This could be fetched from API in the future
        toast.success("Loaded existing client for SDK integration");
      }
    } catch (e) {
      // ignore
    }
  }, [clientId]);

  const handleCopy = (text: string, stepId: string) => {
    navigator.clipboard.writeText(text);
    setCopiedSteps((prev) => new Set([...prev, stepId]));
    setTimeout(() => {
      setCopiedSteps((prev) => {
        const newSet = new Set(prev);
        newSet.delete(stepId);
        return newSet;
      });
    }, 2000);
  };

  const handleDone = () => {
    // Navigate back to dashboard with success state
    navigate("/", {
      state: {
        sdkIntegrationComplete: true,
        from: location.pathname,
        wizardId: locationState?.wizardId,
      },
    });
    toast.success("SDK integration complete!");
  };

  const getInstallCommand = () => {
    return "pip install git+https://github.com/authsec-ai/sdk-authsec.git";
  };

  const getImportCode = () => {
    return "from AuthSec_SDK import protected_by_AuthSec, run_mcp_server_with_oauth";
  };

  const getDecoratorCode = () => {
    return '@protected_by_AuthSec("function-name")';
  };

  const getExampleCode = () => {
    const clientIdForExample = createdClientId ? `${createdClientId}` : "your-client-id";
    return `import json
import math
from datetime import datetime

# Import your auth package and OAuth SDK
from AuthSec_SDK import protected_by_AuthSec, run_mcp_server_with_oauth


@protected_by_AuthSec("secure_calculator")
async def secure_calculator(arguments: dict) -> list:
    """Advanced calculator with user history tracking"""
    user_info = arguments.get("_user_info", {})
    
    operation = arguments.get("operation")
    num1 = float(arguments.get("num1"))
    num2 = float(arguments.get("num2", 1))
    
    operations = {
        "add": lambda a, b: a + b,
        "subtract": lambda a, b: a - b,
        "multiply": lambda a, b: a * b,
        "divide": lambda a, b: a / b if b != 0 else "Cannot divide by zero",
        "power": lambda a, b: a ** b,
        "sqrt": lambda a, b: math.sqrt(a),
        "sin": lambda a, b: math.sin(math.radians(a)),
        "cos": lambda a, b: math.cos(math.radians(a)),
        "log": lambda a, b: math.log10(a) if a > 0 else "Cannot log negative numbers"
    }
    
    if operation not in operations:
        return [{"type": "text", "text": json.dumps({
            "error": f"Unknown operation: {operation}",
            "available_operations": list(operations.keys())
        })}]
    
    try:
        result_value = operations[operation](num1, num2)
        
        calculation = {
            "operation": operation,
            "input": {"num1": num1, "num2": num2 if operation not in ["sqrt", "sin", "cos", "log"] else None},
            "result": result_value,
            "calculated_by": user_info.get("email", "unknown"),
            "org_id": user_info.get("org_id"),
            "timestamp": datetime.now().isoformat(),
            "session_id": arguments.get("session_id", "")[:8] + "...",
            "security_note": "Calculation history is tied to your authenticated account"
        }
        
        return [{"type": "text", "text": json.dumps(calculation, indent=2)}]
        
    except Exception as e:
        return [{"type": "text", "text": json.dumps({
            "error": f"Calculation failed: {str(e)}"
        })}]



# MAIN SERVER STARTUP (User just runs this)
if __name__ == "__main__":
    
    import sys
    # Run MCP server with OAuth protection - using same CLIENT_ID
    run_mcp_server_with_oauth( user_module=sys.modules[__name__], client_id="${clientIdForExample}", app_name="Demo Business Tools Server", host="0.0.0.0", port=3005)`;
  };

  const getRunCode = () => {
    const effectiveClientId = createdClientId;
    const clientId = effectiveClientId ? `${effectiveClientId}` : "your-client-id";
    return `if __name__ == "__main__":
    import sys
    run_mcp_server_with_oauth( user_module=sys.modules[__name__], client_id="${clientId}", app_name="your-server-name", host="0.0.0.0", port=3005)`;
  };

  const getProtectionCode = () => {
    const safeClientName = clientName.replace(/\s+/g, "_").toLowerCase() || "my_client";

    if (clientType === "MCP-Server") {
      return `import json
from datetime import datetime

async def my_protected_function(arguments: dict, session) -> list:
    """
    Template for creating MCP protected functions.
    Replace the 'Your logic here' section with your own code.
    """
 
    # Step 1: Access authenticated user/session info
    user_info = session.user_info or {}
    org_id = session.org_id or "unknown"
    tenant_id = session.tenant_id or "unknown"
    provider = session.provider or "unknown"
 
    # Step 2: Extract and validate arguments
    param1 = arguments.get("param1")           # Required parameter
    param2 = arguments.get("param2", None)     # Optional parameter
 
    if not param1:
        return [{
            "type": "text",
            "text": json.dumps({
                "error": "Missing required parameter: param1"
            })
        }]
 
    # Step 3: Your logic here
    try:
        # Example operation
        result_value = f"Processed {param1} with {param2}" if param2 else f"Processed {param1}"
 
        # Step 4: Prepare secure structured output
        return [{
            "type": "text",
            "text": json.dumps({
                "operation": "example_operation",
                "input": arguments,
                "result": result_value,
                "performed_by": user_info.get("email", "unknown"),
                "org_id": org_id,
                "tenant_id": tenant_id,
                "provider": provider,
                "timestamp": datetime.now().isoformat(),
                "session_id": (session.session_id[:8] + "...") if getattr(session, "session_id", None) else "unknown",
                "security_note": "🔒 This operation is tied to your authenticated account"
            }, indent=2)
        }]
 
    except Exception as e:
        return [{
            "type": "text",
            "text": json.dumps({
                "error": f"Operation failed: {str(e)}"
            })
        }]
 
# =========================================================
# Protected Tool Registration
# =========================================================
 
${safeClientName}.add_protected_tool(
    name="my_protected_function",  # Unique tool name
    description="Describe what your function does here",
    input_schema={
        "type": "object",
        "properties": {
            "param1": {
                "type": "string",
                "description": "First required parameter"
            },
            "param2": {
                "type": "string",
                "description": "Optional second parameter"
            }
        },
        "required": ["param1"]
    },
    handler=my_protected_function
)`;
    } else if (clientType === "AI-Agent") {
      return `@self.auth_client.requires_auth
async def process_request(self, user_message: str):
    """Process requests with authentication"""
    
    # Verify permissions
    if not self.auth_client.verify_permissions(["chat"]):
        return {"error": "Insufficient permissions"}
    
    # Your AI logic here
    response = await self.openai_client.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": user_message}]
    )
    
    return {"response": response.choices[0].message.content}`;
    }
    return "# Select a client type above to see protection code";
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
      <div className="container mx-auto px-8 py-6 max-w-7xl">
        {/* Page Header */}
        <div className="mb-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => (isExistingClient ? navigate("/clients") : navigate(-1))}
              className="h-8 px-2"
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-2xl font-semibold">
                {isExistingClient ? "SDK Integration" : "Onboard New Client"}
              </h1>
              <p className="text-sm text-foreground mt-1">
                {isExistingClient
                  ? `SDK setup instructions for ${existingClientName || "your client"}`
                  : "Create a client and integrate the SDK"}
              </p>
            </div>
          </div>
        </div>

        {/* Timeline Layout */}
        <div className="relative">
          {/* Vertical Timeline Line */}
          <div className="absolute left-6 top-0 bottom-0 w-0.5 bg-gradient-to-b from-primary/50 via-primary/30 to-transparent" />

          {/* Step 1: Create Client */}
          <div className="relative flex gap-6 mb-6">
            {/* Step Circle */}
            <div className="relative z-10 flex-shrink-0">
              <div
                className={`w-12 h-12 rounded-full border-2 flex items-center justify-center transition-all ${
                  createdClientId || isExistingClient
                    ? "bg-primary border-primary shadow-lg shadow-primary/20"
                    : "bg-background border-primary/30"
                }`}
              >
                {createdClientId || isExistingClient ? (
                  <Check className="h-6 w-6 text-primary-foreground" />
                ) : (
                  <span className="text-sm font-semibold text-foreground">1</span>
                )}
              </div>
            </div>

            {/* Step Content */}
            <div className="flex-1 pt-2">
              <p className="text-sm font-medium text-foreground/70 mb-3">Create Client</p>

              {!isExistingClient && !createdClientId ? (
                <form
                  id="create-client-form"
                  className="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-3xl"
                  onSubmit={async (e) => {
                    e.preventDefault();
                    if (!clientName || !email) {
                      toast.error("Please provide name and email");
                      return;
                    }
                    if (!tenantId || !projectId) {
                      toast.error("Missing tenant or project. Please sign in.");
                      return;
                    }
                    try {
                      const res = await registerClient({
                        name: clientName,
                        email,
                        tenant_id: tenantId,
                        project_id: projectId,
                        react_app_url: window.location.origin,
                      }).unwrap();
                      setCreatedClientId(res.client_id);
                      toast.success("Client created successfully!");
                    } catch (err: any) {
                      console.error(err);
                      const msg = err?.data?.message || err?.error || "Failed to create client";
                      toast.error(msg);
                    }
                  }}
                >
                  <div className="space-y-2">
                    <Label htmlFor="clientName">Client Name</Label>
                    <Input
                      id="clientName"
                      placeholder="e.g., My MCP Server"
                      value={clientName}
                      onChange={(e) => setClientName(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="email">Contact Email</Label>
                    <Input
                      id="email"
                      type="email"
                      placeholder="you@example.com"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>Tenant ID</Label>
                    <Input value={tenantId} readOnly className="bg-muted/50" />
                  </div>
                  <div className="space-y-2">
                    <Label>Project ID</Label>
                    <Input value={projectId} readOnly className="bg-muted/50" />
                  </div>
                  <div className="md:col-span-2">
                    <Button type="submit" className="w-full md:w-auto">
                      Create Client & Continue
                    </Button>
                  </div>
                </form>
              ) : (
                <div className="bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded-lg p-3 max-w-3xl">
                  <div className="flex items-center gap-3">
                    <Shield className="h-5 w-5 text-green-600 dark:text-green-400" />
                    <div>
                      <p className="font-medium text-sm text-green-800 dark:text-green-200">
                        Client Created
                      </p>
                      <p className="text-xs text-green-700 dark:text-green-300">
                        Client ID: <span className="font-mono">{createdClientId || clientId}</span>
                      </p>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Step 2: Choose Integration Type */}
          {(createdClientId || isExistingClient) && (
            <div className="relative flex gap-6 mb-6">
              {/* Step Circle */}
              <div className="relative z-10 flex-shrink-0">
                <div
                  className={`w-12 h-12 rounded-full border-2 flex items-center justify-center transition-all ${
                    clientType
                      ? "bg-primary border-primary shadow-lg shadow-primary/20"
                      : "bg-background border-primary/30"
                  }`}
                >
                  {clientType ? (
                    <Check className="h-6 w-6 text-primary-foreground" />
                  ) : (
                    <span className="text-sm font-semibold text-foreground">2</span>
                  )}
                </div>
              </div>

              {/* Step Content */}
              <div className="flex-1 pt-2">
                <p className="text-sm font-medium text-foreground/70 mb-3">
                  {!clientType ? (
                    "Select your client type"
                  ) : (
                    <span>
                      Selected:{" "}
                      <span className="font-semibold text-foreground">
                        {clientType === "MCP-Server" ? "MCP Server" : "AI Agent"}
                      </span>
                      {" · "}
                      <button
                        onClick={() => setClientType(null)}
                        className="text-primary hover:underline text-sm"
                      >
                        Change
                      </button>
                    </span>
                  )}
                </p>

                {!clientType ? (
                  <div className="inline-flex gap-2 p-1 bg-muted/50 rounded-lg max-w-3xl border border-border">
                    <button
                      onClick={() => setClientType("MCP-Server")}
                      className="flex items-center gap-2 px-4 py-2.5 rounded-md hover:bg-background transition-colors text-left flex-1"
                    >
                      <Server className="h-5 w-5 text-primary flex-shrink-0" />
                      <div>
                        <h3 className="font-medium text-sm">MCP Server</h3>
                        <p className="text-xs text-foreground">Model Context Protocol</p>
                      </div>
                    </button>
                    <button
                      onClick={() => setClientType("AI-Agent")}
                      className="flex items-center gap-2 px-4 py-2.5 rounded-md hover:bg-background transition-colors text-left flex-1"
                    >
                      <Bot className="h-5 w-5 text-primary flex-shrink-0" />
                      <div>
                        <h3 className="font-medium text-sm">AI Agent</h3>
                        <p className="text-xs text-foreground">Agents & Chatbots</p>
                      </div>
                    </button>
                  </div>
                ) : null}
              </div>
            </div>
          )}

          {/* Step 3: Install SDK */}
          {(createdClientId || isExistingClient) && clientType && (
            <div className="relative flex gap-6 mb-6">
              {/* Step Circle */}
              <div className="relative z-10 flex-shrink-0">
                <div className="w-12 h-12 rounded-full border-2 bg-background border-primary/30 flex items-center justify-center">
                  <span className="text-sm font-semibold text-foreground">3</span>
                </div>
              </div>

              {/* Step Content */}
              <div className="flex-1 pt-2">
                <p className="text-sm font-medium text-foreground/70 mb-3">Install SDK</p>

                <div className="max-w-3xl">
                  <CodeBlock
                    code={`$ ${getInstallCommand()}`}
                    label="Terminal"
                    onCopy={() => handleCopy(getInstallCommand(), "install")}
                    copied={copiedSteps.has("install")}
                  />
                </div>
              </div>
            </div>
          )}

          {/* Step 4-6: MCP Server Implementation */}
          {(createdClientId || isExistingClient) && clientType === "MCP-Server" && (
            <>
              {/* Step 4: Import */}
              <div className="relative flex gap-6 mb-6">
                <div className="relative z-10 flex-shrink-0">
                  <div className="w-12 h-12 rounded-full border-2 bg-background border-primary/30 flex items-center justify-center">
                    <span className="text-sm font-semibold text-foreground">4</span>
                  </div>
                </div>
                <div className="flex-1 pt-2">
                  <p className="text-sm font-medium text-foreground/70 mb-3">
                    Import SDK Components
                  </p>

                  <div className="max-w-3xl">
                    <CodeBlock
                      code={getImportCode()}
                      label="Python"
                      onCopy={() => handleCopy(getImportCode(), "import")}
                      copied={copiedSteps.has("import")}
                    />
                  </div>
                </div>
              </div>

              {/* Step 5: Decorator */}
              <div className="relative flex gap-6 mb-6">
                <div className="relative z-10 flex-shrink-0">
                  <div className="w-12 h-12 rounded-full border-2 bg-background border-primary/30 flex items-center justify-center">
                    <span className="text-sm font-semibold text-foreground">5</span>
                  </div>
                </div>
                <div className="flex-1 pt-2">
                  <p className="text-sm font-medium text-foreground/70 mb-3">
                    Protect Your Functions
                  </p>

                  <div className="max-w-3xl">
                    <CodeBlock
                      code={getDecoratorCode()}
                      label="Decorator"
                      onCopy={() => handleCopy(getDecoratorCode(), "decorator")}
                      copied={copiedSteps.has("decorator")}
                      onExpand={() => setShowExample(true)}
                    />
                    <p className="text-xs text-foreground mt-2">
                      Replace "function-name" with your actual function name.{" "}
                      <button
                        onClick={() => setShowExample(true)}
                        className="text-primary hover:underline inline-flex items-center gap-1"
                      >
                        <Eye className="h-3 w-3" />
                        View full example
                      </button>
                    </p>
                  </div>
                </div>
              </div>

              {/* Step 6: Run Server */}
              <div className="relative flex gap-6 mb-6">
                <div className="relative z-10 flex-shrink-0">
                  <div className="w-12 h-12 rounded-full border-2 bg-background border-primary/30 flex items-center justify-center">
                    <span className="text-sm font-semibold text-foreground">6</span>
                  </div>
                </div>
                <div className="flex-1 pt-2">
                  <p className="text-sm font-medium text-foreground/70 mb-3">Run Your Server</p>

                  <div className="max-w-3xl space-y-2">
                    <CodeBlock
                      code={getRunCode()}
                      label="Server Startup"
                      onCopy={() => handleCopy(getRunCode(), "run")}
                      copied={copiedSteps.has("run")}
                    />

                    <div className="bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded-lg p-3">
                      <h4 className="font-medium text-sm text-green-800 dark:text-green-200 mb-1 flex items-center gap-2">
                        <Check className="h-4 w-4" />
                        All Set!
                      </h4>
                      <p className="text-xs text-green-700 dark:text-green-300">
                        Your MCP server now has enterprise-grade authentication. Users will be
                        securely authenticated before accessing your protected functions.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </>
          )}

          {/* AI Agent Implementation */}
          {(createdClientId || isExistingClient) && clientType === "AI-Agent" && (
            <div className="relative flex gap-6 mb-6">
              <div className="relative z-10 flex-shrink-0">
                <div className="w-12 h-12 rounded-full border-2 bg-background border-primary/30 flex items-center justify-center">
                  <span className="text-sm font-semibold text-foreground">4</span>
                </div>
              </div>
              <div className="flex-1 pt-2">
                <p className="text-sm font-medium text-foreground/70 mb-3">Add Protection</p>

                <div className="max-w-3xl space-y-2">
                  <CodeBlock
                    code={getProtectionCode()}
                    label="Protected Function"
                    onCopy={() => handleCopy(getProtectionCode(), "protection")}
                    copied={copiedSteps.has("protection")}
                    showTabs={true}
                  />

                  <div className="bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded-lg p-3">
                    <h4 className="font-medium text-sm text-green-800 dark:text-green-200 mb-1 flex items-center gap-2">
                      <Check className="h-4 w-4" />
                      All Set!
                    </h4>
                    <p className="text-xs text-green-700 dark:text-green-300">
                      Your AI agent now has enterprise-grade authentication. Users will be securely
                      authenticated before accessing your protected functions.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Voice Agent Integration Section */}
        {(createdClientId || isExistingClient) && clientType && (
          <div className="mt-8 max-w-3xl">
            <div className="p-6 rounded-xl border border-primary/20 bg-gradient-to-br from-primary/5 via-background to-primary/5">
              <div className="flex items-start gap-4">
                <div className="p-3 rounded-xl bg-primary/10 border border-primary/20">
                  <Mic className="h-6 w-6 text-primary" />
                </div>
                <div className="flex-1">
                  <h3 className="text-lg font-semibold mb-1">Voice Agent Integration</h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    Enable passwordless authentication for voice assistants using CIBA (push
                    notifications) and TOTP (6-digit codes). Perfect for hands-free authentication
                    flows.
                  </p>

                  <div className="flex flex-wrap gap-3 mb-4">
                    <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-background border border-border">
                      <Smartphone className="h-4 w-4 text-primary" />
                      <span className="text-xs font-medium">Push Notifications</span>
                    </div>
                    <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-background border border-border">
                      <KeyRound className="h-4 w-4 text-primary" />
                      <span className="text-xs font-medium">TOTP Fallback</span>
                    </div>
                  </div>

                  <div className="border border-border/60 rounded-lg overflow-hidden bg-background/50 mb-4">
                    <div className="flex items-center justify-between border-b border-border/50 bg-muted/40 px-2.5 py-1">
                      <span className="text-[11px] font-semibold text-foreground/75">
                        Quick Start
                      </span>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-5 w-5 p-0 hover:bg-background/60 text-foreground/60 hover:text-foreground"
                        onClick={() => {
                          const code = `from AuthSec_SDK import CIBAClient

client = CIBAClient(client_id="${createdClientId || clientId}")

# Send push notification
result = client.initiate_app_approval("user@example.com")
approval = client.poll_for_approval("user@example.com", result["auth_req_id"])`;
                          navigator.clipboard.writeText(code);
                          toast.success("Copied to clipboard");
                        }}
                      >
                        <Copy className="h-2.5 w-2.5" />
                      </Button>
                    </div>
                    <div className="p-3 bg-background/30">
                      <pre className="text-xs font-mono text-foreground/90 whitespace-pre-wrap leading-relaxed">
                        {`from AuthSec_SDK import CIBAClient

client = CIBAClient(client_id="${createdClientId || clientId}")

# Send push notification
result = client.initiate_app_approval("user@example.com")
approval = client.poll_for_approval("user@example.com", result["auth_req_id"])`}
                      </pre>
                    </div>
                  </div>

                  <Button
                    onClick={() =>
                      navigate(`/clients/voice-agent?clientId=${createdClientId || clientId}`)
                    }
                    variant="outline"
                    className="admin-tonal-cta gap-2"
                    data-tone="voice"
                  >
                    <Mic className="h-4 w-4" />
                    Configure Voice Agent
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Client ID Panel removed per requirements */}

        {/* No registration actions; simple tutorial only */}

        {/* Wizard "Done" Button */}
        {isFromWizard && (createdClientId || isExistingClient) && clientType && (
          <div className="flex justify-center mt-8">
            <Button
              onClick={handleDone}
              size="sm"
              variant="outline"
              className="bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
            >
              Complete
              <ChevronRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        )}

        {/* Example Code Modal */}
        {showExample && (
          <div
            className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50"
            onClick={() => setShowExample(false)}
          >
            <div
              className="bg-background rounded-lg shadow-xl max-w-4xl w-full max-h-[80vh] overflow-hidden flex flex-col"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex items-center justify-between p-4 border-b flex-shrink-0">
                <h3 className="text-lg font-semibold">Complete MCP Server Example</h3>
                <Button size="sm" variant="ghost" onClick={() => setShowExample(false)}>
                  <X className="h-4 w-4" />
                </Button>
              </div>
              <div className="p-4 overflow-y-auto">
                <CodeBlock
                  code={getExampleCode()}
                  label="example_server.py"
                  onCopy={() => handleCopy(getExampleCode(), "example")}
                  copied={copiedSteps.has("example")}
                />
                <p className="text-center text-sm text-foreground mt-3">
                  This is a complete working example with a secure calculator function
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

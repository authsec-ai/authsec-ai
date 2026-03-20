"use client";

import React, { useState, useEffect, useCallback, useMemo } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { Switch } from "../../components/ui/switch";
import {
  ArrowLeft,
  CheckCircle,
  Loader2,
  Check,
  ChevronRight,
  X,
  Mic,
  Smartphone,
  KeyRound,
  Copy,
  Bot,
  Phone,
  MessageSquare,
  Zap,
  Settings,
  Code,
  Sparkles,
} from "lucide-react";
import { toast } from "../../lib/toast";
import { cn } from "../../lib/utils";
import { useGetAllClientsQuery, type EnhancedClientData } from "../../app/api/clientApi";
import { SessionManager } from "../../utils/sessionManager";

// ============================================================================
// Types & Constants
// ============================================================================

interface VoiceAgentConfig {
  clientId: string;
  clientName: string;
  cibaEnabled: boolean;
  totpEnabled: boolean;
}

const WIZARD_STEPS = [
  { id: "client", label: "Select Client", icon: Bot },
  { id: "methods", label: "Auth Methods", icon: Settings },
  { id: "integration", label: "Integration", icon: Code },
];

// ============================================================================
// Sub-Components
// ============================================================================

// Client Selection Card
const ClientCard = ({
  client,
  selected,
  onSelect,
}: {
  client: EnhancedClientData;
  selected: boolean;
  onSelect: () => void;
}) => (
  <button
    type="button"
    onClick={onSelect}
    className={cn(
      "group relative flex items-start gap-4 overflow-hidden rounded-xl border p-4 text-left transition-all duration-200",
      "hover:border-primary/60 hover:shadow-md focus-visible:outline-2 focus-visible:outline-primary/60",
      selected
        ? "border-primary bg-primary/5 ring-2 ring-primary/20 shadow-md"
        : "border-border/60 bg-card/50 hover:bg-card/80",
    )}
  >
    <div
      className={cn(
        "flex h-12 w-12 shrink-0 items-center justify-center rounded-xl transition-colors",
        selected
          ? "bg-primary/15 text-primary"
          : "bg-muted/80 text-muted-foreground group-hover:bg-muted",
      )}
    >
      <Bot className="h-6 w-6" />
    </div>
    <div className="flex-1 min-w-0">
      <div className="flex items-center gap-2 mb-1">
        <span className="font-semibold text-[15px] text-foreground truncate">
          {client.name || client.client_name || "Unnamed Client"}
        </span>
        {client.active && (
          <Badge variant="secondary" className="text-[10px] px-1.5 py-0 h-5">
            Active
          </Badge>
        )}
      </div>
      <p className="text-xs text-muted-foreground font-mono truncate">{client.client_id}</p>
    </div>
    {selected && (
      <span className="absolute right-3 top-3 inline-flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-sm">
        <Check className="h-3.5 w-3.5" />
      </span>
    )}
  </button>
);

// Auth Method Card
const AuthMethodCard = ({
  icon: Icon,
  title,
  description,
  badge,
  badgeVariant = "secondary",
  features,
  enabled,
  onToggle,
}: {
  icon: React.ElementType;
  title: string;
  description: string;
  badge?: string;
  badgeVariant?: "secondary" | "outline";
  features: { icon: React.ElementType; label: string }[];
  enabled: boolean;
  onToggle: (enabled: boolean) => void;
}) => (
  <div
    className={cn(
      "relative rounded-xl border p-5 transition-all duration-200",
      enabled ? "border-primary/50 bg-primary/[0.03] shadow-sm" : "border-border/60 bg-card/30",
    )}
  >
    <div className="flex items-start justify-between gap-4">
      <div className="flex items-start gap-4">
        <div
          className={cn(
            "flex h-12 w-12 shrink-0 items-center justify-center rounded-xl transition-colors",
            enabled ? "bg-primary/15 text-primary" : "bg-muted/80 text-muted-foreground",
          )}
        >
          <Icon className="h-6 w-6" />
        </div>
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <h4 className="font-semibold text-[15px] text-foreground">{title}</h4>
            {badge && (
              <Badge variant={badgeVariant} className="text-[10px] px-1.5 py-0 h-5">
                {badge}
              </Badge>
            )}
          </div>
          <p className="text-sm text-muted-foreground leading-relaxed max-w-md">{description}</p>
          <div className="flex flex-wrap gap-2 pt-2">
            {features.map((feature, i) => (
              <div
                key={i}
                className={cn(
                  "flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-colors",
                  enabled ? "bg-primary/10 text-primary" : "bg-muted/60 text-muted-foreground",
                )}
              >
                <feature.icon className="h-3 w-3" />
                {feature.label}
              </div>
            ))}
          </div>
        </div>
      </div>
      <Switch checked={enabled} onCheckedChange={onToggle} className="mt-1" />
    </div>
  </div>
);

// Code Block Component
const CodeBlock = ({
  code,
  label,
  onCopy,
  copied,
}: {
  code: string;
  label?: string;
  onCopy?: () => void;
  copied?: boolean;
}) => (
  <div className="rounded-xl border border-border/60 overflow-hidden bg-card/50 shadow-sm">
    <div className="flex items-center justify-between border-b border-border/40 bg-muted/30 px-4 py-2">
      <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
        {label}
      </span>
      {onCopy && (
        <Button
          size="sm"
          variant="ghost"
          className="h-7 px-2 gap-1.5 text-xs text-muted-foreground hover:text-foreground"
          onClick={onCopy}
        >
          {copied ? (
            <>
              <Check className="h-3 w-3 text-green-500" />
              Copied
            </>
          ) : (
            <>
              <Copy className="h-3 w-3" />
              Copy
            </>
          )}
        </Button>
      )}
    </div>
    <div className="p-4 overflow-x-auto bg-[color-mix(in_oklab,var(--background)_97%,black)]">
      <pre className="text-[13px] font-mono text-foreground/90 whitespace-pre leading-relaxed">
        {code}
      </pre>
    </div>
  </div>
);

// ============================================================================
// Main Page Component
// ============================================================================

export default function VoiceAgentWizardPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  // State
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [config, setConfig] = useState<VoiceAgentConfig>({
    clientId: "",
    clientName: "",
    cibaEnabled: true,
    totpEnabled: true,
  });
  const [copiedSteps, setCopiedSteps] = useState<Set<string>>(new Set());

  const currentStep = WIZARD_STEPS[currentStepIndex];

  // Get tenant ID
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id || "";

  // Fetch clients
  const { data: clientsResponse, isLoading: clientsLoading } = useGetAllClientsQuery(
    { tenant_id: tenantId },
    { skip: !tenantId },
  );

  const clients = useMemo(() => clientsResponse?.clients || [], [clientsResponse]);

  // Auto-select client from URL
  useEffect(() => {
    const preSelectedClientId = searchParams.get("clientId");
    if (preSelectedClientId && clients.length > 0 && !config.clientId) {
      const client = clients.find((c) => c.client_id === preSelectedClientId);
      if (client) {
        setConfig((prev) => ({
          ...prev,
          clientId: client.client_id,
          clientName: client.name || client.client_name || "Unnamed",
        }));
        // Skip to methods step if client pre-selected
        setCurrentStepIndex(1);
      }
    }
  }, [searchParams, clients, config.clientId]);

  // Copy handler
  const handleCopy = useCallback((text: string, stepId: string) => {
    navigator.clipboard.writeText(text);
    setCopiedSteps((prev) => new Set([...prev, stepId]));
    toast.success("Copied to clipboard");
    setTimeout(() => {
      setCopiedSteps((prev) => {
        const newSet = new Set(prev);
        newSet.delete(stepId);
        return newSet;
      });
    }, 2000);
  }, []);

  // Navigation
  const handleBack = useCallback(() => {
    if (currentStepIndex > 0) {
      setCurrentStepIndex(currentStepIndex - 1);
    } else {
      navigate("/clients");
    }
  }, [currentStepIndex, navigate]);

  const handleNext = useCallback(() => {
    if (currentStepIndex === 0 && !config.clientId) {
      toast.error("Please select a client");
      return;
    }
    if (currentStepIndex === 1 && !config.cibaEnabled && !config.totpEnabled) {
      toast.error("Please enable at least one authentication method");
      return;
    }
    if (currentStepIndex < WIZARD_STEPS.length - 1) {
      setCurrentStepIndex(currentStepIndex + 1);
    }
  }, [currentStepIndex, config]);

  const handleFinish = useCallback(() => {
    toast.success("Voice agent configuration complete!");
    navigate("/clients");
  }, [navigate]);

  const canProceed = useMemo(() => {
    if (currentStepIndex === 0) return Boolean(config.clientId);
    if (currentStepIndex === 1) return config.cibaEnabled || config.totpEnabled;
    return true;
  }, [currentStepIndex, config]);

  // Code snippets
  const getInstallCode = () => `pip install git+https://github.com/authsec-ai/sdk-authsec.git`;

  const getCibaCode = () => `from AuthSec_SDK import CIBAClient

# Initialize with your client ID
client = CIBAClient(client_id="${config.clientId || "your-client-id"}")

# Send push notification to user's AuthSec mobile app
result = client.initiate_app_approval("user@example.com")

# Wait for user approval (blocks until response or timeout)
approval = client.poll_for_approval(
    email="user@example.com",
    auth_req_id=result["auth_req_id"],
    timeout=60  # seconds
)

if approval["status"] == "approved":
    print(f"✅ Authenticated! Token: {approval['token'][:50]}...")
elif approval["status"] == "access_denied":
    print("❌ User denied the request")
elif approval["status"] == "timeout":
    print("⏱️ Request timed out")`;

  const getTotpCode = () => `# TOTP Verification (6-digit code fallback)
result = client.verify_totp("user@example.com", "123456")

if result["success"]:
    print(f"✅ Valid! Token: {result['token'][:50]}...")
else:
    print(f"❌ Invalid. {result['remaining']} attempts remaining")`;

  const getVoiceAssistantCode = () => `from AuthSec_SDK import CIBAClient

class VoiceAssistant:
    def __init__(self):
        self.ciba = CIBAClient(client_id="${config.clientId || "your-client-id"}")
    
    def authenticate_user(self, email: str) -> str | None:
        """Handle voice authentication with CIBA + TOTP fallback"""
        
        method = self.ask_user("Approve via app or use a code?")
        
        if "app" in method.lower():
            # CIBA: Push notification flow
            self.speak("I've sent a notification to your AuthSec app.")
            result = self.ciba.initiate_app_approval(email)
            approval = self.ciba.poll_for_approval(
                email, result["auth_req_id"], timeout=60
            )
            
            if approval["status"] == "approved":
                self.speak("You're authenticated!")
                return approval["token"]
            self.speak(f"Authentication {approval['status']}.")
            return None
        else:
            # TOTP: 6-digit code flow
            self.speak("Please say your 6-digit code.")
            code = self.listen_for_digits()
            result = self.ciba.verify_totp(email, code)
            
            if result["success"]:
                self.speak("You're authenticated!")
                return result["token"]
            self.speak(f"Invalid. {result['remaining']} attempts left.")
            return None`;

  // Step subtitle
  const getStepSubtitle = () => {
    switch (currentStep.id) {
      case "client":
        return "Choose which client to configure for voice authentication";
      case "methods":
        return "Select the authentication methods for your voice agent";
      case "integration":
        return "Copy the integration code to get started";
      default:
        return "";
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-64px)] w-full bg-gradient-to-b from-background to-muted/20">
      {/* Fixed Header */}
      <div className="flex-shrink-0 border-b border-border/60 bg-background/80 backdrop-blur-sm py-5 px-8">
        <div className="flex items-center justify-between max-w-5xl mx-auto">
          <div className="flex items-center gap-4">
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary/10 text-primary">
              <Mic className="h-5 w-5" />
            </div>
            <div>
              <h1 className="text-xl font-semibold tracking-tight">Configure Voice Agent</h1>
              <p className="text-sm text-muted-foreground mt-0.5">{getStepSubtitle()}</p>
            </div>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate("/clients")}
            className="h-9 w-9 rounded-full text-muted-foreground hover:text-foreground hover:bg-muted"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Scrollable Content Area */}
      <div className="flex-1 overflow-y-auto px-8 py-8 min-h-0">
        <div className="max-w-5xl mx-auto">
          {/* Step 0: Client Selection */}
          {currentStepIndex === 0 && (
            <div className="space-y-6">
              {/* Info Banner */}
              <div className="flex items-start gap-4 p-4 rounded-xl bg-primary/5 border border-primary/20">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/15 text-primary">
                  <Sparkles className="h-5 w-5" />
                </div>
                <div>
                  <h3 className="font-semibold text-sm text-foreground">
                    Voice Agent Authentication
                  </h3>
                  <p className="text-sm text-muted-foreground mt-1 leading-relaxed">
                    Enable passwordless authentication for voice assistants using CIBA (push
                    notifications) and TOTP (6-digit codes). Perfect for hands-free, IoT, and
                    conversational interfaces.
                  </p>
                </div>
              </div>

              {/* Client Selection */}
              <div className="space-y-3">
                <h3 className="text-sm font-semibold text-foreground">Select a Client</h3>
                {clientsLoading ? (
                  <div className="flex items-center justify-center py-12">
                    <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                  </div>
                ) : clients.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-12 text-center">
                    <Bot className="h-12 w-12 text-muted-foreground/50 mb-3" />
                    <p className="text-sm text-muted-foreground">
                      No clients found. Create a client first.
                    </p>
                    <Button
                      variant="outline"
                      className="admin-tonal-cta mt-4"
                      data-tone="voice"
                      onClick={() => navigate("/clients")}
                    >
                      Go to Clients
                    </Button>
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {clients.map((client) => (
                      <ClientCard
                        key={client.client_id}
                        client={client}
                        selected={config.clientId === client.client_id}
                        onSelect={() =>
                          setConfig((prev) => ({
                            ...prev,
                            clientId: client.client_id,
                            clientName: client.name || client.client_name || "Unnamed",
                          }))
                        }
                      />
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Step 1: Auth Methods */}
          {currentStepIndex === 1 && (
            <div className="space-y-6">
              {/* Selected Client Summary */}
              <div className="flex items-center gap-3 p-4 rounded-xl bg-muted/50 border border-border/60">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Bot className="h-5 w-5" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-semibold text-sm text-foreground">{config.clientName}</p>
                  <p className="text-xs text-muted-foreground font-mono truncate">
                    {config.clientId}
                  </p>
                </div>
                <Badge variant="secondary" className="text-xs">
                  Selected
                </Badge>
              </div>

              {/* Best Practice Banner */}
              <div className="flex items-start gap-3 p-4 rounded-xl bg-blue-500/5 border border-blue-500/20">
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-blue-500/15 text-blue-600 dark:text-blue-400">
                  <Sparkles className="h-4 w-4" />
                </div>
                <div>
                  <p className="text-sm font-medium text-blue-700 dark:text-blue-300">
                    Recommended: Enable both methods
                  </p>
                  <p className="text-xs text-blue-600/80 dark:text-blue-400/80 mt-0.5">
                    TOTP provides a fallback when users don't have the AuthSec mobile app installed.
                  </p>
                </div>
              </div>

              {/* Auth Method Cards */}
              <div className="space-y-4">
                <AuthMethodCard
                  icon={Smartphone}
                  title="CIBA Push Notifications"
                  description="Send push notifications to the user's AuthSec mobile app for seamless, hands-free approval."
                  badge="Recommended"
                  badgeVariant="secondary"
                  features={[
                    { icon: Phone, label: "Voice Assistants" },
                    { icon: Zap, label: "IoT Devices" },
                    { icon: Sparkles, label: "Hands-free" },
                  ]}
                  enabled={config.cibaEnabled}
                  onToggle={(enabled) => setConfig((prev) => ({ ...prev, cibaEnabled: enabled }))}
                />

                <AuthMethodCard
                  icon={KeyRound}
                  title="TOTP 6-Digit Codes"
                  description="Verify time-based one-time passwords from authenticator apps as a reliable fallback option."
                  badge="Fallback"
                  badgeVariant="outline"
                  features={[
                    { icon: MessageSquare, label: "CLI Tools" },
                    { icon: Zap, label: "Backup Auth" },
                  ]}
                  enabled={config.totpEnabled}
                  onToggle={(enabled) => setConfig((prev) => ({ ...prev, totpEnabled: enabled }))}
                />
              </div>
            </div>
          )}

          {/* Step 2: Integration Code */}
          {currentStepIndex === 2 && (
            <div className="space-y-6">
              {/* Configuration Summary */}
              <div className="flex items-center gap-4 p-4 rounded-xl bg-green-500/5 border border-green-500/20">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-green-500/15 text-green-600 dark:text-green-400">
                  <CheckCircle className="h-5 w-5" />
                </div>
                <div className="flex-1">
                  <p className="text-sm font-medium text-green-700 dark:text-green-300">
                    Configuration Complete
                  </p>
                  <p className="text-xs text-green-600/80 dark:text-green-400/80 mt-0.5">
                    {config.clientName} • {config.cibaEnabled && "CIBA"}
                    {config.cibaEnabled && config.totpEnabled && " + "}
                    {config.totpEnabled && "TOTP"}
                  </p>
                </div>
              </div>

              {/* Install */}
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/15 text-primary text-xs font-semibold">
                    1
                  </span>
                  <h3 className="text-sm font-semibold text-foreground">Install the SDK</h3>
                </div>
                <CodeBlock
                  code={`$ ${getInstallCode()}`}
                  label="Terminal"
                  onCopy={() => handleCopy(getInstallCode(), "install")}
                  copied={copiedSteps.has("install")}
                />
              </div>

              {/* CIBA Code */}
              {config.cibaEnabled && (
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/15 text-primary text-xs font-semibold">
                      2
                    </span>
                    <h3 className="text-sm font-semibold text-foreground">
                      CIBA Push Authentication
                    </h3>
                  </div>
                  <CodeBlock
                    code={getCibaCode()}
                    label="Python"
                    onCopy={() => handleCopy(getCibaCode(), "ciba")}
                    copied={copiedSteps.has("ciba")}
                  />
                </div>
              )}

              {/* TOTP Code */}
              {config.totpEnabled && (
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/15 text-primary text-xs font-semibold">
                      {config.cibaEnabled ? "3" : "2"}
                    </span>
                    <h3 className="text-sm font-semibold text-foreground">
                      TOTP Code Verification
                    </h3>
                  </div>
                  <CodeBlock
                    code={getTotpCode()}
                    label="Python"
                    onCopy={() => handleCopy(getTotpCode(), "totp")}
                    copied={copiedSteps.has("totp")}
                  />
                </div>
              )}

              {/* Voice Assistant Example */}
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/15 text-primary">
                    <Mic className="h-3 w-3" />
                  </span>
                  <h3 className="text-sm font-semibold text-foreground">
                    Complete Voice Assistant Example
                  </h3>
                </div>
                <CodeBlock
                  code={getVoiceAssistantCode()}
                  label="voice_assistant.py"
                  onCopy={() => handleCopy(getVoiceAssistantCode(), "voice")}
                  copied={copiedSteps.has("voice")}
                />
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Fixed Footer */}
      <div className="flex-shrink-0 border-t border-border/60 bg-background/80 backdrop-blur-sm py-4 px-8">
        <div className="flex items-center justify-between max-w-5xl mx-auto">
          {/* Back Button */}
          <Button
            variant="outline"
            onClick={handleBack}
            className="admin-tonal-cta gap-2"
            data-tone="voice"
          >
            <ArrowLeft className="h-4 w-4" />
            {currentStepIndex > 0 ? "Back" : "Cancel"}
          </Button>

          {/* Progress Stepper */}
          <div className="flex items-center gap-1">
            {WIZARD_STEPS.map((step, index) => {
              const StepIcon = step.icon;
              const isActive = index === currentStepIndex;
              const isCompleted = index < currentStepIndex;

              return (
                <React.Fragment key={step.id}>
                  <div
                    className={cn(
                      "flex items-center gap-2 rounded-lg px-3 py-2 transition-colors",
                      isActive && "bg-primary/10",
                    )}
                  >
                    <div
                      className={cn(
                        "flex h-7 w-7 items-center justify-center rounded-full text-xs font-medium transition-colors",
                        isCompleted && "bg-primary text-primary-foreground",
                        isActive && "bg-primary/20 text-primary",
                        !isActive && !isCompleted && "bg-muted text-muted-foreground",
                      )}
                    >
                      {isCompleted ? (
                        <Check className="h-3.5 w-3.5" />
                      ) : (
                        <StepIcon className="h-3.5 w-3.5" />
                      )}
                    </div>
                    <span
                      className={cn(
                        "text-sm font-medium hidden sm:block",
                        isActive && "text-foreground",
                        !isActive && "text-muted-foreground",
                      )}
                    >
                      {step.label}
                    </span>
                  </div>
                  {index < WIZARD_STEPS.length - 1 && (
                    <ChevronRight className="h-4 w-4 text-muted-foreground/50" />
                  )}
                </React.Fragment>
              );
            })}
          </div>

          {/* Next/Finish Button */}
          {currentStepIndex < WIZARD_STEPS.length - 1 ? (
            <Button onClick={handleNext} disabled={!canProceed} className="gap-2">
              Continue
              <ChevronRight className="h-4 w-4" />
            </Button>
          ) : (
            <Button onClick={handleFinish} className="gap-2">
              <CheckCircle className="h-4 w-4" />
              Done
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

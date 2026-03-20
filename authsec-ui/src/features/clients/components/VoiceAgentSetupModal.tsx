"use client";

import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Mic,
  Copy,
  Check,
  ArrowRight,
  Smartphone,
  KeyRound,
  Zap,
  ExternalLink,
  Info,
  Phone,
  MessageSquare,
  Bot,
} from "lucide-react";
import { toast } from "react-hot-toast";
import { useNavigate } from "react-router-dom";
import type { EnhancedClientData } from "@/app/api/clientApi";

interface VoiceAgentSetupModalProps {
  isOpen: boolean;
  onClose: () => void;
  client?: EnhancedClientData | null;
  clients?: EnhancedClientData[];
  onSuccess?: (clientId: string) => void;
}

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
  <div className="border border-border/60 rounded-lg overflow-hidden bg-background/50 backdrop-blur-sm">
    <div className="flex items-center justify-between border-b border-border/50 bg-muted/40 px-2.5 py-1">
      <span className="text-[11px] font-semibold text-foreground/75">{label}</span>
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
    <div className="p-3 overflow-x-auto bg-background/30">
      <pre className="text-sm font-mono text-foreground/90 whitespace-pre-wrap leading-relaxed">
        {code}
      </pre>
    </div>
  </div>
);

export function VoiceAgentSetupModal({
  isOpen,
  onClose,
  client,
  clients = [],
  onSuccess,
}: VoiceAgentSetupModalProps) {
  const navigate = useNavigate();
  const [step, setStep] = useState(1);
  const [selectedClientId, setSelectedClientId] = useState<string>("");
  const [cibaEnabled, setCibaEnabled] = useState(true);
  const [totpEnabled, setTotpEnabled] = useState(true);
  const [copiedSteps, setCopiedSteps] = useState<Set<string>>(new Set());

  // Reset state when modal opens
  useEffect(() => {
    if (isOpen) {
      setStep(client ? 2 : 1);
      setSelectedClientId(client?.client_id || "");
      setCibaEnabled(true);
      setTotpEnabled(true);
      setCopiedSteps(new Set());
    }
  }, [isOpen, client]);

  const handleCopy = (text: string, stepId: string) => {
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
  };

  const activeClientId = client?.client_id || selectedClientId;
  const activeClient = client || clients.find((c) => c.client_id === selectedClientId);

  const getInstallCode = () => `pip install git+https://github.com/authsec-ai/sdk-authsec.git`;

  const getQuickStartCode = () => `from AuthSec_SDK import CIBAClient

# Initialize with your client ID
client = CIBAClient(client_id="${activeClientId || "your-client-id"}")

# CIBA Flow: Push notification to user's app
result = client.initiate_app_approval("user@example.com")
approval = client.poll_for_approval("user@example.com", result["auth_req_id"])

if approval["status"] == "approved":
    print(f"✅ Authenticated! Token: {approval['token']}")`;

  const getTotpCode = () => `# TOTP Flow: 6-digit code verification
result = client.verify_totp("user@example.com", "123456")

if result["success"]:
    print(f"✅ Authenticated! Token: {result['token']}")
else:
    print(f"❌ Invalid. {result['remaining']} attempts left")`;

  const getVoiceAssistantCode = () => `from AuthSec_SDK import CIBAClient

class VoiceAssistant:
    def __init__(self):
        self.ciba = CIBAClient(client_id="${activeClientId || "your-client-id"}")
    
    def authenticate_user(self, email):
        """Handle voice authentication"""
        method = self.ask_user("Approve via app or use a code?")
        
        if "app" in method.lower():
            # CIBA flow - Push notification
            self.speak("I've sent a notification to your AuthSec app.")
            result = self.ciba.initiate_app_approval(email)
            approval = self.ciba.poll_for_approval(
                email, 
                result["auth_req_id"], 
                timeout=60
            )
            
            if approval["status"] == "approved":
                self.speak("You're authenticated!")
                return approval["token"]
            else:
                self.speak(f"Authentication {approval['status']}. Please try again.")
                return None
        else:
            # TOTP flow - 6-digit code
            self.speak("Please tell me your 6-digit code.")
            code = self.listen_for_digits()
            result = self.ciba.verify_totp(email, code)
            
            if result["success"]:
                self.speak("Perfect! You're authenticated.")
                return result["token"]
            else:
                self.speak(f"Invalid code. {result['remaining']} attempts left.")
                return None`;

  const handleComplete = () => {
    toast.success("Voice agent configuration saved!");
    onSuccess?.(activeClientId);
    onClose();
  };

  const handleViewFullDocs = () => {
    onClose();
    navigate(`/sdk/clients/${activeClientId}?module=ciba`);
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-xl">
            <div className="p-2 rounded-lg bg-primary/10">
              <Mic className="h-5 w-5 text-primary" />
            </div>
            Voice Agent SDK Integration
          </DialogTitle>
          <DialogDescription>
            Enable passwordless authentication for voice assistants and IoT devices using CIBA (push
            notifications) and TOTP (6-digit codes).
          </DialogDescription>
        </DialogHeader>

        {/* Step indicator */}
        <div className="flex items-center gap-2 py-2">
          {[1, 2, 3].map((s) => (
            <div key={s} className="flex items-center">
              <div
                className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors ${
                  step >= s
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-muted-foreground"
                }`}
              >
                {step > s ? <Check className="h-4 w-4" /> : s}
              </div>
              {s < 3 && (
                <div className={`w-12 h-0.5 mx-1 ${step > s ? "bg-primary" : "bg-muted"}`} />
              )}
            </div>
          ))}
          <span className="ml-2 text-sm text-muted-foreground">
            {step === 1 && "Select Client"}
            {step === 2 && "Configure Methods"}
            {step === 3 && "Integration Code"}
          </span>
        </div>

        {/* Step 1: Select Client */}
        {step === 1 && (
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="client-select">Select a Client</Label>
              <Select value={selectedClientId} onValueChange={setSelectedClientId}>
                <SelectTrigger id="client-select">
                  <SelectValue placeholder="Choose a client to configure" />
                </SelectTrigger>
                <SelectContent>
                  {clients.map((c) => (
                    <SelectItem key={c.client_id} value={c.client_id}>
                      <div className="flex items-center gap-2">
                        <Bot className="h-4 w-4 text-muted-foreground" />
                        <span>{c.name || c.client_name || "Unnamed"}</span>
                        <span className="text-xs text-muted-foreground">
                          ({c.client_id.slice(0, 8)}...)
                        </span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {selectedClientId && (
              <div className="p-4 rounded-lg border border-border bg-muted/30">
                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-lg bg-primary/10">
                    <Bot className="h-5 w-5 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <h4 className="font-medium text-sm">
                      {activeClient?.name || "Selected Client"}
                    </h4>
                    <p className="text-xs text-muted-foreground font-mono mt-1 truncate">
                      {activeClientId}
                    </p>
                  </div>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-8 w-8 p-0"
                    onClick={() => handleCopy(activeClientId, "client-id")}
                  >
                    {copiedSteps.has("client-id") ? (
                      <Check className="h-4 w-4 text-green-500" />
                    ) : (
                      <Copy className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              </div>
            )}

            <div className="flex justify-end pt-4">
              <Button onClick={() => setStep(2)} disabled={!selectedClientId} className="gap-2">
                Continue
                <ArrowRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}

        {/* Step 2: Configure Methods */}
        {step === 2 && (
          <div className="space-y-4 py-4">
            {/* Info Banner */}
            <div className="flex items-start gap-3 p-4 rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800">
              <Info className="h-5 w-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
              <div className="text-sm">
                <p className="font-medium text-blue-800 dark:text-blue-200">Best Practice</p>
                <p className="text-blue-700 dark:text-blue-300 mt-1">
                  Enable both CIBA and TOTP for maximum compatibility. TOTP serves as a fallback
                  when users don't have the AuthSec mobile app.
                </p>
              </div>
            </div>

            {/* Auth Methods */}
            <div className="space-y-3">
              {/* CIBA Option */}
              <div
                className={`p-4 rounded-lg border transition-colors ${
                  cibaEnabled ? "border-primary/50 bg-primary/5" : "border-border"
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-3">
                    <div className={`p-2 rounded-lg ${cibaEnabled ? "bg-primary/10" : "bg-muted"}`}>
                      <Smartphone className="h-5 w-5 text-primary" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h4 className="font-medium">CIBA Push Notifications</h4>
                        <Badge variant="secondary" className="text-xs">
                          Recommended
                        </Badge>
                      </div>
                      <p className="text-sm text-muted-foreground mt-1">
                        Send push notifications to user's AuthSec mobile app for approval. Best for
                        voice assistants and hands-free authentication.
                      </p>
                      <div className="flex flex-wrap gap-2 mt-2">
                        <Badge variant="outline" className="text-xs gap-1 bg-background">
                          <Phone className="h-3 w-3" /> Voice Assistants
                        </Badge>
                        <Badge variant="outline" className="text-xs gap-1 bg-background">
                          <Zap className="h-3 w-3" /> IoT Devices
                        </Badge>
                      </div>
                    </div>
                  </div>
                  <Switch checked={cibaEnabled} onCheckedChange={setCibaEnabled} />
                </div>
              </div>

              {/* TOTP Option */}
              <div
                className={`p-4 rounded-lg border transition-colors ${
                  totpEnabled ? "border-primary/50 bg-primary/5" : "border-border"
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-3">
                    <div className={`p-2 rounded-lg ${totpEnabled ? "bg-primary/10" : "bg-muted"}`}>
                      <KeyRound className="h-5 w-5 text-primary" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h4 className="font-medium">TOTP 6-Digit Codes</h4>
                        <Badge variant="outline" className="text-xs">
                          Fallback
                        </Badge>
                      </div>
                      <p className="text-sm text-muted-foreground mt-1">
                        Verify 6-digit codes from authenticator apps. Perfect fallback when push
                        notifications aren't available.
                      </p>
                      <div className="flex flex-wrap gap-2 mt-2">
                        <Badge variant="outline" className="text-xs gap-1 bg-background">
                          <MessageSquare className="h-3 w-3" /> CLI Tools
                        </Badge>
                        <Badge variant="outline" className="text-xs gap-1 bg-background">
                          <Zap className="h-3 w-3" /> Backup Auth
                        </Badge>
                      </div>
                    </div>
                  </div>
                  <Switch checked={totpEnabled} onCheckedChange={setTotpEnabled} />
                </div>
              </div>
            </div>

            <div className="flex justify-between pt-4">
              <Button variant="outline" onClick={() => setStep(1)}>
                Back
              </Button>
              <Button
                onClick={() => setStep(3)}
                disabled={!cibaEnabled && !totpEnabled}
                className="gap-2"
              >
                View Integration Code
                <ArrowRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}

        {/* Step 3: Integration Code */}
        {step === 3 && (
          <div className="space-y-4 py-4">
            {/* Install */}
            <div className="space-y-2">
              <Label className="text-sm font-medium flex items-center gap-2">
                <span className="w-5 h-5 rounded-full bg-primary/10 text-primary text-xs flex items-center justify-center">
                  1
                </span>
                Install the SDK
              </Label>
              <CodeBlock
                code={`$ ${getInstallCode()}`}
                label="Terminal"
                onCopy={() => handleCopy(getInstallCode(), "install")}
                copied={copiedSteps.has("install")}
              />
            </div>

            {/* Quick Start */}
            {cibaEnabled && (
              <div className="space-y-2">
                <Label className="text-sm font-medium flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-primary/10 text-primary text-xs flex items-center justify-center">
                    2
                  </span>
                  CIBA Authentication (Push Notification)
                </Label>
                <CodeBlock
                  code={getQuickStartCode()}
                  label="Python"
                  onCopy={() => handleCopy(getQuickStartCode(), "quickstart")}
                  copied={copiedSteps.has("quickstart")}
                />
              </div>
            )}

            {/* TOTP */}
            {totpEnabled && (
              <div className="space-y-2">
                <Label className="text-sm font-medium flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-primary/10 text-primary text-xs flex items-center justify-center">
                    {cibaEnabled ? "3" : "2"}
                  </span>
                  TOTP Verification (6-Digit Code)
                </Label>
                <CodeBlock
                  code={getTotpCode()}
                  label="Python"
                  onCopy={() => handleCopy(getTotpCode(), "totp")}
                  copied={copiedSteps.has("totp")}
                />
              </div>
            )}

            {/* Voice Assistant Example */}
            <div className="space-y-2">
              <Label className="text-sm font-medium flex items-center gap-2">
                <span className="w-5 h-5 rounded-full bg-primary/10 text-primary text-xs flex items-center justify-center">
                  <Mic className="h-3 w-3" />
                </span>
                Complete Voice Assistant Example
              </Label>
              <CodeBlock
                code={getVoiceAssistantCode()}
                label="voice_assistant.py"
                onCopy={() => handleCopy(getVoiceAssistantCode(), "voice-assistant")}
                copied={copiedSteps.has("voice-assistant")}
              />
            </div>

            {/* Success Banner */}
            <div className="flex items-start gap-3 p-4 rounded-lg bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800">
              <Check className="h-5 w-5 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
              <div className="text-sm">
                <p className="font-medium text-green-800 dark:text-green-200">Ready to go!</p>
                <p className="text-green-700 dark:text-green-300 mt-1">
                  Your voice agent can now authenticate users securely using CIBA push notifications
                  {totpEnabled && " and TOTP codes as fallback"}.
                </p>
              </div>
            </div>

            <div className="flex justify-between pt-4">
              <Button variant="outline" onClick={() => setStep(2)}>
                Back
              </Button>
              <div className="flex gap-2">
                <Button variant="outline" onClick={handleViewFullDocs} className="gap-2">
                  Full SDK Docs
                  <ExternalLink className="h-4 w-4" />
                </Button>
                <Button onClick={handleComplete} className="gap-2">
                  Done
                  <Check className="h-4 w-4" />
                </Button>
              </div>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Copy, ExternalLink, Check, Code } from "lucide-react";
import type { RoleFormData } from "../types";
import { toast } from "@/lib/toast";

interface SDKSnippetProps {
  roleData: RoleFormData;
  className?: string;
}

export function SDKSnippet({ roleData, className }: SDKSnippetProps) {
  const [copied, setCopied] = useState(false);

  const generatePythonCode = () => {
    const roleId = roleData.roleId || "NEW_ROLE";
    const grants = roleData.grants.map((grant) => {
      const scopesStr = grant.scopes.map((s) => `"${s}"`).join(",");
      return `        {"resource": "${grant.resource}", "scopes": [${scopesStr}]}`;
    });

    const grantsStr = grants.length > 0 ? grants.join(",\n") : "        # No grants configured yet";

    return `from authsec import RBAC

rbac = RBAC(workspace_id="acme-prod")

role = rbac.create_role(
    role_id   = "${roleId}",
    grants    = [
${grantsStr}
    ]
)`;
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(generatePythonCode());
      setCopied(true);
      toast.success("Code copied to clipboard");
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      toast.error("Failed to copy code");
    }
  };

  const handleDocsClick = () => {
    // Open docs in new tab
    window.open("/sdk/rbac?module=rbac", "_blank");
  };

  return (
    <Card className={className}>
      <CardHeader className="pb-2">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-primary/10">
            <Code className="h-5 w-5 text-primary" />
          </div>
          <div>
            <CardTitle className="text-xl font-semibold text-foreground">
              Role SDK Integration Code
            </CardTitle>
            <CardDescription className="text-base text-foreground">
              Copy and paste this code into your application. Updates automatically as you configure
              role settings.
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        <div className="relative bg-muted rounded-lg p-6 font-mono text-sm whitespace-pre-wrap text-left shadow-inner border border-border">
          <code>{generatePythonCode()}</code>
          <Button
            onClick={handleCopy}
            size="lg"
            className="absolute top-4 right-4 h-10 px-6 font-semibold shadow-md"
            variant={copied ? "secondary" : "default"}
            aria-label="Copy snippet"
          >
            {copied ? (
              <>
                <Check className="h-5 w-5 mr-2" />
                Copied!
              </>
            ) : (
              <>
                <Copy className="h-5 w-5 mr-2" />
                Copy Code
              </>
            )}
          </Button>
        </div>

        <div className="flex items-center gap-2 mt-4">
          <span className="font-mono text-xs text-foreground">Role ID:</span>
          <span className="font-mono text-xs bg-muted px-2 py-1 rounded select-all">
            {roleData.roleId || "NEW_ROLE"}
          </span>
          <span className="font-mono text-xs text-foreground">•</span>
          <span className="font-mono text-xs text-foreground">
            {roleData.grants.length} grant{roleData.grants.length !== 1 ? "s" : ""}
          </span>
          <button
            className="p-1 rounded hover:bg-accent ml-auto"
            onClick={handleDocsClick}
            aria-label="Open documentation"
          >
            <ExternalLink className="h-4 w-4" />
          </button>
        </div>
      </CardContent>
    </Card>
  );
}

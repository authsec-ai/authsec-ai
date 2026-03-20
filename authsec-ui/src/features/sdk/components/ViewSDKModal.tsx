import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Code2, Check, Copy, ExternalLink, Terminal, BookOpen } from "lucide-react";
import { cn } from "@/lib/utils";
import type { SDKLanguage } from "../types";
import {
  buildSDKHubLink,
  getModuleFromLegacyDocsLink,
  inferHubModuleFromTitle,
} from "../utils/hub-routing";
import "../sdk-editorial-theme.css";

interface CodeStep {
  label: string;
  code: string;
  description?: string;
}

interface ViewSDKModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  entityType: string;
  entityName: string;
  pythonCode: CodeStep[];
  typescriptCode?: CodeStep[];
  docsLink?: string;
}

const CodeBlock = ({
  code,
  label,
  description,
  onCopy,
  copied,
}: {
  code: string;
  label?: string;
  description?: string;
  onCopy?: () => void;
  copied?: boolean;
}) => {
  return (
    <div
      className="border border-neutral-700 rounded-lg overflow-hidden bg-neutral-800 backdrop-blur-sm"
      data-sdk-code-block="true"
    >
      <div className="flex items-center justify-between border-b border-neutral-700 bg-neutral-800 px-3 py-1.5">
        <div className="flex items-center gap-2">
          <Terminal className="h-3.5 w-3.5 text-neutral-500" />
          <span className="text-[12px] font-semibold text-neutral-300">
            {label}
          </span>
        </div>
        <div className="flex items-center">
          {onCopy && (
            <Button
              size="sm"
              variant="ghost"
              className="h-6 px-2 hover:bg-neutral-700 text-neutral-400 hover:text-neutral-300 text-xs"
              onClick={onCopy}
            >
              {copied ? (
                <>
                  <Check className="h-3 w-3 mr-1 text-green-500" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="h-3 w-3 mr-1" />
                  Copy
                </>
              )}
            </Button>
          )}
        </div>
      </div>
      {description && (
        <div className="px-3 py-2 bg-neutral-850 border-b border-neutral-700">
          <p className="text-xs text-neutral-400">{description}</p>
        </div>
      )}
      <div className="p-4 overflow-x-auto bg-neutral-900">
        <pre className="text-sm font-mono text-neutral-300 whitespace-pre leading-relaxed">
          {code}
        </pre>
      </div>
    </div>
  );
};

export function ViewSDKModal({
  open,
  onOpenChange,
  title,
  description,
  entityType,
  entityName,
  pythonCode,
  typescriptCode = [],
  docsLink,
}: ViewSDKModalProps) {
  const [copiedIndex, setCopiedIndex] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<SDKLanguage>("python");
  const moduleFromLegacyDocs = getModuleFromLegacyDocsLink(docsLink);
  const inferredModule = inferHubModuleFromTitle(`${title} ${entityType}`);
  const effectiveDocsLink =
    docsLink && !docsLink.startsWith("/docs/sdk/")
      ? docsLink
      : buildSDKHubLink({ module: moduleFromLegacyDocs ?? inferredModule });

  const handleCopy = (code: string, language: string, index: number) => {
    navigator.clipboard.writeText(code);
    setCopiedIndex(`${language}-${index}`);
    setTimeout(() => setCopiedIndex(null), 2000);
  };

  const handleCopyAll = () => {
    const allCode = activeTab === "python"
      ? pythonCode.map(s => s.code).join("\n\n")
      : typescriptCode.map(s => s.code).join("\n\n");
    navigator.clipboard.writeText(allCode);
    setCopiedIndex("all");
    setTimeout(() => setCopiedIndex(null), 2000);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="!max-w-none w-[70vw] max-h-[90vh] overflow-hidden flex flex-col"
        data-dashboard="overview"
        data-sdk-surface="sdk-view-modal"
      >
        <DialogHeader className="pb-4 border-b">
          <div className="flex items-start justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <DialogTitle className="text-xl font-semibold">
                  {title}
                </DialogTitle>
                <Badge variant="secondary" className="text-xs">
                  {entityType}
                </Badge>
              </div>
              <DialogDescription className="text-sm">
                {description}
              </DialogDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="sdk-editorial-outline-btn gap-2"
              onClick={() => window.open(effectiveDocsLink, "_blank")}
            >
              <BookOpen className="h-4 w-4" />
              View SDK Hub
              <ExternalLink className="h-3 w-3" />
            </Button>
          </div>

          {/* Entity Info Bar */}
          <div className="sdk-editorial-subpanel mt-4 p-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-medium text-foreground">Entity:</span>
                  <code className="px-2 py-0.5 bg-background/80 rounded text-sm font-mono">
                    {entityName}
                  </code>
                </div>
              </div>
              <Button
                variant="secondary"
                size="sm"
                className="sdk-editorial-outline-btn gap-2"
                onClick={handleCopyAll}
              >
                {copiedIndex === "all" ? (
                  <>
                    <Check className="h-3.5 w-3.5 text-green-500" />
                    Copied All
                  </>
                ) : (
                  <>
                    <Copy className="h-3.5 w-3.5" />
                    Copy All Code
                  </>
                )}
              </Button>
            </div>
          </div>
        </DialogHeader>

        <Tabs
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as SDKLanguage)}
          className="flex-1 flex flex-col overflow-hidden"
        >
          <TabsList className="grid w-full grid-cols-2 mb-4">
            <TabsTrigger value="python" className="gap-2">
              <Code2 className="h-4 w-4" />
              Python SDK
            </TabsTrigger>
            <TabsTrigger
              value="typescript"
              className={cn(
                "gap-2",
                typescriptCode.length === 0 && "opacity-50 cursor-not-allowed"
              )}
              disabled={typescriptCode.length === 0}
            >
              <Code2 className="h-4 w-4" />
              TypeScript SDK
              {typescriptCode.length === 0 && (
                <Badge variant="outline" className="ml-1 text-[10px]">Coming Soon</Badge>
              )}
            </TabsTrigger>
          </TabsList>

          <div className="flex-1 overflow-y-auto">
            <TabsContent value="python" className="mt-0 space-y-4">
              {pythonCode.map((step, index) => (
                <CodeBlock
                  key={index}
                  label={step.label}
                  code={step.code}
                  description={step.description}
                  onCopy={() => handleCopy(step.code, "python", index)}
                  copied={copiedIndex === `python-${index}`}
                />
              ))}
            </TabsContent>

            <TabsContent value="typescript" className="mt-0 space-y-4">
              {typescriptCode.map((step, index) => (
                <CodeBlock
                  key={index}
                  label={step.label}
                  code={step.code}
                  description={step.description}
                  onCopy={() => handleCopy(step.code, "typescript", index)}
                  copied={copiedIndex === `typescript-${index}`}
                />
              ))}
            </TabsContent>
          </div>
        </Tabs>

        {/* Footer with helpful tips */}
        <div className="pt-4 border-t mt-4">
          <div className="flex items-start gap-2 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
            <Code2 className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
            <div className="text-sm text-blue-800 dark:text-blue-200">
              <strong>Pro Tip:</strong> Install the SDK dependencies with{" "}
              <code className="px-1.5 py-0.5 bg-blue-100 dark:bg-blue-800 rounded text-xs font-mono">
                pip install requests PyJWT
              </code>{" "}
              to get started quickly.
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

import { useState } from "react";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerFooter,
  DrawerClose,
} from "../../../../components/ui/drawer";
import {
  Accordion,
  AccordionItem,
  AccordionTrigger,
  AccordionContent,
} from "../../../../components/ui/accordion";
import { Badge } from "../../../../components/ui/badge";
import { Button } from "../../../../components/ui/button";
import { Terminal, Copy, Check, X } from "lucide-react";
import type { AuthLog } from "../../../../types/entities";

interface AuthLogDetailDrawerProps {
  log: AuthLog | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AuthLogDetailDrawer({
  log,
  open,
  onOpenChange,
}: AuthLogDetailDrawerProps) {
  const [copied, setCopied] = useState(false);

  if (!log) return null;

  // Skip empty sections automatically
  const renderAccordionSection = (title: string, data: any) => {
    if (!data) return null;
    if (typeof data === "object" && Object.keys(data).length === 0) return null;
    if (typeof data === "string" && data.trim() === "") return null;

    const sectionId = title.toLowerCase().replace(/\s+/g, "-");

    return (
      <AccordionItem value={sectionId} key={sectionId}>
        <AccordionTrigger className="text-sm font-semibold">
          {title}
        </AccordionTrigger>
        <AccordionContent>
          <div className="rounded-lg bg-muted p-4 max-h-96 overflow-y-auto">
            <pre className="text-xs overflow-x-auto whitespace-pre-wrap break-words">
              {typeof data === "string" ? data : JSON.stringify(data, null, 2)}
            </pre>
          </div>
        </AccordionContent>
      </AccordionItem>
    );
  };

  const copyJSON = () => {
    const jsonString = JSON.stringify(log?.rawPayload || log, null, 2);
    navigator.clipboard.writeText(jsonString);
    setCopied(true);

    // Reset the copied state after 2 seconds
    setTimeout(() => {
      setCopied(false);
    }, 2000);
  };

  // Determine badge variant based on status
  const statusVariant =
    {
      success: "default" as const,
      failure: "destructive" as const,
      denied: "secondary" as const,
      suspicious: "outline" as const,
    }[log.status] || ("default" as const);

  return (
    <Drawer open={open} onOpenChange={onOpenChange} direction="right">
      <DrawerContent className="h-full w-full max-w-3xl border-l">
        <DrawerHeader className="border-b p-6">
          <div className="flex items-start justify-between">
            <div className="space-y-2">
              <DrawerTitle className="flex items-center gap-2 text-xl">
                <Terminal className="h-5 w-5" />
                Auth Log Details
                <Badge variant={statusVariant} className="text-xs">
                  {log.status.toUpperCase()}
                </Badge>
              </DrawerTitle>
              <div className="flex flex-wrap items-center gap-4 text-xs text-foreground">
                <span>{new Date(log.timestamp).toLocaleString()}</span>
                <span>{log.logType.toUpperCase()}</span>
                {log.rawPayload?.debug?.internal?.correlation_id && (
                  <code className="bg-muted px-1.5 py-0.5 rounded">
                    {log.rawPayload.debug.internal.correlation_id}
                  </code>
                )}
              </div>
            </div>
            <DrawerClose asChild>
              <Button size="icon" variant="ghost" aria-label="Close">
                <X className="h-4 w-4" />
              </Button>
            </DrawerClose>
          </div>
        </DrawerHeader>

        <div className="flex-1 overflow-y-auto p-6">
          {/* Quick Summary - Always Visible */}
          <div className="mb-6 rounded-lg border bg-card p-4">
            <h3 className="text-sm font-semibold mb-3">Summary</h3>
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div>
                <span className="text-foreground">User:</span>{" "}
                <span className="font-mono">
                  {log.username || log.email || "—"}
                </span>
              </div>
              <div>
                <span className="text-foreground">IP:</span>{" "}
                <span className="font-mono">{log.ipAddress}</span>
              </div>
              <div>
                <span className="text-foreground">Client:</span>{" "}
                <span className="font-mono">{log.clientName}</span>
              </div>
              <div>
                <span className="text-foreground">Auth Method:</span>{" "}
                <span className="font-mono">{log.authMethod}</span>
              </div>
              {log.location && (
                <div>
                  <span className="text-foreground">Location:</span>{" "}
                  <span>{log.location}</span>
                </div>
              )}
              <div>
                <span className="text-foreground">MFA:</span>{" "}
                <span>{log.mfaUsed ? "Yes" : "No"}</span>
              </div>
            </div>
          </div>

          {/* Collapsible Sections */}
          <Accordion
            type="multiple"
            defaultValue={[
              "event-information",
              "actor-details",
              "result-&-status",
            ]}
            className="space-y-1"
          >
            {renderAccordionSection(
              "Event Information",
              log?.rawPayload?.event
            )}
            {renderAccordionSection("Actor Details", log?.rawPayload?.actor)}
            {renderAccordionSection(
              "Client Information",
              log?.rawPayload?.client
            )}
            {renderAccordionSection(
              "Device Information",
              log?.rawPayload?.device
            )}
            {renderAccordionSection(
              "Authentication Context",
              log?.rawPayload?.authentication_context
            )}
            {renderAccordionSection(
              "Protocol Details",
              log?.rawPayload?.protocol
            )}
            {renderAccordionSection("Result & Status", log?.rawPayload?.result)}
            {renderAccordionSection(
              "Security Context",
              log?.rawPayload?.security_context
            )}
            {renderAccordionSection(
              "Policy Information",
              log?.rawPayload?.policy
            )}
            {renderAccordionSection(
              "Transaction Details",
              log?.rawPayload?.transaction
            )}
            {renderAccordionSection(
              "Debug Information",
              log?.rawPayload?.debug
            )}
            {renderAccordionSection(
              "Request Details",
              log?.rawPayload?.request
            )}
            {log?.rawPayload?.message &&
              renderAccordionSection("Raw Message", log.rawPayload.message)}
          </Accordion>
        </div>

        <DrawerFooter className="border-t p-4 flex flex-row justify-end gap-2">
          <Button onClick={copyJSON} variant="outline" size="sm">
            {copied ? (
              <>
                <Check className="h-4 w-4 mr-2" />
                Copied!
              </>
            ) : (
              <>
                <Copy className="h-4 w-4 mr-2" />
                Copy Full JSON
              </>
            )}
          </Button>
          <DrawerClose asChild>
            <Button variant="default" size="sm">
              Close
            </Button>
          </DrawerClose>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}

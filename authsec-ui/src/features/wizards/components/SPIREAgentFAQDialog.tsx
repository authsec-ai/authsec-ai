import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Check, Copy, Loader2 } from "lucide-react";
import { SPIRE_FAQ_DATA } from "./spire-faq-data";

interface SPIREAgentFAQDialogProps {
  open: boolean;
  onDone: () => void;
  isFetching?: boolean;
}

// CodeBlock component for displaying code snippets or images
const CodeBlock = ({
  code,
  label,
  onCopy,
  copied,
  image,
  imageAlt,
}: {
  code?: string;
  label?: string;
  onCopy?: () => void;
  copied?: boolean;
  image?: string;
  imageAlt?: string;
}) => {
  return (
    <div className="border border-neutral-700 rounded-lg overflow-hidden bg-neutral-800 backdrop-blur-sm">
      <div className="flex items-center justify-between border-b border-neutral-700 bg-neutral-800 px-2.5 py-0.5">
        <span className="text-[11px] font-semibold text-neutral-400">
          {label}
        </span>
        <div className="flex items-center">
          {onCopy && code && (
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
          )}
        </div>
      </div>
      <div className="p-3 overflow-x-auto bg-neutral-900">
        {image ? (
          <img
            src={image}
            alt={imageAlt || label || "Screenshot"}
            className="w-full h-auto rounded border border-neutral-700"
          />
        ) : (
          <pre className="text-sm font-mono text-neutral-300 whitespace-pre-wrap leading-relaxed">
            {code}
          </pre>
        )}
      </div>
    </div>
  );
};

export function SPIREAgentFAQDialog({
  open,
  onDone,
  isFetching = false,
}: SPIREAgentFAQDialogProps) {
  const [activeSubTab, setActiveSubTab] = useState<number>(0);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const handleCopy = (code: string, id: string) => {
    navigator.clipboard.writeText(code);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  // Prevent closing via backdrop or escape key
  const handleOpenChange = (isOpen: boolean) => {
    // Only allow programmatic close via "Done" button
    if (!isOpen) {
      return; // Prevent closing
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        className="!max-w-none w-[65vw] max-h-[90vh] overflow-y-auto"
        onEscapeKeyDown={(e) => e.preventDefault()} // Prevent ESC close
        onPointerDownOutside={(e) => e.preventDefault()} // Prevent backdrop close
      >
        <DialogHeader>
          <DialogTitle className="text-lg font-semibold">
            SPIRE Agent Setup Guide
          </DialogTitle>
          <DialogDescription>
            Deploy a SPIRE agent to your infrastructure using Kubernetes,
            Docker, or VM. Once deployed, click "Done" to verify and continue.
          </DialogDescription>
        </DialogHeader>

        {/* Render FAQ tabs */}
        <Tabs
          value={String(activeSubTab)}
          onValueChange={(v) => setActiveSubTab(Number(v))}
          className="mt-4"
        >
          <TabsList
            className="grid w-full"
            style={{
              gridTemplateColumns: `repeat(${SPIRE_FAQ_DATA.length}, 1fr)`,
            }}
          >
            {SPIRE_FAQ_DATA.map((faq, idx) => (
              <TabsTrigger key={idx} value={String(idx)}>
                {faq.question
                  .replace("How do I deploy my agent and workloads on ", "")
                  .replace("How do I deploy my agent on ", "")
                  .replace("?", "")
                  .replace("kubernetes cluster", "Kubernetes")
                  .replace("docker", "Docker")
                  .replace("VM(Virtual Machine)", "VM")}
              </TabsTrigger>
            ))}
          </TabsList>

          {SPIRE_FAQ_DATA.map((faq, faqIdx) => (
            <TabsContent
              key={faqIdx}
              value={String(faqIdx)}
              className="mt-4 space-y-4"
            >
              {faq.code?.python.map((method, methodIdx) => (
                <div key={methodIdx} className="space-y-4">
                  <h3 className="font-semibold text-sm">{method.methodName}</h3>
                  {method.steps.map((step, stepIdx) => (
                    <CodeBlock
                      key={stepIdx}
                      label={step.label}
                      code={step.code}
                      image={step.image}
                      imageAlt={step.imageAlt}
                      onCopy={
                        step.code
                          ? () =>
                              handleCopy(
                                step.code!,
                                `${faqIdx}-${methodIdx}-${stepIdx}`
                              )
                          : undefined
                      }
                      copied={copiedId === `${faqIdx}-${methodIdx}-${stepIdx}`}
                    />
                  ))}
                </div>
              ))}
            </TabsContent>
          ))}
        </Tabs>

        {/* Custom footer with "Done" button */}
        <DialogFooter>
          <Button
            onClick={onDone}
            disabled={isFetching}
            className="bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90"
          >
            {isFetching ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Verifying...
              </>
            ) : (
              <>Done</>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

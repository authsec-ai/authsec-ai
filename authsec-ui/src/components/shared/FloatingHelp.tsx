import React, { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Check, Code2, Copy, HelpCircle, X, ExternalLink } from "lucide-react";
import { cn } from "@/lib/utils";
import "@/features/sdk/sdk-editorial-theme.css";

export interface HelpCodeStep {
  label: string;
  code?: string;
  image?: string;
  imageAlt?: string;
}

export interface HelpCodeGroup {
  methodName: string;
  steps: HelpCodeStep[];
}

export type HelpCodeValue = HelpCodeStep[] | HelpCodeGroup[];

export type HelpCodeExample = Record<string, HelpCodeValue | undefined>;

export interface FloatingHelpItem {
  id: string;
  question: string;
  description: string;
  code?: HelpCodeExample;
  customContent?: React.ReactNode;
  languageTabs?: FloatingHelpLanguageTab[];
  docsLink?: string;
  disabled?: boolean;
}

export interface FloatingHelpLanguageTab {
  key: string;
  label: string;
  disabled?: boolean;
  disabledLabel?: string;
}

export interface FloatingHelpProps {
  items: FloatingHelpItem[];
  tooltipLabel?: string;
  defaultOpen?: boolean;
  defaultLanguage?: string;
  languageTabs?: FloatingHelpLanguageTab[];
  visualVariant?: "default" | "editorial";
}

export const DEFAULT_HELP_LANGUAGE_TABS: FloatingHelpLanguageTab[] = [
  { key: "python", label: "Python" },
  {
    key: "typescript",
    label: "TypeScript",
    disabled: true,
    disabledLabel: "TypeScript (Coming Soon)",
  },
];

const formatLanguageLabel = (key: string) => {
  switch (key.toLowerCase()) {
    case "python":
      return "Python";
    case "typescript":
      return "TypeScript";
    case "go":
      return "Go";
    case "java":
      return "Java";
    default:
      return `${key.charAt(0).toUpperCase()}${key.slice(1)}`;
  }
};

const isGroupedSteps = (value: HelpCodeValue): value is HelpCodeGroup[] => {
  return value.length > 0 && "methodName" in value[0];
};

const CodeBlock = ({
  code,
  label,
  image,
  imageAlt,
  onCopy,
  copied,
}: {
  code?: string;
  label?: string;
  image?: string;
  imageAlt?: string;
  onCopy?: () => void;
  copied?: boolean;
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

export function FloatingHelp({
  items,
  tooltipLabel = "Quick Help",
  defaultOpen = false,
  defaultLanguage = "python",
  languageTabs,
  visualVariant = "default",
}: FloatingHelpProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const [showTooltip, setShowTooltip] = useState(false);
  const [selectedItem, setSelectedItem] = useState<FloatingHelpItem | null>(
    null
  );
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [activeLanguage, setActiveLanguage] = useState(defaultLanguage);
  const [activeSubTab, setActiveSubTab] = useState<number>(0);
  const [hoveredId, setHoveredId] = useState<string | null>(null);

  const resolveTabsForItem = (item: FloatingHelpItem) => {
    if (item.languageTabs && item.languageTabs.length > 0)
      return item.languageTabs;
    if (languageTabs && languageTabs.length > 0) return languageTabs;
    if (!item.code) return [];
    return Object.keys(item.code).map((key) => ({
      key,
      label: formatLanguageLabel(key),
    }));
  };

  const hasContent = (item: FloatingHelpItem, language: string) => {
    const value = item.code?.[language];
    return Array.isArray(value) && value.length > 0;
  };

  const resolveInitialLanguage = (item: FloatingHelpItem) => {
    if (item.code && hasContent(item, defaultLanguage)) {
      return defaultLanguage;
    }
    const tabs = resolveTabsForItem(item);
    const firstWithContent = tabs.find(
      (tab) => !tab.disabled && hasContent(item, tab.key)
    );
    if (firstWithContent) return firstWithContent.key;
    const firstEnabled = tabs.find((tab) => !tab.disabled);
    if (firstEnabled) return firstEnabled.key;
    return Object.keys(item.code ?? {})[0] ?? defaultLanguage;
  };

  const handleQuestionClick = (item: FloatingHelpItem) => {
    setSelectedItem(item);
    setCopiedId(null);
    setActiveSubTab(0);
    setActiveLanguage(resolveInitialLanguage(item));
  };

  const handleCloseModal = () => {
    setSelectedItem(null);
    setCopiedId(null);
  };

  const handleCopy = (code: string, id: string) => {
    navigator.clipboard.writeText(code);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const toggleHelp = () => {
    setIsOpen((prev) => !prev);
    setShowTooltip(false);
  };

  const resolvedTabs = useMemo(() => {
    if (!selectedItem) return languageTabs ?? [];
    return resolveTabsForItem(selectedItem);
  }, [languageTabs, selectedItem]);

  const showLanguageTabs = resolvedTabs.length > 1;

  return (
    <>
      <div
        data-dashboard={visualVariant === "editorial" ? "overview" : undefined}
        data-sdk-surface={visualVariant === "editorial" ? "help-fab" : undefined}
        className={cn(
          "fixed bottom-6 right-6 z-40 flex gap-2",
          isOpen ? "items-end" : "items-center"
        )}
      >
        {showTooltip && !isOpen && (
          <div
            className={cn(
              "px-3 py-2 rounded-lg shadow-lg animate-in fade-in duration-200",
              visualVariant === "editorial"
                ? "sdk-help-tooltip"
                : "bg-slate-900 dark:bg-neutral-800 text-white",
            )}
          >
            <span className="text-sm font-medium whitespace-nowrap">
              {tooltipLabel}
            </span>
          </div>
        )}

        {isOpen && (
          <div
            className={cn(
              "p-4 w-80 animate-in slide-in-from-right-2 fade-in duration-200",
              visualVariant === "editorial" && "sdk-help-popover",
            )}
          >
            <div className="space-y-2">
              {items.map((item) => (
                <button
                  key={item.id}
                  onClick={() => !item.disabled && handleQuestionClick(item)}
                  onMouseEnter={() => setHoveredId(item.id)}
                  onMouseLeave={() => setHoveredId(null)}
                  disabled={item.disabled}
                  className={cn(
                    "w-full text-left p-3 rounded-md transition-colors border",
                    visualVariant === "editorial"
                      ? "sdk-help-item"
                      : "bg-slate-50 dark:bg-neutral-800 border-slate-200 dark:border-neutral-700",
                    item.disabled
                      ? "opacity-100 cursor-not-allowed"
                      : visualVariant === "editorial"
                        ? "cursor-pointer"
                        : "hover:bg-slate-100 dark:hover:bg-neutral-700 cursor-pointer"
                  )}
                >
                  <div className="flex items-start gap-2">
                    <Code2
                      className={cn(
                        "h-4 w-4 mt-0.5 flex-shrink-0",
                        visualVariant === "editorial" && !item.disabled && "sdk-help-item-icon",
                        item.disabled
                          ? "text-slate-400 dark:text-neutral-600"
                          : visualVariant === "editorial"
                            ? ""
                            : "text-blue-600 dark:text-blue-400"
                      )}
                    />
                    <span
                      className={cn(
                        "text-sm transition-all duration-100 ease-out",
                        item.disabled
                          ? "text-slate-500 dark:text-neutral-500"
                          : visualVariant === "editorial"
                            ? "dash-text-1"
                            : "text-slate-900 dark:text-neutral-100",
                        hoveredId === item.id && !item.disabled
                          ? "opacity-100 font-normal"
                          : "opacity-85 font-normal"
                      )}
                    >
                      {item.question}
                    </span>
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        <Button
          data-slot="floating-help-fab"
          className={cn(
            "h-10 w-10 !rounded-full shadow-lg transition-all duration-200 flex-shrink-0",
            visualVariant === "editorial"
              ? "sdk-help-fab"
              : "bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-500 dark:text-white dark:hover:bg-blue-600",
            isOpen && "rotate-180",
            "cursor-pointer"
          )}
          size="icon"
          onClick={toggleHelp}
          onMouseEnter={() => setShowTooltip(true)}
          onMouseLeave={() => setShowTooltip(false)}
        >
          {isOpen ? (
            <X className="h-4 w-4 text-white" />
          ) : (
            <HelpCircle className="h-4 w-4 text-white" />
          )}
        </Button>
      </div>

      <Dialog
        open={!!selectedItem}
        onOpenChange={(open) => !open && handleCloseModal()}
      >
        <DialogContent
          className="!max-w-none w-[65vw] max-h-[90vh] overflow-y-scroll [&::-webkit-scrollbar]:w-2 [&::-webkit-scrollbar-track]:bg-neutral-200 dark:[&::-webkit-scrollbar-track]:bg-neutral-900 [&::-webkit-scrollbar-thumb]:bg-neutral-400 dark:[&::-webkit-scrollbar-thumb]:bg-neutral-700 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:hover:bg-neutral-500 dark:[&::-webkit-scrollbar-thumb]:hover:bg-neutral-600 [scrollbar-width:thin] [scrollbar-color:theme(colors.neutral.400)_theme(colors.neutral.200)] dark:[scrollbar-color:theme(colors.neutral.700)_theme(colors.neutral.900)]"
          data-dashboard={visualVariant === "editorial" ? "overview" : undefined}
          data-sdk-surface={visualVariant === "editorial" ? "help-dialog" : undefined}
        >
          <DialogHeader>
            <div className="flex items-start justify-between gap-4 pr-8">
              <div className="flex-1">
                <DialogTitle className="text-lg font-semibold">
                  {selectedItem?.question}
                </DialogTitle>
                <DialogDescription>
                  {selectedItem?.description}
                </DialogDescription>
              </div>
              {selectedItem?.docsLink && (
                // <a
                //   href={selectedItem.docsLink}
                //   target="_blank"
                //   rel="noopener noreferrer"
                //   className="inline-flex items-center gap-2 text-lg text-primary hover:underline transition-all"
                // >
                //   View Docs
                //   <svg
                //     className="h-5 w-5"
                //     fill="none"
                //     viewBox="0 0 24 24"
                //     stroke="currentColor"
                //   >
                //     <path
                //       strokeLinecap="round"
                //       strokeLinejoin="round"
                //       strokeWidth={2}
                //       d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                //     />
                //   </svg>
                // </a>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => window.open(selectedItem.docsLink, "_blank")}
                  className={cn(
                    "flex items-center gap-2 shrink-0 mt-0.5",
                    visualVariant === "editorial" && "sdk-editorial-outline-btn",
                  )}
                >
                  <ExternalLink className="h-4 w-4" />
                  View Docs
                </Button>
              )}
            </div>
          </DialogHeader>

          {selectedItem && (
            <>
              {selectedItem.customContent ? (
                <div className="mt-4">{selectedItem.customContent}</div>
              ) : selectedItem.code ? (
                <Tabs
                  value={activeLanguage}
                  onValueChange={(value) => {
                    setActiveLanguage(value);
                    setActiveSubTab(0);
                  }}
                  className="mt-4"
                >
                  {showLanguageTabs && (
                    <TabsList
                      className="grid w-full"
                      style={{
                        gridTemplateColumns: `repeat(${resolvedTabs.length}, 1fr)`,
                      }}
                    >
                      {resolvedTabs.map((tab) => {
                        const hasLanguageContent = selectedItem
                          ? hasContent(selectedItem, tab.key)
                          : false;
                        const isDisabled =
                          tab.disabled ||
                          (!hasLanguageContent && showLanguageTabs);
                        return (
                          <TabsTrigger
                            key={tab.key}
                            value={tab.key}
                            className="gap-2"
                            disabled={isDisabled}
                          >
                            <Code2 className="h-4 w-4" />
                            {isDisabled && tab.disabledLabel
                              ? tab.disabledLabel
                              : tab.label}
                          </TabsTrigger>
                        );
                      })}
                    </TabsList>
                  )}

                  {resolvedTabs.map((tab) => {
                    const codeValue = selectedItem.code?.[tab.key];
                    if (!codeValue || codeValue.length === 0) return null;
                    return (
                      <TabsContent
                        key={tab.key}
                        value={tab.key}
                        className="mt-4"
                      >
                        {isGroupedSteps(codeValue) ? (
                          <Tabs
                            value={String(activeSubTab)}
                            onValueChange={(value) =>
                              setActiveSubTab(Number(value))
                            }
                            className="mt-2"
                          >
                            <TabsList
                              className="grid w-full"
                              style={{
                                gridTemplateColumns: `repeat(${codeValue.length}, 1fr)`,
                              }}
                            >
                              {codeValue.map((method, idx) => (
                                <TabsTrigger key={idx} value={String(idx)}>
                                  {method.methodName}
                                </TabsTrigger>
                              ))}
                            </TabsList>

                            {codeValue.map((method, methodIdx) => (
                              <TabsContent
                                key={methodIdx}
                                value={String(methodIdx)}
                                className="mt-4 space-y-4"
                              >
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
                                              step.code,
                                              `${tab.key}-${methodIdx}-${stepIdx}`
                                            )
                                        : undefined
                                    }
                                    copied={
                                      copiedId ===
                                      `${tab.key}-${methodIdx}-${stepIdx}`
                                    }
                                  />
                                ))}
                              </TabsContent>
                            ))}
                          </Tabs>
                        ) : (
                          <div className="space-y-4">
                            {codeValue.map((step, index) => (
                              <CodeBlock
                                key={index}
                                label={step.label}
                                code={step.code}
                                image={step.image}
                                imageAlt={step.imageAlt}
                                onCopy={
                                  step.code
                                    ? () =>
                                        handleCopy(
                                          step.code,
                                          `${tab.key}-${index}`
                                        )
                                    : undefined
                                }
                                copied={copiedId === `${tab.key}-${index}`}
                              />
                            ))}
                          </div>
                        )}
                      </TabsContent>
                    );
                  })}
                </Tabs>
              ) : null}
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}

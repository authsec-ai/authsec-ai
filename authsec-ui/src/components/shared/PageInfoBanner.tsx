import { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ChevronDown, Check, Info, X, Undo2, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

// New interfaces for enhanced layout
export interface PageInfoBannerFeature {
  text: string;
  icon?: LucideIcon; // Optional custom icon (defaults to Check)
}

export interface PageInfoBannerAction {
  label: string;
  onClick: () => void;
  variant?: "default" | "secondary" | "outline";
  className?: string;
  icon?: LucideIcon;
}

export interface PageInfoBannerFAQ {
  id: string;
  question: string;
  answer?: string;
  customContent?: React.ReactNode;
}

// Legacy interface (deprecated but maintained for backward compatibility)
export interface PageInfoBannerSection {
  icon: LucideIcon;
  title: string;
  description: string;
}

export interface PageInfoBannerProps {
  // New API - Main content (left side)
  title?: string;
  description?: string;

  // New API - Features list (right side)
  features?: PageInfoBannerFeature[];
  featuresTitle?: string;

  // New API - Actions (below description)
  primaryAction?: PageInfoBannerAction;
  secondaryAction?: PageInfoBannerAction;

  // New API - FAQ section
  faqs?: PageInfoBannerFAQ[];
  faqsTitle?: string;

  // Banner behavior
  storageKey?: string;
  dismissible?: boolean; // Allow dismissing the banner

  // Styling
  className?: string;
  dismissButtonClassName?: string;

  // Legacy API (deprecated but maintained for backward compatibility)
  /** @deprecated Use title instead */
  summary?: string;
  /** @deprecated Use the new features/actions API instead */
  sections?: PageInfoBannerSection[];
}

export function PageInfoBanner({
  title,
  description,
  features,
  featuresTitle,
  primaryAction,
  secondaryAction,
  faqs,
  faqsTitle,
  storageKey,
  dismissible = false,
  className,
  dismissButtonClassName,
  // Legacy props
  summary,
  sections,
}: PageInfoBannerProps) {
  // Single dismiss state for the entire banner
  const [isDismissed, setIsDismissed] = useState<boolean>(() => {
    if (!storageKey || !dismissible) return false;
    try {
      const saved = localStorage.getItem(
        `pageInfoBanner_dismissed_${storageKey}`,
      );
      return saved === "true";
    } catch {
      return false;
    }
  });

  // Undo state management
  const [showUndo, setShowUndo] = useState(false);
  const [undoTimeLeft, setUndoTimeLeft] = useState(3);
  const undoTimerRef = useRef<NodeJS.Timeout | null>(null);
  const undoCountdownRef = useRef<NodeJS.Timeout | null>(null);
  const UNDO_DISMISS_SECONDS = 3;

  const handleDismiss = () => {
    setShowUndo(true);
    setUndoTimeLeft(UNDO_DISMISS_SECONDS);
    undoCountdownRef.current = setInterval(() => {
      setUndoTimeLeft((prev) => {
        if (prev <= 1) {
          if (undoCountdownRef.current) {
            clearInterval(undoCountdownRef.current);
          }
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    undoTimerRef.current = setTimeout(() => {
      setIsDismissed(true);
      setShowUndo(false);
      if (undoCountdownRef.current) {
        clearInterval(undoCountdownRef.current);
      }
    }, UNDO_DISMISS_SECONDS * 1000);
  };
  const handleUndo = () => {
    setShowUndo(false);
    if (undoTimerRef.current) {
      clearTimeout(undoTimerRef.current);
    }
    if (undoCountdownRef.current) {
      clearInterval(undoCountdownRef.current);
    }
  };
  useEffect(() => {
    return () => {
      if (undoTimerRef.current) {
        clearTimeout(undoTimerRef.current);
      }
      if (undoCountdownRef.current) {
        clearInterval(undoCountdownRef.current);
      }
    };
  }, []);
  useEffect(() => {
    if (storageKey && dismissible) {
      try {
        localStorage.setItem(
          `pageInfoBanner_dismissed_${storageKey}`,
          isDismissed ? "true" : "false",
        );
      } catch {
        // Silently fail if localStorage is unavailable
      }
    }
  }, [isDismissed, storageKey, dismissible]);

  const [expandedFAQId, setExpandedFAQId] = useState<string | null>(null);

  const handleFAQClick = (faqId: string) => {
    setExpandedFAQId(expandedFAQId === faqId ? null : faqId);
  };

  if (isDismissed && !showUndo) {
    return null;
  }

  if (showUndo) {
    return (
      <div data-slot="page-info-banner" data-state="undo" className={cn("relative", className)}>
        <Card className="border-[var(--editorial-border-soft)] bg-[var(--editorial-panel-soft)] shadow-none">
          <div className="p-4 flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <div className="rounded-md border border-[var(--editorial-border-soft)] bg-[var(--editorial-panel)] p-2">
                <X className="h-4 w-4 text-[var(--editorial-text-2)]" />
              </div>
              <div>
                <p className="text-sm font-medium text-[var(--editorial-text-1)]">
                  Banner dismissed
                </p>
                <p className="text-xs text-[var(--editorial-text-3)]">
                  removing in {undoTimeLeft} second
                  {undoTimeLeft !== 1 ? "s" : ""}
                </p>
              </div>
            </div>
            <Button
              onClick={handleUndo}
              size="sm"
              variant="outline"
              className="border-[var(--editorial-border-soft)] bg-[var(--editorial-panel)] text-[var(--editorial-text-1)] hover:bg-[var(--editorial-panel-soft)] shadow-none"
            >
              <Undo2 className="h-3.5 w-3.5 mr-1.5" />
              Undo
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  const displayTitle = title || summary || "";

  const isNewLayout = !!(
    features ||
    description ||
    primaryAction ||
    secondaryAction
  );
  const hasLegacySections = !!(sections && sections.length > 0);

  const content = (
    <>
      {isNewLayout && (
        <Card
          data-page-info-banner-layout="structured"
          className="relative overflow-hidden border-[var(--editorial-border-soft)] bg-[var(--editorial-panel)] shadow-none"
        >
          {dismissible && (
            <Button
              data-slot="page-info-banner-dismiss"
              variant="ghost"
              size="sm"
              onClick={handleDismiss}
              className={cn(
                "absolute top-2 right-2 z-10 h-7 w-7 rounded-md p-0 text-[var(--editorial-text-2)] hover:bg-[var(--editorial-panel-soft)] transition-colors",
                dismissButtonClassName,
              )}
              aria-label="Dismiss banner"
            >
              <X className="h-3.5 w-3.5" />
            </Button>
          )}

          <div className="flex flex-col md:flex-row">
            {(title || description || primaryAction || secondaryAction) && (
              <div
                data-slot="page-info-banner-intro"
                data-pane-kind="intro"
                className="flex-1 bg-[var(--editorial-panel-soft)] p-6"
              >
                <div className="space-y-4">
                  <div
                    data-slot="page-info-banner-eyebrow"
                    className="inline-flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--editorial-text-3)]"
                  >
                    <Info className="h-3.5 w-3.5 text-[var(--editorial-accent)]" />
                    <span data-slot="page-info-banner-eyebrow-label">Help Guide</span>
                  </div>
                  {title && (
                    <h2 className="text-base lg:text-lg font-bold leading-tight text-[var(--editorial-text-1)] [font-family:Manrope,var(--font-family-sans)]">
                      {title}
                    </h2>
                  )}
                  {description && (
                    <p className="text-sm leading-relaxed text-[var(--editorial-text-2)]">
                      {description}
                    </p>
                  )}
                  {(primaryAction || secondaryAction) && (
                    <div className="flex flex-wrap gap-3 items-center">
                      {primaryAction && (
                        <div data-slot="page-info-banner-primary-action">
                          <Button
                            onClick={primaryAction.onClick}
                            size="sm"
                            className={cn(
                              "border border-[var(--editorial-border-soft)] bg-[var(--editorial-panel)] px-4 py-2 text-sm font-medium text-[var(--editorial-text-1)] shadow-none transition-colors hover:bg-[var(--editorial-panel-alt)] cursor-pointer",
                              primaryAction.className,
                            )}
                          >
                            {primaryAction.label}
                          </Button>
                        </div>
                      )}
                      {secondaryAction && (
                        <div data-slot="page-info-banner-secondary-action">
                          <button
                            onClick={secondaryAction.onClick}
                            className={cn(
                              "text-sm font-medium text-[var(--editorial-accent)] hover:opacity-90 underline underline-offset-4 transition-colors",
                              secondaryAction.className,
                            )}
                          >
                            {secondaryAction.label}
                          </button>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </div>
            )}
            {(title || description || primaryAction || secondaryAction) &&
              features &&
              features.length > 0 && (
                <div
                  data-slot="page-info-banner-divider"
                  className="border-t border-[var(--editorial-border-soft)] md:border-t-0 md:border-l"
                />
              )}
            {features && features.length > 0 && (
              <div
                data-slot="page-info-banner-features"
                data-pane-kind="features"
                className="flex-1 bg-[var(--editorial-panel-alt)] p-6"
              >
                <div className="space-y-3">
                  {featuresTitle && (
                    <h3 className="text-sm font-semibold text-[var(--editorial-text-1)] [font-family:Manrope,var(--font-family-sans)]">
                      {featuresTitle}
                    </h3>
                  )}
                  <ul className="space-y-2">
                    {features.map((feature, index) => {
                      const IconComponent = feature.icon || Check;
                      return (
                        <li key={index} className="flex items-start gap-2">
                          <div className="flex-shrink-0 mt-0.5">
                            <div className="flex h-4 w-4 items-center justify-center rounded-full border border-[var(--editorial-border-soft)] bg-[var(--editorial-panel)]">
                              <IconComponent
                                className="h-3 w-3 text-[var(--editorial-accent)]"
                                strokeWidth={2.5}
                              />
                            </div>
                          </div>
                          <span className="text-sm leading-relaxed text-[var(--editorial-text-1)]">
                            {feature.text}
                          </span>
                        </li>
                      );
                    })}
                  </ul>
                </div>
              </div>
            )}
            {features && features.length > 0 && faqs && faqs.length > 0 && (
              <div
                data-slot="page-info-banner-divider"
                className="border-t border-[var(--editorial-border-soft)] md:border-t-0 md:border-l"
              />
            )}
            {faqs && faqs.length > 0 && (
              <div
                data-slot="page-info-banner-faqs"
                data-pane-kind="faqs"
                className="flex-1 bg-[var(--editorial-panel-soft)] p-6"
              >
                <div className="space-y-3">
                  {faqsTitle && (
                    <h3 className="text-sm font-semibold text-[var(--editorial-text-1)] [font-family:Manrope,var(--font-family-sans)]">
                      {faqsTitle}
                    </h3>
                  )}
                  <div className="space-y-1.5">
                    {faqs.map((faq) => {
                      const isExpanded = expandedFAQId === faq.id;
                      return (
                        <div
                          key={faq.id}
                          className="overflow-hidden rounded-md border border-[var(--editorial-border-soft)] bg-[var(--editorial-panel)]"
                        >
                          <button
                            onClick={() => handleFAQClick(faq.id)}
                            className="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm font-medium text-[var(--editorial-text-1)] transition-colors duration-200 cursor-pointer hover:bg-[var(--editorial-panel-soft)]"
                          >
                            <span className="text-xs">{faq.question}</span>
                            <ChevronDown
                              className={cn(
                                "ml-2 h-3.5 w-3.5 flex-shrink-0 text-[var(--editorial-text-3)] transition-transform duration-300",
                                isExpanded && "rotate-180",
                              )}
                            />
                          </button>
                          <div
                            className={cn(
                              "grid transition-all duration-300 ease-in-out",
                              isExpanded
                                ? "grid-rows-[1fr] opacity-100"
                                : "grid-rows-[0fr] opacity-0",
                            )}
                          >
                            <div className="overflow-hidden">
                              <div className="px-3 pb-2 pt-1">
                                {faq.customContent || (
                                  <p className="text-xs leading-relaxed text-[var(--editorial-text-2)]">
                                    {faq.answer}
                                  </p>
                                )}
                              </div>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              </div>
            )}
          </div>
        </Card>
      )}
      {hasLegacySections && (
        <div
          className={cn(
            isNewLayout &&
              "mt-8 border-t border-[var(--editorial-border-soft)] pt-8",
          )}
        >
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {sections!.map((section) => {
              const IconComponent = section.icon;
              return (
                <div
                  key={section.title}
                  className="relative rounded-md border border-[var(--editorial-border-soft)] bg-[var(--editorial-panel)] p-4 transition-colors duration-200 hover:bg-[var(--editorial-panel-soft)]"
                >
                  <div className="flex items-center gap-2.5 mb-3">
                    <div className="rounded-md border border-[var(--editorial-border-soft)] bg-[var(--editorial-panel-soft)] p-2">
                      <IconComponent className="h-4 w-4 text-[var(--editorial-accent)]" />
                    </div>
                    <h3 className="text-sm font-bold text-[var(--editorial-text-1)] [font-family:Manrope,var(--font-family-sans)]">
                      {section.title}
                    </h3>
                  </div>
                  <p className="pl-9 text-sm leading-relaxed text-[var(--editorial-text-2)]">
                    {section.description}
                  </p>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </>
  );
  return (
    <div data-slot="page-info-banner" data-state="default" className={cn("relative", className)}>
      {content}
    </div>
  );
}

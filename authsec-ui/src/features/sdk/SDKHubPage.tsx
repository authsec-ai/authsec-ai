import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { ArrowUpRight, Maximize2, Minimize2 } from "lucide-react";
import { PrismLight as SyntaxHighlighter } from "react-syntax-highlighter";
import { oneDark } from "react-syntax-highlighter/dist/esm/styles/prism";
import tsLanguage from "react-syntax-highlighter/dist/esm/languages/prism/typescript";
import pythonLanguage from "react-syntax-highlighter/dist/esm/languages/prism/python";
import bashLanguage from "react-syntax-highlighter/dist/esm/languages/prism/bash";
import jsonLanguage from "react-syntax-highlighter/dist/esm/languages/prism/json";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";

import {
  SDK_CATALOG,
  SDK_CATALOG_MAP,
  SDK_MODULE_LABELS,
  type SDKCatalogItem,
  type SDKCodeLanguage,
  type SDKHubSection,
  type SDKHubSnippet,
  type SDKId,
} from "./data/sdk-hub-catalog";
import { inferHubModuleFromSurface, isSDKHubModule } from "./utils/hub-routing";
import "./sdk-editorial-theme.css";

const SDK_TAB_ORDER: SDKId[] = ["typescript", "python"];
const SNIPPET_DISPLAY_ORDER: SDKCodeLanguage[] = ["typescript", "python", "bash", "json"];

const LANGUAGE_LABELS: Record<SDKCodeLanguage, string> = {
  bash: "Bash",
  typescript: "TypeScript",
  python: "Python",
  json: "JSON",
};

const SDK_TAB_LABELS: Record<SDKId, string> = {
  typescript: "TypeScript",
  python: "Python",
};

const SYNTAX_LANGUAGE_MAP: Record<SDKCodeLanguage, string> = {
  typescript: "typescript",
  python: "python",
  bash: "bash",
  json: "json",
};

const LANGUAGE_TABS_LIST_CLASS =
  "sdk-editorial-tabs-list inline-flex h-9 max-w-full justify-start overflow-x-auto overflow-y-hidden scrollbar-hide rounded-md border border-border/70 bg-background/80 p-1";
const LANGUAGE_TABS_TRIGGER_CLASS =
  "sdk-editorial-tabs-trigger sdk-editorial-tabs-trigger--language h-7 min-w-[98px] rounded-sm border border-transparent px-2.5 text-sm font-medium text-muted-foreground transition-all hover:text-foreground data-[state=active]:border-border data-[state=active]:bg-foreground data-[state=active]:text-background";
const SNIPPET_TABS_LIST_CLASS =
  "sdk-editorial-tabs-list sdk-editorial-tabs-list--snippet h-8 w-full justify-start overflow-x-auto overflow-y-hidden scrollbar-hide rounded-md border border-border/60 bg-muted/25 p-1";
const SNIPPET_TABS_TRIGGER_CLASS =
  "sdk-editorial-tabs-trigger sdk-editorial-tabs-trigger--snippet h-6 min-w-[118px] rounded-sm border border-transparent px-2.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-border/70 data-[state=active]:bg-background data-[state=active]:text-foreground";

SyntaxHighlighter.registerLanguage("typescript", tsLanguage);
SyntaxHighlighter.registerLanguage("python", pythonLanguage);
SyntaxHighlighter.registerLanguage("bash", bashLanguage);
SyntaxHighlighter.registerLanguage("json", jsonLanguage);

interface SDKCodePanelProps {
  label: string;
  language: SDKCodeLanguage;
  code: string;
  fillHeight?: boolean;
}

interface SDKOptionCard {
  key: string;
  sdk: SDKCatalogItem;
  section: SDKHubSection;
  snippets: SDKHubSnippet[];
}

interface SDKHubHeaderProps {
  contextBadge: string | null;
  availableSDKTabs: SDKId[];
  activeSDKId: SDKId;
  onLanguageTabChange: (value: string) => void;
}

interface SDKModuleNavProps {
  activeSDKName: string;
  optionCards: SDKOptionCard[];
  selectedCardKey?: string;
  onSelectCard: (card: SDKOptionCard) => void;
}

interface SDKGuideHeaderProps {
  card: SDKOptionCard;
}

interface InstallCommandBarProps {
  command: string;
  sdkName: string;
}

interface SnippetTabsProps {
  card: SDKOptionCard;
  selectedSnippet: SDKHubSnippet | null;
  onSnippetChange: (snippetId: string) => void;
  fillHeight?: boolean;
}

function isSDKId(value: string | null | undefined): value is SDKId {
  return value === "typescript" || value === "python";
}

function buildSDKBasePath(surface?: string, entityId?: string): string {
  return surface
    ? `/sdk/${surface}${entityId ? `/${encodeURIComponent(entityId)}` : ""}`
    : "/sdk";
}

function getSnippetPriority(language: SDKCodeLanguage): number {
  const index = SNIPPET_DISPLAY_ORDER.indexOf(language);
  return index === -1 ? Number.MAX_SAFE_INTEGER : index;
}

function getCardSnippets(section: SDKHubSection, sdkId: SDKId): SDKHubSnippet[] {
  return section.snippets
    .filter(
      (snippet) =>
        snippet.language === sdkId ||
        snippet.language === "bash" ||
        snippet.language === "json",
    )
    .sort((a, b) => getSnippetPriority(a.language) - getSnippetPriority(b.language));
}

function openExternal(url: string): void {
  window.open(url, "_blank", "noopener,noreferrer");
}

function getInstallTokenClass(token: string, index: number): string {
  if (token === "|") return "text-slate-200";
  if (token.startsWith("http://") || token.startsWith("https://")) {
    return "text-orange-300";
  }
  if (token.startsWith("-")) return "text-sky-300";
  if (token.startsWith("@")) return "text-indigo-300";
  if (index === 0) return "text-lime-200";
  return "text-slate-100";
}

function getSnippetHeaderComment(language: SDKCodeLanguage, label: string): string {
  const prefix = language === "python" || language === "bash" ? "#" : "//";
  return `${prefix} ${label}`;
}

function SDKHubHeader({
  contextBadge,
  availableSDKTabs,
  activeSDKId,
  onLanguageTabChange,
}: SDKHubHeaderProps) {
  return (
    <section className="sdk-hub-header flex flex-col gap-2 px-3 py-2.5 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <div className="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-1">
          <h1 className="text-xl font-semibold tracking-tight dash-text-1">SDK Hub</h1>
          <p className="text-sm dash-text-2">Language + module examples</p>
        </div>
      </div>

      <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center sm:gap-3">
        <div className="min-w-0 space-y-0.5">
          <p className="sdk-hub-label">
            Language
          </p>
          <Tabs value={activeSDKId} onValueChange={onLanguageTabChange}>
            <TabsList className={LANGUAGE_TABS_LIST_CLASS}>
              {availableSDKTabs.map((sdkId) => (
                <TabsTrigger key={sdkId} value={sdkId} className={LANGUAGE_TABS_TRIGGER_CLASS}>
                  {SDK_TAB_LABELS[sdkId] ?? sdkId}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </div>

        {contextBadge ? (
          <div className="flex items-center gap-2 sm:pl-1">
            <span className="sdk-hub-label">
              Context
            </span>
            <Badge
              variant="outline"
              className="sdk-hub-context-badge w-fit rounded-md text-xs"
            >
              {contextBadge}
            </Badge>
          </div>
        ) : null}
      </div>
    </section>
  );
}

function SDKModuleNav({
  activeSDKName,
  optionCards,
  selectedCardKey,
  onSelectCard,
}: SDKModuleNavProps) {
  return (
    <aside className="flex h-full min-h-0 flex-col">
      <div className="space-y-2.5">
        <div className="space-y-1 px-1">
          <p className="sdk-hub-label">
            Modules
          </p>
          <p className="text-sm dash-text-2">
            {optionCards.length} in {activeSDKName}
          </p>
        </div>

        <nav
          className="min-h-0 flex-1 space-y-1 overflow-y-auto pr-1 scrollbar-hide"
          aria-label={`${activeSDKName} SDK modules`}
        >
          {optionCards.map((card, index) => {
            const isSelected = selectedCardKey === card.key;

            return (
              <button
                key={card.key}
                type="button"
                onClick={() => onSelectCard(card)}
                title={card.section.summary}
                data-selected={isSelected ? "true" : "false"}
                className={cn(
                  "sdk-hub-module-button group relative w-full px-3 py-2 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500/40",
                )}
                aria-current={isSelected ? "page" : undefined}
              >
                <span
                  aria-hidden="true"
                  data-selected={isSelected ? "true" : "false"}
                  className={cn(
                    "sdk-hub-module-indicator absolute inset-y-1.5 left-0 w-0.5 rounded-r-full transition-colors",
                    !isSelected && "bg-transparent",
                  )}
                />
                <span
                  className="sdk-hub-module-index block text-[10px] font-semibold tracking-[0.14em]"
                >
                  {String(index + 1).padStart(2, "0")}
                </span>
                <span
                  className="sdk-hub-module-title mt-0.5 block text-sm font-medium leading-5"
                >
                  {SDK_MODULE_LABELS[card.section.key]}
                </span>
              </button>
            );
          })}
        </nav>
      </div>
    </aside>
  );
}

function SDKGuideHeader({ card }: SDKGuideHeaderProps) {
  const tooltipText = [card.section.summary, ...card.section.highlights]
    .filter(Boolean)
    .join(" | ");

  return (
    <section className="sdk-hub-guide-header flex flex-col gap-3 pb-3">
      <div className="flex flex-col gap-2 xl:flex-row xl:items-center xl:justify-between">
        <div className="min-w-0 space-y-1.5" title={tooltipText}>
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-xl font-semibold tracking-tight dash-text-1">{card.section.title}</h2>
            <Badge
              variant="outline"
              className="sdk-hub-guide-badge rounded-md px-2 py-0 text-[10px] uppercase tracking-[0.14em]"
            >
              {card.sdk.name}
            </Badge>
          </div>
        </div>

        <div className="flex items-center xl:shrink-0">
          <Button
            variant="secondary"
            size="sm"
            className="sdk-hub-docs-button h-6 min-h-0 gap-1 px-2 py-0 text-[10px] leading-none font-medium"
            onClick={() => openExternal(card.sdk.docsUrl)}
            title={`Open package docs (${card.sdk.docsUrl})`}
            aria-label={`Open package docs for ${card.sdk.name}`}
          >
            Package docs
            <ArrowUpRight className="h-2.5 w-2.5" />
          </Button>
        </div>
      </div>
    </section>
  );
}

function InstallCommandBar({ command, sdkName }: InstallCommandBarProps) {
  return (
    <section className="space-y-2">
      <p className="sdk-hub-label">
        Install (macOS, Linux, WSL)
      </p>

      <div className="rounded-sm border border-slate-800/80 bg-[linear-gradient(180deg,#081021,#050a13)] px-4 py-3 shadow-[0_10px_28px_rgba(2,6,23,0.22)]">
        <div className="flex items-center gap-3">
          <div className="min-w-0 flex-1 overflow-x-auto scrollbar-hide font-mono text-sm whitespace-nowrap">
            {command.split(" ").map((token, index) => (
              <span key={`${token}-${index}`} className={getInstallTokenClass(token, index)}>
                {index > 0 ? " " : ""}
                {token}
              </span>
            ))}
          </div>

          <CopyButton
            text={command}
            label={`${sdkName} install command`}
            variant="ghost"
            className="h-8 w-8 rounded-md border border-slate-700/80 text-slate-300 hover:bg-slate-800/70 hover:text-slate-100"
          />
        </div>
      </div>
    </section>
  );
}

function SnippetTabs({ card, selectedSnippet, onSnippetChange, fillHeight }: SnippetTabsProps) {
  if (!selectedSnippet) {
    return null;
  }

  if (card.snippets.length <= 1) {
    return (
      <SDKCodePanel
        label={selectedSnippet.label}
        language={selectedSnippet.language}
        code={selectedSnippet.code}
        fillHeight={fillHeight}
      />
    );
  }

  return (
    <Tabs
      value={selectedSnippet.id}
      onValueChange={onSnippetChange}
      className={cn("space-y-3", fillHeight && "flex min-h-0 flex-1 flex-col")}
    >
      <div className="space-y-2">
        <div className="flex items-center justify-between gap-3">
          <p className="sdk-hub-label">
            Examples
          </p>
          <span className="text-xs dash-text-3">
            {card.snippets.length} snippets
          </span>
        </div>

        <TabsList className={SNIPPET_TABS_LIST_CLASS}>
          {card.snippets.map((snippet) => (
            <TabsTrigger key={snippet.id} value={snippet.id} className={SNIPPET_TABS_TRIGGER_CLASS}>
              {snippet.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </div>

      {card.snippets.map((snippet) => (
        <TabsContent
          key={snippet.id}
          value={snippet.id}
          className={cn("mt-0", fillHeight && "min-h-0 flex-1")}
        >
          <SDKCodePanel
            label={snippet.label}
            language={snippet.language}
            code={snippet.code}
            fillHeight={fillHeight}
          />
        </TabsContent>
      ))}
    </Tabs>
  );
}

function SDKCodePanel({ label, language, code, fillHeight = false }: SDKCodePanelProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const displayCode = `${getSnippetHeaderComment(language, label)}\n\n${code}`;

  useEffect(() => {
    if (!isExpanded) return;

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsExpanded(false);
      }
    };
    window.addEventListener("keydown", handleEscape);

    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", handleEscape);
    };
  }, [isExpanded]);

  return (
    <>
      {isExpanded && (
        <button
          type="button"
          aria-label="Close expanded code view"
          className="fixed inset-0 z-[110] bg-slate-950/60 backdrop-blur-[1px]"
          onClick={() => setIsExpanded(false)}
        />
      )}

      <div
        data-sdk-code-block="true"
        className={cn(
          "overflow-hidden rounded-sm border border-slate-700/70 bg-[#0a1222] shadow-[0_10px_30px_rgba(2,6,23,0.3)] transition-[width,height,transform]",
          fillHeight && "flex h-full min-h-0 flex-col",
          isExpanded &&
            "fixed inset-5 z-[120] w-auto max-w-none rounded-sm border-slate-600/80 shadow-[0_24px_80px_rgba(2,6,23,0.75)]",
        )}
      >
        <div className="flex items-center justify-between gap-3 border-b border-slate-700/70 bg-[#10192d] px-3 py-2">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="truncate text-sm font-medium text-slate-100/95">{label}</span>
              <Badge
                variant="outline"
                className="border-slate-600/80 bg-slate-800/60 text-[10px] uppercase tracking-wide text-slate-200"
              >
                {LANGUAGE_LABELS[language]}
              </Badge>
            </div>
          </div>

          <div className="flex items-center gap-1.5">
            <CopyButton
              text={code}
              label={label}
              variant="outline"
              className="border-slate-600/80 bg-slate-800/80 text-slate-200 hover:bg-slate-700 hover:text-slate-100"
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={() => setIsExpanded((prev) => !prev)}
              aria-label={isExpanded ? "Collapse code panel" : "Expand code panel"}
              className="h-8 w-8 border-slate-600/80 bg-slate-800/80 text-slate-300 hover:bg-slate-700 hover:text-slate-100"
              title={isExpanded ? "Collapse code panel" : "Maximize code panel"}
            >
              {isExpanded ? (
                <Minimize2 className="h-3.5 w-3.5" />
              ) : (
                <Maximize2 className="h-3.5 w-3.5" />
              )}
            </Button>
          </div>
        </div>

        <div
          className={cn(
            "overflow-auto scrollbar-hide bg-[#0a1222] px-3 py-3 font-mono text-[13px] leading-7 text-slate-100 md:text-sm",
            fillHeight && !isExpanded && "min-h-0 flex-1",
            isExpanded ? "max-h-[calc(100vh-6rem)]" : !fillHeight && "max-h-[460px]",
          )}
        >
          <SyntaxHighlighter
            language={SYNTAX_LANGUAGE_MAP[language]}
            style={oneDark}
            showLineNumbers
            wrapLongLines={false}
            customStyle={{
              margin: 0,
              padding: 0,
              background: "transparent",
              fontSize: "0.92rem",
              lineHeight: 1.7,
            }}
            lineNumberStyle={{
              minWidth: "2.5em",
              paddingRight: "1em",
              color: "rgba(148, 163, 184, 0.6)",
              userSelect: "none",
            }}
            PreTag="div"
            codeTagProps={{
              style: {
                fontFamily:
                  "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, monospace",
              },
            }}
          >
            {displayCode}
          </SyntaxHighlighter>
        </div>
      </div>
    </>
  );
}

export function SDKHubPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { surface, entityId } = useParams<{
    surface?: string;
    entityId?: string;
  }>();

  const fallbackModule = inferHubModuleFromSurface(surface);

  const langParam = searchParams.get("lang");
  const sdkParam = searchParams.get("sdk");
  const moduleParam = searchParams.get("module");
  const snippetParam = searchParams.get("snippet");

  const availableSDKTabs = useMemo(() => {
    const preferred = SDK_TAB_ORDER.filter((sdkId) => {
      const sdk = SDK_CATALOG_MAP[sdkId];
      return Boolean(sdk?.sections.length);
    });

    const additional = SDK_CATALOG.map((sdk) => sdk.id).filter(
      (sdkId) => !preferred.includes(sdkId),
    );

    return [...preferred, ...additional].filter((sdkId) => {
      const sdk = SDK_CATALOG_MAP[sdkId];
      return Boolean(sdk?.sections.some((section) => section.snippets.length > 0));
    });
  }, []);

  const defaultSDK = availableSDKTabs[0] ?? "typescript";
  const activeSDKId: SDKId = isSDKId(langParam)
    ? langParam
    : isSDKId(sdkParam)
      ? sdkParam
      : defaultSDK;
  const activeSDK = SDK_CATALOG_MAP[activeSDKId] ?? SDK_CATALOG[0];

  const contextBadge = useMemo(() => {
    if (!surface) return null;

    if (surface.toLowerCase() === "clients") {
      return entityId ? `Client: ${entityId}` : "Client SDK context";
    }

    if (surface.toLowerCase() === "external-services") {
      return entityId ? `Service: ${entityId}` : "External services SDK context";
    }

    if (surface.toLowerCase() === "rbac") {
      return entityId ? `RBAC Entity: ${entityId}` : "RBAC SDK context";
    }

    return entityId ? `${surface}: ${entityId}` : surface;
  }, [entityId, surface]);

  const updateQuery = (
    updates: Partial<Record<"lang" | "sdk" | "module" | "snippet", string | undefined>>,
  ) => {
    const nextParams = new URLSearchParams(searchParams);

    for (const [key, value] of Object.entries(updates)) {
      if (!value) {
        nextParams.delete(key);
      } else {
        nextParams.set(key, value);
      }
    }

    const basePath = buildSDKBasePath(surface, entityId);
    const query = nextParams.toString();

    navigate(query ? `${basePath}?${query}` : basePath, { replace: true });
  };

  const optionCards = useMemo<SDKOptionCard[]>(() => {
    return activeSDK.sections
      .map((section) => {
        const snippets = getCardSnippets(section, activeSDK.id);

        if (snippets.length === 0) {
          return null;
        }

        return {
          key: `${activeSDK.id}:${section.key}`,
          sdk: activeSDK,
          section,
          snippets,
        };
      })
      .filter((card): card is SDKOptionCard => card !== null);
  }, [activeSDK]);

  const selectedCard = useMemo(() => {
    if (optionCards.length === 0) {
      return null;
    }

    if (isSDKHubModule(moduleParam)) {
      const moduleMatch = optionCards.find((card) => card.section.key === moduleParam);
      if (moduleMatch) {
        return moduleMatch;
      }
    }

    if (fallbackModule && isSDKHubModule(fallbackModule)) {
      const fallbackMatch = optionCards.find(
        (card) => card.section.key === fallbackModule,
      );
      if (fallbackMatch) {
        return fallbackMatch;
      }
    }

    return optionCards[0] ?? null;
  }, [fallbackModule, moduleParam, optionCards]);

  const selectedSnippet = useMemo(() => {
    if (!selectedCard) {
      return null;
    }

    const matched = selectedCard.snippets.find((snippet) => snippet.id === snippetParam);
    return matched ?? selectedCard.snippets[0] ?? null;
  }, [selectedCard, snippetParam]);

  const handleLanguageTabChange = (value: string) => {
    if (!isSDKId(value)) return;

    const nextSDK = SDK_CATALOG_MAP[value];
    const hasCurrentModule =
      isSDKHubModule(moduleParam) &&
      Boolean(nextSDK?.sections.some((section) => section.key === moduleParam));

    updateQuery({
      lang: value,
      sdk: value,
      module: hasCurrentModule ? moduleParam : undefined,
      snippet: undefined,
    });
  };

  return (
    <div
      className="h-full w-full overflow-hidden"
      data-dashboard="overview"
      data-sdk-surface="hub"
    >
      <div className="dash-page h-full w-full p-3 sm:p-4">
        <div className="flex h-full flex-col gap-3">
          <SDKHubHeader
            contextBadge={contextBadge}
            availableSDKTabs={availableSDKTabs}
            activeSDKId={activeSDKId}
            onLanguageTabChange={handleLanguageTabChange}
          />

          {optionCards.length === 0 ? (
            <div className="sdk-hub-empty flex min-h-0 flex-1 items-center justify-center px-4 text-center text-sm">
              No SDK cards available for {activeSDK.name}.
            </div>
          ) : (
            <section className="sdk-hub-main-shell min-h-0 flex-1 overflow-hidden">
              <div className="grid h-full min-h-0 gap-0 lg:grid-cols-[252px_minmax(0,1fr)] xl:grid-cols-[268px_minmax(0,1fr)]">
                <div className="sdk-hub-nav-shell min-h-0 border-b px-3 py-3 lg:border-r lg:border-b-0 lg:px-3">
                  <SDKModuleNav
                    activeSDKName={activeSDK.name}
                    optionCards={optionCards}
                    selectedCardKey={selectedCard?.key}
                    onSelectCard={(card) =>
                      updateQuery({
                        lang: card.sdk.id,
                        sdk: card.sdk.id,
                        module: card.section.key,
                        snippet: undefined,
                      })
                    }
                  />
                </div>

                {selectedCard ? (
                  <section className="sdk-hub-content-shell min-h-0 min-w-0 p-3 sm:p-4">
                    <div className="flex h-full min-h-0 flex-col gap-3">
                      <SDKGuideHeader card={selectedCard} />

                      <InstallCommandBar
                        command={selectedCard.sdk.installCommand}
                        sdkName={selectedCard.sdk.name}
                      />

                      <SnippetTabs
                        card={selectedCard}
                        selectedSnippet={selectedSnippet}
                        onSnippetChange={(nextSnippetId) => updateQuery({ snippet: nextSnippetId })}
                        fillHeight
                      />
                    </div>
                  </section>
                ) : null}
              </div>
            </section>
          )}
        </div>
      </div>
    </div>
  );
}

export default SDKHubPage;

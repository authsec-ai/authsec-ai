import {
  DEFAULT_HELP_LANGUAGE_TABS,
  FloatingHelp,
  type FloatingHelpItem,
} from "@/components/shared/FloatingHelp";
import type { SDKHelpItem } from "../types";
import {
  buildSDKHubLink,
  getModuleFromLegacyDocsLink,
  inferHubModuleFromTitle,
  isSDKHubModule,
} from "../utils/hub-routing";

interface SDKQuickHelpProps {
  helpItems: SDKHelpItem[];
  title?: string;
}

export function SDKQuickHelp({
  helpItems,
  title = "SDK Integration",
}: SDKQuickHelpProps) {
  const items: FloatingHelpItem[] = helpItems.map((item) => {
    const moduleFromLegacyLink = getModuleFromLegacyDocsLink(item.docsLink);
    const moduleFromItem = isSDKHubModule(item.hubModule)
      ? item.hubModule
      : inferHubModuleFromTitle(`${title} ${item.question}`);
    const module = moduleFromLegacyLink ?? moduleFromItem;

    const docsLink =
      item.docsLink && !item.docsLink.startsWith("/docs/sdk/")
        ? item.docsLink
        : buildSDKHubLink({ module });

    return {
      id: item.id,
      question: item.question,
      description: item.description,
      code: item.code,
      docsLink,
    };
  });

  return (
    <FloatingHelp
      items={items}
      tooltipLabel={title}
      defaultLanguage="python"
      languageTabs={DEFAULT_HELP_LANGUAGE_TABS}
      visualVariant="editorial"
    />
  );
}

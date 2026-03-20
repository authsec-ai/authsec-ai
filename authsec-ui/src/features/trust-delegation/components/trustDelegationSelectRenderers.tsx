import { Check } from "lucide-react";
import type { SearchableSelectOption } from "@/components/ui/searchable-select";

export function renderTrustDelegationSingleSelectOption(
  option: SearchableSelectOption,
  isSelected: boolean,
) {
  return (
    <div className="flex w-full items-start justify-between gap-3">
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm font-medium text-foreground">
          {option.label}
        </div>
        {option.description ? (
          <div className="mt-0.5 truncate text-xs text-muted-foreground">
            {option.description}
          </div>
        ) : null}
      </div>
      <div className="mt-0.5 flex h-4 w-4 items-center justify-center">
        {isSelected ? <Check className="h-4 w-4 text-primary" /> : null}
      </div>
    </div>
  );
}

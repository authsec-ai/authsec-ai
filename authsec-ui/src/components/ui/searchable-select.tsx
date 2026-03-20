import * as React from "react";
import { Check, ChevronsUpDown, X, Search } from "lucide-react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

export interface SearchableSelectOption {
  value: string;
  label: string;
  description?: string;
  icon?: React.ReactNode;
  disabled?: boolean;
  group?: string;
}

interface BaseSearchableSelectProps {
  /** Available options to select from */
  options: SearchableSelectOption[];
  /** Placeholder text when nothing is selected */
  placeholder?: string;
  /** Placeholder for the search input */
  searchPlaceholder?: string;
  /** Text to show when no options match the search */
  emptyText?: string;
  /** Whether the select is disabled */
  disabled?: boolean;
  /** Additional className for the trigger button */
  className?: string;
  /** Additional className for the popover content */
  contentClassName?: string;
  /** Whether to show the search input */
  searchable?: boolean;
  /** Whether to show clear button (single select only) */
  clearable?: boolean;
  /** Custom render function for options */
  renderOption?: (option: SearchableSelectOption, isSelected: boolean) => React.ReactNode;
  /** Custom render function for selected value display */
  renderValue?: (selected: SearchableSelectOption | SearchableSelectOption[]) => React.ReactNode;
  /** Maximum height of the dropdown */
  maxHeight?: number;
  /** Align the popover */
  align?: "start" | "center" | "end";
  /** Side of the popover */
  side?: "top" | "bottom" | "left" | "right";
}

interface SingleSelectProps extends BaseSearchableSelectProps {
  /** Enable multi-select mode */
  multiple?: false;
  /** Currently selected value */
  value?: string;
  /** Callback when selection changes */
  onChange?: (value: string | undefined) => void;
}

interface MultiSelectProps extends BaseSearchableSelectProps {
  /** Enable multi-select mode */
  multiple: true;
  /** Currently selected values */
  value?: string[];
  /** Callback when selection changes */
  onChange?: (value: string[]) => void;
  /** Maximum number of items that can be selected */
  maxItems?: number;
  /** Whether to show "Select All" option */
  showSelectAll?: boolean;
  /** Maximum number of badges to show before collapsing */
  maxBadges?: number;
}

export type SearchableSelectProps = SingleSelectProps | MultiSelectProps;

const SearchableSelect = React.forwardRef<HTMLButtonElement, SearchableSelectProps>(
  (props, ref) => {
    const {
      options,
      placeholder = "Select...",
      searchPlaceholder = "Search...",
      emptyText = "No results found",
      disabled = false,
      className,
      contentClassName,
      searchable = true,
      clearable = true,
      renderOption,
      renderValue,
      maxHeight = 300,
      align = "start",
      side = "bottom",
    } = props;

    const [open, setOpen] = React.useState(false);
    const [search, setSearch] = React.useState("");

    const isMultiple = props.multiple === true;
    const value = props.value;
    const onChange = props.onChange;
    const maxItems = isMultiple ? (props as MultiSelectProps).maxItems : undefined;
    const showSelectAll = isMultiple ? (props as MultiSelectProps).showSelectAll : false;
    const maxBadges = isMultiple ? ((props as MultiSelectProps).maxBadges ?? 3) : 1;

    // Group options by their group property
    const groupedOptions = React.useMemo(() => {
      const groups: Record<string, SearchableSelectOption[]> = {};
      const ungrouped: SearchableSelectOption[] = [];

      options.forEach((option) => {
        if (option.group) {
          if (!groups[option.group]) {
            groups[option.group] = [];
          }
          groups[option.group].push(option);
        } else {
          ungrouped.push(option);
        }
      });

      return { groups, ungrouped };
    }, [options]);

    // Get selected options
    const selectedOptions = React.useMemo(() => {
      if (isMultiple) {
        const values = (value as string[]) || [];
        return options.filter((opt) => values.includes(opt.value));
      } else {
        const singleValue = value as string | undefined;
        return singleValue ? options.filter((opt) => opt.value === singleValue) : [];
      }
    }, [options, value, isMultiple]);

    // Filter options based on search
    const filteredOptions = React.useMemo(() => {
      if (!search.trim()) return options;
      const query = search.toLowerCase();
      return options.filter(
        (opt) =>
          opt.label.toLowerCase().includes(query) ||
          (opt.description && opt.description.toLowerCase().includes(query))
      );
    }, [options, search]);

    // Handle selection
    const handleSelect = (optionValue: string) => {
      if (isMultiple) {
        const currentValues = (value as string[]) || [];
        const isSelected = currentValues.includes(optionValue);

        let newValues: string[];
        if (isSelected) {
          newValues = currentValues.filter((v) => v !== optionValue);
        } else {
          if (maxItems && currentValues.length >= maxItems) {
            return;
          }
          newValues = [...currentValues, optionValue];
        }

        (onChange as (value: string[]) => void)?.(newValues);
      } else {
        const currentValue = value as string | undefined;
        const newValue = currentValue === optionValue ? undefined : optionValue;
        (onChange as (value: string | undefined) => void)?.(newValue);
        setOpen(false);
      }
      setSearch("");
    };

    // Handle select all
    const handleSelectAll = () => {
      if (!isMultiple) return;

      const currentValues = (value as string[]) || [];
      const allValues = options.filter((opt) => !opt.disabled).map((opt) => opt.value);
      const allSelected = allValues.every((v) => currentValues.includes(v));

      if (allSelected) {
        (onChange as (value: string[]) => void)?.([]);
      } else {
        const newValues = maxItems ? allValues.slice(0, maxItems) : allValues;
        (onChange as (value: string[]) => void)?.(newValues);
      }
    };

    // Handle remove badge
    const handleRemove = (optionValue: string, e: React.MouseEvent) => {
      e.stopPropagation();
      if (isMultiple) {
        const currentValues = (value as string[]) || [];
        (onChange as (value: string[]) => void)?.(
          currentValues.filter((v) => v !== optionValue)
        );
      } else {
        (onChange as (value: string | undefined) => void)?.(undefined);
      }
    };

    // Handle clear all
    const handleClear = (e: React.MouseEvent) => {
      e.stopPropagation();
      if (isMultiple) {
        (onChange as (value: string[]) => void)?.([]);
      } else {
        (onChange as (value: string | undefined) => void)?.(undefined);
      }
    };

    // Render the trigger content
    const renderTriggerContent = () => {
      if (renderValue && selectedOptions.length > 0) {
        return renderValue(isMultiple ? selectedOptions : selectedOptions[0]);
      }

      if (selectedOptions.length === 0) {
        return (
          <span className="text-foreground/50">{placeholder}</span>
        );
      }

      if (isMultiple) {
        const visibleBadges = selectedOptions.slice(0, maxBadges);
        const remainingCount = selectedOptions.length - maxBadges;

        return (
          <div className="flex flex-wrap gap-1 flex-1">
            {visibleBadges.map((opt) => (
              <Badge
                key={opt.value}
                variant="secondary"
                className="gap-1 px-2 py-0.5 text-xs bg-black/5 dark:bg-white/10 hover:bg-black/10 dark:hover:bg-white/15 border-0"
              >
                {opt.icon}
                <span className="truncate max-w-[100px]">{opt.label}</span>
                <button
                  type="button"
                  onClick={(e) => handleRemove(opt.value, e)}
                  className="ml-0.5 rounded-full hover:bg-black/10 dark:hover:bg-white/20 p-0.5"
                >
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            ))}
            {remainingCount > 0 && (
              <Badge
                variant="secondary"
                className="px-2 py-0.5 text-xs bg-black/5 dark:bg-white/10 border-0"
              >
                +{remainingCount}
              </Badge>
            )}
          </div>
        );
      }

      const selected = selectedOptions[0];
      return (
        <div className="flex items-center gap-2 flex-1 truncate">
          {selected.icon}
          <span className="truncate">{selected.label}</span>
        </div>
      );
    };

    // Render option item
    const renderOptionItem = (option: SearchableSelectOption) => {
      const isSelected = isMultiple
        ? ((value as string[]) || []).includes(option.value)
        : (value as string) === option.value;

      if (renderOption) {
        return renderOption(option, isSelected);
      }

      return (
        <div className="flex items-start gap-2 w-full">
          <div
            className={cn(
              "flex h-4 w-4 items-center justify-center rounded-sm border border-primary mt-0.5 flex-shrink-0",
              isSelected
                ? "bg-primary text-primary-foreground"
                : "opacity-50"
            )}
          >
            <Check className={cn("h-3 w-3", isSelected ? "opacity-100" : "opacity-0")} />
          </div>
          <div className="flex flex-col flex-1 min-w-0">
            <div className="flex items-center gap-2">
              {option.icon}
              <span className="font-medium text-sm truncate">{option.label}</span>
            </div>
            {option.description && (
              <span className="text-xs text-foreground/60 mt-0.5 line-clamp-2">
                {option.description}
              </span>
            )}
          </div>
        </div>
      );
    };

    const hasValue = isMultiple
      ? ((value as string[]) || []).length > 0
      : !!(value as string);

    const allSelected = isMultiple && showSelectAll
      ? options.filter((opt) => !opt.disabled).every((opt) => ((value as string[]) || []).includes(opt.value))
      : false;

    return (
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            ref={ref}
            variant="outline"
            role="combobox"
            aria-expanded={open}
            disabled={disabled}
            className={cn(
              "w-full justify-between h-10 px-3 font-normal",
              "bg-background hover:bg-background",
              "border-input",
              !hasValue && "text-foreground/50",
              className
            )}
          >
            {renderTriggerContent()}
            <div className="flex items-center gap-1 ml-2 flex-shrink-0">
              {clearable && hasValue && (
                <div
                  role="button"
                  tabIndex={-1}
                  onClick={handleClear}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      handleClear(e as any);
                    }
                  }}
                  className="rounded-full hover:bg-black/10 dark:hover:bg-white/20 p-0.5 cursor-pointer"
                >
                  <X className="h-4 w-4 text-foreground/50" />
                </div>
              )}
              <ChevronsUpDown className="h-4 w-4 text-foreground/50" />
            </div>
          </Button>
        </PopoverTrigger>
        <PopoverContent
          className={cn("p-0", contentClassName)}
          style={{ width: "var(--radix-popover-trigger-width)" }}
          align={align}
          side={side}
          onOpenAutoFocus={(e) => e.preventDefault()}
          onWheel={(e) => e.stopPropagation()}
        >
          {searchable && (
            <div className="flex items-center border-b px-3">
              <Search className="mr-2 h-4 w-4 shrink-0 opacity-50" />
              <input
                placeholder={searchPlaceholder}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="flex h-11 w-full rounded-md bg-transparent py-3 text-sm outline-none placeholder:text-foreground/50 disabled:cursor-not-allowed disabled:opacity-50"
              />
              {search && (
                <button
                  type="button"
                  onClick={() => setSearch("")}
                  className="p-1 hover:bg-accent rounded"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>
          )}
          <div
            className="overflow-y-auto overflow-x-hidden p-1 overscroll-contain"
            style={{ maxHeight }}
            onWheel={(e) => {
              // Ensure scroll works within the dropdown
              const target = e.currentTarget;
              const { scrollTop, scrollHeight, clientHeight } = target;
              const isAtTop = scrollTop === 0 && e.deltaY < 0;
              const isAtBottom = scrollTop + clientHeight >= scrollHeight && e.deltaY > 0;

              // Only stop propagation if we can scroll
              if (!isAtTop && !isAtBottom) {
                e.stopPropagation();
              }
            }}
          >
            {filteredOptions.length === 0 ? (
              <div className="py-6 text-center text-sm text-foreground/60">
                {emptyText}
              </div>
            ) : (
              <>
                {isMultiple && showSelectAll && filteredOptions.length > 0 && (
                  <>
                    <div
                      role="option"
                      onClick={handleSelectAll}
                      className="relative flex cursor-pointer select-none items-center rounded-sm px-2 py-2 text-sm outline-none hover:bg-accent hover:text-accent-foreground"
                    >
                      <div className="flex items-center gap-2">
                        <div
                          className={cn(
                            "flex h-4 w-4 items-center justify-center rounded-sm border border-primary",
                            allSelected
                              ? "bg-primary text-primary-foreground"
                              : "opacity-50"
                          )}
                        >
                          <Check className={cn("h-3 w-3", allSelected ? "opacity-100" : "opacity-0")} />
                        </div>
                        <span className="font-medium text-sm">
                          {allSelected ? "Deselect All" : "Select All"}
                        </span>
                      </div>
                    </div>
                    <div className="-mx-1 my-1 h-px bg-border" />
                  </>
                )}

                {/* Render options */}
                {filteredOptions.map((option) => (
                  <div
                    key={option.value}
                    role="option"
                    aria-selected={
                      isMultiple
                        ? ((value as string[]) || []).includes(option.value)
                        : (value as string) === option.value
                    }
                    data-disabled={option.disabled}
                    onClick={() => {
                      if (!option.disabled) {
                        handleSelect(option.value);
                      }
                    }}
                    className={cn(
                      "relative flex cursor-pointer select-none items-center rounded-sm px-2 py-2 text-sm outline-none",
                      "hover:bg-accent hover:text-accent-foreground",
                      option.disabled && "pointer-events-none opacity-50"
                    )}
                  >
                    {renderOptionItem(option)}
                  </div>
                ))}
              </>
            )}
          </div>
        </PopoverContent>
      </Popover>
    );
  }
);

SearchableSelect.displayName = "SearchableSelect";

export { SearchableSelect };

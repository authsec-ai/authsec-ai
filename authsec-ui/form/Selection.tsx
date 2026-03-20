import * as React from "react";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Check, Search, X, Plus, User, Users } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";

export interface SelectionItem {
  id: string;
  label: string;
  description?: string;
  image?: string;
  icon?: React.ReactNode;
  disabled?: boolean;
  metadata?: Record<string, any>;
}

interface SelectionProps {
  items: SelectionItem[];
  selectedIds: string[];
  onSelectionChange: (ids: string[]) => void;
  title?: string;
  description?: string;
  placeholder?: string;
  emptyMessage?: string;
  multiSelect?: boolean;
  className?: string;
  isLoading?: boolean;
  maxHeight?: string | number;
}

export function Selection({
  items,
  selectedIds,
  onSelectionChange,
  title,
  description,
  placeholder = "Search...",
  emptyMessage = "No items found",
  multiSelect = true,
  className,
  isLoading,
  maxHeight = "300px",
}: SelectionProps) {
  const [searchQuery, setSearchQuery] = React.useState("");

  const filteredItems = React.useMemo(() => {
    if (!searchQuery) return items;
    const lowerQuery = searchQuery.toLowerCase();
    return items.filter(
      (item) =>
        item.label.toLowerCase().includes(lowerQuery) ||
        item.description?.toLowerCase().includes(lowerQuery)
    );
  }, [items, searchQuery]);

  const handleToggle = (id: string) => {
    if (multiSelect) {
      if (selectedIds.includes(id)) {
        onSelectionChange(selectedIds.filter((i) => i !== id));
      } else {
        onSelectionChange([...selectedIds, id]);
      }
    } else {
      onSelectionChange(selectedIds.includes(id) ? [] : [id]);
    }
  };

  const handleRemove = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    onSelectionChange(selectedIds.filter((i) => i !== id));
  };

  const selectedItems = items.filter((item) => selectedIds.includes(item.id));

  return (
    <div className={cn("space-y-4", className)}>
      {(title || description) && (
        <div className="space-y-1">
          {title && <h3 className="text-sm font-medium leading-none">{title}</h3>}
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
      )}

      {/* Selected Items Summary (Chips) */}
      {selectedItems.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-2">
          {selectedItems.map((item) => (
            <Badge
              key={item.id}
              variant="secondary"
              className="pl-2 pr-1 py-1 flex items-center gap-1 transition-all hover:bg-secondary/80"
            >
              {item.icon && <span className="w-3 h-3 mr-1">{item.icon}</span>}
              <span className="max-w-[150px] truncate">{item.label}</span>
              <button
                onClick={(e) => handleRemove(item.id, e)}
                className="ml-1 hover:bg-background/50 rounded-full p-0.5 transition-colors"
              >
                <X className="h-3 w-3" />
                <span className="sr-only">Remove {item.label}</span>
              </button>
            </Badge>
          ))}
          <div className="text-xs text-muted-foreground flex items-center ml-1">
            {selectedItems.length} selected
          </div>
        </div>
      )}

      <div className="border rounded-lg bg-card shadow-sm overflow-hidden">
        {/* Search Bar */}
        <div className="p-3 border-b bg-muted/30">
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder={placeholder}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9 bg-background border-muted-foreground/20 focus-visible:ring-1"
            />
          </div>
        </div>

        {/* List */}
        <ScrollArea style={{ height: maxHeight }} className="w-full">
          <div className="p-2 space-y-1">
            {isLoading ? (
              <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
                Loading items...
              </div>
            ) : filteredItems.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-8 text-sm text-muted-foreground text-center px-4">
                <div className="bg-muted/50 p-3 rounded-full mb-2">
                  <Search className="h-5 w-5 opacity-50" />
                </div>
                <p>{emptyMessage}</p>
              </div>
            ) : (
              filteredItems.map((item) => {
                const isSelected = selectedIds.includes(item.id);
                return (
                  <div
                    key={item.id}
                    onClick={() => !item.disabled && handleToggle(item.id)}
                    className={cn(
                      "group flex items-center gap-3 p-3 rounded-md cursor-pointer transition-all duration-200 border border-transparent",
                      isSelected
                        ? "bg-primary/5 border-primary/10"
                        : "hover:bg-muted/50 hover:border-border/50",
                      item.disabled && "opacity-50 cursor-not-allowed"
                    )}
                  >
                    {/* Checkbox / Radio Indicator */}
                    <div
                      className={cn(
                        "flex items-center justify-center w-5 h-5 rounded border transition-colors",
                        isSelected
                          ? "bg-primary border-primary text-primary-foreground"
                          : "border-muted-foreground/30 bg-background group-hover:border-primary/50",
                        !multiSelect && "rounded-full"
                      )}
                    >
                      {isSelected && <Check className="h-3.5 w-3.5" />}
                    </div>

                    {/* Avatar / Icon */}
                    {(item.image || item.icon) && (
                      <div className="flex-shrink-0">
                        {item.image ? (
                          <Avatar className="h-9 w-9 border border-border/50">
                            <AvatarImage src={item.image} />
                            <AvatarFallback>{item.label.substring(0, 2).toUpperCase()}</AvatarFallback>
                          </Avatar>
                        ) : (
                          <div className="h-9 w-9 rounded-full bg-muted/50 flex items-center justify-center text-muted-foreground">
                            {item.icon}
                          </div>
                        )}
                      </div>
                    )}

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <p className={cn("font-medium text-sm truncate", isSelected && "text-primary")}>
                          {item.label}
                        </p>
                        {item.metadata?.badge && (
                          <Badge variant="outline" className="text-[10px] h-5 px-1.5">
                            {item.metadata.badge}
                          </Badge>
                        )}
                      </div>
                      {item.description && (
                        <p className="text-xs text-muted-foreground truncate">{item.description}</p>
                      )}
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </ScrollArea>
      </div>
    </div>
  );
}

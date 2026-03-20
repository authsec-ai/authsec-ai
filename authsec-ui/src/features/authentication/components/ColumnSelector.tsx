import React from "react";
import { Button } from "../../../components/ui/button";
import { 
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  DropdownMenuCheckboxItem,
} from "../../../components/ui/dropdown-menu";
import { Settings2, Eye, EyeOff, Sliders } from "lucide-react";

export interface ColumnConfig {
  id: string;
  label: string;
  description?: string;
  isVisible: boolean;
  isRequired?: boolean;
  category?: "basic" | "configuration" | "technical" | "settings" | "activity" | "actions";
}

interface ColumnSelectorProps {
  availableColumns: ColumnConfig[];
  selectedColumns: ColumnConfig[];
  onColumnsChange: (columns: ColumnConfig[]) => void;
  onReset?: () => void;
}

export function ColumnSelector({ 
  availableColumns,
  selectedColumns, 
  onColumnsChange, 
  onReset 
}: ColumnSelectorProps) {
  const handleToggleColumn = (columnId: string) => {
    const updatedColumns = selectedColumns.map(col => 
      col.id === columnId ? { ...col, isVisible: !col.isVisible } : col
    );
    onColumnsChange(updatedColumns);
  };

  const visibleCount = selectedColumns.filter(col => col.isVisible).length;
  const totalCount = selectedColumns.length;

  // Group columns by category
  const groupedColumns = selectedColumns.reduce((acc, column) => {
    const category = column.category || "basic";
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(column);
    return acc;
  }, {} as Record<string, ColumnConfig[]>);

  const categoryLabels = {
    basic: "Basic Information",
    configuration: "OAuth Configuration", 
    technical: "Technical Details",
    settings: "Settings & Preferences",
    activity: "Activity & Timeline",
    actions: "Actions"
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className="h-8">
          <Sliders className="h-4 w-4 mr-2" />
          Columns
          <span className="ml-1 text-xs text-foreground">
            ({visibleCount}/{totalCount})
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80">
        <DropdownMenuLabel className="flex items-center justify-between">
          <span>Customize Columns</span>
          {onReset && (
            <Button 
              variant="ghost" 
              size="sm" 
              onClick={onReset}
              className="h-6 px-2 text-xs"
            >
              Reset
            </Button>
          )}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        
        {Object.entries(groupedColumns).map(([category, categoryColumns]) => (
          <div key={category}>
            <DropdownMenuLabel className="text-xs font-medium text-foreground px-2 py-1">
              {categoryLabels[category as keyof typeof categoryLabels] || category}
            </DropdownMenuLabel>
            {categoryColumns.map((column) => (
              <DropdownMenuCheckboxItem
                key={column.id}
                checked={column.isVisible}
                onCheckedChange={() => handleToggleColumn(column.id)}
                disabled={column.isRequired}
                className="flex items-start gap-2 py-2"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm">{column.label}</span>
                    {column.isRequired && (
                      <span className="text-xs text-foreground">(Required)</span>
                    )}
                  </div>
                  {column.description && (
                    <div className="text-xs text-foreground mt-1">
                      {column.description}
                    </div>
                  )}
                </div>
                {column.isVisible ? (
                  <Eye className="h-3 w-3 text-foreground" />
                ) : (
                  <EyeOff className="h-3 w-3 text-foreground" />
                )}
              </DropdownMenuCheckboxItem>
            ))}
          </div>
        ))}
        
        <DropdownMenuSeparator />
        <DropdownMenuLabel className="text-xs text-foreground">
          {visibleCount} of {totalCount} columns visible
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <div className="px-2 py-1">
          <div className="text-xs text-foreground leading-relaxed">
            💡 <strong>Tip:</strong> Use the expand button (⚙️) to view detailed provider configuration
          </div>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
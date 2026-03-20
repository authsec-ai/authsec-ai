import { useState } from "react";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../components/ui/table";
import {
  ResponsiveTableProvider,
  useResponsiveTableContext,
} from "../../../components/ui/responsive-table";
import { Checkbox } from "../../../components/ui/checkbox";
import { Button } from "../../../components/ui/button";
import { Badge } from "../../../components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../../components/ui/dropdown-menu";
import { IconGripVertical } from "@tabler/icons-react";
import {
  ChevronDown,
  ChevronRight,
  MoreHorizontal,
  Edit,
  Copy,
  Trash2,
  Eye,
  EyeOff,
  Key,
  Lock,
  Shield,
  Database,
  Server,
  Calendar,
  AlertTriangle,
  RefreshCw,
  Download,
} from "lucide-react";

// Define the Secret type based on the mock data structure
interface Secret {
  id: string;
  name: string;
  type: "api_key" | "password" | "certificate" | "connection_string" | "token";
  description: string;
  tags: string[];
  createdAt: string;
  updatedAt: string;
  expiresAt: string;
  accessCount: number;
  lastAccessed: string;
  isExpired: boolean;
}

interface EnhancedVaultTableProps {
  data: Secret[];
  selectedSecrets: string[];
  onSelectAll: () => void;
  onSelectSecret: (secretId: string) => void;
  onEditSecret: (secretId: string) => void;
  onCopySecret: (secretId: string) => void;
  onDeleteSecret: (secretId: string) => void;
  onToggleVisibility: (secretId: string) => void;
  visibleSecrets: Set<string>;
  onCreateSecret: () => void;
}

interface DraggableRowProps {
  row: Secret;
  isExpanded: boolean;
  onToggleExpansion: (id: string) => void;
  selectedSecrets: string[];
  onSelectSecret: (secretId: string) => void;
  onEditSecret: (secretId: string) => void;
  onCopySecret: (secretId: string) => void;
  onDeleteSecret: (secretId: string) => void;
  onToggleVisibility: (secretId: string) => void;
  visibleSecrets: Set<string>;
}

function DraggableRow({
  row,
  isExpanded,
  onToggleExpansion,
  selectedSecrets,
  onSelectSecret,
  onEditSecret,
  onCopySecret,
  onDeleteSecret,
  onToggleVisibility,
  visibleSecrets,
}: DraggableRowProps) {
  const { visibleColumns } = useResponsiveTableContext();
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: row.id,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  /**
   * Get secret type icon
   */
  const getTypeIcon = (type: Secret["type"]) => {
    switch (type) {
      case "api_key":
        return Key;
      case "password":
        return Lock;
      case "certificate":
        return Shield;
      case "connection_string":
        return Database;
      case "token":
        return Server;
      default:
        return Key;
    }
  };

  /**
   * Get secret type color
   */
  const getTypeColor = (type: Secret["type"]) => {
    switch (type) {
      case "api_key":
        return "bg-blue-100 text-blue-800 border-blue-200";
      case "password":
        return "bg-red-100 text-red-800 border-red-200";
      case "certificate":
        return "bg-blue-100 text-blue-800 border-blue-200";
      case "connection_string":
        return "bg-green-100 text-green-800 border-green-200";
      case "token":
        return "bg-orange-100 text-orange-800 border-orange-200";
      default:
        return "bg-gray-100 text-gray-800 border-gray-200";
    }
  };

  /**
   * Format timestamp for display
   */
  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffInDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24));

    if (diffInDays === 0) {
      return "Today";
    } else if (diffInDays === 1) {
      return "Yesterday";
    } else if (diffInDays < 7) {
      return `${diffInDays}d ago`;
    } else {
      return date.toLocaleDateString();
    }
  };

  /**
   * Check if secret is expiring soon (within 30 days)
   */
  const isExpiringSoon = (expiresAt: string) => {
    const expiry = new Date(expiresAt);
    const now = new Date();
    const diffInDays = Math.floor((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
    return diffInDays <= 30 && diffInDays > 0;
  };

  const TypeIcon = getTypeIcon(row.type);
  const isVisible = visibleSecrets.has(row.id);
  const expiringSoon = isExpiringSoon(row.expiresAt);

  return (
    <>
      <TableRow
        ref={setNodeRef}
        style={style}
        className={`relative group hover:bg-muted/30 transition-all duration-200 border-b h-16 ${
          selectedSecrets.includes(row.id) ? "bg-muted/50" : ""
        } ${isDragging ? "shadow-lg bg-background border-2 border-primary/20" : ""}`}
      >
        {/* Drag Handle */}
        {visibleColumns.dragHandle && (
          <TableCell className="w-8 px-2">
            <Button
              {...attributes}
              {...listeners}
              variant="ghost"
              size="icon"
              className="text-foreground size-7 hover:bg-muted cursor-grab active:cursor-grabbing opacity-50 group-hover:opacity-100 transition-opacity"
            >
              <IconGripVertical className="text-foreground size-3" />
              <span className="sr-only">Drag to reorder</span>
            </Button>
          </TableCell>
        )}

        {/* Checkbox */}
        {visibleColumns.checkbox && (
          <TableCell className="w-12 px-3">
            <div className="flex items-center justify-center">
              <Checkbox
                checked={selectedSecrets.includes(row.id)}
                onCheckedChange={() => onSelectSecret(row.id)}
                aria-label={`Select ${row.name}`}
              />
            </div>
          </TableCell>
        )}

        {/* Secret */}
        {visibleColumns.secret && (
          <TableCell className="px-4 min-w-[180px]">
            <div className="flex items-center gap-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted flex-shrink-0">
                <TypeIcon className="h-4 w-4" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="font-medium sm:truncate" title={row.name}>
                  {row.name}
                </div>
                <div className="text-sm text-foreground sm:truncate" title={row.description}>
                  {row.description}
                </div>
              </div>
            </div>
          </TableCell>
        )}

        {/* Type & Tags */}
        {visibleColumns.typeTags && (
          <TableCell className="px-4">
            <div className="space-y-1">
              <Badge className={getTypeColor(row.type)} variant="outline">
                {row.type.replace("_", " ")}
              </Badge>
              <div className="flex gap-1">
                {row.tags.slice(0, 2).map((tag) => (
                  <Badge key={tag} variant="outline" className="text-xs">
                    {tag}
                  </Badge>
                ))}
                {row.tags.length > 2 && (
                  <Badge variant="outline" className="text-xs">
                    +{row.tags.length - 2}
                  </Badge>
                )}
              </div>
            </div>
          </TableCell>
        )}

        {/* Status */}
        {visibleColumns.status && (
          <TableCell className="px-4">
            {row.isExpired ? (
              <Badge variant="destructive">Expired</Badge>
            ) : expiringSoon ? (
              <Badge variant="secondary">
                <AlertTriangle className="h-3 w-3 mr-1" />
                Expiring Soon
              </Badge>
            ) : (
              <Badge variant="default">Active</Badge>
            )}
          </TableCell>
        )}

        {/* Activity */}
        {visibleColumns.activity && (
          <TableCell className="px-4">
            <div className="space-y-1">
              <div className="text-sm font-medium">{row.accessCount} accesses</div>
              <div className="flex items-center gap-1 text-sm text-foreground">
                <Calendar className="h-3 w-3" />
                <span>{formatTimestamp(row.lastAccessed)}</span>
              </div>
            </div>
          </TableCell>
        )}

        {/* Expires */}
        {visibleColumns.expires && (
          <TableCell className="px-4">
            <div className="text-sm">{new Date(row.expiresAt).toLocaleDateString()}</div>
          </TableCell>
        )}

        {/* Actions */}
        {visibleColumns.actions && (
          <TableCell className="w-32 px-4 text-center">
            <div className="flex items-center justify-end gap-1">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onToggleVisibility(row.id)}
                className="admin-row-icon-btn h-8 w-8 p-0"
              >
                {isVisible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" visualVariant="row-actions">
                  <DropdownMenuItem onClick={() => onCopySecret(row.id)}>
                    <Copy className="mr-2 h-4 w-4" />
                    Copy Value
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => onEditSecret(row.id)}>
                    <Edit className="mr-2 h-4 w-4" />
                    Edit Secret
                  </DropdownMenuItem>
                  <DropdownMenuItem>
                    <RefreshCw className="mr-2 h-4 w-4" />
                    Rotate Secret
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem>
                    <Download className="mr-2 h-4 w-4" />
                    Export
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => onDeleteSecret(row.id)}
                    className="text-destructive"
                  >
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>

              {/* Accordion Toggle Button - At the far right */}
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onToggleExpansion(row.id)}
                className="h-8 w-8 p-0 opacity-70 hover:opacity-100 transition-opacity duration-200"
              >
                {isExpanded ? (
                  <ChevronDown className="h-4 w-4" />
                ) : (
                  <ChevronRight className="h-4 w-4" />
                )}
              </Button>
            </div>
          </TableCell>
        )}
      </TableRow>

      {/* Expanded Row */}
      {isExpanded && <ExpandedRow secret={row} />}
    </>
  );
}

interface ExpandedRowProps {
  secret: Secret;
}

function ExpandedRow({ secret }: ExpandedRowProps) {
  const { visibleColumns } = useResponsiveTableContext();
  const colSpan = Object.values(visibleColumns).filter(Boolean).length;

  return (
    <TableRow>
      <TableCell colSpan={colSpan} className="p-0 bg-muted/20">
        <div className="p-6 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Secret Details */}
            <div className="space-y-3">
              <h4 className="font-medium flex items-center gap-2">
                <Key className="h-4 w-4" />
                Secret Details
              </h4>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-foreground">Secret ID:</span>
                  <span className="font-mono">{secret.id}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-foreground">Name:</span>
                  <span>{secret.name}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-foreground">Type:</span>
                  <Badge variant="outline" className="text-xs">
                    {secret.type.replace("_", " ")}
                  </Badge>
                </div>
                <div className="flex justify-between">
                  <span className="text-foreground">Description:</span>
                  <span className="text-right max-w-48">{secret.description}</span>
                </div>
              </div>
            </div>

            {/* Access & Security */}
            <div className="space-y-3">
              <h4 className="font-medium flex items-center gap-2">
                <Shield className="h-4 w-4" />
                Access & Security
              </h4>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-foreground">Access Count:</span>
                  <span>{secret.accessCount}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-foreground">Last Accessed:</span>
                  <span>{new Date(secret.lastAccessed).toLocaleDateString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-foreground">Status:</span>
                  <Badge variant={secret.isExpired ? "destructive" : "default"} className="text-xs">
                    {secret.isExpired ? "Expired" : "Active"}
                  </Badge>
                </div>
                <div className="flex justify-between">
                  <span className="text-foreground">Tags:</span>
                  <div className="flex gap-1 flex-wrap">
                    {secret.tags.map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </div>
              </div>
            </div>

            {/* Lifecycle */}
            <div className="space-y-3">
              <h4 className="font-medium flex items-center gap-2">
                <Calendar className="h-4 w-4" />
                Lifecycle
              </h4>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-foreground">Created:</span>
                  <span>{new Date(secret.createdAt).toLocaleDateString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-foreground">Updated:</span>
                  <span>{new Date(secret.updatedAt).toLocaleDateString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-foreground">Expires:</span>
                  <span>{new Date(secret.expiresAt).toLocaleDateString()}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Quick Actions */}
          <div className="flex gap-2 pt-4 border-t">
            <Button variant="outline" size="sm">
              <Copy className="mr-2 h-4 w-4" />
              Copy Value
            </Button>
            <Button variant="outline" size="sm">
              <RefreshCw className="mr-2 h-4 w-4" />
              Rotate Secret
            </Button>
            <Button variant="outline" size="sm">
              <Download className="mr-2 h-4 w-4" />
              Export
            </Button>
          </div>
        </div>
      </TableCell>
    </TableRow>
  );
}

export function EnhancedVaultTable({
  data,
  selectedSecrets,
  onSelectAll,
  onSelectSecret,
  onEditSecret,
  onCopySecret,
  onDeleteSecret,
  onToggleVisibility,
  visibleSecrets,
  onCreateSecret,
}: EnhancedVaultTableProps) {
  return (
    <ResponsiveTableProvider tableType="vault">
      <EnhancedVaultTableContent
        data={data}
        selectedSecrets={selectedSecrets}
        onSelectAll={onSelectAll}
        onSelectSecret={onSelectSecret}
        onEditSecret={onEditSecret}
        onCopySecret={onCopySecret}
        onDeleteSecret={onDeleteSecret}
        onToggleVisibility={onToggleVisibility}
        visibleSecrets={visibleSecrets}
        onCreateSecret={onCreateSecret}
      />
    </ResponsiveTableProvider>
  );
}

function EnhancedVaultTableContent({
  data,
  selectedSecrets,
  onSelectAll,
  onSelectSecret,
  onEditSecret,
  onCopySecret,
  onDeleteSecret,
  onToggleVisibility,
  visibleSecrets,
  onCreateSecret,
}: EnhancedVaultTableProps) {
  const [secrets, setSecrets] = useState(data);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const { visibleColumns } = useResponsiveTableContext();

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      setSecrets((items) => {
        const oldIndex = items.findIndex((item) => item.id === active.id);
        const newIndex = items.findIndex((item) => item.id === over.id);

        return arrayMove(items, oldIndex, newIndex);
      });
    }
  };

  const toggleRowExpansion = (id: string) => {
    setExpandedRows((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(id)) {
        newSet.delete(id);
      } else {
        newSet.add(id);
      }
      return newSet;
    });
  };

  const allSelected = selectedSecrets.length === secrets.length && secrets.length > 0;
  const someSelected = selectedSecrets.length > 0 && selectedSecrets.length < secrets.length;

  return (
    <div className="w-full space-y-6 shadow-xl">
      <div className="relative flex flex-col overflow-auto">
        <div className="overflow-hidden">
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={secrets.map((s) => s.id)}
              strategy={verticalListSortingStrategy}
            >
              <Table>
                <TableHeader className="bg-muted/50 sticky top-0 z-10">
                  <TableRow className="border-b">
                    {visibleColumns.dragHandle && <TableHead className="w-8 px-2"></TableHead>}
                    {visibleColumns.checkbox && (
                      <TableHead className="w-12 px-3">
                        <div className="flex items-center justify-center">
                          <Checkbox
                            checked={allSelected}
                            ref={(el) => {
                              if (el) (el as any).indeterminate = someSelected;
                            }}
                            onCheckedChange={onSelectAll}
                            aria-label="Select all"
                          />
                        </div>
                      </TableHead>
                    )}
                    {visibleColumns.secret && (
                      <TableHead className="px-4 font-semibold">Secret</TableHead>
                    )}
                    {visibleColumns.typeTags && (
                      <TableHead className="px-4 font-semibold">Type & Tags</TableHead>
                    )}
                    {visibleColumns.status && (
                      <TableHead className="px-4 font-semibold">Status</TableHead>
                    )}
                    {visibleColumns.activity && (
                      <TableHead className="px-4 font-semibold">Activity</TableHead>
                    )}
                    {visibleColumns.expires && (
                      <TableHead className="px-4 font-semibold">Expires</TableHead>
                    )}
                    {visibleColumns.actions && (
                      <TableHead className="w-32 px-4 font-semibold text-center">Actions</TableHead>
                    )}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {secrets.length > 0 ? (
                    secrets.map((secret) => {
                      const isExpanded = expandedRows.has(secret.id);
                      return (
                        <DraggableRow
                          key={secret.id}
                          row={secret}
                          isExpanded={isExpanded}
                          onToggleExpansion={toggleRowExpansion}
                          selectedSecrets={selectedSecrets}
                          onSelectSecret={onSelectSecret}
                          onEditSecret={onEditSecret}
                          onCopySecret={onCopySecret}
                          onDeleteSecret={onDeleteSecret}
                          onToggleVisibility={onToggleVisibility}
                          visibleSecrets={visibleSecrets}
                        />
                      );
                    })
                  ) : (
                    <TableRow>
                      <TableCell
                        colSpan={Object.values(visibleColumns).filter(Boolean).length}
                        className="h-24 text-center"
                      >
                        <div className="flex flex-col items-center gap-2">
                          <Key className="h-8 w-8 text-foreground" />
                          <div className="text-lg font-medium">No secrets found</div>
                          <div className="text-sm text-foreground">
                            Get started by creating your first secret
                          </div>
                          <Button onClick={onCreateSecret} className="mt-2">
                            <Key className="mr-2 h-4 w-4" />
                            Create Secret
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </SortableContext>
          </DndContext>
        </div>
      </div>
    </div>
  );
}

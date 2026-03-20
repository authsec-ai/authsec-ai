import React, { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  MoreHorizontal,
  Edit,
  Copy,
  Trash2,
  Server,
  Plus,
  Check,
  Clock,
  AlertCircle,
  CheckCircle,
} from "lucide-react";
import type { ExternalService } from "@/types/entities";

// Format date utility
const formatDate = (dateString?: string) => {
  if (!dateString) return "—";
  return new Date(dateString).toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
};

// Format relative time utility
const formatRelativeTime = (dateString?: string) => {
  if (!dateString) return "Never";
  const date = new Date(dateString);
  const now = new Date();
  const diffInDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24));
  
  if (diffInDays === 0) return "Today";
  if (diffInDays === 1) return "Yesterday";
  if (diffInDays < 7) return `${diffInDays} days ago`;
  if (diffInDays < 30) return `${Math.floor(diffInDays / 7)} weeks ago`;
  if (diffInDays < 365) return `${Math.floor(diffInDays / 30)} months ago`;
  return `${Math.floor(diffInDays / 365)} years ago`;
};

// Get status badge color
const getStatusBadgeColor = (status: string) => {
  switch (status) {
    case "connected":
      return "text-green-700 bg-green-50 border-green-200 dark:text-green-400 dark:bg-green-950/30 dark:border-green-800";
    case "needs_consent":
      return "text-amber-700 bg-amber-50 border-amber-200 dark:text-amber-400 dark:bg-amber-950/30 dark:border-amber-800";
    case "error":
      return "text-red-700 bg-red-50 border-red-200 dark:text-red-400 dark:bg-red-950/30 dark:border-red-800";
    default:
      return "text-gray-700 bg-gray-50 border-gray-200 dark:text-gray-400 dark:bg-gray-950/30 dark:border-gray-800";
  }
};

// Get provider icon
const getProviderIcon = (provider?: string) => {
  switch (provider?.toLowerCase()) {
    case "github":
    case "google":
    case "microsoft":
    case "slack":
      return CheckCircle;
    default:
      return Server;
  }
};


interface SimpleExternalServicesTableProps {
  data: ExternalService[];
  selectedServices: string[];
  onSelectAll: () => void;
  onSelectService: (serviceId: string) => void;
  onEditService: (serviceId: string) => void;
  onDuplicateService: (serviceId: string) => void;
  onDeleteService: (serviceId: string) => void;
  onCreateService: () => void;
}

export function SimpleExternalServicesTable({
  data = [],
  selectedServices = [],
  onSelectAll,
  onSelectService,
  onEditService,
  onDuplicateService,
  onDeleteService,
  onCreateService,
}: SimpleExternalServicesTableProps) {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  const toggleRowExpansion = (serviceId: string) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(serviceId)) {
      newExpanded.delete(serviceId);
    } else {
      newExpanded.add(serviceId);
    }
    setExpandedRows(newExpanded);
  };

  // Handle empty state
  if (!data || data.length === 0) {
    return (
      <div className="border rounded-sm">
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <Server className="w-12 h-12 text-slate-400 dark:text-neutral-500 mb-4" />
          <h3 className="text-lg font-semibold text-slate-900 dark:text-neutral-100 mb-2">
            No external services found
          </h3>
          <p className="text-slate-500 dark:text-neutral-400 mb-4 max-w-sm">
            Get started by connecting your first external service to enable integrations.
          </p>
          <Button onClick={onCreateService} className="flex items-center gap-2">
            <Plus className="w-4 h-4" />
            Add External Service
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="border rounded-sm overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow className="bg-slate-50/50 dark:bg-neutral-800/50 hover:bg-slate-50/50 dark:hover:bg-neutral-800/50">
            <TableHead className="w-12">
              <Checkbox
                checked={selectedServices.length === data.length && data.length > 0}
                indeterminate={selectedServices.length > 0 && selectedServices.length < data.length}
                onCheckedChange={onSelectAll}
                aria-label="Select all services"
              />
            </TableHead>
            <TableHead>Service</TableHead>
            <TableHead>Provider</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Last Sync</TableHead>
            <TableHead>Users</TableHead>
            <TableHead className="w-12"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((service) => {
            const ProviderIcon = getProviderIcon(service.provider);
            const isSelected = selectedServices.includes(service.id);
            const isExpanded = expandedRows.has(service.id);

            return (
              <React.Fragment key={service.id}>
                <TableRow 
                  className={`cursor-pointer transition-colors ${
                    isSelected ? 'bg-blue-50/50 dark:bg-blue-950/20' : 'hover:bg-slate-50/50 dark:hover:bg-neutral-800/30'
                  }`}
                  onClick={() => toggleRowExpansion(service.id)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => onSelectService(service.id)}
                      aria-label={`Select ${service.name}`}
                    />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center space-x-3">
                      <div className="flex-shrink-0">
                        <div className="w-8 h-8 rounded-full bg-slate-100 dark:bg-neutral-700 flex items-center justify-center">
                          <ProviderIcon className="w-4 h-4 text-slate-600 dark:text-neutral-300" />
                        </div>
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="font-medium text-slate-900 dark:text-neutral-100 truncate">
                          {service.name}
                        </div>
                        <div className="text-sm text-slate-500 dark:text-neutral-400 truncate">
                          {service.provider?.toUpperCase()} • {service.id}
                        </div>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="font-mono text-xs">
                      {service.provider}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant="secondary"
                      className={`${getStatusBadgeColor(service.status)} font-medium border`}
                    >
                      {service.status?.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="text-sm">
                      <div className="font-medium text-slate-900 dark:text-neutral-100">
                        {formatRelativeTime(service.lastSync)}
                      </div>
                      {service.lastSync && (
                        <div className="text-xs text-slate-500 dark:text-neutral-400">
                          {formatDate(service.lastSync)}
                        </div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center space-x-2">
                      <div className="font-medium">
                        {service.userTokenCount || 0}
                      </div>
                    </div>
                  </TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
                          <MoreHorizontal className="h-4 w-4" />
                          <span className="sr-only">Open menu</span>
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" visualVariant="row-actions" className="w-48">
                        <DropdownMenuItem onClick={() => onEditService(service.id)}>
                          <Edit className="mr-2 h-4 w-4" />
                          Edit
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => onDuplicateService(service.id)}>
                          <Copy className="mr-2 h-4 w-4" />
                          Duplicate
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => onDeleteService(service.id)}
                          className="text-red-600 dark:text-red-400"
                        >
                          <Trash2 className="mr-2 h-4 w-4" />
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
                
                {/* Expanded Row */}
                {isExpanded && (
                  <TableRow>
                    <TableCell colSpan={7} className="p-0">
                      <div className="p-6 bg-slate-50/50 dark:bg-neutral-900/50 border-t border-slate-200/50 dark:border-neutral-700/50">
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                          {/* Service Details */}
                          <div className="space-y-3">
                            <h4 className="font-semibold text-slate-900 dark:text-neutral-100 flex items-center gap-2">
                              <Server className="w-4 h-4" />
                              Service Details
                            </h4>
                            <div className="space-y-2 text-sm">
                              <div className="flex justify-between">
                                <span className="text-slate-600 dark:text-neutral-400">Service ID:</span>
                                <code className="bg-slate-100 dark:bg-neutral-800 px-2 py-1 rounded text-xs">
                                  {service.id}
                                </code>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-slate-600 dark:text-neutral-400">Provider:</span>
                                <span className="font-medium">{service.provider}</span>
                              </div>
                            </div>
                          </div>

                          {/* Status & Health */}
                          <div className="space-y-3">
                            <h4 className="font-semibold text-slate-900 dark:text-neutral-100 flex items-center gap-2">
                              <CheckCircle className="w-4 h-4" />
                              Status & Health
                            </h4>
                            <div className="space-y-2 text-sm">
                              <div className="flex justify-between">
                                <span className="text-slate-600 dark:text-neutral-400">Current Status:</span>
                                <Badge
                                  variant="secondary"
                                  className={`${getStatusBadgeColor(service.status)} text-xs`}
                                >
                                  {service.status?.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())}
                                </Badge>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-slate-600 dark:text-neutral-400">User Tokens:</span>
                                <span className="font-medium">{service.userTokenCount || 0}</span>
                              </div>
                            </div>
                          </div>

                          {/* Sync Information */}
                          <div className="space-y-3">
                            <h4 className="font-semibold text-slate-900 dark:text-neutral-100 flex items-center gap-2">
                              <Clock className="w-4 h-4" />
                              Sync Information
                            </h4>
                            <div className="space-y-2 text-sm">
                              <div className="flex justify-between">
                                <span className="text-slate-600 dark:text-neutral-400">Last Sync:</span>
                                <span className="font-medium">{formatDate(service.lastSync)}</span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-slate-600 dark:text-neutral-400">Sync Status:</span>
                                <span className="font-medium">{formatRelativeTime(service.lastSync)}</span>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </React.Fragment>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { ServerCog, Cloud, Mail, Plus, RefreshCw, AlertCircle, Trash2 } from "lucide-react";
import { useListSyncConfigsQuery, useDeleteSyncConfigMutation } from "@/app/api/syncConfigsApi";
import type { SyncConfig, SyncType } from "@/app/api/syncConfigsApi";
import {
  useSyncActiveDirectoryMutation,
  useSyncEntraIDMutation,
} from "@/app/api/enduser/invitesApi";
import { SyncConfigCard } from "./SyncConfigCard";
import { toast } from "@/lib/toast";

interface AddUsersModalProps {
  isOpen: boolean;
  onClose: () => void;
  audience: 'admin' | 'endUser';
  onOpenInviteModal: () => void;
  onOpenSyncConfigModal?: (config?: SyncConfig, syncType?: SyncType) => void;
}

export function AddUsersModal({
  isOpen,
  onClose,
  audience,
  onOpenInviteModal,
  onOpenSyncConfigModal,
}: AddUsersModalProps) {
  const { data: configsData, isLoading, error } = useListSyncConfigsQuery();
  const [deleteSyncConfig] = useDeleteSyncConfigMutation();
  const [syncAD] = useSyncActiveDirectoryMutation();
  const [syncEntra] = useSyncEntraIDMutation();
  const [syncingAll, setSyncingAll] = useState<'ad' | 'entra' | null>(null);
  const [deleteConfirmConfig, setDeleteConfirmConfig] = useState<SyncConfig | null>(null);

  const configs = configsData?.configs || [];
  const adConfigs = configs.filter((c) => c.sync_type === 'active_directory');
  const entraConfigs = configs.filter((c) => c.sync_type === 'entra_id');

  const audienceLabel = audience === 'admin' ? 'Admin' : 'End User';

  const handleDelete = (config: SyncConfig) => {
    setDeleteConfirmConfig(config);
  };

  const confirmDelete = async () => {
    if (!deleteConfirmConfig?.id) return;

    try {
      await deleteSyncConfig({
        id: deleteConfirmConfig.id,
      }).unwrap();
      toast.success('Configuration deleted');
      setDeleteConfirmConfig(null);
    } catch (error: any) {
      toast.error(`Delete failed: ${error.data?.message || error.message}`);
    }
  };

  const handleSyncAll = async (type: 'ad' | 'entra') => {
    const targetConfigs = type === 'ad' ? adConfigs : entraConfigs;

    if (targetConfigs.length === 0) {
      toast.error('No configurations to sync');
      return;
    }

    setSyncingAll(type);

    try {
      const syncMutation = type === 'ad' ? syncAD : syncEntra;
      const results = await Promise.allSettled(
        targetConfigs.map((config) =>
          syncMutation({
            provider: type,
            config_id: config.id!,
            dry_run: false,
            audience,
          }).unwrap()
        )
      );

      const successful = results.filter((r) => r.status === 'fulfilled').length;
      const failed = results.filter((r) => r.status === 'rejected').length;

      if (failed === 0) {
        toast.success(`Synced ${successful} ${type.toUpperCase()} config${successful > 1 ? 's' : ''}`);
      } else {
        toast.warning(`${successful} succeeded, ${failed} failed`);
      }
    } catch (error: any) {
      toast.error(`Sync failed: ${error.data?.message || error.message}`);
    } finally {
      setSyncingAll(null);
    }
  };

  return (
    <>
      <Dialog open={isOpen} onOpenChange={onClose}>
        <DialogContent className="sm:max-w-2xl max-h-[85vh] overflow-y-auto">
          <DialogHeader className="pb-4">
            <DialogTitle className="text-xl font-semibold">Add {audienceLabel}s</DialogTitle>
          </DialogHeader>

        <div className="space-y-6">
          {/* Loading State */}
          {isLoading && (
            <div className="flex items-center justify-center py-8">
              <div className="flex items-center gap-2">
                <RefreshCw className="h-3.5 w-3.5 animate-spin text-foreground" />
                <p className="text-xs text-foreground">Loading...</p>
              </div>
            </div>
          )}

          {/* Error State */}
          {error && (
            <div className="flex items-center justify-center py-8 text-red-600">
              <div className="flex items-center gap-2">
                <AlertCircle className="h-3.5 w-3.5" />
                <p className="text-xs">Failed to load configurations</p>
              </div>
            </div>
          )}

          {/* Active Directory Section */}
          {!isLoading && !error && (
            <>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="p-1.5 rounded-md bg-primary/10">
                      <ServerCog className="h-4 w-4 text-primary" />
                    </div>
                    <h3 className="text-sm font-semibold">Active Directory</h3>
                  </div>
                  <div className="flex items-center gap-2">
                    {adConfigs.length > 1 && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleSyncAll('ad')}
                        disabled={syncingAll === 'ad'}
                        className="h-8 text-xs px-3"
                      >
                        {syncingAll === 'ad' ? (
                          <>
                            <RefreshCw className="h-3 w-3 mr-1.5 animate-spin" />
                            Syncing...
                          </>
                        ) : (
                          <>
                            <RefreshCw className="h-3 w-3 mr-1.5" />
                            Sync All
                          </>
                        )}
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onOpenSyncConfigModal?.(undefined, 'active_directory')}
                      className="h-8 w-8 border border-dashed rounded-full text-foreground hover:text-foreground"
                      aria-label="Add Active Directory configuration"
                    >
                      <Plus className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                {adConfigs.length === 0 ? (
                  <p className="text-xs text-foreground px-1">No configurations yet. Use the plus to add one.</p>
                ) : (
                  <div className="space-y-2">
                    {adConfigs.map((config) => (
                      <SyncConfigCard
                        key={config.id}
                        config={config}
                        onEdit={(c) => onOpenSyncConfigModal?.(c, c.sync_type)}
                        onDelete={handleDelete}
                        audience={audience}
                      />
                    ))}
                  </div>
                )}
              </div>

              {/* Azure Entra ID Section */}
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="p-1.5 rounded-md bg-blue-500/10">
                      <Cloud className="h-4 w-4 text-blue-500" />
                    </div>
                    <h3 className="text-sm font-semibold">Azure Entra ID</h3>
                  </div>
                  <div className="flex items-center gap-2">
                    {entraConfigs.length > 1 && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleSyncAll('entra')}
                        disabled={syncingAll === 'entra'}
                        className="h-8 text-xs px-3"
                      >
                        {syncingAll === 'entra' ? (
                          <>
                            <RefreshCw className="h-3 w-3 mr-1.5 animate-spin" />
                            Syncing...
                          </>
                        ) : (
                          <>
                            <RefreshCw className="h-3 w-3 mr-1.5" />
                            Sync All
                          </>
                        )}
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onOpenSyncConfigModal?.(undefined, 'entra_id')}
                      className="h-8 w-8 border border-dashed rounded-full text-foreground hover:text-foreground"
                      aria-label="Add Azure Entra ID configuration"
                    >
                      <Plus className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                {entraConfigs.length === 0 ? (
                  <p className="text-xs text-foreground px-1">No configurations yet. Use the plus to add one.</p>
                ) : (
                  <div className="space-y-2">
                    {entraConfigs.map((config) => (
                      <SyncConfigCard
                        key={config.id}
                        config={config}
                        onEdit={(c) => onOpenSyncConfigModal?.(c, c.sync_type)}
                        onDelete={handleDelete}
                        audience={audience}
                      />
                    ))}
                  </div>
                )}
              </div>

              {/* Email Invite Section */}
              <div className="space-y-3 pt-3 border-t">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-md bg-blue-500/10">
                    <Mail className="h-4 w-4 text-blue-500" />
                  </div>
                  <h3 className="text-sm font-semibold">Email Invite</h3>
                </div>
                <Button
                  variant="outline"
                  onClick={onOpenInviteModal}
                  className="w-full h-12 text-sm justify-start hover:bg-accent"
                >
                  <Mail className="h-4 w-4 mr-2" />
                  Send Email Invite
                </Button>
              </div>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteConfirmConfig} onOpenChange={() => setDeleteConfirmConfig(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Trash2 className="h-5 w-5 text-red-600" />
              Delete Configuration
            </DialogTitle>
            <DialogDescription className="pt-2">
              Are you sure you want to delete <span className="font-semibold text-foreground">"{deleteConfirmConfig?.config_name}"</span>?
              <div className="mt-3 p-3 bg-muted rounded-md text-xs">
                Users synced from this source will remain but won't be associated with this configuration.
              </div>
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="flex-row justify-end gap-2 sm:gap-2">
            <Button
              variant="outline"
              onClick={() => setDeleteConfirmConfig(null)}
              className="flex-1 sm:flex-none"
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDelete}
              className="flex-1 sm:flex-none"
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

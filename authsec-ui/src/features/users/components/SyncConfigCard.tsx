import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { SyncConfig } from "@/app/api/syncConfigsApi";
import {
  useSyncActiveDirectoryMutation,
  useSyncEntraIDMutation,
  useSyncAdminUsersActiveDirectoryMutation,
  useSyncAdminUsersEntraIDMutation,
} from "@/app/api/enduser/invitesApi";
import { RefreshCw, Pencil, Trash2, CheckCircle2, XCircle, Loader2, Copy, Check, Users, UserCog } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { toast } from "@/lib/toast";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface SyncConfigCardProps {
  config: SyncConfig;
  onEdit: (config: SyncConfig) => void;
  onDelete: (config: SyncConfig) => void;
  audience?: 'admin' | 'endUser';
}

export function SyncConfigCard({
  config,
  onEdit,
  onDelete,
  audience = 'admin',
}: SyncConfigCardProps) {
  const [syncing, setSyncing] = useState(false);
  const [syncResult, setSyncResult] = useState<{
    users_found?: number;
    users_created?: number;
    users_updated?: number;
  } | null>(null);
  const [confirmModalOpen, setConfirmModalOpen] = useState(false);
  const [pendingSyncTarget, setPendingSyncTarget] = useState<'admin' | 'endUser' | null>(null);

  const [syncAD] = useSyncActiveDirectoryMutation();
  const [syncEntra] = useSyncEntraIDMutation();
  const [syncAdminAD] = useSyncAdminUsersActiveDirectoryMutation();
  const [syncAdminEntra] = useSyncAdminUsersEntraIDMutation();

  const handleSyncClick = (targetAudience: 'admin' | 'endUser') => {
    setPendingSyncTarget(targetAudience);
    setConfirmModalOpen(true);
  };

  const handleConfirmSync = async () => {
    if (!config.id || !pendingSyncTarget) return;

    setSyncing(true);
    setSyncResult(null);

    try {
      let result;

      // Determine which mutation to use based on sync type and target audience
      if (pendingSyncTarget === 'admin') {
        // Sync to Admin Users list
        if (config.sync_type === 'active_directory') {
          result = await syncAdminAD({
            provider: 'ad',
            config_id: config.id,
            dry_run: false,
          }).unwrap();
        } else {
          result = await syncAdminEntra({
            provider: 'entra',
            config_id: config.id,
            dry_run: false,
          }).unwrap();
        }
      } else {
        // Sync to End Users list
        if (config.sync_type === 'active_directory') {
          result = await syncAD({
            provider: 'ad',
            config_id: config.id,
            dry_run: false,
            audience: 'endUser',
          }).unwrap();
        } else {
          result = await syncEntra({
            provider: 'entra',
            config_id: config.id,
            dry_run: false,
            audience: 'endUser',
          }).unwrap();
        }
      }

      setSyncResult(result);
      const targetList = pendingSyncTarget === 'admin' ? 'Admin Users' : 'End Users';
      toast.success(
        `Synced to ${targetList}: ${result.users_found || 0} found · ${result.users_created || 0} created · ${result.users_updated || 0} updated`
      );

      setConfirmModalOpen(false);
      setPendingSyncTarget(null);
      setTimeout(() => setSyncResult(null), 5000);
    } catch (error: any) {
      toast.error(`Sync failed: ${error.data?.message || error.message || 'Unknown error'}`);
    } finally {
      setSyncing(false);
    }
  };

  const formatLastSync = (lastSyncAt?: string) => {
    if (!lastSyncAt) return null;
    try {
      return formatDistanceToNow(new Date(lastSyncAt), { addSuffix: true });
    } catch {
      return null;
    }
  };

  const lastSyncText = formatLastSync(config.last_sync_at);
  const isSuccess = config.last_sync_status === "success";
  const isError = config.last_sync_status === "error" || config.last_sync_status === "failed";

  // Get client ID to display
  const clientId = config.entra_client_id || config.client_id || config.entra_config?.client_id;

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard');
  };

  return (
    <div className={`
      group relative border rounded-md p-3 transition-all
      ${syncing
        ? 'border-blue-200 bg-blue-50/50 dark:border-blue-800 dark:bg-blue-950/50'
        : 'hover:border-gray-300 dark:hover:border-gray-600 hover:shadow-sm bg-card'
      }
    `}>
      <div className="flex items-center justify-between gap-3">
        {/* Left: Config Info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3">
            <h4 className="font-medium text-sm truncate">
              {config.config_name}
            </h4>
            {clientId && (
              <>
                <span className="text-foreground text-xs truncate max-w-[200px]">{clientId}</span>
                <button
                  onClick={() => handleCopy(clientId)}
                  className="p-0.5 text-foreground hover:text-foreground transition-colors shrink-0"
                  title="Copy Client ID"
                >
                  <Copy className="h-3 w-3" />
                </button>
              </>
            )}
            {syncing ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-600 shrink-0" />
            ) : isSuccess ? (
              <CheckCircle2 className="h-3.5 w-3.5 text-green-600 shrink-0" />
            ) : isError ? (
              <XCircle className="h-3.5 w-3.5 text-red-600 shrink-0" />
            ) : null}
          </div>

          <div className="flex items-center gap-2 text-xs mt-1">
            {syncResult ? (
              <span className="text-green-600 font-medium">
                {syncResult.users_found || 0} found · {syncResult.users_created || 0} created
              </span>
            ) : lastSyncText ? (
              <span className="text-foreground">{lastSyncText}</span>
            ) : null}
            {config.last_sync_users_count !== undefined && !syncResult && (
              <>
                {lastSyncText && <span className="text-foreground">·</span>}
                <Badge variant="secondary" className="h-4 px-1.5 text-[10px]">
                  {config.last_sync_users_count} users
                </Badge>
              </>
            )}
          </div>
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-1.5">
          {/* Sync button - shown based on audience context */}
          {audience === 'admin' ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => handleSyncClick('admin')}
              disabled={syncing}
              className="h-7 px-2 text-xs gap-1.5 border-blue-200 text-blue-700 hover:bg-blue-50 dark:border-blue-800 dark:text-blue-400 dark:hover:bg-blue-950/30"
              title="Sync to Admin Users list"
            >
              <UserCog className="h-3 w-3" />
              <span className="hidden sm:inline">Sync Admin</span>
            </Button>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={() => handleSyncClick('endUser')}
              disabled={syncing}
              className="h-7 px-2 text-xs gap-1.5 border-blue-200 text-blue-700 hover:bg-blue-50 dark:border-blue-800 dark:text-blue-400 dark:hover:bg-blue-950/30"
              title="Sync to End Users list"
            >
              <Users className="h-3 w-3" />
              <span className="hidden sm:inline">Sync End Users</span>
            </Button>
          )}

          <Button
            variant="ghost"
            size="sm"
            onClick={() => onEdit(config)}
            disabled={syncing}
            className="h-8 w-8 p-0"
            title="Edit configuration"
          >
            <Pencil className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onDelete(config)}
            disabled={syncing}
            className="h-8 w-8 p-0 text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950/30"
            title="Delete configuration"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      {/* Confirmation Dialog */}
      <Dialog open={confirmModalOpen} onOpenChange={setConfirmModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Sync</DialogTitle>
            <DialogDescription>
              Are you sure you want to sync "{config.config_name}" ({config.sync_type}) to {pendingSyncTarget === 'admin' ? 'Admin Users' : 'End Users'}?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmModalOpen(false)} disabled={syncing}>
              Cancel
            </Button>
            <Button onClick={handleConfirmSync} disabled={syncing}>
              {syncing ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Syncing...
                </>
              ) : (
                'Confirm'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

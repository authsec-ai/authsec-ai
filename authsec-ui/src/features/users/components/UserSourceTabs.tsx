import React from "react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ServerCog, Cloud, Shield, Plus } from "lucide-react";

export type UserSource = "all" | "ad" | "entra" | "authsec";

interface UserSourceTabsProps {
  selectedSource: UserSource;
  onSourceChange: (source: UserSource) => void;
  onAddUsersClick: () => void;
  adCount?: number;
  entraCount?: number;
  authsecCount?: number;
  adConfigured: boolean;
  entraConfigured: boolean;
}

export function UserSourceTabs({
  selectedSource,
  onSourceChange,
  onAddUsersClick,
  adCount,
  entraCount,
  authsecCount,
  adConfigured,
  entraConfigured,
}: UserSourceTabsProps) {
  const totalCount = (adCount || 0) + (entraCount || 0) + (authsecCount || 0);

  return (
    <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 w-full">
      <Tabs
        value={selectedSource}
        onValueChange={(v) => onSourceChange(v as UserSource)}
        className="flex-1 min-w-0"
      >
        <TabsList className="w-full sm:w-auto grid grid-cols-4 sm:inline-flex h-auto">
          <TabsTrigger value="all">
            <span className="hidden sm:inline">All</span>
            <span className="sm:hidden">All</span>
            {totalCount > 0 && <span className="ml-1 text-xs opacity-60">{totalCount}</span>}
          </TabsTrigger>

          <TabsTrigger value="ad">
            <ServerCog className="h-3.5 w-3.5 sm:mr-1" />
            <span className="hidden sm:inline">AD</span>
            {adCount !== undefined && adCount > 0 && (
              <span className="ml-1 text-xs opacity-60">{adCount}</span>
            )}
            {!adConfigured && (
              <span className="ml-1 text-[10px] text-amber-500 hidden sm:inline">Setup</span>
            )}
          </TabsTrigger>

          <TabsTrigger value="entra">
            <Cloud className="h-3.5 w-3.5 sm:mr-1" />
            <span className="hidden sm:inline">Entra</span>
            {entraCount !== undefined && entraCount > 0 && (
              <span className="ml-1 text-xs opacity-60">{entraCount}</span>
            )}
            {!entraConfigured && (
              <span className="ml-1 text-[10px] text-amber-500 hidden sm:inline">Setup</span>
            )}
          </TabsTrigger>

          <TabsTrigger value="authsec">
            <Shield className="h-3.5 w-3.5 sm:mr-1" />
            <span className="hidden sm:inline">AuthSec</span>
            {authsecCount !== undefined && authsecCount > 0 && (
              <span className="ml-1 text-xs opacity-60">{authsecCount}</span>
            )}
          </TabsTrigger>
        </TabsList>
      </Tabs>

      <Button
        size="sm"
        onClick={onAddUsersClick}
        className="shrink-0 text-white"
        data-tour-id="invite-user-button"
      >
        <Plus className="h-4 w-4 sm:mr-1.5" />
        Add User
      </Button>
    </div>
  );
}

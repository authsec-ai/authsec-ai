import React from "react";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerDescription,
  DrawerClose,
} from "@/components/ui/drawer";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Shield, Copy, Trash2, Edit2 } from "lucide-react";
import type { EnhancedRole } from "@/types/entities";

interface RoleDrawerProps {
  role: EnhancedRole | null;
  onClose: () => void;
  onEdit: (id: string) => void;
  onClone: (id: string) => void;
  onDelete: (id: string) => void;
  onAssignUsers: (id: string) => void;
}

export function RoleDrawer({
  role,
  onClose,
  onEdit,
  onClone,
  onDelete,
  onAssignUsers,
}: RoleDrawerProps) {
  if (!role) return null;

  return (
    <Drawer open={!!role} onOpenChange={(open) => !open && onClose()} direction="right">
      <DrawerContent className="h-full w-full max-w-lg border-l">
        <DrawerHeader className="border-b p-6 flex items-start justify-between">
          <div className="space-y-1">
            <DrawerTitle className="flex items-center gap-2 text-xl font-bold">
              <Shield className="h-5 w-5" /> {role.name}
              {role.version > 1 && <Badge variant="outline">v{role.version}</Badge>}
            </DrawerTitle>
            {role.description && <DrawerDescription>{role.description}</DrawerDescription>}
          </div>
          <div className="flex gap-1">
            <Button size="icon" variant="ghost" onClick={() => onEdit(role.id)}>
              <Edit2 className="h-4 w-4" />
            </Button>
            <Button size="icon" variant="ghost" onClick={() => onClone(role.id)}>
              <Copy className="h-4 w-4" />
            </Button>
            <Button
              size="icon"
              variant="ghost"
              onClick={() => onDelete(role.id)}
              disabled={role.isBuiltIn}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
            <DrawerClose asChild>
              <Button size="icon" variant="ghost" aria-label="Close">
                ✕
              </Button>
            </DrawerClose>
          </div>
        </DrawerHeader>

        {/* Body */}
        <div className="flex-1 overflow-auto p-6">
          <Tabs defaultValue="definition" className="w-full">
            <TabsList>
              <TabsTrigger value="definition">Definition</TabsTrigger>
              <TabsTrigger value="assignments">Assignments</TabsTrigger>
              <TabsTrigger value="versions">Version History</TabsTrigger>
              <TabsTrigger value="audit">Audit Log</TabsTrigger>
            </TabsList>
            <TabsContent value="definition" className="mt-4 space-y-4">
              {/* Permissions matrix placeholder */}
              <div className="border rounded-lg p-4 text-sm text-foreground">
                Matrix preview coming soon.
              </div>
            </TabsContent>
            <TabsContent value="assignments" className="mt-4 space-y-4">
              <p className="text-sm text-foreground">Assignments UI coming soon.</p>
              <Button size="sm" onClick={() => onAssignUsers(role.id)}>
                Assign Users / Groups
              </Button>
            </TabsContent>
            <TabsContent value="versions" className="mt-4 space-y-4">
              <p className="text-sm text-foreground">Version history coming soon.</p>
            </TabsContent>
            <TabsContent value="audit" className="mt-4 space-y-4">
              <p className="text-sm text-foreground">Audit log coming soon.</p>
            </TabsContent>
          </Tabs>
        </div>
      </DrawerContent>
    </Drawer>
  );
}

import { useMemo, useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SearchableSelect, type SearchableSelectOption } from "@/components/ui/searchable-select";
import { Shield, Send, ExternalLink, User } from "lucide-react";
import { cn } from "@/lib/utils";
import { toast } from "@/lib/toast";
import { useGetAuthSecRolesQuery } from "@/app/api/rolesApi";
import { useInviteAdminUserMutation } from "@/app/api/admin/invitesApi";
import { useInviteEndUserMutation } from "@/app/api/enduser/invitesApi";
import { SessionManager } from "@/utils/sessionManager";
import { resolveTenantId } from "@/utils/workspace";
import { useNavigate } from "react-router-dom";

const emailRx = /^[^\s@]+@[^\s@]+\.[^\s@]+$/i;

interface InviteUserModalProps {
  isOpen: boolean;
  onClose: () => void;
  audience: 'admin' | 'endUser';
  onSuccess?: () => void;
}

export function InviteUserModal({ isOpen, onClose, audience, onSuccess }: InviteUserModalProps) {
  const isAdmin = audience === 'admin';
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [selectedRoleIds, setSelectedRoleIds] = useState<string[]>([]);
  const [emailTouched, setEmailTouched] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            title: "Invite Admin User",
            subtitle: "Send a secure invitation to a new administrator and pre-assign their controls.",
            sendCta: "Send Admin Invite",
            sendingLabel: "Sending admin invite...",
          }
        : {
            title: "Invite End User",
            subtitle: "Welcome a customer or member and assign the experiences they should see first.",
            sendCta: "Send End-user Invite",
            sendingLabel: "Sending end-user invite...",
          },
    [isAdmin]
  );

  const emailValid = emailRx.test(email);
  const canSend = emailValid && selectedRoleIds.length > 0;

  const sessionData = SessionManager.getSession();
  const tenantId =
    resolveTenantId() ??
    sessionData?.tenant_id ??
    (sessionData as any)?.tenantId ??
    '';

  const { data: rolesResponse = [] } = useGetAuthSecRolesQuery({
    tenant_id: tenantId || "",
    audience,
  });
  const [inviteAdminUser, { isLoading: busySendAdmin }] = useInviteAdminUserMutation();
  const [inviteEndUser, { isLoading: busySendEndUser }] = useInviteEndUserMutation();
  const busySend = isAdmin ? busySendAdmin : busySendEndUser;

  // Convert roles to SearchableSelectOption format
  const roleOptions = useMemo<SearchableSelectOption[]>(() => {
    if (!Array.isArray(rolesResponse)) {
      return [];
    }

    const normalized = rolesResponse
      .map((role: any, index: number) => {
        const rawId =
          role?.id ??
          role?.role_id ??
          role?.roleId ??
          role?.roleID ??
          role?.uuid ??
          role?.uid ??
          role?.external_id ??
          role?.slug ??
          role?.name ??
          role?.role_name ??
          `role-${index}`;

        const rawName =
          role?.name ??
          role?.role_name ??
          role?.roleName ??
          role?.label ??
          role?.display_name ??
          role?.role ??
          `Role ${index + 1}`;

        if (!rawId || !rawName) {
          return null;
        }

        const description =
          role?.description ??
          role?.details ??
          role?.summary ??
          role?.meta?.description;

        return {
          value: String(rawId),
          label: String(rawName),
          description: description ? String(description) : undefined,
        };
      })
      .filter((role): role is SearchableSelectOption => Boolean(role));

    // Deduplicate by value
    const uniqueById = new Map<string, SearchableSelectOption>();
    normalized.forEach((role) => {
      if (!uniqueById.has(role.value)) {
        uniqueById.set(role.value, role);
      }
    });

    return Array.from(uniqueById.values());
  }, [rolesResponse]);

  // Prefill default role(s) based on audience
  useEffect(() => {
    if (!roleOptions.length || selectedRoleIds.length > 0) {
      return;
    }

    if (isAdmin) {
      const adminDefaults = roleOptions.filter((role) =>
        role.label.toLowerCase().includes("admin")
      );

      if (adminDefaults.length > 0) {
        setSelectedRoleIds(adminDefaults.map(r => r.value));
        return;
      }
    } else {
      const endUserDefault = roleOptions.find((role) => role.label.toLowerCase() === "end user");
      if (endUserDefault) {
        setSelectedRoleIds([endUserDefault.value]);
        return;
      }
    }

    if (roleOptions.length > 0) {
      setSelectedRoleIds([roleOptions[0].value]);
    }
  }, [roleOptions, selectedRoleIds.length, isAdmin]);

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setEmail("");
      setUsername("");
      setFirstName("");
      setLastName("");
      setSelectedRoleIds([]);
      setEmailTouched(false);
      setErrorMessage(null);
    }
  }, [isOpen]);

  const tenantDomain = typeof window !== "undefined" ? window.location.hostname : undefined;
  const clientId =
    sessionData?.client_id ??
    (sessionData as any)?.clientId ??
    sessionData?.project_id ??
    (sessionData as any)?.projectId ??
    "";
  const projectId =
    sessionData?.project_id ??
    (sessionData as any)?.projectId ??
    sessionData?.client_id ??
    (sessionData as any)?.clientId ??
    "";

  const sendInvite = async () => {
    if (!canSend) return;
    try {
      setErrorMessage(null);
      const payload = {
        email,
        username: username || email,
        first_name: firstName,
        last_name: lastName,
        roles: selectedRoleIds,
        tenant_domain:
          tenantDomain ?? sessionData?.tenant_domain ?? (sessionData as any)?.tenantDomain ?? "",
        tenant_id: tenantId || "",
        client_id: clientId,
        project_id: projectId,
      };

      if (isAdmin) {
        await inviteAdminUser(payload).unwrap();
      } else {
        await inviteEndUser(payload).unwrap();
      }

      toast.success(`Invitation sent to ${email}`);
      onClose();
      onSuccess?.();
    } catch (err: any) {
      const message =
        err?.data?.message ||
        err?.data?.error ||
        err?.error ||
        err?.message ||
        "Failed to send invite";
      setErrorMessage(message);
      toast.error(message);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="text-2xl font-bold">{audienceCopy.title}</DialogTitle>
          <DialogDescription>{audienceCopy.subtitle}</DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* User Info Section */}
          <section className="space-y-4">
            <Label className="text-sm font-semibold text-foreground">
              <div className="flex items-center gap-2">
                <User className="h-4 w-4 text-foreground" />
                User Information
              </div>
            </Label>

            {/* Email */}
            <div className="space-y-2">
              <Label htmlFor="email" className="text-sm text-foreground">Email Address *</Label>
              <Input
                id="email"
                type="email"
                placeholder="colleague@company.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                onBlur={() => setEmailTouched(true)}
                className={cn(
                  "h-11 text-base border bg-black/[0.02] dark:bg-white/[0.02] focus:bg-background transition-colors px-4 rounded-lg",
                  emailTouched && !emailValid && "border-destructive"
                )}
                required
              />
              {emailTouched && !emailValid && (
                <p className="text-sm text-destructive">Enter a valid email address</p>
              )}
            </div>

            {/* Username */}
            <div className="space-y-2">
              <Label htmlFor="username" className="text-sm text-foreground">Username</Label>
              <Input
                id="username"
                type="text"
                placeholder="johndoe (optional, defaults to email)"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="h-11 text-base border bg-black/[0.02] dark:bg-white/[0.02] focus:bg-background transition-colors px-4 rounded-lg"
              />
            </div>

            {/* First Name and Last Name */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="firstName" className="text-sm text-foreground">First Name</Label>
                <Input
                  id="firstName"
                  type="text"
                  placeholder="John"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  className="h-11 text-base border bg-black/[0.02] dark:bg-white/[0.02] focus:bg-background transition-colors px-4 rounded-lg"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="lastName" className="text-sm text-foreground">Last Name</Label>
                <Input
                  id="lastName"
                  type="text"
                  placeholder="Doe"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  className="h-11 text-base border bg-black/[0.02] dark:bg-white/[0.02] focus:bg-background transition-colors px-4 rounded-lg"
                />
              </div>
            </div>
          </section>

          <section className="space-y-3">
            <Label className="text-sm font-semibold text-foreground flex items-center gap-2">
              <Shield className="h-4 w-4 text-foreground" />
              Roles & Permissions
            </Label>

            {/* Role Selection using SearchableSelect */}
            <SearchableSelect
              multiple
              options={roleOptions}
              value={selectedRoleIds}
              onChange={setSelectedRoleIds}
              placeholder="Select roles..."
              searchPlaceholder="Search roles..."
              emptyText="No roles found"
              maxBadges={5}
              className="h-11"
            />

            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  onClose();
                  navigate('/roles');
                }}
                className="h-9 px-3 gap-2 text-foreground hover:opacity-70"
              >
                <ExternalLink className="h-3.5 w-3.5" />
                Manage Roles
              </Button>
            </div>
            {selectedRoleIds.length === 0 && (
              <p className="text-sm text-destructive">Select at least one role</p>
            )}
          </section>
        </div>

        {errorMessage && (
          <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {errorMessage}
          </div>
        )}

        <DialogFooter className="gap-3">
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={sendInvite} disabled={!canSend || busySend} className="min-w-[140px]">
            {busySend ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-2 border-primary-foreground border-t-transparent mr-2" />
                Sending...
              </>
            ) : (
              <>
                <Send className="h-4 w-4 mr-2" />
                {audienceCopy.sendCta}
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

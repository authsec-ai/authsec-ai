import { useState } from "react";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Globe,
  CheckCircle2,
  Clock,
  Star,
  Copy,
  Check,
  Trash2,
  MoreVertical,
  RefreshCw,
  Key,
  Loader2,
} from "lucide-react";
import {
  useVerifyDomainMutation,
  useSetPrimaryDomainMutation,
} from "@/app/api/domainApi";
import { SessionManager } from "@/utils/sessionManager";
import { toast } from "@/lib/toast";
import type { CustomDomain } from "@/app/api/domainApi";

interface DomainCardProps {
  domain: CustomDomain;
  onDelete: (domain: CustomDomain) => void;
}

// Format relative time
function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins} minute${diffMins > 1 ? "s" : ""} ago`;
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? "s" : ""} ago`;
  if (diffDays < 30) return `${diffDays} day${diffDays > 1 ? "s" : ""} ago`;

  return date.toLocaleDateString();
}

export function DomainCard({ domain, onDelete }: DomainCardProps) {
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const [verifyDomain, { isLoading: isVerifying }] = useVerifyDomainMutation();
  const [setPrimaryDomain, { isLoading: isSettingPrimary }] =
    useSetPrimaryDomainMutation();

  const isLoading = isVerifying || isSettingPrimary;

  const handleCopy = async (text: string, field: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(field);
      toast.success("Copied to clipboard");
      setTimeout(() => setCopiedField(null), 2000);
    } catch {
      toast.error("Failed to copy to clipboard");
    }
  };

  const handleVerify = async () => {
    const session = SessionManager.getSession();
    if (!session?.tenant_id) {
      toast.error("Session expired. Please log in again.");
      return;
    }

    try {
      await verifyDomain({
        tenant_id: session.tenant_id,
        domain_id: domain.id,
      }).unwrap();

      toast.success("Domain verified successfully!");
    } catch (error: any) {
      const message =
        error?.data?.error ||
        error?.data?.details ||
        error?.data?.message ||
        "DNS verification failed. Please check your DNS records and try again.";
      toast.error(message);
    }
  };

  const handleSetPrimary = async () => {
    if (!domain.is_verified) {
      toast.error("Domain must be verified before setting as primary");
      return;
    }

    const session = SessionManager.getSession();
    if (!session?.tenant_id) {
      toast.error("Session expired. Please log in again.");
      return;
    }

    try {
      await setPrimaryDomain({
        tenant_id: session.tenant_id,
        domain_id: domain.id,
      }).unwrap();

      toast.success("Primary domain updated successfully!");
    } catch (error: any) {
      const message =
        error?.data?.error ||
        error?.data?.message ||
        "Failed to set primary domain. Please try again.";
      toast.error(message);
    }
  };

  return (
    <Card className="relative overflow-hidden">
      {/* Primary indicator stripe */}
      {domain.is_primary && (
        <div className="absolute top-0 left-0 right-0 h-1 bg-primary" />
      )}

      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <Globe className="h-5 w-5 text-muted-foreground" />
            <span className="font-semibold text-lg">{domain.domain}</span>
          </div>

          {/* Actions Menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" disabled={isLoading}>
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {domain.is_verified && !domain.is_primary && (
                <DropdownMenuItem
                  onClick={handleSetPrimary}
                  disabled={isSettingPrimary}
                >
                  <Star className="h-4 w-4 mr-2" />
                  Set as Primary
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => onDelete(domain)}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {/* Status Badges */}
        <div className="flex items-center gap-2 mt-2 flex-wrap">
          {domain.is_verified ? (
            <Badge
              variant="default"
              className="bg-green-500/10 text-green-600 border-green-500/30"
            >
              <CheckCircle2 className="h-3 w-3 mr-1" />
              Verified
            </Badge>
          ) : (
            <Badge
              variant="secondary"
              className="bg-yellow-500/10 text-yellow-600 border-yellow-500/30"
            >
              <Clock className="h-3 w-3 mr-1" />
              Pending
            </Badge>
          )}

          {domain.is_primary && (
            <Badge
              variant="default"
              className="bg-blue-500/10 text-blue-600 border-blue-500/30"
            >
              <Star className="h-3 w-3 mr-1 fill-current" />
              Primary
            </Badge>
          )}

          <Badge variant="outline" className="text-xs">
            {domain.kind === "custom" ? "Custom" : "Platform"}
          </Badge>
        </div>

        {/* Timestamps */}
        <div className="text-xs text-muted-foreground mt-2">
          Added {formatRelativeTime(domain.created_at)}
          {domain.verified_at && (
            <> • Verified {formatRelativeTime(domain.verified_at)}</>
          )}
        </div>
      </CardHeader>

      {/* Verification Section (for pending domains) */}
      {!domain.is_verified && (
        <CardContent className="pt-0">
          <div className="mt-4 rounded-lg border bg-muted/30 p-4 space-y-4">
            <div className="flex items-center gap-2 text-sm font-medium">
              <Key className="h-4 w-4 text-primary" />
              DNS Verification Required
            </div>

            <p className="text-xs text-muted-foreground">
              Add the following TXT record to your DNS configuration to verify
              domain ownership.
            </p>

            {/* TXT Record Name */}
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground">
                TXT Record Name
              </label>
              <div className="flex items-center gap-2">
                <code className="flex-1 text-xs bg-background p-2 rounded border font-mono truncate">
                  {domain.verification_txt_name}
                </code>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="outline"
                        size="icon"
                        className="shrink-0"
                        onClick={() =>
                          handleCopy(domain.verification_txt_name, "name")
                        }
                      >
                        {copiedField === "name" ? (
                          <Check className="h-4 w-4 text-green-500" />
                        ) : (
                          <Copy className="h-4 w-4" />
                        )}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Copy to clipboard</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
            </div>

            {/* TXT Record Value */}
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground">
                TXT Record Value
              </label>
              <div className="flex items-center gap-2">
                <code className="flex-1 text-xs bg-background p-2 rounded border font-mono truncate">
                  {domain.verification_txt_value}
                </code>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="outline"
                        size="icon"
                        className="shrink-0"
                        onClick={() =>
                          handleCopy(domain.verification_txt_value, "value")
                        }
                      >
                        {copiedField === "value" ? (
                          <Check className="h-4 w-4 text-green-500" />
                        ) : (
                          <Copy className="h-4 w-4" />
                        )}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Copy to clipboard</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
            </div>

            {/* Verify Button */}
            <Button
              onClick={handleVerify}
              disabled={isVerifying}
              className="w-full"
            >
              {isVerifying ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Verifying...
                </>
              ) : (
                <>
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Verify DNS
                </>
              )}
            </Button>

            <p className="text-xs text-muted-foreground text-center">
              DNS changes may take 5-15 minutes to propagate
            </p>
          </div>
        </CardContent>
      )}
    </Card>
  );
}

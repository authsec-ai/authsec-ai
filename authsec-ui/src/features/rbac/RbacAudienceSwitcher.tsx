import { Crown, UserCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";

interface RbacAudienceSwitcherProps {
  className?: string;
}

export function RbacAudienceSwitcher({ className }: RbacAudienceSwitcherProps) {
  const { audience, setAudience } = useRbacAudience();
  const isAdmin = audience === "admin";

  return (
    <div
      className={cn(
        "flex items-center gap-1 p-1 rounded-full bg-black/5 dark:bg-white/5 border border-black/5 dark:border-white/10",
        className
      )}
    >
      {/* Admin Option */}
      <button
        type="button"
        onClick={() => setAudience("admin")}
        className={cn(
          "flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium transition-all duration-200",
          isAdmin
            ? "bg-[var(--brand-blue-600)] text-white shadow-sm"
            : "text-foreground/60 hover:text-foreground hover:bg-black/5 dark:hover:bg-white/5"
        )}
        aria-label="Switch to Admin context"
      >
        <Crown className="h-3.5 w-3.5" />
        <span>Admin</span>
      </button>

      {/* End User Option */}
      <button
        type="button"
        onClick={() => setAudience("endUser")}
        className={cn(
          "flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium transition-all duration-200",
          !isAdmin
            ? "bg-[var(--brand-blue-600)] text-white shadow-sm"
            : "text-foreground/60 hover:text-foreground hover:bg-black/5 dark:hover:bg-white/5"
        )}
        aria-label="Switch to End User context"
      >
        <UserCircle className="h-3.5 w-3.5" />
        <span>Customer Identities</span>
      </button>
    </div>
  );
}

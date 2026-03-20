import { useCallback, useMemo } from "react";
import { useLocation, useNavigate, type NavigateOptions } from "react-router-dom";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";

function normalizePath(path: string): string {
  if (!path) return "";
  if (path.startsWith("/")) {
    return path.slice(1);
  }
  return path;
}

export function useContextualNavigate() {
  const navigate = useNavigate();
  const location = useLocation();
  const { audience } = useRbacAudience();

  const activeContext = useMemo<"admin" | "enduser" | null>(() => {
    const match = location.pathname.match(/^\/(admin|enduser)(?:\/|$)/);
    return match ? (match[1] as "admin" | "enduser") : null;
  }, [location.pathname]);

  return useCallback(
    (path: string, options?: NavigateOptions) => {
      const normalized = normalizePath(path);
      const [pathOnly, ...restParts] = normalized.split(/(?=[?#])/);
      const remainder = restParts.join("");
      const hasExplicitContext =
        pathOnly.startsWith("admin/") || pathOnly.startsWith("enduser/");
      const contextSegment = activeContext ?? (audience === "admin" ? "admin" : "enduser");
      const prefix = `/${contextSegment}`;
      const buildPath = () => {
        if (!pathOnly) {
          return `${prefix}${remainder || ""}`;
        }
        if (hasExplicitContext) {
          return `/${pathOnly}${remainder}`;
        }
        const cleanPrefix = prefix.replace(/\/+$/, "");
        return `${cleanPrefix}/${pathOnly}${remainder}`.replace(/\/{2,}/g, "/");
      };

      navigate(buildPath(), options);
    },
    [activeContext, audience, navigate]
  );
}

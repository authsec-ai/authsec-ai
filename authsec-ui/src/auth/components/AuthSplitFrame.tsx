import React, { useEffect, useRef } from "react";
import { cn } from "@/lib/utils";
import { useTheme } from "next-themes";

export interface AuthSplitFrameProps {
  valuePanel: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  valuePanelClassName?: string;
  actionPanelClassName?: string;
}

export function AuthSplitFrame({
  valuePanel,
  children,
  className,
  valuePanelClassName,
  actionPanelClassName,
}: AuthSplitFrameProps) {
  const { setTheme, theme } = useTheme();
  const previousThemeRef = useRef<string | undefined>(undefined);

  useEffect(() => {
    const previousColorScheme = document.documentElement.style.colorScheme;
    previousThemeRef.current = theme;

    setTheme("light");
    document.documentElement.style.colorScheme = "light";

    return () => {
      document.documentElement.style.colorScheme = previousColorScheme;

      if (previousThemeRef.current && previousThemeRef.current !== "light") {
        setTheme(previousThemeRef.current);
      }
    };
  }, [setTheme]);

  return (
    <div className={cn("auth-shell", className)}>
      <div className="auth-shell__stage">
        <div className="auth-shell__grid">
          <aside className={cn("auth-shell__value", valuePanelClassName)}>
            {valuePanel}
          </aside>
          <section className={cn("auth-shell__action", actionPanelClassName)}>
            {children}
          </section>
        </div>
      </div>
    </div>
  );
}

export default AuthSplitFrame;

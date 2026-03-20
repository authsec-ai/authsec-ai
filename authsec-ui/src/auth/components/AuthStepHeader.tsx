import React from "react";
import { cn } from "@/lib/utils";

export interface AuthStepHeaderProps {
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  meta?: React.ReactNode;
  align?: "left" | "center";
  className?: string;
}

export function AuthStepHeader({
  title,
  subtitle,
  meta,
  align = "left",
  className,
}: AuthStepHeaderProps) {
  return (
    <header
      className={cn(
        "auth-step-header",
        align === "center" && "auth-step-header--center",
        className,
      )}
    >
      <h1 className="auth-step-header__title">{title}</h1>
      {subtitle ? <p className="auth-step-header__subtitle">{subtitle}</p> : null}
      {meta ? <div className="auth-step-header__meta">{meta}</div> : null}
    </header>
  );
}

export default AuthStepHeader;

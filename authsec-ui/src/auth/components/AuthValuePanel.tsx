import React from "react";
import { Check } from "lucide-react";
import { cn } from "@/lib/utils";

export interface AuthValuePanelProps {
  eyebrow?: React.ReactNode;
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  points?: string[];
  trustLabel?: React.ReactNode;
  trustItems?: React.ReactNode;
  className?: string;
  children?: React.ReactNode;
}

export function AuthValuePanel({
  eyebrow,
  title,
  subtitle,
  points = [],
  trustLabel,
  trustItems,
  className,
  children,
}: AuthValuePanelProps) {
  return (
    <div className={cn("auth-value-panel", className)}>
      {eyebrow ? <p className="auth-value-panel__eyebrow">{eyebrow}</p> : null}
      <h2 className="auth-value-panel__title">{title}</h2>
      {subtitle ? <p className="auth-value-panel__subtitle">{subtitle}</p> : null}

      {points.length > 0 ? (
        <ul className="auth-value-panel__list" aria-label="Highlights">
          {points.map((point) => (
            <li className="auth-value-panel__list-item" key={point}>
              <Check className="auth-value-panel__list-icon" aria-hidden="true" />
              <span>{point}</span>
            </li>
          ))}
        </ul>
      ) : null}

      {children}

      {trustLabel ? <p className="auth-value-panel__trust-label">{trustLabel}</p> : null}
      {trustItems ? <div className="auth-value-panel__trust-items">{trustItems}</div> : null}
    </div>
  );
}

export default AuthValuePanel;

import React from "react";
import { cn } from "@/lib/utils";

export interface AuthActionPanelProps {
  children: React.ReactNode;
  className?: string;
}

export function AuthActionPanel({ children, className }: AuthActionPanelProps) {
  return <div className={cn("auth-action-panel", className)}>{children}</div>;
}

export default AuthActionPanel;

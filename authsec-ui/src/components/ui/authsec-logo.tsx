import React from "react";
import { useTheme } from "next-themes";
import authsecLogoBlack from "../../logos/AuthSec Logo Black.png";
import authsecLogoWhite from "../../logos/AuthSec Logo White.png";

interface AuthSecLogoProps {
  className?: string;
}

export function AuthSecLogo({ className = "h-8 w-8" }: AuthSecLogoProps) {
  const { resolvedTheme } = useTheme();
  const logoSrc = resolvedTheme === "dark" ? authsecLogoWhite : authsecLogoBlack;

  return (
    <svg
      viewBox="0 0 1024 1024"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-label="AuthSec Logo"
      role="img"
    >
      <image
        href={logoSrc}
        x="0"
        y="0"
        width="1024"
        height="1024"
        preserveAspectRatio="xMidYMid meet"
      />
    </svg>
  );
}

interface AuthSecMarkProps {
  className?: string;
}

export function AuthSecMark({ className = "h-12 w-12" }: AuthSecMarkProps) {
  const { resolvedTheme } = useTheme();
  const logoSrc = resolvedTheme === "dark" ? authsecLogoWhite : authsecLogoBlack;

  return (
    <svg
      viewBox="0 0 1024 1024"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-label="AuthSec Mark"
      role="img"
    >
      <image
        href={logoSrc}
        x="0"
        y="0"
        width="1024"
        height="1024"
        preserveAspectRatio="xMidYMid meet"
      />
    </svg>
  );
}

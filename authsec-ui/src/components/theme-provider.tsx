"use client";

import { useMemo } from "react";
import { ThemeProvider as NextThemesProvider, type ThemeProviderProps } from "next-themes";
import "../theme/tokens.css";
import { ThemeTokensProvider, themeTokens } from "../theme";

export function ThemeProvider({ children, ...props }: ThemeProviderProps) {
  const value = useMemo(() => themeTokens, []);

  return (
    <ThemeTokensProvider value={value}>
      <NextThemesProvider attribute="class" defaultTheme="dark" enableSystem={true} {...props}>
        {children}
      </NextThemesProvider>
    </ThemeTokensProvider>
  );
}

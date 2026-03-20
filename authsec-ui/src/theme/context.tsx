import { createContext, useContext } from "react";
import type { ThemeTokens } from "./tokens";
import { themeTokens } from "./tokens";

const ThemeTokensContext = createContext<ThemeTokens>(themeTokens);

export const ThemeTokensProvider = ThemeTokensContext.Provider;

export function useThemeTokens() {
  return useContext(ThemeTokensContext);
}

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import "./index.css";
import App from "./App.tsx";
import { store } from "./app/store.ts";
import { ThemeProvider } from "./components/theme-provider.tsx";
import { ErrorBoundary } from "./components/ErrorBoundary.tsx";
import { TooltipProvider } from "./components/ui/tooltip.tsx";
import { checkAndShowTokenInjector } from "./utils/devTokenInjector.ts";

// Check if running on localhost and show token injector if needed
checkAndShowTokenInjector();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Provider store={store}>
      <ErrorBoundary>
        <ThemeProvider
          defaultTheme="dark"
          storageKey="authsec-ui-theme"
          enableSystem
        >
          <TooltipProvider>
            <App />
          </TooltipProvider>
        </ThemeProvider>
      </ErrorBoundary>
    </Provider>
  </StrictMode>,
);

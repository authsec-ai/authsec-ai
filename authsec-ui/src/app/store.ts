import { configureStore } from "@reduxjs/toolkit";
import { setupListeners } from "@reduxjs/toolkit/query";

// Import API slices
import { baseApi } from "./api/baseApi";
// Import APIs that inject endpoints into baseApi (imports needed for side effects)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { authApi } from "./api/authApi"; // User authentication endpoints
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { webauthnApi } from "./api/webauthnApi"; // WebAuthn/MFA endpoints
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { externalServiceApi } from "./api/externalServiceApi"; // External services endpoints
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { workloadsApi } from "./api/workloadsApi"; // SPIRE workloads endpoints
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { dashboardApi } from "./api/dashboardApi"; // Dashboard endpoints

// New segregated authentication APIs
import { userAuthApi } from "./api/userAuthApi"; // Direct email/password login
import { oidcApi } from "./api/oidcApi"; // OIDC/OAuth flows
import { deviceApi } from "./api/deviceApi"; // Device management (TOTP/CIBA)

// Import regular slices
import uiSlice from "./slices/uiSlice";

// Import auth slices from new location
import authSlice from "../auth/slices/authSlice";
import adminWebAuthnSlice from "../auth/slices/adminWebAuthnSlice";
import oidcWebAuthnSlice from "../auth/slices/oidcWebAuthnSlice";

/**
 * Redux store configuration with RTK Query integration
 *
 * Configures the main application store with:
 * - RTK Query API slices for data fetching
 * - UI state management
 * - Authentication state
 * - Development tools in dev mode
 */
export const store = configureStore({
  reducer: {
    // RTK Query API slices - baseApi with injected endpoints
    [baseApi.reducerPath]: baseApi.reducer, // Contains authApi, webauthnApi, and externalServiceApi endpoints

    // Segregated authentication APIs (still separate)
    [userAuthApi.reducerPath]: userAuthApi.reducer, // Direct login
    [oidcApi.reducerPath]: oidcApi.reducer, // OIDC/OAuth
    [deviceApi.reducerPath]: deviceApi.reducer, // Device management

    // Regular slices
    ui: uiSlice,
    auth: authSlice,
    adminWebAuthn: adminWebAuthnSlice,
    oidcWebAuthn: oidcWebAuthnSlice,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware({
      serializableCheck: {
        ignoredActions: [
          // Ignore these action types from RTK Query
          "persist/PERSIST",
          "persist/REHYDRATE",
        ],
      },
    }).concat(
      // Add RTK Query middleware
      baseApi.middleware, // Contains authApi, webauthnApi, and externalServiceApi endpoints
      // Segregated authentication middleware
      userAuthApi.middleware,
      oidcApi.middleware,
      deviceApi.middleware,
    ),
  devTools: process.env.NODE_ENV !== "production",
});

// Enable listener behavior for the store
setupListeners(store.dispatch);

// Infer types from the store itself
export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

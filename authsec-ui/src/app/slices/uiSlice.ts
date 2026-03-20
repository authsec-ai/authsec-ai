import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";

interface UIState {
  sidebarCollapsed: boolean;
  currentPage: string;
  loading: boolean;
  error: string | null;
  theme: "light" | "dark";
}

const initialState: UIState = {
  sidebarCollapsed: false,
  currentPage: "dashboard",
  loading: false,
  error: null,
  theme: "dark",
};

/**
 * UI slice for managing application-wide UI state
 * Handles sidebar, navigation, loading states, and theme
 */
const uiSlice = createSlice({
  name: "ui",
  initialState,
  reducers: {
    toggleSidebar: (state) => {
      state.sidebarCollapsed = !state.sidebarCollapsed;
    },
    setSidebarCollapsed: (state, action: PayloadAction<boolean>) => {
      state.sidebarCollapsed = action.payload;
    },
    setCurrentPage: (state, action: PayloadAction<string>) => {
      state.currentPage = action.payload;
    },
    setLoading: (state, action: PayloadAction<boolean>) => {
      state.loading = action.payload;
    },
    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
    clearError: (state) => {
      state.error = null;
    },
    setTheme: (state, action: PayloadAction<"light" | "dark">) => {
      state.theme = action.payload;
    },
    toggleTheme: (state) => {
      state.theme = state.theme === "light" ? "dark" : "light";
    },
  },
});

export const {
  toggleSidebar,
  setSidebarCollapsed,
  setCurrentPage,
  setLoading,
  setError,
  clearError,
  setTheme,
  toggleTheme,
} = uiSlice.actions;

export default uiSlice.reducer;

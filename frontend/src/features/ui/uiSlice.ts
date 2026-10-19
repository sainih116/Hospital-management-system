import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

export type ThemeMode = "light" | "dark";

interface UiState {
  sidebarOpen: boolean;
  themeMode: ThemeMode;
}

const THEME_STORAGE_KEY = "hms.themeMode";

function loadStoredThemeMode(): ThemeMode {
  if (typeof window === "undefined") return "light";
  const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
  return stored === "dark" ? "dark" : "light";
}

const initialState: UiState = {
  sidebarOpen: true,
  themeMode: loadStoredThemeMode(),
};

const uiSlice = createSlice({
  name: "ui",
  initialState,
  reducers: {
    toggleSidebar(state) {
      state.sidebarOpen = !state.sidebarOpen;
    },
    setSidebarOpen(state, action: PayloadAction<boolean>) {
      state.sidebarOpen = action.payload;
    },
    toggleThemeMode(state) {
      state.themeMode = state.themeMode === "light" ? "dark" : "light";
      window.localStorage.setItem(THEME_STORAGE_KEY, state.themeMode);
    },
  },
});

export const { toggleSidebar, setSidebarOpen, toggleThemeMode } = uiSlice.actions;
export default uiSlice.reducer;

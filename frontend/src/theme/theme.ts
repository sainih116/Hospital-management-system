import { createTheme, type ThemeOptions } from "@mui/material/styles";

import type { ThemeMode } from "@/features/ui/uiSlice";

const shared: ThemeOptions = {
  shape: {
    borderRadius: 10,
  },
  typography: {
    fontFamily: [
      "Inter",
      "Roboto",
      "-apple-system",
      "BlinkMacSystemFont",
      "Segoe UI",
      "Helvetica Neue",
      "Arial",
      "sans-serif",
    ].join(","),
    h1: { fontWeight: 700 },
    h2: { fontWeight: 700 },
    h3: { fontWeight: 700 },
    h4: { fontWeight: 600 },
    h5: { fontWeight: 600 },
    h6: { fontWeight: 600 },
    button: { fontWeight: 600, textTransform: "none" },
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: { borderRadius: 8 },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: { backgroundImage: "none" },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: { boxShadow: "none" },
      },
    },
  },
};

const lightPalette: ThemeOptions["palette"] = {
  mode: "light",
  primary: { main: "#0f4c81" },
  secondary: { main: "#0f9d8f" },
  error: { main: "#d32f2f" },
  warning: { main: "#ed6c02" },
  success: { main: "#2e7d32" },
  background: { default: "#f4f6f8", paper: "#ffffff" },
};

const darkPalette: ThemeOptions["palette"] = {
  mode: "dark",
  primary: { main: "#4d8fc4" },
  secondary: { main: "#3fc7b7" },
  error: { main: "#f2726f" },
  warning: { main: "#ffa726" },
  success: { main: "#66bb6a" },
  background: { default: "#0f1620", paper: "#161f2c" },
};

export function buildTheme(mode: ThemeMode) {
  return createTheme({
    ...shared,
    palette: mode === "dark" ? darkPalette : lightPalette,
  });
}

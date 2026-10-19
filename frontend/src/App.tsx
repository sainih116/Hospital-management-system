import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import CssBaseline from "@mui/material/CssBaseline";
import { ThemeProvider } from "@mui/material/styles";
import { useEffect, useMemo } from "react";

import { useAppDispatch, useAppSelector } from "@/app/hooks";
import ErrorBoundary from "@/components/common/ErrorBoundary";
import { bootstrapAuth, sessionExpired } from "@/features/auth/authSlice";
import AppRoutes from "@/routes/AppRoutes";
import { setSessionExpiredHandler } from "@/services/apiClient";
import { buildTheme } from "@/theme/theme";

export default function App() {
  const dispatch = useAppDispatch();
  const themeMode = useAppSelector((state) => state.ui.themeMode);
  const bootstrapped = useAppSelector((state) => state.auth.bootstrapped);
  const theme = useMemo(() => buildTheme(themeMode), [themeMode]);

  useEffect(() => {
    setSessionExpiredHandler(() => dispatch(sessionExpired()));
    void dispatch(bootstrapAuth());
  }, [dispatch]);

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <ErrorBoundary>
        {bootstrapped ? (
          <AppRoutes />
        ) : (
          <Box
            sx={{
              minHeight: "100vh",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              bgcolor: "background.default",
            }}
          >
            <CircularProgress />
          </Box>
        )}
      </ErrorBoundary>
    </ThemeProvider>
  );
}

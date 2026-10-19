import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
}

/**
 * Top-level safety net: React unmounts the whole tree below the nearest
 * error boundary when a render throws, and this app previously had none —
 * any uncaught exception in any page component (an unexpected API shape,
 * a null field access, ...) white-screened the entire app with no
 * recovery short of the user guessing to reload. Must be a class
 * component; React has no hooks-based equivalent to
 * getDerivedStateFromError/componentDidCatch.
 */
export default class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // No error-reporting service is wired up in this build — console.error
    // at least keeps the failure visible in the browser devtools instead
    // of disappearing silently along with the crashed tree.
    console.error("Unhandled UI error:", error, info.componentStack);
  }

  private handleReload = (): void => {
    window.location.assign("/dashboard");
  };

  render(): ReactNode {
    if (!this.state.hasError) {
      return this.props.children;
    }

    return (
      <Box
        sx={{
          minHeight: "100vh",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          p: 3,
          bgcolor: "background.default",
        }}
      >
        <Stack spacing={2} alignItems="center" textAlign="center" maxWidth={420}>
          <Typography variant="h5" fontWeight={700}>
            Something went wrong
          </Typography>
          <Typography variant="body2" color="text.secondary">
            An unexpected error occurred while rendering this page. Your work on other tabs/sessions is unaffected —
            reloading will return you to the dashboard.
          </Typography>
          <Button variant="contained" onClick={this.handleReload}>
            Reload
          </Button>
        </Stack>
      </Box>
    );
  }
}

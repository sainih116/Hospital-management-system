import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import { useNavigate } from "react-router-dom";

/**
 * Shown when an authenticated user navigates (by URL, not through the nav —
 * the sidebar already hides items the current role can't use) to a route
 * their role lacks the backend permission for. Renders inside
 * DashboardLayout, unlike NotFoundPage, so the sidebar/navbar stay usable.
 */
export default function ForbiddenPage() {
  const navigate = useNavigate();

  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 2,
        textAlign: "center",
        py: 10,
      }}
    >
      <Typography variant="h2" fontWeight={800} color="primary">
        403
      </Typography>
      <Typography variant="h6">Access denied</Typography>
      <Typography variant="body2" color="text.secondary" maxWidth={420}>
        Your role does not have permission to view this page. If you believe this is a mistake, contact an
        administrator.
      </Typography>
      <Button variant="contained" onClick={() => navigate("/dashboard")}>
        Back to Dashboard
      </Button>
    </Box>
  );
}

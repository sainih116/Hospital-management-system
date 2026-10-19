import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import { useNavigate } from "react-router-dom";

export default function NotFoundPage() {
  const navigate = useNavigate();

  return (
    <Box
      sx={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 2,
        textAlign: "center",
        p: 3,
      }}
    >
      <Typography variant="h2" fontWeight={800} color="primary">
        404
      </Typography>
      <Typography variant="h6">Page not found</Typography>
      <Typography variant="body2" color="text.secondary" maxWidth={420}>
        The page you are looking for does not exist or you may not have access to it.
      </Typography>
      <Button variant="contained" onClick={() => navigate("/dashboard")}>
        Back to Dashboard
      </Button>
    </Box>
  );
}

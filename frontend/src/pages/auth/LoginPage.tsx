import { zodResolver } from "@hookform/resolvers/zod";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useLocation, useNavigate, type Location } from "react-router-dom";
import { z } from "zod";

import { useAppDispatch, useAppSelector } from "@/app/hooks";
import { login } from "@/features/auth/authSlice";

const loginSchema = z.object({
  email: z.string().min(1, "Email is required").email("Enter a valid email address"),
  password: z.string().min(1, "Password is required"),
});

type LoginFormValues = z.infer<typeof loginSchema>;

interface LocationState {
  from?: Location;
}

export default function LoginPage() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const location = useLocation();

  const isAuthenticated = useAppSelector((state) => state.auth.isAuthenticated);
  const authStatus = useAppSelector((state) => state.auth.status);
  const authError = useAppSelector((state) => state.auth.error);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });

  useEffect(() => {
    if (!isAuthenticated) return;
    const state = location.state as LocationState | null;
    const destination = state?.from?.pathname ?? "/dashboard";
    navigate(destination, { replace: true });
  }, [isAuthenticated, location.state, navigate]);

  const onSubmit = (values: LoginFormValues) => {
    void dispatch(login(values));
  };

  return (
    <Box
      sx={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        bgcolor: "background.default",
        p: 2,
      }}
    >
      <Paper variant="outlined" sx={{ p: 4, width: "100%", maxWidth: 400 }}>
        <Stack spacing={1} alignItems="center" sx={{ mb: 3 }}>
          <Typography variant="h5" fontWeight={700} color="primary">
            HMS
          </Typography>
          <Typography variant="body2" color="text.secondary" textAlign="center">
            Sign in with your staff account.
          </Typography>
        </Stack>

        {authError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {authError}
          </Alert>
        )}

        <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
          <Stack spacing={2}>
            <TextField
              label="Email"
              type="email"
              autoComplete="username"
              fullWidth
              autoFocus
              error={!!errors.email}
              helperText={errors.email?.message}
              {...register("email")}
            />
            <TextField
              label="Password"
              type="password"
              autoComplete="current-password"
              fullWidth
              error={!!errors.password}
              helperText={errors.password?.message}
              {...register("password")}
            />
            <Button type="submit" variant="contained" fullWidth disabled={authStatus === "loading"} sx={{ py: 1.2 }}>
              {authStatus === "loading" ? <CircularProgress size={22} color="inherit" /> : "Sign In"}
            </Button>
          </Stack>
        </Box>
      </Paper>
    </Box>
  );
}

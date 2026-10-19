import EventRepeatRoundedIcon from "@mui/icons-material/EventRepeatRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  buildReschedulePayload,
  rescheduleFormSchema,
  type RescheduleFormInput,
} from "@/features/appointments/appointmentSchema";
import { appointmentsApi, type Appointment, type AppointmentStatus } from "@/features/appointments/appointmentsApi";
import {
  DOCTOR_ALLOWED_STATUSES,
  STATUS_COLORS,
  STATUS_LABELS,
  allowedNextStatuses,
} from "@/features/appointments/appointmentStateMachine";
import { useAppSelector } from "@/app/hooks";
import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { ApiRequestError } from "@/services/apiClient";

interface FieldProps {
  label: string;
  value?: string;
}

function Field({ label, value }: FieldProps) {
  return (
    <Box>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      <Typography variant="body1">{value || "—"}</Typography>
    </Box>
  );
}

const REASON_REQUIRED_STATUSES: AppointmentStatus[] = ["CANCELLED", "NO_SHOW"];

export default function AppointmentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const role = useAppSelector((state) => state.auth.user?.role);
  const canManage = useHasPermission(PERMISSIONS.APPOINTMENT_MANAGE);

  const [appointment, setAppointment] = useState<Appointment | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [pendingStatus, setPendingStatus] = useState<AppointmentStatus | null>(null);
  const [reasonText, setReasonText] = useState("");
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const [rescheduleOpen, setRescheduleOpen] = useState(false);
  const [rescheduleForm, setRescheduleForm] = useState<RescheduleFormInput>({ startAt: "", durationMinutes: 30 });
  const [rescheduleErrors, setRescheduleErrors] = useState<Record<string, string>>({});
  const [rescheduleError, setRescheduleError] = useState<string | null>(null);
  const [rescheduling, setRescheduling] = useState(false);

  const load = () => {
    if (!id) return;
    setLoading(true);
    appointmentsApi
      .getById(id)
      .then(setAppointment)
      .catch((err: unknown) => setError(err instanceof ApiRequestError ? err.message : "Failed to load appointment."))
      .finally(() => setLoading(false));
  };

  useEffect(load, [id]);

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !appointment) {
    return <Alert severity="error">{error ?? "Appointment not found."}</Alert>;
  }

  let visibleNextStatuses = allowedNextStatuses(appointment.status);
  if (!canManage && role === "DOCTOR") {
    visibleNextStatuses = visibleNextStatuses.filter((s) => DOCTOR_ALLOWED_STATUSES.includes(s));
  } else if (!canManage) {
    visibleNextStatuses = [];
  }

  const startStatusChange = (target: AppointmentStatus) => {
    setActionError(null);
    setReasonText("");
    if (REASON_REQUIRED_STATUSES.includes(target)) {
      setPendingStatus(target);
      return;
    }
    void applyStatusChange(target);
  };

  const applyStatusChange = async (target: AppointmentStatus, reason?: string) => {
    if (!id) return;
    setActionLoading(true);
    setActionError(null);
    try {
      const updated = await appointmentsApi.changeStatus(id, {
        status: target as Exclude<AppointmentStatus, "SCHEDULED" | "RESCHEDULED">,
        reason,
      });
      setAppointment(updated);
      setPendingStatus(null);
    } catch (err) {
      setActionError(err instanceof ApiRequestError ? err.message : "Failed to update status.");
    } finally {
      setActionLoading(false);
    }
  };

  const submitReschedule = async () => {
    const parsed = rescheduleFormSchema.safeParse(rescheduleForm);
    if (!parsed.success) {
      const fieldErrors: Record<string, string> = {};
      for (const issue of parsed.error.issues) {
        fieldErrors[String(issue.path[0])] = issue.message;
      }
      setRescheduleErrors(fieldErrors);
      return;
    }
    setRescheduleErrors({});
    setRescheduleError(null);
    setRescheduling(true);
    try {
      const created = await appointmentsApi.reschedule(id!, buildReschedulePayload(parsed.data));
      navigate(`/appointments/${created.id}`);
    } catch (err) {
      setRescheduleError(err instanceof ApiRequestError ? err.message : "Failed to reschedule.");
    } finally {
      setRescheduling(false);
    }
  };

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>
            {appointment.patientName} with {appointment.doctorName}
          </Typography>
          <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
            <Typography variant="body2" color="text.secondary">
              {new Date(appointment.startAt).toLocaleString()}
            </Typography>
            <Chip size="small" label={STATUS_LABELS[appointment.status]} color={STATUS_COLORS[appointment.status]} />
          </Stack>
        </Box>
        {canManage && !["CANCELLED", "COMPLETED", "NO_SHOW", "RESCHEDULED"].includes(appointment.status) && (
          <Button variant="outlined" startIcon={<EventRepeatRoundedIcon />} onClick={() => setRescheduleOpen(true)}>
            Reschedule
          </Button>
        )}
      </Stack>

      {actionError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {actionError}
        </Alert>
      )}

      {visibleNextStatuses.length > 0 && (
        <Paper variant="outlined" sx={{ p: 2, mb: 3 }}>
          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            Actions
          </Typography>
          <Stack direction="row" spacing={1} flexWrap="wrap">
            {visibleNextStatuses.map((target) => (
              <Button
                key={target}
                variant="outlined"
                color={target === "CANCELLED" || target === "NO_SHOW" ? "error" : "primary"}
                disabled={actionLoading}
                onClick={() => startStatusChange(target)}
              >
                {STATUS_LABELS[target]}
              </Button>
            ))}
          </Stack>
        </Paper>
      )}

      <Paper variant="outlined" sx={{ p: 3 }}>
        <Grid container spacing={3}>
          <Grid item xs={12} sm={6}>
            <Field label="Patient" value={appointment.patientName} />
          </Grid>
          <Grid item xs={12} sm={6}>
            <Field label="Doctor" value={appointment.doctorName} />
          </Grid>
          <Grid item xs={12} sm={6}>
            <Field label="Start" value={new Date(appointment.startAt).toLocaleString()} />
          </Grid>
          <Grid item xs={12} sm={6}>
            <Field label="End" value={new Date(appointment.endAt).toLocaleString()} />
          </Grid>
          <Grid item xs={12}>
            <Field label="Reason" value={appointment.reason} />
          </Grid>
          {appointment.cancelReason && (
            <Grid item xs={12}>
              <Field label="Cancellation / No-show reason" value={appointment.cancelReason} />
            </Grid>
          )}
        </Grid>
      </Paper>

      <Dialog open={pendingStatus !== null} onClose={() => setPendingStatus(null)}>
        <DialogTitle>{pendingStatus === "CANCELLED" ? "Cancel appointment" : "Mark as no-show"}</DialogTitle>
        <DialogContent>
          <TextField
            label="Reason (optional)"
            fullWidth
            multiline
            minRows={2}
            sx={{ mt: 1, minWidth: 320 }}
            value={reasonText}
            onChange={(e) => setReasonText(e.target.value)}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPendingStatus(null)} disabled={actionLoading}>
            Back
          </Button>
          <Button
            color="error"
            variant="contained"
            disabled={actionLoading}
            onClick={() => pendingStatus && applyStatusChange(pendingStatus, reasonText || undefined)}
          >
            {actionLoading ? <CircularProgress size={20} color="inherit" /> : "Confirm"}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={rescheduleOpen} onClose={() => setRescheduleOpen(false)}>
        <DialogTitle>Reschedule appointment</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1, minWidth: 320 }}>
            {rescheduleError && <Alert severity="error">{rescheduleError}</Alert>}
            <TextField
              label="New Start Time"
              type="datetime-local"
              fullWidth
              required
              slotProps={{ inputLabel: { shrink: true } }}
              value={rescheduleForm.startAt}
              onChange={(e) => setRescheduleForm((prev) => ({ ...prev, startAt: e.target.value }))}
              error={!!rescheduleErrors.startAt}
              helperText={rescheduleErrors.startAt}
            />
            <TextField
              label="Duration (minutes)"
              type="number"
              fullWidth
              required
              value={rescheduleForm.durationMinutes}
              onChange={(e) => setRescheduleForm((prev) => ({ ...prev, durationMinutes: Number(e.target.value) }))}
              error={!!rescheduleErrors.durationMinutes}
              helperText={rescheduleErrors.durationMinutes}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRescheduleOpen(false)} disabled={rescheduling}>
            Cancel
          </Button>
          <Button variant="contained" onClick={submitReschedule} disabled={rescheduling}>
            {rescheduling ? <CircularProgress size={20} color="inherit" /> : "Reschedule"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

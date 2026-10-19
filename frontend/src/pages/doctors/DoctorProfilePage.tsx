import DeleteRoundedIcon from "@mui/icons-material/DeleteRounded";
import EditRoundedIcon from "@mui/icons-material/EditRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { DAYS_OF_WEEK } from "@/features/doctors/doctorSchema";
import { doctorsApi, type Doctor } from "@/features/doctors/doctorsApi";
import { ApiRequestError } from "@/services/apiClient";

interface FieldProps {
  label: string;
  value?: string | number;
}

function Field({ label, value }: FieldProps) {
  return (
    <Box>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      <Typography variant="body1">{value === undefined || value === "" ? "—" : value}</Typography>
    </Box>
  );
}

export default function DoctorProfilePage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const canManage = useHasPermission(PERMISSIONS.DOCTOR_MANAGE);

  const [doctor, setDoctor] = useState<Doctor | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;

    doctorsApi
      .getById(id)
      .then((data) => {
        if (!cancelled) setDoctor(data);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof ApiRequestError ? err.message : "Failed to load doctor.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [id]);

  const handleDeactivate = async () => {
    if (!id) return;
    setDeleting(true);
    setDeleteError(null);
    try {
      await doctorsApi.remove(id);
      navigate("/doctors");
    } catch (err) {
      setDeleteError(err instanceof ApiRequestError ? err.message : "Failed to deactivate doctor.");
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !doctor) {
    return <Alert severity="error">{error ?? "Doctor not found."}</Alert>;
  }

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>
            {doctor.name}
          </Typography>
          <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
            <Typography variant="body2" color="text.secondary">
              {doctor.specialization} · {doctor.departmentName}
            </Typography>
            <Chip size="small" label={doctor.status} color={doctor.status === "ACTIVE" ? "success" : "default"} />
          </Stack>
        </Box>
        {canManage && (
          <Stack direction="row" spacing={1}>
            <Button variant="outlined" startIcon={<EditRoundedIcon />} onClick={() => navigate(`/doctors/${doctor.id}/edit`)}>
              Edit
            </Button>
            {doctor.status === "ACTIVE" && (
              <Button color="error" variant="outlined" startIcon={<DeleteRoundedIcon />} onClick={() => setConfirmOpen(true)}>
                Deactivate
              </Button>
            )}
          </Stack>
        )}
      </Stack>

      <Paper variant="outlined" sx={{ p: 3, mb: 3 }}>
        <Typography variant="subtitle2" color="text.secondary" gutterBottom>
          Profile
        </Typography>
        <Grid container spacing={3}>
          <Grid item xs={12} sm={4}>
            <Field label="Email" value={doctor.email} />
          </Grid>
          <Grid item xs={12} sm={4}>
            <Field label="License Number" value={doctor.licenseNo} />
          </Grid>
          <Grid item xs={12} sm={4}>
            <Field label="Consultation Fee" value={doctor.fee} />
          </Grid>
          <Grid item xs={12}>
            <Field label="Bio" value={doctor.bio} />
          </Grid>
        </Grid>
      </Paper>

      <Paper variant="outlined" sx={{ p: 3 }}>
        <Typography variant="subtitle2" color="text.secondary" gutterBottom>
          Weekly Availability
        </Typography>
        {doctor.availability && doctor.availability.length > 0 ? (
          <Stack spacing={1}>
            {doctor.availability
              .slice()
              .sort((a, b) => a.dayOfWeek - b.dayOfWeek)
              .map((slot) => (
                <Stack key={slot.dayOfWeek} direction="row" spacing={2}>
                  <Typography sx={{ width: 100 }} fontWeight={600}>
                    {DAYS_OF_WEEK.find((d) => d.value === slot.dayOfWeek)?.label}
                  </Typography>
                  <Typography color="text.secondary">
                    {slot.startTime} – {slot.endTime}
                  </Typography>
                </Stack>
              ))}
          </Stack>
        ) : (
          <Typography color="text.secondary">No recurring availability configured.</Typography>
        )}
      </Paper>

      <Dialog open={confirmOpen} onClose={() => setConfirmOpen(false)}>
        <DialogTitle>Deactivate doctor?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {doctor.name} will be hidden from doctor search and appointment booking. Their record is retained.
          </DialogContentText>
          {deleteError && (
            <Alert severity="error" sx={{ mt: 2 }}>
              {deleteError}
            </Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmOpen(false)} disabled={deleting}>
            Cancel
          </Button>
          <Button color="error" variant="contained" onClick={handleDeactivate} disabled={deleting}>
            {deleting ? <CircularProgress size={20} color="inherit" /> : "Deactivate"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

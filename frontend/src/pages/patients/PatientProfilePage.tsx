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
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { patientsApi, type Patient } from "@/features/patients/patientsApi";
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

export default function PatientProfilePage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const canUpdate = useHasPermission(PERMISSIONS.PATIENT_UPDATE);
  const canDelete = useHasPermission(PERMISSIONS.PATIENT_DELETE);

  const [patient, setPatient] = useState<Patient | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;

    setLoading(true);
    patientsApi
      .getById(id)
      .then((data) => {
        if (!cancelled) setPatient(data);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof ApiRequestError ? err.message : "Failed to load patient.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [id]);

  const handleDelete = async () => {
    if (!id) return;
    setDeleting(true);
    setDeleteError(null);
    try {
      await patientsApi.remove(id);
      navigate("/patients");
    } catch (err) {
      setDeleteError(err instanceof ApiRequestError ? err.message : "Failed to deactivate patient.");
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

  if (error || !patient) {
    return <Alert severity="error">{error ?? "Patient not found."}</Alert>;
  }

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>
            {patient.name}
          </Typography>
          <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
            <Typography variant="body2" color="text.secondary">
              {patient.patientCode}
            </Typography>
            <Chip size="small" label={patient.status} color={patient.status === "ACTIVE" ? "success" : "default"} />
            <Chip size="small" variant="outlined" label={patient.type} />
          </Stack>
        </Box>
        <Stack direction="row" spacing={1}>
          {canUpdate && (
            <Button
              variant="outlined"
              startIcon={<EditRoundedIcon />}
              onClick={() => navigate(`/patients/${patient.id}/edit`)}
            >
              Edit
            </Button>
          )}
          {canDelete && patient.status === "ACTIVE" && (
            <Button color="error" variant="outlined" startIcon={<DeleteRoundedIcon />} onClick={() => setConfirmOpen(true)}>
              Deactivate
            </Button>
          )}
        </Stack>
      </Stack>

      <Paper variant="outlined" sx={{ p: 3 }}>
        <Typography variant="subtitle2" color="text.secondary" gutterBottom>
          Demographics
        </Typography>
        <Grid container spacing={3} sx={{ mb: 2 }}>
          <Grid item xs={12} sm={4}>
            <Field label="Date of Birth" value={patient.dob} />
          </Grid>
          <Grid item xs={12} sm={4}>
            <Field label="Gender" value={patient.gender} />
          </Grid>
          <Grid item xs={12} sm={4}>
            <Field label="Blood Group" value={patient.bloodGroup} />
          </Grid>
          <Grid item xs={12}>
            <Field label="Allergies" value={patient.allergies?.join(", ")} />
          </Grid>
        </Grid>

        <Divider sx={{ my: 2 }} />
        <Typography variant="subtitle2" color="text.secondary" gutterBottom>
          Contact
        </Typography>
        <Grid container spacing={3} sx={{ mb: 2 }}>
          <Grid item xs={12} sm={6}>
            <Field label="Phone" value={patient.phone} />
          </Grid>
          <Grid item xs={12} sm={6}>
            <Field label="Email" value={patient.email} />
          </Grid>
          <Grid item xs={12}>
            <Field label="Address" value={patient.address} />
          </Grid>
        </Grid>

        <Divider sx={{ my: 2 }} />
        <Typography variant="subtitle2" color="text.secondary" gutterBottom>
          Emergency Contact
        </Typography>
        <Grid container spacing={3} sx={{ mb: 2 }}>
          <Grid item xs={12} sm={4}>
            <Field label="Name" value={patient.emergencyContact?.name} />
          </Grid>
          <Grid item xs={12} sm={4}>
            <Field label="Phone" value={patient.emergencyContact?.phone} />
          </Grid>
          <Grid item xs={12} sm={4}>
            <Field label="Relationship" value={patient.emergencyContact?.relationship} />
          </Grid>
        </Grid>

        <Divider sx={{ my: 2 }} />
        <Typography variant="subtitle2" color="text.secondary" gutterBottom>
          Insurance
        </Typography>
        <Grid container spacing={3}>
          <Grid item xs={12} sm={6}>
            <Field label="Provider" value={patient.insurance?.provider} />
          </Grid>
          <Grid item xs={12} sm={6}>
            <Field label="Policy Number" value={patient.insurance?.policyNumber} />
          </Grid>
        </Grid>
      </Paper>

      <Dialog open={confirmOpen} onClose={() => setConfirmOpen(false)}>
        <DialogTitle>Deactivate patient?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {patient.name} ({patient.patientCode}) will be hidden from patient search and lists. Their record is
            retained and can be restored by an administrator if needed.
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
          <Button color="error" variant="contained" onClick={handleDelete} disabled={deleting}>
            {deleting ? <CircularProgress size={20} color="inherit" /> : "Deactivate"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

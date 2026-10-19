import CancelRoundedIcon from "@mui/icons-material/CancelRounded";
import LocalPharmacyRoundedIcon from "@mui/icons-material/LocalPharmacyRounded";
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
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { prescriptionsApi, type Prescription } from "@/features/prescriptions/prescriptionsApi";
import { ApiRequestError } from "@/services/apiClient";

export default function PrescriptionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const canWrite = useHasPermission(PERMISSIONS.PRESCRIPTION_WRITE);
  const canDispense = useHasPermission(PERMISSIONS.PHARMACY_DISPENSE);

  const [prescription, setPrescription] = useState<Prescription | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    prescriptionsApi
      .getById(id)
      .then((data) => {
        if (!cancelled) setPrescription(data);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof ApiRequestError ? err.message : "Failed to load prescription.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  const handleCancel = async () => {
    if (!id) return;
    setCancelling(true);
    setCancelError(null);
    try {
      const updated = await prescriptionsApi.cancel(id);
      setPrescription(updated);
      setConfirmOpen(false);
    } catch (err) {
      setCancelError(err instanceof ApiRequestError ? err.message : "Failed to cancel prescription.");
    } finally {
      setCancelling(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !prescription) {
    return <Alert severity="error">{error ?? "Prescription not found."}</Alert>;
  }

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>
            {prescription.patientName}
          </Typography>
          <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
            <Typography variant="body2" color="text.secondary">
              Prescribed by {prescription.doctorName} on {new Date(prescription.issuedAt).toLocaleString()}
            </Typography>
            <Chip size="small" label={prescription.status} color={prescription.status === "ACTIVE" ? "success" : "default"} />
          </Stack>
        </Box>
        {(canDispense || canWrite) && prescription.status === "ACTIVE" && (
          <Stack direction="row" spacing={2}>
            {canDispense && (
              <Button
                variant="contained"
                startIcon={<LocalPharmacyRoundedIcon />}
                onClick={() => navigate(`/pharmacy/dispense?prescriptionId=${prescription.id}`)}
              >
                Dispense
              </Button>
            )}
            {canWrite && (
              <Button color="error" variant="outlined" startIcon={<CancelRoundedIcon />} onClick={() => setConfirmOpen(true)}>
                Cancel Prescription
              </Button>
            )}
          </Stack>
        )}
      </Stack>

      <Paper variant="outlined">
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Medicine</TableCell>
              <TableCell>Dose</TableCell>
              <TableCell>Frequency</TableCell>
              <TableCell>Duration</TableCell>
              <TableCell>Instructions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {prescription.items.map((item, index) => (
              <TableRow key={index}>
                <TableCell>{item.medicineName}</TableCell>
                <TableCell>{item.dose}</TableCell>
                <TableCell>{item.frequency}</TableCell>
                <TableCell>{item.duration}</TableCell>
                <TableCell>{item.instructions || "—"}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Paper>

      <Dialog open={confirmOpen} onClose={() => setConfirmOpen(false)}>
        <DialogTitle>Cancel prescription?</DialogTitle>
        <DialogContent>
          <DialogContentText>This prescription will be marked cancelled and can no longer be dispensed.</DialogContentText>
          {cancelError && (
            <Alert severity="error" sx={{ mt: 2 }}>
              {cancelError}
            </Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmOpen(false)} disabled={cancelling}>
            Back
          </Button>
          <Button color="error" variant="contained" onClick={handleCancel} disabled={cancelling}>
            {cancelling ? <CircularProgress size={20} color="inherit" /> : "Cancel Prescription"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

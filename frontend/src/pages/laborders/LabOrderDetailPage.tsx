import CancelRoundedIcon from "@mui/icons-material/CancelRounded";
import PrintRoundedIcon from "@mui/icons-material/PrintRounded";
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
import FormControl from "@mui/material/FormControl";
import Grid from "@mui/material/Grid";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { labOrdersApi, type LabOrder, type ResultFlag } from "@/features/laborders/labOrdersApi";
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

const FLAGS: ResultFlag[] = ["NORMAL", "LOW", "HIGH", "CRITICAL"];

export default function LabOrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const canWrite = useHasPermission(PERMISSIONS.LAB_ORDER_WRITE);
  const canVerify = useHasPermission(PERMISSIONS.LAB_RESULT_VERIFY);

  const [order, setOrder] = useState<LabOrder | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [resultValue, setResultValue] = useState("");
  const [flag, setFlag] = useState<ResultFlag>("NORMAL");
  const [notes, setNotes] = useState("");
  const [entering, setEntering] = useState(false);
  const [enterError, setEnterError] = useState<string | null>(null);

  const [verifying, setVerifying] = useState(false);
  const [verifyError, setVerifyError] = useState<string | null>(null);

  const [cancelOpen, setCancelOpen] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState<string | null>(null);

  const load = () => {
    if (!id) return;
    setLoading(true);
    labOrdersApi
      .getById(id)
      .then(setOrder)
      .catch((err: unknown) => setError(err instanceof ApiRequestError ? err.message : "Failed to load lab order."))
      .finally(() => setLoading(false));
  };

  useEffect(load, [id]);

  const handleEnterResult = async () => {
    if (!id) return;
    setEntering(true);
    setEnterError(null);
    try {
      const updated = await labOrdersApi.enterResult(id, { resultValue, flag, notes: notes || undefined });
      setOrder(updated);
    } catch (err) {
      setEnterError(err instanceof ApiRequestError ? err.message : "Failed to enter result.");
    } finally {
      setEntering(false);
    }
  };

  const handleVerify = async () => {
    if (!order?.result) return;
    setVerifying(true);
    setVerifyError(null);
    try {
      const updated = await labOrdersApi.verifyResult(order.result.id);
      setOrder(updated);
    } catch (err) {
      setVerifyError(err instanceof ApiRequestError ? err.message : "Failed to verify result.");
    } finally {
      setVerifying(false);
    }
  };

  const handleCancel = async () => {
    if (!id) return;
    setCancelling(true);
    setCancelError(null);
    try {
      const updated = await labOrdersApi.cancel(id);
      setOrder(updated);
      setCancelOpen(false);
    } catch (err) {
      setCancelError(err instanceof ApiRequestError ? err.message : "Failed to cancel order.");
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

  if (error || !order) {
    return <Alert severity="error">{error ?? "Lab order not found."}</Alert>;
  }

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>
            {order.testName}
          </Typography>
          <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
            <Typography variant="body2" color="text.secondary">
              {order.sampleCode} · {order.patientName} · Ordered by {order.doctorName}
            </Typography>
            <Chip size="small" label={order.status.replace("_", " ")} />
          </Stack>
        </Box>
        <Stack direction="row" spacing={1}>
          {order.status === "COMPLETED" && (
            <Button variant="outlined" startIcon={<PrintRoundedIcon />} onClick={() => navigate(`/laboratory/${order.id}/report`)}>
              View Report
            </Button>
          )}
          {canWrite && order.status === "ORDERED" && (
            <Button color="error" variant="outlined" startIcon={<CancelRoundedIcon />} onClick={() => setCancelOpen(true)}>
              Cancel Order
            </Button>
          )}
        </Stack>
      </Stack>

      <Paper variant="outlined" sx={{ p: 3, mb: 3 }}>
        <Grid container spacing={3}>
          <Grid item xs={12} sm={4}>
            <Field label="Test Code" value={order.testCode} />
          </Grid>
          <Grid item xs={12} sm={4}>
            <Field label="Unit" value={order.unit} />
          </Grid>
          <Grid item xs={12} sm={4}>
            <Field label="Normal Range" value={order.normalRange} />
          </Grid>
        </Grid>
      </Paper>

      {canWrite && order.status === "ORDERED" && (
        <Paper variant="outlined" sx={{ p: 3, mb: 3 }}>
          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            Enter Result
          </Typography>
          {enterError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {enterError}
            </Alert>
          )}
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6}>
              <TextField label="Result Value" fullWidth value={resultValue} onChange={(e) => setResultValue(e.target.value)} />
            </Grid>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth>
                <InputLabel id="flag-label">Flag</InputLabel>
                <Select labelId="flag-label" label="Flag" value={flag} onChange={(e) => setFlag(e.target.value as ResultFlag)}>
                  {FLAGS.map((f) => (
                    <MenuItem key={f} value={f}>
                      {f}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12}>
              <TextField label="Notes" fullWidth multiline minRows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
            </Grid>
          </Grid>
          <Stack direction="row" justifyContent="flex-end" sx={{ mt: 2 }}>
            <Button variant="contained" onClick={handleEnterResult} disabled={entering || !resultValue}>
              {entering ? <CircularProgress size={20} color="inherit" /> : "Submit Result"}
            </Button>
          </Stack>
        </Paper>
      )}

      {order.result && (
        <Paper variant="outlined" sx={{ p: 3 }}>
          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            Result
          </Typography>
          {verifyError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {verifyError}
            </Alert>
          )}
          <Grid container spacing={3} sx={{ mb: 2 }}>
            <Grid item xs={12} sm={4}>
              <Field label="Value" value={order.result.resultValue} />
            </Grid>
            <Grid item xs={12} sm={4}>
              <Field label="Flag" value={order.result.flag} />
            </Grid>
            <Grid item xs={12} sm={4}>
              <Field label="Entered By" value={order.result.enteredByName} />
            </Grid>
            {order.result.notes && (
              <Grid item xs={12}>
                <Field label="Notes" value={order.result.notes} />
              </Grid>
            )}
          </Grid>

          {order.result.verifiedAt ? (
            <>
              <Divider sx={{ my: 2 }} />
              <Field label="Verified By" value={`${order.result.verifiedByName} on ${new Date(order.result.verifiedAt).toLocaleString()}`} />
            </>
          ) : (
            canVerify && (
              <Stack direction="row" justifyContent="flex-end" sx={{ mt: 1 }}>
                <Button variant="contained" onClick={handleVerify} disabled={verifying}>
                  {verifying ? <CircularProgress size={20} color="inherit" /> : "Verify Result"}
                </Button>
              </Stack>
            )
          )}
        </Paper>
      )}

      <Dialog open={cancelOpen} onClose={() => setCancelOpen(false)}>
        <DialogTitle>Cancel lab order?</DialogTitle>
        <DialogContent>
          <DialogContentText>This order will be marked cancelled and can no longer be processed.</DialogContentText>
          {cancelError && (
            <Alert severity="error" sx={{ mt: 2 }}>
              {cancelError}
            </Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCancelOpen(false)} disabled={cancelling}>
            Back
          </Button>
          <Button color="error" variant="contained" onClick={handleCancel} disabled={cancelling}>
            {cancelling ? <CircularProgress size={20} color="inherit" /> : "Cancel Order"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

import AddRoundedIcon from "@mui/icons-material/AddRounded";
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
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { pharmacyApi, type Batch, type Medicine, type ReceiveBatchPayload } from "@/features/pharmacy/pharmacyApi";
import { ApiRequestError } from "@/services/apiClient";

interface FormState {
  batchNo: string;
  expiryDate: string;
  quantity: string;
  purchasePrice: string;
  sellingPrice: string;
}

const emptyForm: FormState = { batchNo: "", expiryDate: "", quantity: "", purchasePrice: "", sellingPrice: "" };

export default function MedicineDetailPage() {
  const { id } = useParams<{ id: string }>();
  const canDispense = useHasPermission(PERMISSIONS.PHARMACY_DISPENSE);

  const [medicine, setMedicine] = useState<Medicine | null>(null);
  const [batches, setBatches] = useState<Batch[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const load = () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    Promise.all([pharmacyApi.getMedicineById(id), pharmacyApi.listBatches(id)])
      .then(([medicineResult, batchesResult]) => {
        setMedicine(medicineResult);
        setBatches(batchesResult);
      })
      .catch((err: unknown) => setError(err instanceof ApiRequestError ? err.message : "Failed to load medicine details."))
      .finally(() => setLoading(false));
  };

  useEffect(load, [id]);

  const handleReceiveBatch = async () => {
    if (!id) return;
    setSaving(true);
    setFormError(null);
    try {
      const payload: ReceiveBatchPayload = {
        batchNo: form.batchNo,
        expiryDate: new Date(form.expiryDate).toISOString(),
        quantity: Number(form.quantity) || 0,
        purchasePrice: Number(form.purchasePrice) || 0,
        sellingPrice: Number(form.sellingPrice) || 0,
      };
      await pharmacyApi.receiveBatch(id, payload);
      setDialogOpen(false);
      setForm(emptyForm);
      load();
    } catch (err) {
      setFormError(err instanceof ApiRequestError ? err.message : "Failed to receive batch.");
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !medicine) {
    return <Alert severity="error">{error ?? "Medicine not found."}</Alert>;
  }

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>
            {medicine.name}
          </Typography>
          <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
            <Typography variant="body2" color="text.secondary">
              {medicine.genericName || "No generic name"} · {medicine.category || "Uncategorized"}
            </Typography>
            {medicine.isLowStock && <Chip size="small" label="Low Stock" color="warning" />}
            <Chip size="small" label={`Stock: ${medicine.currentStock}`} color={medicine.isLowStock ? "warning" : "success"} />
          </Stack>
        </Box>
        {canDispense && (
          <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => setDialogOpen(true)}>
            Receive Batch
          </Button>
        )}
      </Stack>

      <Paper variant="outlined">
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Batch No.</TableCell>
              <TableCell>Expiry Date</TableCell>
              <TableCell align="right">Quantity</TableCell>
              <TableCell align="right">Purchase Price</TableCell>
              <TableCell align="right">Selling Price</TableCell>
              <TableCell>Status</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {batches.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} align="center" sx={{ py: 6 }}>
                  <Typography color="text.secondary">No batches received yet.</Typography>
                </TableCell>
              </TableRow>
            )}
            {batches.map((batch) => (
              <TableRow key={batch.id}>
                <TableCell>{batch.batchNo}</TableCell>
                <TableCell>{new Date(batch.expiryDate).toLocaleDateString()}</TableCell>
                <TableCell align="right">{batch.quantity}</TableCell>
                <TableCell align="right">{batch.purchasePrice.toFixed(2)}</TableCell>
                <TableCell align="right">{batch.sellingPrice.toFixed(2)}</TableCell>
                <TableCell>
                  {batch.isExpired ? (
                    <Chip size="small" label="Expired" color="error" />
                  ) : batch.quantity === 0 ? (
                    <Chip size="small" label="Depleted" color="default" />
                  ) : (
                    <Chip size="small" label="Available" color="success" />
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Paper>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Receive Batch — {medicine.name}</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 0.5 }}>
            {formError && (
              <Grid item xs={12}>
                <Alert severity="error">{formError}</Alert>
              </Grid>
            )}
            <Grid item xs={12} sm={6}>
              <TextField
                label="Batch No."
                value={form.batchNo}
                onChange={(e) => setForm((p) => ({ ...p, batchNo: e.target.value }))}
                fullWidth
                autoFocus
                required
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Expiry Date"
                type="date"
                value={form.expiryDate}
                onChange={(e) => setForm((p) => ({ ...p, expiryDate: e.target.value }))}
                fullWidth
                required
                slotProps={{ inputLabel: { shrink: true } }}
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <TextField
                label="Quantity"
                type="number"
                value={form.quantity}
                onChange={(e) => setForm((p) => ({ ...p, quantity: e.target.value }))}
                fullWidth
                required
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <TextField
                label="Purchase Price"
                type="number"
                value={form.purchasePrice}
                onChange={(e) => setForm((p) => ({ ...p, purchasePrice: e.target.value }))}
                fullWidth
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <TextField
                label="Selling Price"
                type="number"
                value={form.sellingPrice}
                onChange={(e) => setForm((p) => ({ ...p, sellingPrice: e.target.value }))}
                fullWidth
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)} disabled={saving}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleReceiveBatch}
            disabled={saving || !form.batchNo || !form.expiryDate || !form.quantity}
          >
            {saving ? <CircularProgress size={20} color="inherit" /> : "Receive Batch"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

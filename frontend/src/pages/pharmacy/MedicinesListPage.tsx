import AddRoundedIcon from "@mui/icons-material/AddRounded";
import LocalShippingRoundedIcon from "@mui/icons-material/LocalShippingRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import FormControlLabel from "@mui/material/FormControlLabel";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TablePagination from "@mui/material/TablePagination";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { pharmacyApi, type CreateMedicinePayload, type Medicine } from "@/features/pharmacy/pharmacyApi";
import { ApiRequestError } from "@/services/apiClient";

interface FormState {
  name: string;
  genericName: string;
  category: string;
  manufacturer: string;
  unit: string;
  reorderLevel: string;
}

const emptyForm: FormState = { name: "", genericName: "", category: "", manufacturer: "", unit: "", reorderLevel: "" };

export default function MedicinesListPage() {
  const navigate = useNavigate();
  const canDispense = useHasPermission(PERMISSIONS.PHARMACY_DISPENSE);

  const [query, setQuery] = useState("");
  const [lowStockOnly, setLowStockOnly] = useState(false);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(20);

  const [items, setItems] = useState<Medicine[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    setPage(0);
  }, [query, lowStockOnly]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    pharmacyApi
      .searchMedicines({ query: query || undefined, lowStock: lowStockOnly || undefined, page: page + 1, pageSize })
      .then((result) => {
        if (cancelled) return;
        setItems(result.items);
        setTotal(result.total);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof ApiRequestError ? err.message : "Failed to load the medicine catalog.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [query, lowStockOnly, page, pageSize]);

  const reload = () => {
    pharmacyApi
      .searchMedicines({ query: query || undefined, lowStock: lowStockOnly || undefined, page: page + 1, pageSize })
      .then((result) => {
        setItems(result.items);
        setTotal(result.total);
      })
      .catch(() => undefined);
  };

  const handleCreate = async () => {
    setSaving(true);
    setFormError(null);
    try {
      const payload: CreateMedicinePayload = {
        name: form.name,
        genericName: form.genericName || undefined,
        category: form.category || undefined,
        manufacturer: form.manufacturer || undefined,
        unit: form.unit || undefined,
        reorderLevel: Number(form.reorderLevel) || 0,
      };
      await pharmacyApi.createMedicine(payload);
      setDialogOpen(false);
      setForm(emptyForm);
      reload();
    } catch (err) {
      setFormError(err instanceof ApiRequestError ? err.message : "Failed to add medicine.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          Pharmacy
        </Typography>
        {canDispense && (
          <Stack direction="row" spacing={2}>
            <Button variant="outlined" startIcon={<LocalShippingRoundedIcon />} onClick={() => navigate("/pharmacy/dispense")}>
              Dispense Prescription
            </Button>
            <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => setDialogOpen(true)}>
              Add Medicine
            </Button>
          </Stack>
        )}
      </Stack>

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Stack direction={{ xs: "column", md: "row" }} spacing={2} alignItems={{ md: "center" }}>
          <TextField
            label="Search by name or generic name"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            sx={{ minWidth: 280, flexGrow: 1 }}
          />
          <FormControlLabel
            control={<Switch checked={lowStockOnly} onChange={(e) => setLowStockOnly(e.target.checked)} />}
            label="Low stock only"
          />
        </Stack>
      </Paper>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Paper variant="outlined">
        <TableContainer sx={{ overflowX: "auto" }}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Generic Name</TableCell>
                <TableCell>Category</TableCell>
                <TableCell align="right">Current Stock</TableCell>
                <TableCell align="right">Reorder Level</TableCell>
                <TableCell>Status</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading && (
                <TableRow>
                  <TableCell colSpan={6} align="center" sx={{ py: 6 }}>
                    <CircularProgress size={28} />
                  </TableCell>
                </TableRow>
              )}
              {!loading && items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} align="center" sx={{ py: 6 }}>
                    <Typography color="text.secondary">No medicines found.</Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading &&
                items.map((medicine) => (
                  <TableRow
                    key={medicine.id}
                    hover
                    sx={{ cursor: "pointer" }}
                    onClick={() => navigate(`/pharmacy/${medicine.id}`)}
                  >
                    <TableCell>{medicine.name}</TableCell>
                    <TableCell>{medicine.genericName || "—"}</TableCell>
                    <TableCell>{medicine.category || "—"}</TableCell>
                    <TableCell align="right">{medicine.currentStock}</TableCell>
                    <TableCell align="right">{medicine.reorderLevel}</TableCell>
                    <TableCell>
                      <Stack direction="row" spacing={1}>
                        {medicine.isLowStock && <Chip size="small" label="Low Stock" color="warning" />}
                        <Chip
                          size="small"
                          label={medicine.status === "ACTIVE" ? "Active" : "Inactive"}
                          color={medicine.status === "ACTIVE" ? "success" : "default"}
                        />
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
        </TableContainer>
        <TablePagination
          component="div"
          count={total}
          page={page}
          onPageChange={(_, newPage) => setPage(newPage)}
          rowsPerPage={pageSize}
          onRowsPerPageChange={(e) => {
            setPageSize(parseInt(e.target.value, 10));
            setPage(0);
          }}
          rowsPerPageOptions={[10, 20, 50, 100]}
        />
      </Paper>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Add Medicine</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 0.5 }}>
            {formError && (
              <Grid item xs={12}>
                <Alert severity="error">{formError}</Alert>
              </Grid>
            )}
            <Grid item xs={12} sm={6}>
              <TextField
                label="Name"
                value={form.name}
                onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
                fullWidth
                autoFocus
                required
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Generic Name"
                value={form.genericName}
                onChange={(e) => setForm((p) => ({ ...p, genericName: e.target.value }))}
                fullWidth
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Category"
                value={form.category}
                onChange={(e) => setForm((p) => ({ ...p, category: e.target.value }))}
                fullWidth
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Manufacturer"
                value={form.manufacturer}
                onChange={(e) => setForm((p) => ({ ...p, manufacturer: e.target.value }))}
                fullWidth
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField label="Unit" value={form.unit} onChange={(e) => setForm((p) => ({ ...p, unit: e.target.value }))} fullWidth helperText="e.g. Tablet, Capsule, Sachet" />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Reorder Level"
                type="number"
                value={form.reorderLevel}
                onChange={(e) => setForm((p) => ({ ...p, reorderLevel: e.target.value }))}
                fullWidth
                helperText="Alert threshold for low stock"
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)} disabled={saving}>
            Cancel
          </Button>
          <Button variant="contained" onClick={handleCreate} disabled={saving || !form.name}>
            {saving ? <CircularProgress size={20} color="inherit" /> : "Add Medicine"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

import AddRoundedIcon from "@mui/icons-material/AddRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";

import { labTestsApi, type LabTest } from "@/features/labtests/labTestsApi";
import { ApiRequestError } from "@/services/apiClient";

interface FormState {
  code: string;
  name: string;
  category: string;
  sampleType: string;
  unit: string;
  normalRange: string;
  cost: string;
}

const emptyForm: FormState = { code: "", name: "", category: "", sampleType: "", unit: "", normalRange: "", cost: "" };

export default function LabTestsPage() {
  const [tests, setTests] = useState<LabTest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const load = () => {
    setLoading(true);
    labTestsApi
      .list()
      .then(setTests)
      .catch((err: unknown) => setError(err instanceof ApiRequestError ? err.message : "Failed to load lab tests."))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const handleCreate = async () => {
    setSaving(true);
    setFormError(null);
    try {
      await labTestsApi.create({
        code: form.code.toUpperCase(),
        name: form.name,
        category: form.category || undefined,
        sampleType: form.sampleType || undefined,
        unit: form.unit || undefined,
        normalRange: form.normalRange || undefined,
        cost: Number(form.cost) || 0,
      });
      setDialogOpen(false);
      setForm(emptyForm);
      load();
    } catch (err) {
      setFormError(err instanceof ApiRequestError ? err.message : "Failed to create lab test.");
    } finally {
      setSaving(false);
    }
  };

  const handleToggleStatus = async (test: LabTest) => {
    const nextStatus = test.status === "ACTIVE" ? "INACTIVE" : "ACTIVE";
    setTests((prev) => prev.map((t) => (t.id === test.id ? { ...t, status: nextStatus } : t)));
    try {
      await labTestsApi.update(test.id, { status: nextStatus });
    } catch {
      load(); // revert to server state on failure
    }
  };

  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          Lab Test Catalog
        </Typography>
        <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => setDialogOpen(true)}>
          Add Lab Test
        </Button>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Paper variant="outlined">
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Code</TableCell>
              <TableCell>Name</TableCell>
              <TableCell>Category</TableCell>
              <TableCell>Cost</TableCell>
              <TableCell>Active</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading && (
              <TableRow>
                <TableCell colSpan={5} align="center" sx={{ py: 6 }}>
                  <CircularProgress size={28} />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              tests.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>{t.code}</TableCell>
                  <TableCell>{t.name}</TableCell>
                  <TableCell>{t.category || "—"}</TableCell>
                  <TableCell>{t.cost}</TableCell>
                  <TableCell>
                    <Switch checked={t.status === "ACTIVE"} onChange={() => handleToggleStatus(t)} />
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </Paper>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Add Lab Test</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 0.5 }}>
            {formError && (
              <Grid item xs={12}>
                <Alert severity="error">{formError}</Alert>
              </Grid>
            )}
            <Grid item xs={12} sm={6}>
              <TextField
                label="Code"
                value={form.code}
                onChange={(e) => setForm((p) => ({ ...p, code: e.target.value }))}
                fullWidth
                autoFocus
                helperText="Short unique code, e.g. CBC"
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField label="Name" value={form.name} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} fullWidth />
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
                label="Sample Type"
                value={form.sampleType}
                onChange={(e) => setForm((p) => ({ ...p, sampleType: e.target.value }))}
                fullWidth
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <TextField label="Unit" value={form.unit} onChange={(e) => setForm((p) => ({ ...p, unit: e.target.value }))} fullWidth />
            </Grid>
            <Grid item xs={12} sm={4}>
              <TextField
                label="Normal Range"
                value={form.normalRange}
                onChange={(e) => setForm((p) => ({ ...p, normalRange: e.target.value }))}
                fullWidth
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <TextField
                label="Cost"
                type="number"
                value={form.cost}
                onChange={(e) => setForm((p) => ({ ...p, cost: e.target.value }))}
                fullWidth
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)} disabled={saving}>
            Cancel
          </Button>
          <Button variant="contained" onClick={handleCreate} disabled={saving || !form.code || !form.name}>
            {saving ? <CircularProgress size={20} color="inherit" /> : "Create"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

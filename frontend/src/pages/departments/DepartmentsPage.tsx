import AddRoundedIcon from "@mui/icons-material/AddRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
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

import { departmentsApi, type Department } from "@/features/departments/departmentsApi";
import { ApiRequestError } from "@/services/apiClient";

export default function DepartmentsPage() {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const load = () => {
    setLoading(true);
    departmentsApi
      .list()
      .then(setDepartments)
      .catch((err: unknown) => setError(err instanceof ApiRequestError ? err.message : "Failed to load departments."))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const handleCreate = async () => {
    setSaving(true);
    setFormError(null);
    try {
      await departmentsApi.create({ name, code: code.toUpperCase() });
      setDialogOpen(false);
      setName("");
      setCode("");
      load();
    } catch (err) {
      setFormError(err instanceof ApiRequestError ? err.message : "Failed to create department.");
    } finally {
      setSaving(false);
    }
  };

  const handleToggleStatus = async (dept: Department) => {
    const nextStatus = dept.status === "ACTIVE" ? "INACTIVE" : "ACTIVE";
    setDepartments((prev) => prev.map((d) => (d.id === dept.id ? { ...d, status: nextStatus } : d)));
    try {
      await departmentsApi.update(dept.id, { status: nextStatus });
    } catch {
      load(); // revert to server state on failure
    }
  };

  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          Departments
        </Typography>
        <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => setDialogOpen(true)}>
          Add Department
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
              <TableCell>Name</TableCell>
              <TableCell>Code</TableCell>
              <TableCell>Active</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading && (
              <TableRow>
                <TableCell colSpan={3} align="center" sx={{ py: 6 }}>
                  <CircularProgress size={28} />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              departments.map((d) => (
                <TableRow key={d.id}>
                  <TableCell>{d.name}</TableCell>
                  <TableCell>{d.code}</TableCell>
                  <TableCell>
                    <Switch checked={d.status === "ACTIVE"} onChange={() => handleToggleStatus(d)} />
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </Paper>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} fullWidth maxWidth="xs">
        <DialogTitle>Add Department</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            {formError && <Alert severity="error">{formError}</Alert>}
            <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} fullWidth autoFocus />
            <TextField
              label="Code"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              fullWidth
              helperText="Short unique code, e.g. CARD"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)} disabled={saving}>
            Cancel
          </Button>
          <Button variant="contained" onClick={handleCreate} disabled={saving || !name || !code}>
            {saving ? <CircularProgress size={20} color="inherit" /> : "Create"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

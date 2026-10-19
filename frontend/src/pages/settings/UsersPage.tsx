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
import FormControl from "@mui/material/FormControl";
import Grid from "@mui/material/Grid";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select, { type SelectChangeEvent } from "@mui/material/Select";
import Stack from "@mui/material/Stack";
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

import { usersApi, type UserSummary } from "@/features/users/usersApi";
import { ROLES, type RoleCode } from "@/types/roles";
import { ApiRequestError } from "@/services/apiClient";

const STATUS_COLORS: Record<UserSummary["status"], "success" | "default" | "error"> = {
  ACTIVE: "success",
  INACTIVE: "default",
  LOCKED: "error",
};

interface FormState {
  name: string;
  email: string;
  password: string;
  roleCode: RoleCode;
}

const emptyForm: FormState = { name: "", email: "", password: "", roleCode: ROLES.RECEPTIONIST };

/**
 * Minimal staff/user management: list + create only, both backed by
 * endpoints that already existed (GET /users, POST /auth/register) —
 * there was previously no UI path to create a staff account at all.
 * Role reassignment and activate/deactivate are deliberately not
 * offered here: no backend endpoint supports changing a user's role or
 * status after creation (UserRepository has no such method), and adding
 * one is out of scope for this remediation pass (no new API contracts).
 */
export default function UsersPage() {
  const [roleFilter, setRoleFilter] = useState<RoleCode | "">("");
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(20);

  const [items, setItems] = useState<UserSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const load = () => {
    setLoading(true);
    setError(null);
    usersApi
      .list({ role: roleFilter || undefined, page: page + 1, pageSize })
      .then((result) => {
        setItems(result.items);
        setTotal(result.total);
      })
      .catch((err: unknown) => setError(err instanceof ApiRequestError ? err.message : "Failed to load users."))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    setPage(0);
  }, [roleFilter]);

  useEffect(load, [roleFilter, page, pageSize]);

  const handleCreate = async () => {
    setSaving(true);
    setFormError(null);
    try {
      await usersApi.create(form);
      setDialogOpen(false);
      setForm(emptyForm);
      load();
    } catch (err) {
      setFormError(err instanceof ApiRequestError ? err.message : "Failed to create user.");
    } finally {
      setSaving(false);
    }
  };

  const canSubmit = form.name.trim().length >= 2 && form.email.trim().length > 0 && form.password.length >= 8;

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          Staff / Users
        </Typography>
        <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => setDialogOpen(true)}>
          Add User
        </Button>
      </Stack>

      <Alert severity="info" sx={{ mb: 2 }}>
        Role changes and activate/deactivate are not available in this build yet — accounts are created here but
        must otherwise be managed directly until that capability is added.
      </Alert>

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <FormControl sx={{ minWidth: 220 }}>
          <InputLabel id="filter-role-label">Role</InputLabel>
          <Select
            labelId="filter-role-label"
            label="Role"
            value={roleFilter}
            onChange={(e: SelectChangeEvent) => setRoleFilter(e.target.value as RoleCode | "")}
          >
            <MenuItem value="">All roles</MenuItem>
            {Object.values(ROLES).map((r) => (
              <MenuItem key={r} value={r}>
                {r}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
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
                <TableCell>Email</TableCell>
                <TableCell>Role</TableCell>
                <TableCell>Status</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading && (
                <TableRow>
                  <TableCell colSpan={4} align="center" sx={{ py: 6 }}>
                    <CircularProgress size={28} />
                  </TableCell>
                </TableRow>
              )}
              {!loading && items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} align="center" sx={{ py: 6 }}>
                    <Typography color="text.secondary">No users found.</Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading &&
                items.map((user) => (
                  <TableRow key={user.id}>
                    <TableCell>{user.name}</TableCell>
                    <TableCell>{user.email}</TableCell>
                    <TableCell>
                      <Chip size="small" label={user.role} variant="outlined" />
                    </TableCell>
                    <TableCell>
                      <Chip size="small" label={user.status} color={STATUS_COLORS[user.status]} />
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
        <DialogTitle>Add User</DialogTitle>
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
                label="Email"
                type="email"
                value={form.email}
                onChange={(e) => setForm((p) => ({ ...p, email: e.target.value }))}
                fullWidth
                required
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Temporary Password"
                type="password"
                value={form.password}
                onChange={(e) => setForm((p) => ({ ...p, password: e.target.value }))}
                fullWidth
                required
                helperText="At least 8 characters"
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth required>
                <InputLabel id="create-role-label">Role</InputLabel>
                <Select
                  labelId="create-role-label"
                  label="Role"
                  value={form.roleCode}
                  onChange={(e: SelectChangeEvent) => setForm((p) => ({ ...p, roleCode: e.target.value as RoleCode }))}
                >
                  {Object.values(ROLES).map((r) => (
                    <MenuItem key={r} value={r}>
                      {r}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)} disabled={saving}>
            Cancel
          </Button>
          <Button variant="contained" onClick={handleCreate} disabled={saving || !canSubmit}>
            {saving ? <CircularProgress size={20} color="inherit" /> : "Create User"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

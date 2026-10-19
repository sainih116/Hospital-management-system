import AddRoundedIcon from "@mui/icons-material/AddRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import FormControl from "@mui/material/FormControl";
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
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import AsyncAutocomplete, { type AsyncOption } from "@/components/common/AsyncAutocomplete";
import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { patientsApi } from "@/features/patients/patientsApi";
import { prescriptionsApi, type Prescription, type PrescriptionStatus } from "@/features/prescriptions/prescriptionsApi";
import { ApiRequestError } from "@/services/apiClient";

export default function PrescriptionsListPage() {
  const navigate = useNavigate();
  const canWrite = useHasPermission(PERMISSIONS.PRESCRIPTION_WRITE);

  const [patientFilter, setPatientFilter] = useState<AsyncOption | null>(null);
  const [status, setStatus] = useState<PrescriptionStatus | "">("");
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(20);

  const [items, setItems] = useState<Prescription[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setPage(0);
  }, [patientFilter, status]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    prescriptionsApi
      .search({ patientId: patientFilter?.id, status: status || undefined, page: page + 1, pageSize })
      .then((result) => {
        if (cancelled) return;
        setItems(result.items);
        setTotal(result.total);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof ApiRequestError ? err.message : "Failed to load prescriptions.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [patientFilter, status, page, pageSize]);

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          Prescriptions
        </Typography>
        {canWrite && (
          <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => navigate("/prescriptions/new")}>
            New Prescription
          </Button>
        )}
      </Stack>

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Stack direction={{ xs: "column", md: "row" }} spacing={2}>
          <Box sx={{ minWidth: 260, flexGrow: 1 }}>
            <AsyncAutocomplete
              label="Filter by patient"
              value={patientFilter}
              onChange={setPatientFilter}
              fetchOptions={async (query) => {
                const result = await patientsApi.search({ query, pageSize: 10 });
                return result.items.map((p) => ({ id: p.id, label: `${p.name} (${p.patientCode})` }));
              }}
            />
          </Box>
          <FormControl sx={{ minWidth: 160 }}>
            <InputLabel id="filter-status-label">Status</InputLabel>
            <Select
              labelId="filter-status-label"
              label="Status"
              value={status}
              onChange={(e: SelectChangeEvent) => setStatus(e.target.value as PrescriptionStatus | "")}
            >
              <MenuItem value="">All</MenuItem>
              <MenuItem value="ACTIVE">Active</MenuItem>
              <MenuItem value="CANCELLED">Cancelled</MenuItem>
            </Select>
          </FormControl>
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
                <TableCell>Issued</TableCell>
                <TableCell>Patient</TableCell>
                <TableCell>Doctor</TableCell>
                <TableCell>Medicines</TableCell>
                <TableCell>Status</TableCell>
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
              {!loading && items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} align="center" sx={{ py: 6 }}>
                    <Typography color="text.secondary">No prescriptions found.</Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading &&
                items.map((prescription) => (
                  <TableRow
                    key={prescription.id}
                    hover
                    sx={{ cursor: "pointer" }}
                    onClick={() => navigate(`/prescriptions/${prescription.id}`)}
                  >
                    <TableCell>{new Date(prescription.issuedAt).toLocaleDateString()}</TableCell>
                    <TableCell>{prescription.patientName}</TableCell>
                    <TableCell>{prescription.doctorName}</TableCell>
                    <TableCell>{prescription.items.map((i) => i.medicineName).join(", ")}</TableCell>
                    <TableCell>
                      <Chip size="small" label={prescription.status} color={prescription.status === "ACTIVE" ? "success" : "default"} />
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
    </Box>
  );
}

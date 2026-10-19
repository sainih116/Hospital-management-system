import AddRoundedIcon from "@mui/icons-material/AddRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Paper from "@mui/material/Paper";
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
import { medicalRecordsApi, type MedicalRecord } from "@/features/medicalrecords/medicalRecordsApi";
import { patientsApi } from "@/features/patients/patientsApi";
import { ApiRequestError } from "@/services/apiClient";

export default function MedicalRecordsListPage() {
  const navigate = useNavigate();
  const canWrite = useHasPermission(PERMISSIONS.EMR_WRITE);

  const [patientFilter, setPatientFilter] = useState<AsyncOption | null>(null);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(20);

  const [items, setItems] = useState<MedicalRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setPage(0);
  }, [patientFilter]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    medicalRecordsApi
      .search({ patientId: patientFilter?.id, page: page + 1, pageSize })
      .then((result) => {
        if (cancelled) return;
        setItems(result.items);
        setTotal(result.total);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof ApiRequestError ? err.message : "Failed to load medical records.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [patientFilter, page, pageSize]);

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          Medical Records
        </Typography>
        {canWrite && (
          <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => navigate("/emr/new")}>
            New Clinical Note
          </Button>
        )}
      </Stack>

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Box sx={{ maxWidth: 400 }}>
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
                <TableCell>Date</TableCell>
                <TableCell>Patient</TableCell>
                <TableCell>Doctor</TableCell>
                <TableCell>Diagnosis</TableCell>
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
                    <Typography color="text.secondary">
                      {patientFilter ? "No records found for this patient." : "No medical records yet."}
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading &&
                items.map((record) => (
                  <TableRow key={record.id} hover sx={{ cursor: "pointer" }} onClick={() => navigate(`/emr/${record.id}`)}>
                    <TableCell>{new Date(record.createdAt).toLocaleDateString()}</TableCell>
                    <TableCell>{record.patientName}</TableCell>
                    <TableCell>{record.doctorName}</TableCell>
                    <TableCell>{record.diagnosis}</TableCell>
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

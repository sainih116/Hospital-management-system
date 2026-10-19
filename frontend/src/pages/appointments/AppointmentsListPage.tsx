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
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { appointmentsApi, type Appointment, type AppointmentStatus } from "@/features/appointments/appointmentsApi";
import { STATUS_COLORS, STATUS_LABELS } from "@/features/appointments/appointmentStateMachine";
import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import type { Doctor } from "@/features/doctors/doctorsApi";
import { doctorsApi } from "@/features/doctors/doctorsApi";
import { ApiRequestError } from "@/services/apiClient";

const ALL_STATUSES = Object.keys(STATUS_LABELS) as AppointmentStatus[];

export default function AppointmentsListPage() {
  const navigate = useNavigate();
  const canBook = useHasPermission(PERMISSIONS.APPOINTMENT_MANAGE);

  const [doctors, setDoctors] = useState<Doctor[]>([]);
  const [doctorId, setDoctorId] = useState("");
  const [status, setStatus] = useState<AppointmentStatus | "">("");
  const [date, setDate] = useState("");
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(20);

  const [items, setItems] = useState<Appointment[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    doctorsApi.search({ pageSize: 100 }).then((r) => setDoctors(r.items)).catch(() => undefined);
  }, []);

  useEffect(() => {
    setPage(0);
  }, [doctorId, status, date]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    let from: string | undefined;
    let to: string | undefined;
    if (date) {
      from = new Date(`${date}T00:00:00`).toISOString();
      to = new Date(`${date}T23:59:59`).toISOString();
    }

    appointmentsApi
      .search({
        doctorId: doctorId || undefined,
        status: status || undefined,
        from,
        to,
        page: page + 1,
        pageSize,
      })
      .then((result) => {
        if (cancelled) return;
        setItems(result.items);
        setTotal(result.total);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof ApiRequestError ? err.message : "Failed to load appointments.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [doctorId, status, date, page, pageSize]);

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          Appointments
        </Typography>
        {canBook && (
          <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => navigate("/appointments/new")}>
            Book Appointment
          </Button>
        )}
      </Stack>

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Stack direction={{ xs: "column", md: "row" }} spacing={2}>
          <FormControl sx={{ minWidth: 200 }}>
            <InputLabel id="filter-doctor-label">Doctor</InputLabel>
            <Select labelId="filter-doctor-label" label="Doctor" value={doctorId} onChange={(e: SelectChangeEvent) => setDoctorId(e.target.value)}>
              <MenuItem value="">All</MenuItem>
              {doctors.map((d) => (
                <MenuItem key={d.id} value={d.id}>
                  {d.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControl sx={{ minWidth: 160 }}>
            <InputLabel id="filter-status-label">Status</InputLabel>
            <Select
              labelId="filter-status-label"
              label="Status"
              value={status}
              onChange={(e: SelectChangeEvent) => setStatus(e.target.value as AppointmentStatus | "")}
            >
              <MenuItem value="">All</MenuItem>
              {ALL_STATUSES.map((s) => (
                <MenuItem key={s} value={s}>
                  {STATUS_LABELS[s]}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <TextField
            label="Date"
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            slotProps={{ inputLabel: { shrink: true } }}
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
                <TableCell>Date &amp; Time</TableCell>
                <TableCell>Patient</TableCell>
                <TableCell>Doctor</TableCell>
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
                    <Typography color="text.secondary">No appointments match your filters.</Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading &&
                items.map((appt) => (
                  <TableRow key={appt.id} hover sx={{ cursor: "pointer" }} onClick={() => navigate(`/appointments/${appt.id}`)}>
                    <TableCell>{new Date(appt.startAt).toLocaleString()}</TableCell>
                    <TableCell>{appt.patientName}</TableCell>
                    <TableCell>{appt.doctorName}</TableCell>
                    <TableCell>
                      <Chip size="small" label={STATUS_LABELS[appt.status]} color={STATUS_COLORS[appt.status]} />
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

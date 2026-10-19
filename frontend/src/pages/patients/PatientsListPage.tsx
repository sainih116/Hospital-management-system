import AddRoundedIcon from "@mui/icons-material/AddRounded";
import SearchRoundedIcon from "@mui/icons-material/SearchRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import FormControl from "@mui/material/FormControl";
import InputAdornment from "@mui/material/InputAdornment";
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
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { BLOOD_GROUPS, GENDERS, PATIENT_TYPES } from "@/features/patients/patientSchema";
import { patientsApi, type Patient } from "@/features/patients/patientsApi";
import { ApiRequestError } from "@/services/apiClient";

export default function PatientsListPage() {
  const navigate = useNavigate();
  const canCreate = useHasPermission(PERMISSIONS.PATIENT_CREATE);

  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [gender, setGender] = useState("");
  const [bloodGroup, setBloodGroup] = useState("");
  const [type, setType] = useState("");
  const [page, setPage] = useState(0); // zero-based for MUI TablePagination
  const [pageSize, setPageSize] = useState(20);

  const [items, setItems] = useState<Patient[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query), 350);
    return () => clearTimeout(timer);
  }, [query]);

  useEffect(() => {
    setPage(0);
  }, [debouncedQuery, gender, bloodGroup, type]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    patientsApi
      .search({
        query: debouncedQuery || undefined,
        gender: (gender || undefined) as Patient["gender"] | undefined,
        bloodGroup: bloodGroup || undefined,
        type: (type || undefined) as Patient["type"] | undefined,
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
        setError(err instanceof ApiRequestError ? err.message : "Failed to load patients.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [debouncedQuery, gender, bloodGroup, type, page, pageSize]);

  const hasFilters = useMemo(() => Boolean(gender || bloodGroup || type), [gender, bloodGroup, type]);

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          Patients
        </Typography>
        {canCreate && (
          <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => navigate("/patients/new")}>
            New Patient
          </Button>
        )}
      </Stack>

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Stack direction={{ xs: "column", md: "row" }} spacing={2}>
          <TextField
            fullWidth
            placeholder="Search by name, patient code or phone"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            slotProps={{
              input: {
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchRoundedIcon fontSize="small" />
                  </InputAdornment>
                ),
              },
            }}
          />
          <FormControl sx={{ minWidth: 140 }}>
            <InputLabel id="filter-gender-label">Gender</InputLabel>
            <Select
              labelId="filter-gender-label"
              label="Gender"
              value={gender}
              onChange={(e: SelectChangeEvent) => setGender(e.target.value)}
            >
              <MenuItem value="">All</MenuItem>
              {GENDERS.map((g) => (
                <MenuItem key={g} value={g}>
                  {g}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControl sx={{ minWidth: 140 }}>
            <InputLabel id="filter-blood-label">Blood Group</InputLabel>
            <Select
              labelId="filter-blood-label"
              label="Blood Group"
              value={bloodGroup}
              onChange={(e: SelectChangeEvent) => setBloodGroup(e.target.value)}
            >
              <MenuItem value="">All</MenuItem>
              {BLOOD_GROUPS.map((bg) => (
                <MenuItem key={bg} value={bg}>
                  {bg}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControl sx={{ minWidth: 140 }}>
            <InputLabel id="filter-type-label">Type</InputLabel>
            <Select
              labelId="filter-type-label"
              label="Type"
              value={type}
              onChange={(e: SelectChangeEvent) => setType(e.target.value)}
            >
              <MenuItem value="">All</MenuItem>
              {PATIENT_TYPES.map((t) => (
                <MenuItem key={t} value={t}>
                  {t}
                </MenuItem>
              ))}
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
                <TableCell>Patient Code</TableCell>
                <TableCell>Name</TableCell>
                <TableCell>Gender</TableCell>
                <TableCell>Phone</TableCell>
                <TableCell>Type</TableCell>
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
                    <Typography color="text.secondary">
                      {hasFilters || debouncedQuery ? "No patients match your filters." : "No patients registered yet."}
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading &&
                items.map((patient) => (
                  <TableRow
                    key={patient.id}
                    hover
                    sx={{ cursor: "pointer" }}
                    onClick={() => navigate(`/patients/${patient.id}`)}
                  >
                    <TableCell>{patient.patientCode}</TableCell>
                    <TableCell>{patient.name}</TableCell>
                    <TableCell>{patient.gender}</TableCell>
                    <TableCell>{patient.phone}</TableCell>
                    <TableCell>{patient.type}</TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        label={patient.status}
                        color={patient.status === "ACTIVE" ? "success" : "default"}
                      />
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

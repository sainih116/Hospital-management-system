import AddRoundedIcon from "@mui/icons-material/AddRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select, { type SelectChangeEvent } from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
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
import { billingApi, type Bill, type BillStatus } from "@/features/billing/billingApi";
import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { patientsApi } from "@/features/patients/patientsApi";
import { ApiRequestError } from "@/services/apiClient";

const STATUS_LABELS: Record<BillStatus, string> = {
  UNPAID: "Unpaid",
  PARTIALLY_PAID: "Partially Paid",
  PAID: "Paid",
  CANCELLED: "Cancelled",
};

const STATUS_COLORS: Record<BillStatus, "default" | "warning" | "success" | "error"> = {
  UNPAID: "error",
  PARTIALLY_PAID: "warning",
  PAID: "success",
  CANCELLED: "default",
};

export default function BillingListPage() {
  const navigate = useNavigate();
  const canManage = useHasPermission(PERMISSIONS.BILLING_MANAGE);

  const [patientFilter, setPatientFilter] = useState<AsyncOption | null>(null);
  const [status, setStatus] = useState<BillStatus | "">("");
  const [outstandingOnly, setOutstandingOnly] = useState(false);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(20);

  const [items, setItems] = useState<Bill[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setPage(0);
  }, [patientFilter, status, outstandingOnly]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    billingApi
      .search({
        patientId: patientFilter?.id,
        status: outstandingOnly ? undefined : status || undefined,
        outstandingOnly: outstandingOnly || undefined,
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
        setError(err instanceof ApiRequestError ? err.message : "Failed to load bills.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [patientFilter, status, outstandingOnly, page, pageSize]);

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          Billing
        </Typography>
        {canManage && (
          <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => navigate("/billing/new")}>
            New Bill
          </Button>
        )}
      </Stack>

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Stack direction={{ xs: "column", md: "row" }} spacing={2} alignItems={{ md: "center" }}>
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
          <FormControl sx={{ minWidth: 200 }} disabled={outstandingOnly}>
            <InputLabel id="filter-status-label">Status</InputLabel>
            <Select
              labelId="filter-status-label"
              label="Status"
              value={status}
              onChange={(e: SelectChangeEvent) => setStatus(e.target.value as BillStatus | "")}
            >
              <MenuItem value="">All</MenuItem>
              {(Object.keys(STATUS_LABELS) as BillStatus[]).map((s) => (
                <MenuItem key={s} value={s}>
                  {STATUS_LABELS[s]}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControlLabel
            control={<Switch checked={outstandingOnly} onChange={(e) => setOutstandingOnly(e.target.checked)} />}
            label="Outstanding only"
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
                <TableCell>Bill No.</TableCell>
                <TableCell>Patient</TableCell>
                <TableCell align="right">Total</TableCell>
                <TableCell align="right">Paid</TableCell>
                <TableCell align="right">Balance</TableCell>
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
                    <Typography color="text.secondary">No bills found.</Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading &&
                items.map((bill) => (
                  <TableRow key={bill.id} hover sx={{ cursor: "pointer" }} onClick={() => navigate(`/billing/${bill.id}`)}>
                    <TableCell>{bill.billNo}</TableCell>
                    <TableCell>{bill.patientName}</TableCell>
                    <TableCell align="right">{bill.totalAmount.toFixed(2)}</TableCell>
                    <TableCell align="right">{bill.amountPaid.toFixed(2)}</TableCell>
                    <TableCell align="right">{bill.balance.toFixed(2)}</TableCell>
                    <TableCell>
                      <Chip size="small" label={STATUS_LABELS[bill.status]} color={STATUS_COLORS[bill.status]} />
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

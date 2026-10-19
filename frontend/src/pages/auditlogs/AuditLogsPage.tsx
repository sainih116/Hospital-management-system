import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Chip from "@mui/material/Chip";
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
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";

import { auditLogsApi, type AuditLog } from "@/features/auditlogs/auditLogsApi";
import { ApiRequestError } from "@/services/apiClient";

export default function AuditLogsPage() {
  const [entityType, setEntityType] = useState("");
  const [entityId, setEntityId] = useState("");
  const [action, setAction] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(20);

  const [items, setItems] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setPage(0);
  }, [entityType, entityId, action, from, to]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    auditLogsApi
      .search({
        entityType: entityType || undefined,
        entityId: entityId || undefined,
        action: action || undefined,
        from: from ? new Date(from).toISOString() : undefined,
        to: to ? new Date(to + "T23:59:59").toISOString() : undefined,
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
        setError(err instanceof ApiRequestError ? err.message : "Failed to load audit logs.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [entityType, entityId, action, from, to, page, pageSize]);

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} sx={{ mb: 3 }}>
        Audit Logs
      </Typography>

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Stack direction={{ xs: "column", md: "row" }} spacing={2} flexWrap="wrap">
          <TextField label="Entity Type" value={entityType} onChange={(e) => setEntityType(e.target.value)} sx={{ minWidth: 180 }} helperText="e.g. patient, billing" />
          <TextField label="Entity ID" value={entityId} onChange={(e) => setEntityId(e.target.value)} sx={{ minWidth: 220 }} />
          <TextField label="Action" value={action} onChange={(e) => setAction(e.target.value)} sx={{ minWidth: 200 }} helperText="e.g. BILL_CREATED" />
          <TextField
            label="From"
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            slotProps={{ inputLabel: { shrink: true } }}
          />
          <TextField
            label="To"
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
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
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Date/Time</TableCell>
                <TableCell>Actor</TableCell>
                <TableCell>Action</TableCell>
                <TableCell>Entity</TableCell>
                <TableCell>Details</TableCell>
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
                    <Typography color="text.secondary">No audit log entries found.</Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading &&
                items.map((log) => (
                  <TableRow key={log.id} hover>
                    <TableCell sx={{ whiteSpace: "nowrap" }}>{new Date(log.createdAt).toLocaleString()}</TableCell>
                    <TableCell>
                      <Stack spacing={0.25}>
                        <Typography variant="body2">{log.actorEmail || "—"}</Typography>
                        {log.actorRole && <Chip size="small" label={log.actorRole} variant="outlined" sx={{ width: "fit-content" }} />}
                      </Stack>
                    </TableCell>
                    <TableCell>{log.action}</TableCell>
                    <TableCell>
                      {log.entityType}
                      {log.entityId && (
                        <Typography component="span" variant="caption" color="text.secondary" sx={{ display: "block" }}>
                          {log.entityId}
                        </Typography>
                      )}
                    </TableCell>
                    <TableCell>
                      {log.metadata && Object.keys(log.metadata).length > 0 ? (
                        <Typography
                          variant="caption"
                          component="pre"
                          title={JSON.stringify(log.metadata, null, 2)}
                          sx={{ m: 0, maxWidth: 320, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}
                        >
                          {JSON.stringify(log.metadata)}
                        </Typography>
                      ) : (
                        "—"
                      )}
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

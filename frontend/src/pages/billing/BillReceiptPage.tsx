import PrintRoundedIcon from "@mui/icons-material/PrintRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import { billingApi, type Bill } from "@/features/billing/billingApi";
import { ApiRequestError } from "@/services/apiClient";

interface FieldProps {
  label: string;
  value?: string;
}

function Field({ label, value }: FieldProps) {
  return (
    <Box>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      <Typography variant="body1" fontWeight={500}>
        {value || "—"}
      </Typography>
    </Box>
  );
}

/**
 * A printable receipt view (browser print-to-PDF), the same pattern
 * LabReportPage uses — no server-rendered PDF binary, no new backend
 * endpoint. GET /billing/bills/{id} already returns everything a receipt
 * needs (items, totals, and the full payment history), so this page is a
 * pure presentation layer over data BillDetailPage already fetches.
 */
export default function BillReceiptPage() {
  const { id } = useParams<{ id: string }>();

  const [bill, setBill] = useState<Bill | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    billingApi
      .getById(id)
      .then(setBill)
      .catch((err: unknown) => setError(err instanceof ApiRequestError ? err.message : "Bill not found."))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !bill) {
    return <Alert severity="error">{error ?? "Bill not found."}</Alert>;
  }

  return (
    <Box>
      <Stack direction="row" justifyContent="flex-end" sx={{ mb: 2, "@media print": { display: "none" } }}>
        <Button variant="contained" startIcon={<PrintRoundedIcon />} onClick={() => window.print()}>
          Print
        </Button>
      </Stack>

      <Paper variant="outlined" sx={{ p: 4, maxWidth: 800, mx: "auto" }}>
        <Stack direction="row" justifyContent="space-between" alignItems="flex-start" sx={{ mb: 3 }}>
          <Stack spacing={0.5}>
            <Typography variant="h5" fontWeight={700}>
              Payment Receipt
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Bill No: {bill.billNo}
            </Typography>
          </Stack>
          <Chip label={bill.status.replace("_", " ")} color={bill.status === "PAID" ? "success" : bill.status === "CANCELLED" ? "default" : "warning"} />
        </Stack>

        <Grid container spacing={3} sx={{ mb: 3 }}>
          <Grid item xs={6}>
            <Field label="Patient" value={bill.patientName} />
          </Grid>
          <Grid item xs={6}>
            <Field label="Bill Date" value={new Date(bill.createdAt).toLocaleString()} />
          </Grid>
        </Grid>

        <Divider sx={{ mb: 3 }} />

        <Typography variant="subtitle1" fontWeight={600} gutterBottom>
          Charges
        </Typography>
        <Table size="small" sx={{ mb: 3 }}>
          <TableHead>
            <TableRow>
              <TableCell>Description</TableCell>
              <TableCell>Category</TableCell>
              <TableCell align="right">Qty</TableCell>
              <TableCell align="right">Unit Price</TableCell>
              <TableCell align="right">Amount</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {bill.items.map((item, index) => (
              <TableRow key={index}>
                <TableCell>{item.description}</TableCell>
                <TableCell>{item.category}</TableCell>
                <TableCell align="right">{item.quantity}</TableCell>
                <TableCell align="right">{item.unitPrice.toFixed(2)}</TableCell>
                <TableCell align="right">{item.amount.toFixed(2)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>

        <Box sx={{ display: "flex", justifyContent: "flex-end", mb: 3 }}>
          <Stack spacing={0.5} sx={{ minWidth: 260 }}>
            <Stack direction="row" justifyContent="space-between">
              <Typography variant="body2" color="text.secondary">
                Subtotal
              </Typography>
              <Typography variant="body2">{bill.subtotal.toFixed(2)}</Typography>
            </Stack>
            <Stack direction="row" justifyContent="space-between">
              <Typography variant="body2" color="text.secondary">
                Discount
              </Typography>
              <Typography variant="body2">-{bill.discount.toFixed(2)}</Typography>
            </Stack>
            <Stack direction="row" justifyContent="space-between">
              <Typography variant="body2" color="text.secondary">
                Tax
              </Typography>
              <Typography variant="body2">+{bill.tax.toFixed(2)}</Typography>
            </Stack>
            <Divider />
            <Stack direction="row" justifyContent="space-between">
              <Typography variant="subtitle2" fontWeight={700}>
                Total
              </Typography>
              <Typography variant="subtitle2" fontWeight={700}>
                {bill.totalAmount.toFixed(2)}
              </Typography>
            </Stack>
            <Stack direction="row" justifyContent="space-between">
              <Typography variant="body2" color="text.secondary">
                Amount Paid
              </Typography>
              <Typography variant="body2">{bill.amountPaid.toFixed(2)}</Typography>
            </Stack>
            <Stack direction="row" justifyContent="space-between">
              <Typography variant="subtitle2" fontWeight={700}>
                Balance Due
              </Typography>
              <Typography variant="subtitle2" fontWeight={700} color={bill.balance > 0 ? "error.main" : "success.main"}>
                {bill.balance.toFixed(2)}
              </Typography>
            </Stack>
          </Stack>
        </Box>

        <Divider sx={{ mb: 3 }} />

        <Typography variant="subtitle1" fontWeight={600} gutterBottom>
          Payment History
        </Typography>
        {!bill.payments || bill.payments.length === 0 ? (
          <Typography variant="body2" color="text.secondary">
            No payments recorded yet.
          </Typography>
        ) : (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Date</TableCell>
                <TableCell>Method</TableCell>
                <TableCell>Reference</TableCell>
                <TableCell>Received By</TableCell>
                <TableCell align="right">Amount</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {bill.payments.map((payment) => (
                <TableRow key={payment.id}>
                  <TableCell>{new Date(payment.paidAt).toLocaleString()}</TableCell>
                  <TableCell>{payment.method.replace("_", " ")}</TableCell>
                  <TableCell>{payment.referenceNo || "—"}</TableCell>
                  <TableCell>{payment.receivedByName || "—"}</TableCell>
                  <TableCell align="right">{payment.amount.toFixed(2)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Paper>
    </Box>
  );
}

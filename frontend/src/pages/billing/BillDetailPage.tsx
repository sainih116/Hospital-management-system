import CancelRoundedIcon from "@mui/icons-material/CancelRounded";
import PaymentsRoundedIcon from "@mui/icons-material/PaymentsRounded";
import PrintRoundedIcon from "@mui/icons-material/PrintRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import Divider from "@mui/material/Divider";
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
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { billingApi, type Bill, type BillStatus, type PaymentMethod } from "@/features/billing/billingApi";
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

const PAYMENT_METHODS: PaymentMethod[] = ["CASH", "CARD", "UPI", "BANK_TRANSFER", "INSURANCE"];

export default function BillDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const canManage = useHasPermission(PERMISSIONS.BILLING_MANAGE);

  const [bill, setBill] = useState<Bill | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [paymentDialogOpen, setPaymentDialogOpen] = useState(false);
  const [amount, setAmount] = useState("");
  const [method, setMethod] = useState<PaymentMethod>("CASH");
  const [referenceNo, setReferenceNo] = useState("");
  const [paying, setPaying] = useState(false);
  const [paymentError, setPaymentError] = useState<string | null>(null);

  const [cancelDialogOpen, setCancelDialogOpen] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState<string | null>(null);

  const load = () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    billingApi
      .getById(id)
      .then(setBill)
      .catch((err: unknown) => setError(err instanceof ApiRequestError ? err.message : "Failed to load bill."))
      .finally(() => setLoading(false));
  };

  useEffect(load, [id]);

  const handleRecordPayment = async () => {
    if (!id) return;
    setPaying(true);
    setPaymentError(null);
    try {
      const updated = await billingApi.recordPayment(id, {
        amount: Number(amount) || 0,
        method,
        referenceNo: referenceNo || undefined,
      });
      setBill(updated);
      setPaymentDialogOpen(false);
      setAmount("");
      setReferenceNo("");
      setMethod("CASH");
    } catch (err) {
      setPaymentError(err instanceof ApiRequestError ? err.message : "Failed to record payment.");
    } finally {
      setPaying(false);
    }
  };

  const handleCancel = async () => {
    if (!id) return;
    setCancelling(true);
    setCancelError(null);
    try {
      const updated = await billingApi.cancel(id);
      setBill(updated);
      setCancelDialogOpen(false);
    } catch (err) {
      setCancelError(err instanceof ApiRequestError ? err.message : "Failed to cancel bill.");
    } finally {
      setCancelling(false);
    }
  };

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

  const canRecordPayment = canManage && (bill.status === "UNPAID" || bill.status === "PARTIALLY_PAID");
  const canCancel = canManage && bill.amountPaid === 0 && bill.status !== "CANCELLED";

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>
            {bill.billNo}
          </Typography>
          <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
            <Typography variant="body2" color="text.secondary">
              {bill.patientName} · {new Date(bill.createdAt).toLocaleDateString()}
            </Typography>
            <Chip size="small" label={STATUS_LABELS[bill.status]} color={STATUS_COLORS[bill.status]} />
          </Stack>
        </Box>
        <Stack direction="row" spacing={2}>
          <Button variant="outlined" startIcon={<PrintRoundedIcon />} onClick={() => navigate(`/billing/${bill.id}/receipt`)}>
            Print Receipt
          </Button>
          {canRecordPayment && (
            <Button variant="contained" startIcon={<PaymentsRoundedIcon />} onClick={() => setPaymentDialogOpen(true)}>
              Record Payment
            </Button>
          )}
          {canCancel && (
            <Button color="error" variant="outlined" startIcon={<CancelRoundedIcon />} onClick={() => setCancelDialogOpen(true)}>
              Cancel Bill
            </Button>
          )}
        </Stack>
      </Stack>

      <Paper variant="outlined" sx={{ mb: 3 }}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Description</TableCell>
              <TableCell>Category</TableCell>
              <TableCell align="right">Quantity</TableCell>
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

        <Box sx={{ p: 2, display: "flex", justifyContent: "flex-end" }}>
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
            <Divider sx={{ my: 0.5 }} />
            <Stack direction="row" justifyContent="space-between">
              <Typography variant="subtitle1" fontWeight={700}>
                Total
              </Typography>
              <Typography variant="subtitle1" fontWeight={700}>
                {bill.totalAmount.toFixed(2)}
              </Typography>
            </Stack>
            <Stack direction="row" justifyContent="space-between">
              <Typography variant="body2" color="text.secondary">
                Paid
              </Typography>
              <Typography variant="body2">{bill.amountPaid.toFixed(2)}</Typography>
            </Stack>
            <Stack direction="row" justifyContent="space-between">
              <Typography variant="subtitle1" fontWeight={700} color={bill.balance > 0 ? "error.main" : "success.main"}>
                Balance
              </Typography>
              <Typography variant="subtitle1" fontWeight={700} color={bill.balance > 0 ? "error.main" : "success.main"}>
                {bill.balance.toFixed(2)}
              </Typography>
            </Stack>
          </Stack>
        </Box>
      </Paper>

      <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1 }}>
        Payment History
      </Typography>
      <Paper variant="outlined">
        <Table>
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
            {(!bill.payments || bill.payments.length === 0) && (
              <TableRow>
                <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">No payments recorded yet.</Typography>
                </TableCell>
              </TableRow>
            )}
            {bill.payments?.map((payment) => (
              <TableRow key={payment.id}>
                <TableCell>{new Date(payment.paidAt).toLocaleString()}</TableCell>
                <TableCell>{payment.method}</TableCell>
                <TableCell>{payment.referenceNo || "—"}</TableCell>
                <TableCell>{payment.receivedByName || "—"}</TableCell>
                <TableCell align="right">{payment.amount.toFixed(2)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Paper>

      <Dialog open={paymentDialogOpen} onClose={() => setPaymentDialogOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Record Payment</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 0.5 }}>
            {paymentError && (
              <Grid item xs={12}>
                <Alert severity="error">{paymentError}</Alert>
              </Grid>
            )}
            <Grid item xs={12}>
              <Alert severity="info">Outstanding balance: {bill.balance.toFixed(2)}</Alert>
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Amount"
                type="number"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                fullWidth
                autoFocus
                required
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth required>
                <InputLabel id="payment-method-label">Method</InputLabel>
                <Select
                  labelId="payment-method-label"
                  label="Method"
                  value={method}
                  onChange={(e: SelectChangeEvent) => setMethod(e.target.value as PaymentMethod)}
                >
                  {PAYMENT_METHODS.map((m) => (
                    <MenuItem key={m} value={m}>
                      {m}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12}>
              <TextField
                label="Reference No. (optional)"
                value={referenceNo}
                onChange={(e) => setReferenceNo(e.target.value)}
                fullWidth
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPaymentDialogOpen(false)} disabled={paying}>
            Cancel
          </Button>
          <Button variant="contained" onClick={handleRecordPayment} disabled={paying || !amount || Number(amount) <= 0}>
            {paying ? <CircularProgress size={20} color="inherit" /> : "Record Payment"}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={cancelDialogOpen} onClose={() => setCancelDialogOpen(false)}>
        <DialogTitle>Cancel bill?</DialogTitle>
        <DialogContent>
          <DialogContentText>This bill will be marked cancelled. This cannot be undone.</DialogContentText>
          {cancelError && (
            <Alert severity="error" sx={{ mt: 2 }}>
              {cancelError}
            </Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCancelDialogOpen(false)} disabled={cancelling}>
            Back
          </Button>
          <Button color="error" variant="contained" onClick={handleCancel} disabled={cancelling}>
            {cancelling ? <CircularProgress size={20} color="inherit" /> : "Cancel Bill"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

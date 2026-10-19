import PrintRoundedIcon from "@mui/icons-material/PrintRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import { labOrdersApi, type LabOrder } from "@/features/laborders/labOrdersApi";
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
 * A printable report view (browser print-to-PDF) rather than a
 * server-rendered binary — see the backend's GetReport doc comment for why.
 * The action bar is hidden via @media print so only the report content
 * appears in the printed/saved output.
 */
export default function LabReportPage() {
  const { id } = useParams<{ id: string }>();

  const [order, setOrder] = useState<LabOrder | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    labOrdersApi
      .getReport(id)
      .then((report) => setOrder(report.order))
      .catch((err: unknown) => setError(err instanceof ApiRequestError ? err.message : "Report not available."))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !order) {
    return <Alert severity="error">{error ?? "Report not found."}</Alert>;
  }

  return (
    <Box>
      <Stack direction="row" justifyContent="flex-end" sx={{ mb: 2, "@media print": { display: "none" } }}>
        <Button variant="contained" startIcon={<PrintRoundedIcon />} onClick={() => window.print()}>
          Print
        </Button>
      </Stack>

      <Paper variant="outlined" sx={{ p: 4, maxWidth: 800, mx: "auto" }}>
        <Stack spacing={0.5} sx={{ mb: 3 }}>
          <Typography variant="h5" fontWeight={700}>
            Laboratory Report
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Sample: {order.sampleCode}
          </Typography>
        </Stack>

        <Grid container spacing={3} sx={{ mb: 3 }}>
          <Grid item xs={6}>
            <Field label="Patient" value={order.patientName} />
          </Grid>
          <Grid item xs={6}>
            <Field label="Ordering Doctor" value={order.doctorName} />
          </Grid>
        </Grid>

        <Divider sx={{ mb: 3 }} />

        <Typography variant="subtitle1" fontWeight={600} gutterBottom>
          {order.testName} ({order.testCode})
        </Typography>
        <Grid container spacing={3} sx={{ mb: 3 }}>
          <Grid item xs={4}>
            <Field label="Result" value={order.result?.resultValue} />
          </Grid>
          <Grid item xs={4}>
            <Field label="Unit" value={order.unit} />
          </Grid>
          <Grid item xs={4}>
            <Field label="Normal Range" value={order.normalRange} />
          </Grid>
          <Grid item xs={4}>
            <Field label="Flag" value={order.result?.flag} />
          </Grid>
        </Grid>
        {order.result?.notes && (
          <Box sx={{ mb: 3 }}>
            <Field label="Notes" value={order.result.notes} />
          </Box>
        )}

        <Divider sx={{ mb: 3 }} />

        <Grid container spacing={3}>
          <Grid item xs={6}>
            <Field
              label="Entered By"
              value={order.result ? `${order.result.enteredByName} on ${new Date(order.result.enteredAt).toLocaleString()}` : undefined}
            />
          </Grid>
          <Grid item xs={6}>
            <Field
              label="Verified By"
              value={order.result?.verifiedAt ? `${order.result.verifiedByName} on ${new Date(order.result.verifiedAt).toLocaleString()}` : undefined}
            />
          </Grid>
        </Grid>
      </Paper>
    </Box>
  );
}

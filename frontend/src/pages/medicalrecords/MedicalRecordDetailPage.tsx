import EditRoundedIcon from "@mui/icons-material/EditRounded";
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
import { useNavigate, useParams } from "react-router-dom";

import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { medicalRecordsApi, type MedicalRecord } from "@/features/medicalrecords/medicalRecordsApi";
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
      <Typography variant="body1" sx={{ whiteSpace: "pre-wrap" }}>
        {value || "—"}
      </Typography>
    </Box>
  );
}

export default function MedicalRecordDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const canWrite = useHasPermission(PERMISSIONS.EMR_WRITE);

  const [record, setRecord] = useState<MedicalRecord | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    medicalRecordsApi
      .getById(id)
      .then((data) => {
        if (!cancelled) setRecord(data);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof ApiRequestError ? err.message : "Failed to load medical record.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !record) {
    return <Alert severity="error">{error ?? "Medical record not found."}</Alert>;
  }

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>
            {record.patientName}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Seen by {record.doctorName} on {new Date(record.createdAt).toLocaleString()}
          </Typography>
        </Box>
        {canWrite && (
          <Button variant="outlined" startIcon={<EditRoundedIcon />} onClick={() => navigate(`/emr/${record.id}/edit`)}>
            Amend
          </Button>
        )}
      </Stack>

      <Paper variant="outlined" sx={{ p: 3 }}>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Field label="Chief Complaint" value={record.complaint} />
          </Grid>
          <Grid item xs={12}>
            <Field label="History of Present Illness" value={record.hpi} />
          </Grid>
          <Grid item xs={12}>
            <Field label="Past / Family / Social History" value={record.history} />
          </Grid>
          <Grid item xs={12}>
            <Field label="Examination" value={record.examination} />
          </Grid>
          <Grid item xs={12}>
            <Divider />
          </Grid>
          <Grid item xs={12}>
            <Field label="Diagnosis" value={record.diagnosis} />
          </Grid>
          <Grid item xs={12}>
            <Field label="Treatment" value={record.treatment} />
          </Grid>
          <Grid item xs={12}>
            <Field label="Follow-up" value={record.followUp} />
          </Grid>
        </Grid>
      </Paper>
    </Box>
  );
}

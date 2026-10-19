import { zodResolver } from "@hookform/resolvers/zod";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";

import AsyncAutocomplete from "@/components/common/AsyncAutocomplete";
import { doctorsApi } from "@/features/doctors/doctorsApi";
import {
  emptyMedicalRecordFormValues,
  medicalRecordFormSchema,
  type MedicalRecordFormInput,
} from "@/features/medicalrecords/medicalRecordSchema";
import { medicalRecordsApi } from "@/features/medicalrecords/medicalRecordsApi";
import { patientsApi } from "@/features/patients/patientsApi";
import { ApiRequestError } from "@/services/apiClient";

export default function MedicalRecordFormPage() {
  const { id } = useParams<{ id: string }>();
  const isEditMode = Boolean(id);
  const navigate = useNavigate();

  const [loading, setLoading] = useState(isEditMode);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    control,
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<MedicalRecordFormInput>({
    resolver: zodResolver(medicalRecordFormSchema),
    defaultValues: emptyMedicalRecordFormValues,
  });

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    medicalRecordsApi
      .getById(id)
      .then((record) => {
        if (cancelled) return;
        reset({
          patientId: record.patientId,
          patientLabel: record.patientName,
          doctorId: record.doctorId,
          doctorLabel: record.doctorName,
          complaint: record.complaint ?? "",
          hpi: record.hpi ?? "",
          history: record.history ?? "",
          examination: record.examination ?? "",
          diagnosis: record.diagnosis,
          treatment: record.treatment ?? "",
          followUp: record.followUp ?? "",
        });
      })
      .catch((err: unknown) => {
        if (!cancelled) setLoadError(err instanceof ApiRequestError ? err.message : "Failed to load medical record.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id, reset]);

  const onSubmit = async (values: MedicalRecordFormInput) => {
    setSubmitError(null);
    try {
      if (isEditMode && id) {
        await medicalRecordsApi.update(id, {
          complaint: values.complaint || undefined,
          hpi: values.hpi || undefined,
          history: values.history || undefined,
          examination: values.examination || undefined,
          diagnosis: values.diagnosis,
          treatment: values.treatment || undefined,
          followUp: values.followUp || undefined,
        });
        navigate(`/emr/${id}`);
      } else {
        const created = await medicalRecordsApi.create({
          patientId: values.patientId,
          doctorId: values.doctorId,
          complaint: values.complaint || undefined,
          hpi: values.hpi || undefined,
          history: values.history || undefined,
          examination: values.examination || undefined,
          diagnosis: values.diagnosis,
          treatment: values.treatment || undefined,
          followUp: values.followUp || undefined,
        });
        navigate(`/emr/${created.id}`);
      }
    } catch (err) {
      setSubmitError(err instanceof ApiRequestError ? err.message : "Failed to save medical record.");
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (loadError) {
    return <Alert severity="error">{loadError}</Alert>;
  }

  const patientLabel = watch("patientLabel");
  const doctorLabel = watch("doctorLabel");

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} sx={{ mb: 3 }}>
        {isEditMode ? "Amend Clinical Note" : "New Clinical Note"}
      </Typography>

      {submitError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {submitError}
        </Alert>
      )}

      <Paper variant="outlined" sx={{ p: 3, maxWidth: 800 }}>
        <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
          {isEditMode ? (
            <Grid container spacing={2} sx={{ mb: 2 }}>
              <Grid item xs={12} sm={6}>
                <TextField label="Patient" fullWidth value={patientLabel} disabled />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField label="Doctor" fullWidth value={doctorLabel} disabled />
              </Grid>
            </Grid>
          ) : (
            <Grid container spacing={2} sx={{ mb: 2 }}>
              <Grid item xs={12} sm={6}>
                <AsyncAutocomplete
                  label="Patient"
                  required
                  value={patientLabel ? { id: watch("patientId"), label: patientLabel } : null}
                  onChange={(option) => {
                    setValue("patientId", option?.id ?? "");
                    setValue("patientLabel", option?.label ?? "");
                  }}
                  error={!!errors.patientId}
                  helperText={errors.patientId?.message}
                  fetchOptions={async (query) => {
                    const result = await patientsApi.search({ query, pageSize: 10 });
                    return result.items.map((p) => ({ id: p.id, label: `${p.name} (${p.patientCode})` }));
                  }}
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <AsyncAutocomplete
                  label="Doctor"
                  required
                  value={doctorLabel ? { id: watch("doctorId"), label: doctorLabel } : null}
                  onChange={(option) => {
                    setValue("doctorId", option?.id ?? "");
                    setValue("doctorLabel", option?.label ?? "");
                  }}
                  error={!!errors.doctorId}
                  helperText={errors.doctorId?.message}
                  fetchOptions={async (query) => {
                    const result = await doctorsApi.search({ query, pageSize: 10 });
                    return result.items.map((d) => ({ id: d.id, label: `${d.name} — ${d.specialization}` }));
                  }}
                />
              </Grid>
            </Grid>
          )}

          <Divider sx={{ my: 2 }} />

          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={12}>
              <TextField label="Chief Complaint" fullWidth multiline minRows={2} {...register("complaint")} />
            </Grid>
            <Grid item xs={12}>
              <TextField label="History of Present Illness" fullWidth multiline minRows={2} {...register("hpi")} />
            </Grid>
            <Grid item xs={12}>
              <TextField label="Past / Family / Social History" fullWidth multiline minRows={2} {...register("history")} />
            </Grid>
            <Grid item xs={12}>
              <TextField label="Examination" fullWidth multiline minRows={2} {...register("examination")} />
            </Grid>
            <Grid item xs={12}>
              <Controller
                name="diagnosis"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Diagnosis"
                    fullWidth
                    required
                    multiline
                    minRows={2}
                    error={!!errors.diagnosis}
                    helperText={errors.diagnosis?.message}
                  />
                )}
              />
            </Grid>
            <Grid item xs={12}>
              <TextField label="Treatment" fullWidth multiline minRows={2} {...register("treatment")} />
            </Grid>
            <Grid item xs={12}>
              <TextField label="Follow-up" fullWidth multiline minRows={2} {...register("followUp")} />
            </Grid>
          </Grid>

          <Stack direction="row" spacing={2} justifyContent="flex-end">
            <Button variant="outlined" onClick={() => navigate(-1)} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type="submit" variant="contained" disabled={isSubmitting}>
              {isSubmitting ? <CircularProgress size={22} color="inherit" /> : "Save"}
            </Button>
          </Stack>
        </Box>
      </Paper>
    </Box>
  );
}

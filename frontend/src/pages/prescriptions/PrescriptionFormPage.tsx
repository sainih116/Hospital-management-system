import AddRoundedIcon from "@mui/icons-material/AddRounded";
import DeleteRoundedIcon from "@mui/icons-material/DeleteRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useNavigate } from "react-router-dom";
import { useFieldArray, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";

import AsyncAutocomplete from "@/components/common/AsyncAutocomplete";
import { doctorsApi } from "@/features/doctors/doctorsApi";
import { patientsApi } from "@/features/patients/patientsApi";
import {
  emptyPrescriptionFormValues,
  emptyPrescriptionItem,
  prescriptionFormSchema,
  type PrescriptionFormInput,
} from "@/features/prescriptions/prescriptionSchema";
import { prescriptionsApi } from "@/features/prescriptions/prescriptionsApi";
import { ApiRequestError } from "@/services/apiClient";

export default function PrescriptionFormPage() {
  const navigate = useNavigate();
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    control,
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<PrescriptionFormInput>({
    resolver: zodResolver(prescriptionFormSchema),
    defaultValues: emptyPrescriptionFormValues,
  });

  const { fields, append, remove } = useFieldArray({ control, name: "items" });

  const patientLabel = watch("patientLabel");
  const doctorLabel = watch("doctorLabel");

  const onSubmit = async (values: PrescriptionFormInput) => {
    setSubmitError(null);
    try {
      const created = await prescriptionsApi.create({
        patientId: values.patientId,
        doctorId: values.doctorId,
        items: values.items.map((i) => ({
          medicineName: i.medicineName,
          dose: i.dose,
          frequency: i.frequency,
          duration: i.duration,
          instructions: i.instructions || undefined,
        })),
      });
      navigate(`/prescriptions/${created.id}`);
    } catch (err) {
      setSubmitError(err instanceof ApiRequestError ? err.message : "Failed to save prescription.");
    }
  };

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} sx={{ mb: 3 }}>
        New Prescription
      </Typography>

      {submitError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {submitError}
        </Alert>
      )}

      <Paper variant="outlined" sx={{ p: 3, maxWidth: 900 }}>
        <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
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

          <Divider sx={{ my: 2 }} />
          <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
            <Typography variant="subtitle2" color="text.secondary">
              Medicines
            </Typography>
            <Button size="small" startIcon={<AddRoundedIcon />} onClick={() => append(emptyPrescriptionItem)}>
              Add Medicine
            </Button>
          </Stack>
          {errors.items?.message && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {errors.items.message}
            </Alert>
          )}

          <Stack spacing={2} sx={{ mb: 3 }}>
            {fields.map((field, index) => (
              <Paper key={field.id} variant="outlined" sx={{ p: 2 }}>
                <Grid container spacing={2} alignItems="flex-start">
                  <Grid item xs={12} sm={4}>
                    <TextField
                      label="Medicine"
                      fullWidth
                      required
                      error={!!errors.items?.[index]?.medicineName}
                      helperText={errors.items?.[index]?.medicineName?.message}
                      {...register(`items.${index}.medicineName`)}
                    />
                  </Grid>
                  <Grid item xs={6} sm={2}>
                    <TextField
                      label="Dose"
                      fullWidth
                      required
                      error={!!errors.items?.[index]?.dose}
                      helperText={errors.items?.[index]?.dose?.message}
                      {...register(`items.${index}.dose`)}
                    />
                  </Grid>
                  <Grid item xs={6} sm={2}>
                    <TextField
                      label="Frequency"
                      fullWidth
                      required
                      error={!!errors.items?.[index]?.frequency}
                      helperText={errors.items?.[index]?.frequency?.message}
                      {...register(`items.${index}.frequency`)}
                    />
                  </Grid>
                  <Grid item xs={6} sm={2}>
                    <TextField
                      label="Duration"
                      fullWidth
                      required
                      error={!!errors.items?.[index]?.duration}
                      helperText={errors.items?.[index]?.duration?.message}
                      {...register(`items.${index}.duration`)}
                    />
                  </Grid>
                  <Grid item xs={6} sm={1}>
                    <IconButton
                      color="error"
                      onClick={() => remove(index)}
                      disabled={fields.length === 1}
                      aria-label="Remove medicine"
                    >
                      <DeleteRoundedIcon />
                    </IconButton>
                  </Grid>
                  <Grid item xs={12}>
                    <TextField label="Instructions" fullWidth {...register(`items.${index}.instructions`)} />
                  </Grid>
                </Grid>
              </Paper>
            ))}
          </Stack>

          <Stack direction="row" spacing={2} justifyContent="flex-end">
            <Button variant="outlined" onClick={() => navigate(-1)} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type="submit" variant="contained" disabled={isSubmitting}>
              {isSubmitting ? <CircularProgress size={22} color="inherit" /> : "Issue Prescription"}
            </Button>
          </Stack>
        </Box>
      </Paper>
    </Box>
  );
}

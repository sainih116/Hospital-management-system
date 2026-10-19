import { zodResolver } from "@hookform/resolvers/zod";
import Alert from "@mui/material/Alert";
import Autocomplete from "@mui/material/Autocomplete";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useMemo, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";

import { appointmentsApi } from "@/features/appointments/appointmentsApi";
import { buildBookingPayload, bookingFormSchema, emptyBookingFormValues, type BookingFormInput } from "@/features/appointments/appointmentSchema";
import type { Doctor } from "@/features/doctors/doctorsApi";
import { doctorsApi } from "@/features/doctors/doctorsApi";
import type { Patient } from "@/features/patients/patientsApi";
import { patientsApi } from "@/features/patients/patientsApi";
import { ApiRequestError } from "@/services/apiClient";

interface Option {
  id: string;
  label: string;
}

function useDebouncedValue(value: string, delayMs: number): string {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(timer);
  }, [value, delayMs]);
  return debounced;
}

export default function AppointmentBookingPage() {
  const navigate = useNavigate();
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    control,
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<BookingFormInput>({
    resolver: zodResolver(bookingFormSchema),
    defaultValues: emptyBookingFormValues,
  });

  const [patientQuery, setPatientQuery] = useState("");
  const [patientOptions, setPatientOptions] = useState<Patient[]>([]);
  const [patientLoading, setPatientLoading] = useState(false);
  const debouncedPatientQuery = useDebouncedValue(patientQuery, 300);

  const [doctorQuery, setDoctorQuery] = useState("");
  const [doctorOptions, setDoctorOptions] = useState<Doctor[]>([]);
  const [doctorLoading, setDoctorLoading] = useState(false);
  const debouncedDoctorQuery = useDebouncedValue(doctorQuery, 300);

  useEffect(() => {
    if (!debouncedPatientQuery) {
      setPatientOptions([]);
      return;
    }
    setPatientLoading(true);
    patientsApi
      .search({ query: debouncedPatientQuery, pageSize: 10 })
      .then((r) => setPatientOptions(r.items))
      .catch(() => setPatientOptions([]))
      .finally(() => setPatientLoading(false));
  }, [debouncedPatientQuery]);

  useEffect(() => {
    if (!debouncedDoctorQuery) {
      setDoctorOptions([]);
      return;
    }
    setDoctorLoading(true);
    doctorsApi
      .search({ query: debouncedDoctorQuery, pageSize: 10 })
      .then((r) => setDoctorOptions(r.items))
      .catch(() => setDoctorOptions([]))
      .finally(() => setDoctorLoading(false));
  }, [debouncedDoctorQuery]);

  const patientOptionList: Option[] = useMemo(
    () => patientOptions.map((p) => ({ id: p.id, label: `${p.name} (${p.patientCode})` })),
    [patientOptions],
  );
  const doctorOptionList: Option[] = useMemo(
    () => doctorOptions.map((d) => ({ id: d.id, label: `${d.name} — ${d.specialization}` })),
    [doctorOptions],
  );

  const patientLabel = watch("patientLabel");
  const doctorLabel = watch("doctorLabel");

  const onSubmit = async (values: BookingFormInput) => {
    setSubmitError(null);
    try {
      const created = await appointmentsApi.book(buildBookingPayload(values));
      navigate(`/appointments/${created.id}`);
    } catch (err) {
      setSubmitError(err instanceof ApiRequestError ? err.message : "Failed to book appointment.");
    }
  };

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} sx={{ mb: 3 }}>
        Book Appointment
      </Typography>

      {submitError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {submitError}
        </Alert>
      )}

      <Paper variant="outlined" sx={{ p: 3, maxWidth: 700 }}>
        <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={12}>
              <Autocomplete
                options={patientOptionList}
                loading={patientLoading}
                filterOptions={(x) => x}
                value={patientLabel ? { id: watch("patientId"), label: patientLabel } : null}
                onInputChange={(_, newValue) => setPatientQuery(newValue)}
                onChange={(_, option) => {
                  setValue("patientId", option?.id ?? "");
                  setValue("patientLabel", option?.label ?? "");
                }}
                isOptionEqualToValue={(option, value) => option.id === value.id}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    label="Patient"
                    required
                    error={!!errors.patientId}
                    helperText={errors.patientId?.message ?? "Search by name, patient code or phone"}
                  />
                )}
              />
            </Grid>
            <Grid item xs={12}>
              <Autocomplete
                options={doctorOptionList}
                loading={doctorLoading}
                filterOptions={(x) => x}
                value={doctorLabel ? { id: watch("doctorId"), label: doctorLabel } : null}
                onInputChange={(_, newValue) => setDoctorQuery(newValue)}
                onChange={(_, option) => {
                  setValue("doctorId", option?.id ?? "");
                  setValue("doctorLabel", option?.label ?? "");
                }}
                isOptionEqualToValue={(option, value) => option.id === value.id}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    label="Doctor"
                    required
                    error={!!errors.doctorId}
                    helperText={errors.doctorId?.message ?? "Search by name or specialization"}
                  />
                )}
              />
            </Grid>
            <Grid item xs={12} sm={7}>
              <Controller
                name="startAt"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Start Time"
                    type="datetime-local"
                    fullWidth
                    required
                    slotProps={{ inputLabel: { shrink: true } }}
                    error={!!errors.startAt}
                    helperText={errors.startAt?.message}
                  />
                )}
              />
            </Grid>
            <Grid item xs={12} sm={5}>
              <TextField
                label="Duration (minutes)"
                type="number"
                fullWidth
                required
                error={!!errors.durationMinutes}
                helperText={errors.durationMinutes?.message}
                {...register("durationMinutes")}
              />
            </Grid>
            <Grid item xs={12}>
              <TextField label="Reason" fullWidth multiline minRows={2} {...register("reason")} />
            </Grid>
          </Grid>

          <Stack direction="row" spacing={2} justifyContent="flex-end">
            <Button variant="outlined" onClick={() => navigate(-1)} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type="submit" variant="contained" disabled={isSubmitting}>
              {isSubmitting ? <CircularProgress size={22} color="inherit" /> : "Book Appointment"}
            </Button>
          </Stack>
        </Box>
      </Paper>
    </Box>
  );
}

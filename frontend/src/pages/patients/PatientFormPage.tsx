import { zodResolver } from "@hookform/resolvers/zod";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Divider from "@mui/material/Divider";
import FormControl from "@mui/material/FormControl";
import FormHelperText from "@mui/material/FormHelperText";
import Grid from "@mui/material/Grid";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";

import {
  BLOOD_GROUPS,
  GENDERS,
  PATIENT_TYPES,
  buildPatientPayload,
  emptyPatientFormValues,
  patientFormSchema,
  patientToFormValues,
  type PatientFormInput,
} from "@/features/patients/patientSchema";
import { patientsApi } from "@/features/patients/patientsApi";
import { ApiRequestError } from "@/services/apiClient";

export default function PatientFormPage() {
  const { id } = useParams<{ id: string }>();
  const isEditMode = Boolean(id);
  const navigate = useNavigate();

  const [loadingPatient, setLoadingPatient] = useState(isEditMode);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    control,
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<PatientFormInput>({
    resolver: zodResolver(patientFormSchema),
    defaultValues: emptyPatientFormValues,
  });

  useEffect(() => {
    if (!id) return;
    let cancelled = false;

    patientsApi
      .getById(id)
      .then((patient) => {
        if (cancelled) return;
        reset(patientToFormValues(patient));
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setLoadError(err instanceof ApiRequestError ? err.message : "Failed to load patient.");
      })
      .finally(() => {
        if (!cancelled) setLoadingPatient(false);
      });

    return () => {
      cancelled = true;
    };
  }, [id, reset]);

  const onSubmit = async (values: PatientFormInput) => {
    setSubmitError(null);
    try {
      const payload = buildPatientPayload(values);
      if (isEditMode && id) {
        await patientsApi.update(id, payload);
        navigate(`/patients/${id}`);
      } else {
        const created = await patientsApi.create(payload);
        navigate(`/patients/${created.id}`);
      }
    } catch (err) {
      setSubmitError(err instanceof ApiRequestError ? err.message : "Failed to save patient.");
    }
  };

  if (loadingPatient) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (loadError) {
    return <Alert severity="error">{loadError}</Alert>;
  }

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} sx={{ mb: 3 }}>
        {isEditMode ? "Edit Patient" : "Register Patient"}
      </Typography>

      {submitError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {submitError}
        </Alert>
      )}

      <Paper variant="outlined" sx={{ p: 3, maxWidth: 800 }}>
        <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            Demographics
          </Typography>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Full Name"
                fullWidth
                required
                error={!!errors.name}
                helperText={errors.name?.message}
                {...register("name")}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Date of Birth"
                type="date"
                fullWidth
                required
                slotProps={{ inputLabel: { shrink: true } }}
                error={!!errors.dob}
                helperText={errors.dob?.message}
                {...register("dob")}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <Controller
                name="gender"
                control={control}
                render={({ field }) => (
                  <FormControl fullWidth required error={!!errors.gender}>
                    <InputLabel id="gender-label">Gender</InputLabel>
                    <Select labelId="gender-label" label="Gender" {...field}>
                      {GENDERS.map((g) => (
                        <MenuItem key={g} value={g}>
                          {g}
                        </MenuItem>
                      ))}
                    </Select>
                    {errors.gender && <FormHelperText>{errors.gender.message}</FormHelperText>}
                  </FormControl>
                )}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <Controller
                name="type"
                control={control}
                render={({ field }) => (
                  <FormControl fullWidth required error={!!errors.type}>
                    <InputLabel id="type-label">Patient Type</InputLabel>
                    <Select labelId="type-label" label="Patient Type" {...field}>
                      {PATIENT_TYPES.map((t) => (
                        <MenuItem key={t} value={t}>
                          {t}
                        </MenuItem>
                      ))}
                    </Select>
                    {errors.type && <FormHelperText>{errors.type.message}</FormHelperText>}
                  </FormControl>
                )}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <Controller
                name="bloodGroup"
                control={control}
                render={({ field }) => (
                  <FormControl fullWidth error={!!errors.bloodGroup}>
                    <InputLabel id="bloodgroup-label">Blood Group</InputLabel>
                    <Select labelId="bloodgroup-label" label="Blood Group" {...field}>
                      <MenuItem value="">Unknown</MenuItem>
                      {BLOOD_GROUPS.map((bg) => (
                        <MenuItem key={bg} value={bg}>
                          {bg}
                        </MenuItem>
                      ))}
                    </Select>
                    {errors.bloodGroup && <FormHelperText>{errors.bloodGroup.message}</FormHelperText>}
                  </FormControl>
                )}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField label="Allergies (comma-separated)" fullWidth {...register("allergiesText")} />
            </Grid>
          </Grid>

          <Divider sx={{ my: 2 }} />
          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            Contact
          </Typography>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Phone"
                fullWidth
                required
                error={!!errors.phone}
                helperText={errors.phone?.message}
                {...register("phone")}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Email"
                type="email"
                fullWidth
                error={!!errors.email}
                helperText={errors.email?.message}
                {...register("email")}
              />
            </Grid>
            <Grid item xs={12}>
              <TextField label="Address" fullWidth multiline minRows={2} {...register("address")} />
            </Grid>
          </Grid>

          <Divider sx={{ my: 2 }} />
          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            Emergency Contact (optional)
          </Typography>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={12} sm={4}>
              <TextField label="Contact Name" fullWidth {...register("emergencyContactName")} />
            </Grid>
            <Grid item xs={12} sm={4}>
              <TextField
                label="Contact Phone"
                fullWidth
                error={!!errors.emergencyContactPhone}
                helperText={errors.emergencyContactPhone?.message}
                {...register("emergencyContactPhone")}
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <TextField label="Relationship" fullWidth {...register("emergencyContactRelationship")} />
            </Grid>
          </Grid>

          <Divider sx={{ my: 2 }} />
          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            Insurance (optional)
          </Typography>
          <Grid container spacing={2} sx={{ mb: 3 }}>
            <Grid item xs={12} sm={6}>
              <TextField label="Provider" fullWidth {...register("insuranceProvider")} />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField label="Policy Number" fullWidth {...register("insurancePolicyNumber")} />
            </Grid>
          </Grid>

          <Stack direction="row" spacing={2} justifyContent="flex-end">
            <Button variant="outlined" onClick={() => navigate(-1)} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type="submit" variant="contained" disabled={isSubmitting}>
              {isSubmitting ? <CircularProgress size={22} color="inherit" /> : "Save Patient"}
            </Button>
          </Stack>
        </Box>
      </Paper>
    </Box>
  );
}

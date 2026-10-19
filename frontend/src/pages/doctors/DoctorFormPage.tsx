import { zodResolver } from "@hookform/resolvers/zod";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
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
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";

import type { Department } from "@/features/departments/departmentsApi";
import { departmentsApi } from "@/features/departments/departmentsApi";
import {
  DAYS_OF_WEEK,
  buildCreateDoctorPayload,
  buildUpdateDoctorPayload,
  doctorFormSchema,
  doctorToFormValues,
  emptyDoctorFormValues,
  type DoctorFormInput,
} from "@/features/doctors/doctorSchema";
import { doctorsApi } from "@/features/doctors/doctorsApi";
import { usersApi, type UserSummary } from "@/features/users/usersApi";
import { ApiRequestError } from "@/services/apiClient";

export default function DoctorFormPage() {
  const { id } = useParams<{ id: string }>();
  const isEditMode = Boolean(id);
  const navigate = useNavigate();

  const [departments, setDepartments] = useState<Department[]>([]);
  const [doctorUsers, setDoctorUsers] = useState<UserSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    control,
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<DoctorFormInput>({
    resolver: zodResolver(doctorFormSchema),
    defaultValues: emptyDoctorFormValues(),
  });

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        const [depts] = await Promise.all([departmentsApi.list()]);
        if (cancelled) return;
        setDepartments(depts);

        if (isEditMode && id) {
          const doctor = await doctorsApi.getById(id);
          if (cancelled) return;
          reset(doctorToFormValues(doctor));
        } else {
          const users = await usersApi.list({ role: "DOCTOR", pageSize: 100 });
          if (cancelled) return;
          setDoctorUsers(users.items);
        }
      } catch (err) {
        if (!cancelled) setLoadError(err instanceof ApiRequestError ? err.message : "Failed to load form data.");
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [id, isEditMode, reset]);

  const onSubmit = async (values: DoctorFormInput) => {
    setSubmitError(null);
    try {
      if (isEditMode && id) {
        await doctorsApi.update(id, buildUpdateDoctorPayload(values));
        navigate(`/doctors/${id}`);
      } else {
        if (!values.userId) {
          setSubmitError("Select a user account to attach this doctor profile to.");
          return;
        }
        const created = await doctorsApi.create(buildCreateDoctorPayload(values));
        navigate(`/doctors/${created.id}`);
      }
    } catch (err) {
      setSubmitError(err instanceof ApiRequestError ? err.message : "Failed to save doctor profile.");
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

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} sx={{ mb: 3 }}>
        {isEditMode ? "Edit Doctor" : "New Doctor Profile"}
      </Typography>

      {!isEditMode && doctorUsers.length === 0 && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          No available DOCTOR-role accounts found. Create one first via user registration, then attach a clinical
          profile here.
        </Alert>
      )}

      {submitError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {submitError}
        </Alert>
      )}

      <Paper variant="outlined" sx={{ p: 3, maxWidth: 900 }}>
        <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
          {!isEditMode && (
            <>
              <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                Linked Account
              </Typography>
              <Grid container spacing={2} sx={{ mb: 2 }}>
                <Grid item xs={12} sm={6}>
                  <Controller
                    name="userId"
                    control={control}
                    render={({ field }) => (
                      <FormControl fullWidth required>
                        <InputLabel id="user-label">User Account</InputLabel>
                        <Select labelId="user-label" label="User Account" {...field}>
                          {doctorUsers.map((u) => (
                            <MenuItem key={u.id} value={u.id}>
                              {u.name} ({u.email})
                            </MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                    )}
                  />
                </Grid>
              </Grid>
              <Divider sx={{ my: 2 }} />
            </>
          )}

          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            Clinical Profile
          </Typography>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={12} sm={6}>
              <Controller
                name="departmentId"
                control={control}
                render={({ field }) => (
                  <FormControl fullWidth required error={!!errors.departmentId}>
                    <InputLabel id="department-label">Department</InputLabel>
                    <Select labelId="department-label" label="Department" {...field}>
                      {departments.map((d) => (
                        <MenuItem key={d.id} value={d.id}>
                          {d.name}
                        </MenuItem>
                      ))}
                    </Select>
                    {errors.departmentId && <FormHelperText>{errors.departmentId.message}</FormHelperText>}
                  </FormControl>
                )}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="License Number"
                fullWidth
                required
                error={!!errors.licenseNo}
                helperText={errors.licenseNo?.message}
                {...register("licenseNo")}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Specialization"
                fullWidth
                required
                error={!!errors.specialization}
                helperText={errors.specialization?.message}
                {...register("specialization")}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Consultation Fee"
                type="number"
                fullWidth
                required
                error={!!errors.fee}
                helperText={errors.fee?.message}
                {...register("fee")}
              />
            </Grid>
            <Grid item xs={12}>
              <TextField label="Bio" fullWidth multiline minRows={2} {...register("bio")} />
            </Grid>
          </Grid>

          <Divider sx={{ my: 2 }} />
          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            Weekly Availability
          </Typography>
          <Table size="small" sx={{ mb: 3 }}>
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox" />
                <TableCell>Day</TableCell>
                <TableCell>Start</TableCell>
                <TableCell>End</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {DAYS_OF_WEEK.map((day, index) => (
                <Controller
                  key={day.value}
                  name={`availability.${index}`}
                  control={control}
                  render={({ field }) => (
                    <TableRow>
                      <TableCell padding="checkbox">
                        <Checkbox
                          checked={field.value.enabled}
                          onChange={(e) => field.onChange({ ...field.value, enabled: e.target.checked })}
                        />
                      </TableCell>
                      <TableCell>{day.label}</TableCell>
                      <TableCell>
                        <TextField
                          type="time"
                          size="small"
                          disabled={!field.value.enabled}
                          value={field.value.startTime}
                          onChange={(e) => field.onChange({ ...field.value, startTime: e.target.value })}
                          error={!!errors.availability?.[index]?.startTime}
                        />
                      </TableCell>
                      <TableCell>
                        <TextField
                          type="time"
                          size="small"
                          disabled={!field.value.enabled}
                          value={field.value.endTime}
                          onChange={(e) => field.onChange({ ...field.value, endTime: e.target.value })}
                          error={!!errors.availability?.[index]?.endTime}
                          helperText={errors.availability?.[index]?.endTime?.message}
                        />
                      </TableCell>
                    </TableRow>
                  )}
                />
              ))}
            </TableBody>
          </Table>

          <Stack direction="row" spacing={2} justifyContent="flex-end">
            <Button variant="outlined" onClick={() => navigate(-1)} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type="submit" variant="contained" disabled={isSubmitting}>
              {isSubmitting ? <CircularProgress size={22} color="inherit" /> : "Save Doctor"}
            </Button>
          </Stack>
        </Box>
      </Paper>
    </Box>
  );
}

import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import FormControl from "@mui/material/FormControl";
import Grid from "@mui/material/Grid";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import AsyncAutocomplete, { type AsyncOption } from "@/components/common/AsyncAutocomplete";
import { doctorsApi } from "@/features/doctors/doctorsApi";
import { labOrdersApi } from "@/features/laborders/labOrdersApi";
import { labTestsApi, type LabTest } from "@/features/labtests/labTestsApi";
import { patientsApi } from "@/features/patients/patientsApi";
import { ApiRequestError } from "@/services/apiClient";

export default function LabOrderFormPage() {
  const navigate = useNavigate();

  const [tests, setTests] = useState<LabTest[]>([]);
  const [testId, setTestId] = useState("");
  const [patient, setPatient] = useState<AsyncOption | null>(null);
  const [doctor, setDoctor] = useState<AsyncOption | null>(null);

  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    labTestsApi
      .list()
      .then((all) => setTests(all.filter((t) => t.status === "ACTIVE")))
      .catch(() => undefined);
  }, []);

  const canSubmit = Boolean(patient && doctor && testId) && !submitting;

  const handleSubmit = async () => {
    if (!patient || !doctor || !testId) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      const order = await labOrdersApi.create({ patientId: patient.id, doctorId: doctor.id, testId });
      navigate(`/laboratory/${order.id}`);
    } catch (err) {
      setSubmitError(err instanceof ApiRequestError ? err.message : "Failed to create lab order.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} sx={{ mb: 3 }}>
        New Lab Order
      </Typography>

      {submitError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {submitError}
        </Alert>
      )}

      <Paper variant="outlined" sx={{ p: 3, maxWidth: 700 }}>
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6}>
            <AsyncAutocomplete
              label="Patient"
              required
              value={patient}
              onChange={setPatient}
              fetchOptions={async (query) => {
                const result = await patientsApi.search({ query, pageSize: 10 });
                return result.items.map((p) => ({ id: p.id, label: `${p.name} (${p.patientCode})` }));
              }}
            />
          </Grid>
          <Grid item xs={12} sm={6}>
            <AsyncAutocomplete
              label="Ordering Doctor"
              required
              value={doctor}
              onChange={setDoctor}
              fetchOptions={async (query) => {
                const result = await doctorsApi.search({ query, pageSize: 10 });
                return result.items.map((d) => ({ id: d.id, label: `${d.name} — ${d.specialization}` }));
              }}
            />
          </Grid>
          <Grid item xs={12}>
            <FormControl fullWidth required>
              <InputLabel id="test-label">Test</InputLabel>
              <Select labelId="test-label" label="Test" value={testId} onChange={(e) => setTestId(e.target.value)}>
                {tests.map((t) => (
                  <MenuItem key={t.id} value={t.id}>
                    {t.name} ({t.code}) — {t.cost}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
        </Grid>

        <Stack direction="row" spacing={2} justifyContent="flex-end">
          <Button variant="outlined" onClick={() => navigate(-1)} disabled={submitting}>
            Cancel
          </Button>
          <Button variant="contained" onClick={handleSubmit} disabled={!canSubmit}>
            {submitting ? <CircularProgress size={22} color="inherit" /> : "Create Order"}
          </Button>
        </Stack>
      </Paper>
    </Box>
  );
}

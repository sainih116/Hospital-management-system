import CheckCircleRoundedIcon from "@mui/icons-material/CheckCircleRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import AsyncAutocomplete, { type AsyncOption } from "@/components/common/AsyncAutocomplete";
import { pharmacyApi, type DispenseResult } from "@/features/pharmacy/pharmacyApi";
import { patientsApi } from "@/features/patients/patientsApi";
import { prescriptionsApi, type Prescription } from "@/features/prescriptions/prescriptionsApi";
import { ApiRequestError } from "@/services/apiClient";

interface DispenseRow {
  prescribedName: string;
  medicine: AsyncOption | null;
  quantity: string;
}

export default function DispensePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const prefilledPrescriptionId = searchParams.get("prescriptionId");

  const [patient, setPatient] = useState<AsyncOption | null>(null);
  const [activePrescriptions, setActivePrescriptions] = useState<Prescription[]>([]);
  const [loadingPrescriptions, setLoadingPrescriptions] = useState(false);
  const [prescription, setPrescription] = useState<Prescription | null>(null);
  const [rows, setRows] = useState<DispenseRow[]>([]);

  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [result, setResult] = useState<DispenseResult | null>(null);

  useEffect(() => {
    if (prefilledPrescriptionId) {
      prescriptionsApi
        .getById(prefilledPrescriptionId)
        .then((p) => selectPrescription(p))
        .catch(() => undefined);
    }
  }, [prefilledPrescriptionId]);

  useEffect(() => {
    if (!patient) {
      setActivePrescriptions([]);
      return;
    }
    let cancelled = false;
    setLoadingPrescriptions(true);
    prescriptionsApi
      .search({ patientId: patient.id, status: "ACTIVE", pageSize: 50 })
      .then((result) => {
        if (!cancelled) setActivePrescriptions(result.items);
      })
      .catch(() => {
        if (!cancelled) setActivePrescriptions([]);
      })
      .finally(() => {
        if (!cancelled) setLoadingPrescriptions(false);
      });
    return () => {
      cancelled = true;
    };
  }, [patient]);

  const selectPrescription = (p: Prescription) => {
    setPrescription(p);
    setResult(null);
    setSubmitError(null);
    setRows(p.items.map((item) => ({ prescribedName: item.medicineName, medicine: null, quantity: "1" })));
  };

  const canSubmit =
    Boolean(prescription) &&
    rows.length > 0 &&
    rows.every((r) => r.medicine && Number(r.quantity) > 0) &&
    !submitting;

  const handleSubmit = async () => {
    if (!prescription) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      const dispenseResult = await pharmacyApi.dispense({
        prescriptionId: prescription.id,
        items: rows.map((r) => ({ medicineId: r.medicine!.id, quantity: Number(r.quantity) })),
      });
      setResult(dispenseResult);
    } catch (err) {
      setSubmitError(err instanceof ApiRequestError ? err.message : "Failed to dispense prescription.");
    } finally {
      setSubmitting(false);
    }
  };

  if (result) {
    return (
      <Box>
        <Alert severity="success" icon={<CheckCircleRoundedIcon />} sx={{ mb: 3 }}>
          Prescription dispensed successfully on {new Date(result.dispensedAt).toLocaleString()}.
        </Alert>

        <Paper variant="outlined">
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Medicine</TableCell>
                <TableCell align="right">Quantity</TableCell>
                <TableCell>Batches Used (FEFO)</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {result.items.map((item) => (
                <TableRow key={item.medicineId}>
                  <TableCell>{item.medicineName}</TableCell>
                  <TableCell align="right">{item.quantity}</TableCell>
                  <TableCell>
                    <Stack direction="row" spacing={1} flexWrap="wrap">
                      {item.batches.map((b) => (
                        <Chip key={b.batchId} size="small" label={`${b.batchNo}: ${b.quantity}`} />
                      ))}
                    </Stack>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Paper>

        <Stack direction="row" spacing={2} justifyContent="flex-end" sx={{ mt: 3 }}>
          <Button variant="outlined" onClick={() => navigate("/pharmacy")}>
            Back to Catalog
          </Button>
          <Button variant="contained" onClick={() => navigate(`/prescriptions/${result.prescriptionId}`)}>
            View Prescription
          </Button>
        </Stack>
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} sx={{ mb: 3 }}>
        Dispense Prescription
      </Typography>

      {submitError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {submitError}
        </Alert>
      )}

      <Paper variant="outlined" sx={{ p: 3, mb: 3, maxWidth: 900 }}>
        <Grid container spacing={2}>
          <Grid item xs={12} sm={6}>
            <AsyncAutocomplete
              label="Patient"
              value={patient}
              onChange={(p) => {
                setPatient(p);
                setPrescription(null);
                setRows([]);
                setResult(null);
              }}
              fetchOptions={async (query) => {
                const result = await patientsApi.search({ query, pageSize: 10 });
                return result.items.map((p) => ({ id: p.id, label: `${p.name} (${p.patientCode})` }));
              }}
            />
          </Grid>
        </Grid>

        {loadingPrescriptions && (
          <Box sx={{ display: "flex", justifyContent: "center", py: 3 }}>
            <CircularProgress size={24} />
          </Box>
        )}

        {!loadingPrescriptions && patient && activePrescriptions.length === 0 && (
          <Alert severity="info" sx={{ mt: 2 }}>
            This patient has no active (undispensed) prescriptions.
          </Alert>
        )}

        {!loadingPrescriptions && activePrescriptions.length > 0 && !prescription && (
          <Stack spacing={1} sx={{ mt: 2 }}>
            <Typography variant="subtitle2" color="text.secondary">
              Select a prescription to dispense
            </Typography>
            {activePrescriptions.map((p) => (
              <Paper
                key={p.id}
                variant="outlined"
                sx={{ p: 1.5, cursor: "pointer", "&:hover": { borderColor: "primary.main" } }}
                onClick={() => selectPrescription(p)}
              >
                <Stack direction="row" justifyContent="space-between" alignItems="center">
                  <Typography variant="body2">
                    {p.doctorName} — {new Date(p.issuedAt).toLocaleDateString()} — {p.items.length} item(s)
                  </Typography>
                  <Chip size="small" label="ACTIVE" color="success" />
                </Stack>
              </Paper>
            ))}
          </Stack>
        )}
      </Paper>

      {prescription && (
        <Paper variant="outlined" sx={{ p: 3, maxWidth: 900 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
            <Typography variant="subtitle1" fontWeight={600}>
              Prescribed by {prescription.doctorName} on {new Date(prescription.issuedAt).toLocaleDateString()}
            </Typography>
            <Button size="small" onClick={() => setPrescription(null)}>
              Change
            </Button>
          </Stack>

          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Prescribed As</TableCell>
                <TableCell>Dose / Frequency / Duration</TableCell>
                <TableCell sx={{ minWidth: 240 }}>Dispense From Catalog</TableCell>
                <TableCell sx={{ width: 120 }}>Quantity</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {prescription.items.map((item, index) => (
                <TableRow key={index}>
                  <TableCell>{item.medicineName}</TableCell>
                  <TableCell>
                    {item.dose} / {item.frequency} / {item.duration}
                  </TableCell>
                  <TableCell>
                    <AsyncAutocomplete
                      label="Medicine"
                      required
                      value={rows[index]?.medicine ?? null}
                      onChange={(medicine) =>
                        setRows((prev) => prev.map((r, i) => (i === index ? { ...r, medicine } : r)))
                      }
                      fetchOptions={async (query) => {
                        const result = await pharmacyApi.searchMedicines({ query, pageSize: 10 });
                        return result.items
                          .filter((m) => m.status === "ACTIVE")
                          .map((m) => ({ id: m.id, label: `${m.name} (in stock: ${m.currentStock})` }));
                      }}
                    />
                  </TableCell>
                  <TableCell>
                    <TextField
                      type="number"
                      size="small"
                      value={rows[index]?.quantity ?? ""}
                      onChange={(e) =>
                        setRows((prev) => prev.map((r, i) => (i === index ? { ...r, quantity: e.target.value } : r)))
                      }
                      fullWidth
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          <Stack direction="row" spacing={2} justifyContent="flex-end" sx={{ mt: 3 }}>
            <Button variant="outlined" onClick={() => navigate(-1)} disabled={submitting}>
              Cancel
            </Button>
            <Button variant="contained" onClick={handleSubmit} disabled={!canSubmit}>
              {submitting ? <CircularProgress size={22} color="inherit" /> : "Dispense"}
            </Button>
          </Stack>
        </Paper>
      )}
    </Box>
  );
}

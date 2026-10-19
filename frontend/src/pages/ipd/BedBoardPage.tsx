import AddRoundedIcon from "@mui/icons-material/AddRounded";
import BedRoundedIcon from "@mui/icons-material/BedRounded";
import BuildRoundedIcon from "@mui/icons-material/BuildRounded";
import LogoutRoundedIcon from "@mui/icons-material/LogoutRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useEffect, useMemo, useState } from "react";

import AsyncAutocomplete, { type AsyncOption } from "@/components/common/AsyncAutocomplete";
import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { doctorsApi } from "@/features/doctors/doctorsApi";
import { ipdApi, type Admission, type Bed, type Ward } from "@/features/ipd/ipdApi";
import { patientsApi } from "@/features/patients/patientsApi";
import { ApiRequestError } from "@/services/apiClient";

const BED_COLORS: Record<Bed["status"], { bg: string; border: string; label: string }> = {
  AVAILABLE: { bg: "success.light", border: "success.main", label: "Available" },
  OCCUPIED: { bg: "error.light", border: "error.main", label: "Occupied" },
  MAINTENANCE: { bg: "grey.300", border: "grey.500", label: "Maintenance" },
};

function BedTile({ bed, onClick }: { bed: Bed; onClick: () => void }) {
  const colors = BED_COLORS[bed.status];
  return (
    <Paper
      variant="outlined"
      onClick={onClick}
      sx={{
        width: 84,
        height: 72,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        cursor: "pointer",
        bgcolor: colors.bg,
        borderColor: colors.border,
        borderWidth: 2,
        transition: "transform 0.1s",
        "&:hover": { transform: "scale(1.05)" },
      }}
    >
      <BedRoundedIcon fontSize="small" />
      <Typography variant="caption" fontWeight={700}>
        {bed.bedNumber}
      </Typography>
    </Paper>
  );
}

export default function BedBoardPage() {
  const canWrite = useHasPermission(PERMISSIONS.IPD_WRITE);

  const [wards, setWards] = useState<Ward[]>([]);
  const [beds, setBeds] = useState<Bed[]>([]);
  const [activeAdmissions, setActiveAdmissions] = useState<Admission[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [selectedBed, setSelectedBed] = useState<Bed | null>(null);

  const [admitPatientOption, setAdmitPatientOption] = useState<AsyncOption | null>(null);
  const [admitDoctorOption, setAdmitDoctorOption] = useState<AsyncOption | null>(null);
  const [admitReason, setAdmitReason] = useState("");
  const [admitting, setAdmitting] = useState(false);
  const [admitError, setAdmitError] = useState<string | null>(null);

  const [dischargeNote, setDischargeNote] = useState("");
  const [discharging, setDischarging] = useState(false);
  const [dischargeError, setDischargeError] = useState<string | null>(null);

  const [wardDialogOpen, setWardDialogOpen] = useState(false);
  const [newWardName, setNewWardName] = useState("");
  const [newWardCode, setNewWardCode] = useState("");
  const [savingWard, setSavingWard] = useState(false);
  const [wardError, setWardError] = useState<string | null>(null);

  const [bedDialogOpen, setBedDialogOpen] = useState(false);
  const [newBedWardId, setNewBedWardId] = useState("");
  const [newBedNumber, setNewBedNumber] = useState("");
  const [savingBed, setSavingBed] = useState(false);
  const [bedError, setBedError] = useState<string | null>(null);

  const load = () => {
    setLoading(true);
    setError(null);
    Promise.all([ipdApi.listWards(), ipdApi.listBeds(), ipdApi.searchAdmissions({ status: "ADMITTED", pageSize: 200 })])
      .then(([wardsResult, bedsResult, admissionsResult]) => {
        setWards(wardsResult);
        setBeds(bedsResult);
        setActiveAdmissions(admissionsResult.items);
      })
      .catch((err: unknown) => setError(err instanceof ApiRequestError ? err.message : "Failed to load the bed board."))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const admissionByBedId = useMemo(() => {
    const map = new Map<string, Admission>();
    activeAdmissions.forEach((a) => map.set(a.bedId, a));
    return map;
  }, [activeAdmissions]);

  const bedsByWard = useMemo(() => {
    const map = new Map<string, Bed[]>();
    beds.forEach((b) => {
      const list = map.get(b.wardId) ?? [];
      list.push(b);
      map.set(b.wardId, list);
    });
    return map;
  }, [beds]);

  const closeBedDialog = () => {
    setSelectedBed(null);
    setAdmitPatientOption(null);
    setAdmitDoctorOption(null);
    setAdmitReason("");
    setAdmitError(null);
    setDischargeNote("");
    setDischargeError(null);
  };

  const handleAdmit = async () => {
    if (!selectedBed || !admitPatientOption || !admitDoctorOption) return;
    setAdmitting(true);
    setAdmitError(null);
    try {
      await ipdApi.admitPatient({
        patientId: admitPatientOption.id,
        doctorId: admitDoctorOption.id,
        bedId: selectedBed.id,
        reason: admitReason || undefined,
      });
      closeBedDialog();
      load();
    } catch (err) {
      setAdmitError(err instanceof ApiRequestError ? err.message : "Failed to admit patient.");
    } finally {
      setAdmitting(false);
    }
  };

  const handleDischarge = async () => {
    const admission = selectedBed ? admissionByBedId.get(selectedBed.id) : undefined;
    if (!admission) return;
    setDischarging(true);
    setDischargeError(null);
    try {
      await ipdApi.dischargePatient(admission.id, { note: dischargeNote || undefined });
      closeBedDialog();
      load();
    } catch (err) {
      setDischargeError(err instanceof ApiRequestError ? err.message : "Failed to discharge patient.");
    } finally {
      setDischarging(false);
    }
  };

  const handleSetBedStatus = async (status: "AVAILABLE" | "MAINTENANCE") => {
    if (!selectedBed) return;
    setDischarging(true);
    setDischargeError(null);
    try {
      await ipdApi.setBedStatus(selectedBed.id, status);
      closeBedDialog();
      load();
    } catch (err) {
      setDischargeError(err instanceof ApiRequestError ? err.message : "Failed to update the bed.");
    } finally {
      setDischarging(false);
    }
  };

  const handleCreateWard = async () => {
    setSavingWard(true);
    setWardError(null);
    try {
      await ipdApi.createWard({ name: newWardName, code: newWardCode.toUpperCase() });
      setWardDialogOpen(false);
      setNewWardName("");
      setNewWardCode("");
      load();
    } catch (err) {
      setWardError(err instanceof ApiRequestError ? err.message : "Failed to create ward.");
    } finally {
      setSavingWard(false);
    }
  };

  const handleCreateBed = async () => {
    setSavingBed(true);
    setBedError(null);
    try {
      await ipdApi.createBed({ wardId: newBedWardId, bedNumber: newBedNumber });
      setBedDialogOpen(false);
      setNewBedNumber("");
      load();
    } catch (err) {
      setBedError(err instanceof ApiRequestError ? err.message : "Failed to add bed.");
    } finally {
      setSavingBed(false);
    }
  };

  const selectedAdmission = selectedBed ? admissionByBedId.get(selectedBed.id) : undefined;

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          IPD / Wards
        </Typography>
        {canWrite && (
          <Stack direction="row" spacing={2}>
            <Button variant="outlined" startIcon={<AddRoundedIcon />} onClick={() => setWardDialogOpen(true)}>
              Add Ward
            </Button>
            <Button
              variant="outlined"
              startIcon={<AddRoundedIcon />}
              onClick={() => {
                setNewBedWardId(wards[0]?.id ?? "");
                setBedDialogOpen(true);
              }}
              disabled={wards.length === 0}
            >
              Add Bed
            </Button>
          </Stack>
        )}
      </Stack>

      <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
        <Chip size="small" label="Available" sx={{ bgcolor: "success.light" }} />
        <Chip size="small" label="Occupied" sx={{ bgcolor: "error.light" }} />
        <Chip size="small" label="Maintenance" sx={{ bgcolor: "grey.300" }} />
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {loading ? (
        <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
          <CircularProgress />
        </Box>
      ) : wards.length === 0 ? (
        <Alert severity="info">No wards set up yet. {canWrite && 'Use "Add Ward" to get started.'}</Alert>
      ) : (
        <Stack spacing={2} sx={{ mb: 4 }}>
          {wards.map((ward) => (
            <Paper key={ward.id} variant="outlined" sx={{ p: 2 }}>
              <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1.5 }}>
                {ward.name} <Typography component="span" variant="caption" color="text.secondary">({ward.code})</Typography>
              </Typography>
              <Stack direction="row" spacing={1.5} flexWrap="wrap" useFlexGap>
                {(bedsByWard.get(ward.id) ?? []).map((bed) => (
                  <BedTile key={bed.id} bed={bed} onClick={() => setSelectedBed(bed)} />
                ))}
                {(bedsByWard.get(ward.id) ?? []).length === 0 && (
                  <Typography variant="body2" color="text.secondary">
                    No beds in this ward yet.
                  </Typography>
                )}
              </Stack>
            </Paper>
          ))}
        </Stack>
      )}

      <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1 }}>
        Active Admissions
      </Typography>
      <Paper variant="outlined">
        <TableContainer sx={{ overflowX: "auto" }}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Patient</TableCell>
                <TableCell>Doctor</TableCell>
                <TableCell>Bed</TableCell>
                <TableCell>Reason</TableCell>
                <TableCell>Admitted</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {activeAdmissions.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">No patients currently admitted.</Typography>
                  </TableCell>
                </TableRow>
              )}
              {activeAdmissions.map((a) => (
                <TableRow key={a.id}>
                  <TableCell>{a.patientName}</TableCell>
                  <TableCell>{a.doctorName}</TableCell>
                  <TableCell>
                    {a.wardName} · {a.bedNumber}
                  </TableCell>
                  <TableCell>{a.reason || "—"}</TableCell>
                  <TableCell>{new Date(a.admittedAt).toLocaleString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      {/* Bed action dialog: Admit / View+Discharge / Maintenance toggle */}
      <Dialog open={!!selectedBed} onClose={closeBedDialog} fullWidth maxWidth="sm">
        {selectedBed && (
          <>
            <DialogTitle>
              {selectedBed.wardName} — Bed {selectedBed.bedNumber}
            </DialogTitle>
            <DialogContent>
              {selectedBed.status === "AVAILABLE" && canWrite && (
                <Stack spacing={2} sx={{ mt: 0.5 }}>
                  {admitError && <Alert severity="error">{admitError}</Alert>}
                  <AsyncAutocomplete
                    label="Patient"
                    required
                    value={admitPatientOption}
                    onChange={setAdmitPatientOption}
                    fetchOptions={async (query) => {
                      const result = await patientsApi.search({ query, pageSize: 10 });
                      return result.items.map((p) => ({ id: p.id, label: `${p.name} (${p.patientCode})` }));
                    }}
                  />
                  <AsyncAutocomplete
                    label="Admitting Doctor"
                    required
                    value={admitDoctorOption}
                    onChange={setAdmitDoctorOption}
                    fetchOptions={async (query) => {
                      const result = await doctorsApi.search({ query, pageSize: 10 });
                      return result.items.map((d) => ({ id: d.id, label: `${d.name} — ${d.specialization}` }));
                    }}
                  />
                  <TextField label="Reason (optional)" value={admitReason} onChange={(e) => setAdmitReason(e.target.value)} fullWidth multiline minRows={2} />
                </Stack>
              )}

              {selectedBed.status === "AVAILABLE" && !canWrite && (
                <Typography color="text.secondary">This bed is available.</Typography>
              )}

              {selectedBed.status === "OCCUPIED" && selectedAdmission && (
                <Stack spacing={1.5} sx={{ mt: 0.5 }}>
                  {dischargeError && <Alert severity="error">{dischargeError}</Alert>}
                  <Box>
                    <Typography variant="caption" color="text.secondary">Patient</Typography>
                    <Typography variant="body1" fontWeight={600}>{selectedAdmission.patientName}</Typography>
                  </Box>
                  <Box>
                    <Typography variant="caption" color="text.secondary">Doctor</Typography>
                    <Typography variant="body1">{selectedAdmission.doctorName}</Typography>
                  </Box>
                  <Box>
                    <Typography variant="caption" color="text.secondary">Admitted</Typography>
                    <Typography variant="body1">{new Date(selectedAdmission.admittedAt).toLocaleString()}</Typography>
                  </Box>
                  {selectedAdmission.reason && (
                    <Box>
                      <Typography variant="caption" color="text.secondary">Reason</Typography>
                      <Typography variant="body1">{selectedAdmission.reason}</Typography>
                    </Box>
                  )}
                  {canWrite && (
                    <>
                      <Divider />
                      <TextField
                        label="Discharge Note (optional)"
                        value={dischargeNote}
                        onChange={(e) => setDischargeNote(e.target.value)}
                        fullWidth
                        multiline
                        minRows={2}
                      />
                    </>
                  )}
                </Stack>
              )}

              {selectedBed.status === "MAINTENANCE" && (
                <Stack spacing={1.5} sx={{ mt: 0.5 }}>
                  {dischargeError && <Alert severity="error">{dischargeError}</Alert>}
                  <Typography color="text.secondary">This bed is marked under maintenance.</Typography>
                </Stack>
              )}
            </DialogContent>
            <DialogActions>
              <Button onClick={closeBedDialog}>Close</Button>
              {selectedBed.status === "AVAILABLE" && canWrite && (
                <>
                  <Button startIcon={<BuildRoundedIcon />} onClick={() => handleSetBedStatus("MAINTENANCE")} disabled={admitting || discharging}>
                    Mark Maintenance
                  </Button>
                  <Button
                    variant="contained"
                    startIcon={<BedRoundedIcon />}
                    onClick={handleAdmit}
                    disabled={admitting || !admitPatientOption || !admitDoctorOption}
                  >
                    {admitting ? <CircularProgress size={20} color="inherit" /> : "Admit Patient"}
                  </Button>
                </>
              )}
              {selectedBed.status === "OCCUPIED" && canWrite && selectedAdmission && (
                <Button variant="contained" color="warning" startIcon={<LogoutRoundedIcon />} onClick={handleDischarge} disabled={discharging}>
                  {discharging ? <CircularProgress size={20} color="inherit" /> : "Discharge"}
                </Button>
              )}
              {selectedBed.status === "MAINTENANCE" && canWrite && (
                <Button variant="contained" startIcon={<BuildRoundedIcon />} onClick={() => handleSetBedStatus("AVAILABLE")} disabled={discharging}>
                  {discharging ? <CircularProgress size={20} color="inherit" /> : "Mark Available"}
                </Button>
              )}
            </DialogActions>
          </>
        )}
      </Dialog>

      {/* Add Ward dialog */}
      <Dialog open={wardDialogOpen} onClose={() => setWardDialogOpen(false)} fullWidth maxWidth="xs">
        <DialogTitle>Add Ward</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 0.5 }}>
            {wardError && (
              <Grid item xs={12}>
                <Alert severity="error">{wardError}</Alert>
              </Grid>
            )}
            <Grid item xs={12}>
              <TextField label="Name" value={newWardName} onChange={(e) => setNewWardName(e.target.value)} fullWidth autoFocus required />
            </Grid>
            <Grid item xs={12}>
              <TextField label="Code" value={newWardCode} onChange={(e) => setNewWardCode(e.target.value)} fullWidth required helperText="Short unique code, e.g. ICU" />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setWardDialogOpen(false)} disabled={savingWard}>
            Cancel
          </Button>
          <Button variant="contained" onClick={handleCreateWard} disabled={savingWard || !newWardName || !newWardCode}>
            {savingWard ? <CircularProgress size={20} color="inherit" /> : "Create"}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Add Bed dialog */}
      <Dialog open={bedDialogOpen} onClose={() => setBedDialogOpen(false)} fullWidth maxWidth="xs">
        <DialogTitle>Add Bed</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 0.5 }}>
            {bedError && (
              <Grid item xs={12}>
                <Alert severity="error">{bedError}</Alert>
              </Grid>
            )}
            <Grid item xs={12}>
              <TextField select label="Ward" value={newBedWardId} onChange={(e) => setNewBedWardId(e.target.value)} fullWidth required SelectProps={{ native: true }}>
                {wards.map((w) => (
                  <option key={w.id} value={w.id}>
                    {w.name}
                  </option>
                ))}
              </TextField>
            </Grid>
            <Grid item xs={12}>
              <TextField label="Bed Number" value={newBedNumber} onChange={(e) => setNewBedNumber(e.target.value)} fullWidth required helperText="e.g. ICU-04" />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setBedDialogOpen(false)} disabled={savingBed}>
            Cancel
          </Button>
          <Button variant="contained" onClick={handleCreateBed} disabled={savingBed || !newBedWardId || !newBedNumber}>
            {savingBed ? <CircularProgress size={20} color="inherit" /> : "Create"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

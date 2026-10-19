import AssignmentIndRoundedIcon from "@mui/icons-material/AssignmentIndRounded";
import DownloadRoundedIcon from "@mui/icons-material/DownloadRounded";
import EventAvailableRoundedIcon from "@mui/icons-material/EventAvailableRounded";
import LocalPharmacyRoundedIcon from "@mui/icons-material/LocalPharmacyRounded";
import MedicationRoundedIcon from "@mui/icons-material/MedicationRounded";
import PaidRoundedIcon from "@mui/icons-material/PaidRounded";
import PeopleAltRoundedIcon from "@mui/icons-material/PeopleAltRounded";
import PrintRoundedIcon from "@mui/icons-material/PrintRounded";
import ReceiptLongRoundedIcon from "@mui/icons-material/ReceiptLongRounded";
import ScienceRoundedIcon from "@mui/icons-material/ScienceRounded";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import CircularProgress from "@mui/material/CircularProgress";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  type PieLabelRenderProps,
} from "recharts";

import type { SvgIconComponent } from "@/types/mui";
import {
  reportsApi,
  type DoctorWorkload,
  type NameCount,
  type StatusCount,
  type Summary,
  type TrendPoint,
} from "@/features/reports/reportsApi";
import { ApiRequestError } from "@/services/apiClient";

function isoDate(d: Date): string {
  return d.toISOString().slice(0, 10);
}

function defaultFrom(): string {
  const d = new Date();
  d.setDate(d.getDate() - 30);
  return isoDate(d);
}

const currencyFormatter = new Intl.NumberFormat(undefined, { style: "currency", currency: "USD", maximumFractionDigits: 0 });

interface StatCardProps {
  label: string;
  value: string;
  icon: SvgIconComponent;
  color: "primary" | "secondary" | "success" | "warning" | "error";
}

function StatCard({ label, value, icon: Icon, color, loading }: StatCardProps & { loading: boolean }) {
  return (
    <Card variant="outlined" sx={{ height: "100%" }}>
      <CardContent>
        <Stack direction="row" alignItems="center" spacing={2}>
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              width: 44,
              height: 44,
              borderRadius: 2,
              bgcolor: `${color}.main`,
              color: `${color}.contrastText`,
              flexShrink: 0,
            }}
          >
            <Icon fontSize="small" />
          </Box>
          <Box sx={{ minWidth: 0 }}>
            {loading ? (
              <Skeleton variant="text" width={64} height={32} />
            ) : (
              <Typography variant="h6" fontWeight={700} noWrap>
                {value}
              </Typography>
            )}
            <Typography variant="caption" color="text.secondary">
              {label}
            </Typography>
          </Box>
        </Stack>
      </CardContent>
    </Card>
  );
}

function ChartPanel({ title, children, loading, empty }: { title: string; children: React.ReactNode; loading: boolean; empty: boolean }) {
  return (
    <Paper variant="outlined" sx={{ p: 2, height: 360 }}>
      <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1 }}>
        {title}
      </Typography>
      {loading ? (
        <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", height: 280 }}>
          <CircularProgress size={28} />
        </Box>
      ) : empty ? (
        <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", height: 280 }}>
          <Typography color="text.secondary">No data for this range.</Typography>
        </Box>
      ) : (
        <Box sx={{ height: 290 }}>{children}</Box>
      )}
    </Paper>
  );
}

export default function ReportsDashboardPage() {
  const theme = useTheme();

  const [from, setFrom] = useState(defaultFrom());
  const [to, setTo] = useState(isoDate(new Date()));

  const [summary, setSummary] = useState<Summary | null>(null);
  const [appointmentsByStatus, setAppointmentsByStatus] = useState<StatusCount[]>([]);
  const [revenueTrend, setRevenueTrend] = useState<TrendPoint[]>([]);
  const [registrationTrend, setRegistrationTrend] = useState<TrendPoint[]>([]);
  const [doctorWorkload, setDoctorWorkload] = useState<DoctorWorkload[]>([]);
  const [labTestVolume, setLabTestVolume] = useState<NameCount[]>([]);

  const [summaryLoading, setSummaryLoading] = useState(true);
  const [chartsLoading, setChartsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setSummaryLoading(true);
    reportsApi
      .summary()
      .then((result) => {
        if (!cancelled) setSummary(result);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof ApiRequestError ? err.message : "Failed to load the dashboard summary.");
      })
      .finally(() => {
        if (!cancelled) setSummaryLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    setChartsLoading(true);
    setError(null);

    const params = { from, to };
    Promise.all([
      reportsApi.appointmentsByStatus(params),
      reportsApi.revenueTrend(params),
      reportsApi.patientRegistrationTrend(params),
      reportsApi.doctorWorkload({ ...params, limit: 10 }),
      reportsApi.labTestVolume({ ...params, limit: 10 }),
    ])
      .then(([statuses, revenue, registrations, workload, labVolume]) => {
        if (cancelled) return;
        setAppointmentsByStatus(statuses);
        setRevenueTrend(revenue);
        setRegistrationTrend(registrations);
        setDoctorWorkload(workload);
        setLabTestVolume(labVolume);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof ApiRequestError ? err.message : "Failed to load report charts.");
      })
      .finally(() => {
        if (!cancelled) setChartsLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [from, to]);

  const handleExport = async () => {
    setExporting(true);
    setError(null);
    try {
      const blob = await reportsApi.exportWorkbook({ from, to });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `hms-report-${to}.xlsx`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Failed to export the report.");
    } finally {
      setExporting(false);
    }
  };

  const statCards: StatCardProps[] = useMemo(
    () => [
      { label: "Total Patients", value: summary ? summary.totalPatients.toLocaleString() : "—", icon: PeopleAltRoundedIcon, color: "primary" },
      { label: "Active Doctors", value: summary ? summary.totalDoctors.toLocaleString() : "—", icon: AssignmentIndRoundedIcon, color: "secondary" },
      { label: "Appointments Today", value: summary ? summary.appointmentsToday.toLocaleString() : "—", icon: EventAvailableRoundedIcon, color: "primary" },
      {
        label: "Revenue Collected",
        value: summary ? currencyFormatter.format(summary.totalRevenueCollected) : "—",
        icon: PaidRoundedIcon,
        color: "success",
      },
      {
        label: "Outstanding Balance",
        value: summary ? currencyFormatter.format(summary.outstandingBalance) : "—",
        icon: ReceiptLongRoundedIcon,
        color: "warning",
      },
      { label: "Low Stock Medicines", value: summary ? summary.lowStockMedicineCount.toLocaleString() : "—", icon: LocalPharmacyRoundedIcon, color: "error" },
      { label: "Pending Lab Verifications", value: summary ? summary.pendingLabVerifications.toLocaleString() : "—", icon: ScienceRoundedIcon, color: "warning" },
      { label: "Active Prescriptions", value: summary ? summary.activePrescriptions.toLocaleString() : "—", icon: MedicationRoundedIcon, color: "secondary" },
    ],
    [summary],
  );

  const pieColors = [theme.palette.primary.main, theme.palette.secondary.main, theme.palette.success.main, theme.palette.warning.main, theme.palette.error.main];

  return (
    <Box>
      <Stack direction={{ xs: "column", sm: "row" }} justifyContent="space-between" alignItems={{ sm: "center" }} spacing={2} sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>
          Reporting Dashboard
        </Typography>
        <Stack direction="row" spacing={2} sx={{ "@media print": { display: "none" } }}>
          <TextField
            label="From"
            type="date"
            size="small"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            slotProps={{ inputLabel: { shrink: true } }}
          />
          <TextField
            label="To"
            type="date"
            size="small"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            slotProps={{ inputLabel: { shrink: true } }}
          />
          <Button
            variant="outlined"
            startIcon={exporting ? <CircularProgress size={16} color="inherit" /> : <DownloadRoundedIcon />}
            onClick={handleExport}
            disabled={exporting}
          >
            Export to Excel
          </Button>
          <Button variant="outlined" startIcon={<PrintRoundedIcon />} onClick={() => window.print()}>
            Print / Save as PDF
          </Button>
        </Stack>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Grid container spacing={2} sx={{ mb: 3 }}>
        {statCards.map((card) => (
          <Grid key={card.label} item xs={12} sm={6} md={3}>
            <StatCard {...card} loading={summaryLoading} />
          </Grid>
        ))}
      </Grid>

      <Grid container spacing={2}>
        <Grid item xs={12} md={6}>
          <ChartPanel title="Appointments by Status" loading={chartsLoading} empty={appointmentsByStatus.length === 0}>
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={appointmentsByStatus}
                  dataKey="count"
                  nameKey="status"
                  cx="50%"
                  cy="50%"
                  outerRadius={90}
                  label={(entry: PieLabelRenderProps) => {
                    // recharts passes the computed sector geometry here, not
                    // our data row — the original row is carried through on
                    // `.payload` (typed `any` by recharts itself).
                    const { status, count } = entry.payload as StatusCount;
                    return `${status}: ${count}`;
                  }}
                >
                  {appointmentsByStatus.map((entry, index) => (
                    <Cell key={entry.status} fill={pieColors[index % pieColors.length]} />
                  ))}
                </Pie>
                <Tooltip />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </ChartPanel>
        </Grid>

        <Grid item xs={12} md={6}>
          <ChartPanel title="Revenue Collected" loading={chartsLoading} empty={revenueTrend.length === 0}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={revenueTrend}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" tick={{ fontSize: 11 }} />
                <YAxis tick={{ fontSize: 11 }} />
                <Tooltip formatter={(value) => currencyFormatter.format(Number(value))} />
                <Area type="monotone" dataKey="value" name="Revenue" stroke={theme.palette.success.main} fill={theme.palette.success.main} fillOpacity={0.2} />
              </AreaChart>
            </ResponsiveContainer>
          </ChartPanel>
        </Grid>

        <Grid item xs={12} md={6}>
          <ChartPanel title="New Patient Registrations" loading={chartsLoading} empty={registrationTrend.length === 0}>
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={registrationTrend}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" tick={{ fontSize: 11 }} />
                <YAxis tick={{ fontSize: 11 }} allowDecimals={false} />
                <Tooltip />
                <Line type="monotone" dataKey="value" name="Registrations" stroke={theme.palette.primary.main} strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </ChartPanel>
        </Grid>

        <Grid item xs={12} md={6}>
          <ChartPanel title="Top Doctors by Appointment Volume" loading={chartsLoading} empty={doctorWorkload.length === 0}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={doctorWorkload} layout="vertical" margin={{ left: 24 }}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" tick={{ fontSize: 11 }} allowDecimals={false} />
                <YAxis type="category" dataKey="doctorName" tick={{ fontSize: 11 }} width={120} />
                <Tooltip />
                <Bar dataKey="count" name="Appointments" fill={theme.palette.secondary.main} />
              </BarChart>
            </ResponsiveContainer>
          </ChartPanel>
        </Grid>

        <Grid item xs={12}>
          <ChartPanel title="Top Ordered Lab Tests" loading={chartsLoading} empty={labTestVolume.length === 0}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={labTestVolume}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" tick={{ fontSize: 11 }} interval={0} angle={-15} textAnchor="end" height={50} />
                <YAxis tick={{ fontSize: 11 }} allowDecimals={false} />
                <Tooltip />
                <Bar dataKey="count" name="Orders" fill={theme.palette.warning.main} />
              </BarChart>
            </ResponsiveContainer>
          </ChartPanel>
        </Grid>
      </Grid>
    </Box>
  );
}

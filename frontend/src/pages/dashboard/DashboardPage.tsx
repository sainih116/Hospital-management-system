import EventAvailableRoundedIcon from "@mui/icons-material/EventAvailableRounded";
import LocalPharmacyRoundedIcon from "@mui/icons-material/LocalPharmacyRounded";
import PeopleAltRoundedIcon from "@mui/icons-material/PeopleAltRounded";
import ScienceRoundedIcon from "@mui/icons-material/ScienceRounded";
import Box from "@mui/material/Box";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Grid from "@mui/material/Grid";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { Link as RouterLink } from "react-router-dom";
import Link from "@mui/material/Link";

import { PERMISSIONS } from "@/features/auth/permissions";
import { useHasPermission } from "@/features/auth/useHasPermission";
import { reportsApi, type Summary } from "@/features/reports/reportsApi";
import { ApiRequestError } from "@/services/apiClient";
import type { SvgIconComponent } from "@/types/mui";

interface StatCardConfig {
  label: string;
  value: (summary: Summary) => string;
  icon: SvgIconComponent;
  color: "primary" | "secondary" | "success" | "warning";
}

const STAT_CARDS: StatCardConfig[] = [
  { label: "Patients Registered", value: (s) => s.totalPatients.toLocaleString(), icon: PeopleAltRoundedIcon, color: "primary" },
  { label: "Appointments Today", value: (s) => s.appointmentsToday.toLocaleString(), icon: EventAvailableRoundedIcon, color: "secondary" },
  { label: "Pending Lab Verifications", value: (s) => s.pendingLabVerifications.toLocaleString(), icon: ScienceRoundedIcon, color: "warning" },
  { label: "Low Stock Medicines", value: (s) => s.lowStockMedicineCount.toLocaleString(), icon: LocalPharmacyRoundedIcon, color: "success" },
];

function StatCard({ label, value, icon: Icon, color, loading }: { label: string; value: string; icon: SvgIconComponent; color: StatCardConfig["color"]; loading: boolean }) {
  return (
    <Card variant="outlined" sx={{ height: "100%" }}>
      <CardContent>
        <Stack direction="row" alignItems="center" spacing={2}>
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              width: 48,
              height: 48,
              borderRadius: 2,
              bgcolor: `${color}.main`,
              color: `${color}.contrastText`,
            }}
          >
            <Icon />
          </Box>
          <Box>
            {loading ? (
              <Skeleton variant="text" width={48} height={32} />
            ) : (
              <Typography variant="h5" fontWeight={700}>
                {value}
              </Typography>
            )}
            <Typography variant="body2" color="text.secondary">
              {label}
            </Typography>
          </Box>
        </Stack>
      </CardContent>
    </Card>
  );
}

export default function DashboardPage() {
  const canViewReports = useHasPermission(PERMISSIONS.REPORT_EXPORT);

  const [summary, setSummary] = useState<Summary | null>(null);
  const [loading, setLoading] = useState(canViewReports);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!canViewReports) return;
    let cancelled = false;
    setLoading(true);
    reportsApi
      .summary()
      .then((result) => {
        if (!cancelled) setSummary(result);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof ApiRequestError ? err.message : "Failed to load dashboard summary.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [canViewReports]);

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} gutterBottom>
        Dashboard
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        {canViewReports ? (
          error ? (
            error
          ) : (
            <>
              Live snapshot as of now. For trends and breakdowns, see the{" "}
              <Link component={RouterLink} to="/reports">
                full Reporting Dashboard
              </Link>
              .
            </>
          )
        ) : (
          "Summary figures are available to Administrators. Use the module pages in the sidebar to work with your day-to-day records."
        )}
      </Typography>

      <Grid container spacing={2}>
        {STAT_CARDS.map((card) => (
          <Grid key={card.label} item xs={12} sm={6} lg={3}>
            <StatCard
              label={card.label}
              value={canViewReports && summary ? card.value(summary) : "—"}
              icon={card.icon}
              color={card.color}
              loading={canViewReports && loading}
            />
          </Grid>
        ))}
      </Grid>
    </Box>
  );
}

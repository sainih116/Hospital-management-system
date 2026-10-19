import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import { lazy, Suspense } from "react";
import { Navigate, Route, Routes } from "react-router-dom";

import AppointmentBookingPage from "@/pages/appointments/AppointmentBookingPage";
import AppointmentDetailPage from "@/pages/appointments/AppointmentDetailPage";
import AppointmentsListPage from "@/pages/appointments/AppointmentsListPage";
import AuditLogsPage from "@/pages/auditlogs/AuditLogsPage";
import BillDetailPage from "@/pages/billing/BillDetailPage";
import BillFormPage from "@/pages/billing/BillFormPage";
import BillingListPage from "@/pages/billing/BillingListPage";
import BillReceiptPage from "@/pages/billing/BillReceiptPage";
import DashboardLayout from "@/components/layout/DashboardLayout";
import LoginPage from "@/pages/auth/LoginPage";
import DashboardPage from "@/pages/dashboard/DashboardPage";
import DepartmentsPage from "@/pages/departments/DepartmentsPage";
import DoctorFormPage from "@/pages/doctors/DoctorFormPage";
import DoctorProfilePage from "@/pages/doctors/DoctorProfilePage";
import DoctorsListPage from "@/pages/doctors/DoctorsListPage";
import { PERMISSIONS } from "@/features/auth/permissions";
import ForbiddenPage from "@/pages/ForbiddenPage";
import BedBoardPage from "@/pages/ipd/BedBoardPage";
import LabOrderDetailPage from "@/pages/laborders/LabOrderDetailPage";
import LabOrderFormPage from "@/pages/laborders/LabOrderFormPage";
import LabOrdersListPage from "@/pages/laborders/LabOrdersListPage";
import LabReportPage from "@/pages/laborders/LabReportPage";
import LabTestsPage from "@/pages/labtests/LabTestsPage";
import MedicalRecordDetailPage from "@/pages/medicalrecords/MedicalRecordDetailPage";
import MedicalRecordFormPage from "@/pages/medicalrecords/MedicalRecordFormPage";
import MedicalRecordsListPage from "@/pages/medicalrecords/MedicalRecordsListPage";
import NotFoundPage from "@/pages/NotFoundPage";
import DispensePage from "@/pages/pharmacy/DispensePage";
import MedicineDetailPage from "@/pages/pharmacy/MedicineDetailPage";
import MedicinesListPage from "@/pages/pharmacy/MedicinesListPage";
import PatientFormPage from "@/pages/patients/PatientFormPage";
import PatientProfilePage from "@/pages/patients/PatientProfilePage";
import PatientsListPage from "@/pages/patients/PatientsListPage";
import PrescriptionDetailPage from "@/pages/prescriptions/PrescriptionDetailPage";
import PrescriptionFormPage from "@/pages/prescriptions/PrescriptionFormPage";
import PrescriptionsListPage from "@/pages/prescriptions/PrescriptionsListPage";
import UsersPage from "@/pages/settings/UsersPage";
import PermissionRoute from "@/routes/PermissionRoute";
import ProtectedRoute from "@/routes/ProtectedRoute";

// Lazy-loaded: recharts (the only consumer) added ~400KB to the main
// bundle, and this Admin-only page is not something every session loads —
// code-splitting it keeps the cost off everyone else's initial load.
const ReportsDashboardPage = lazy(() => import("@/pages/reports/ReportsDashboardPage"));

function RouteFallback() {
  return (
    <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
      <CircularProgress />
    </Box>
  );
}

/**
 * Phase 10 route table adds the Reporting Dashboard (Recharts-based
 * summary cards + trend/breakdown charts over every prior module's data).
 *
 * Hardening pass: module route groups are wrapped in PermissionRoute
 * (redirects to /forbidden) so direct URL navigation can't reach a page
 * shell the current role has no backend permission for — previously only
 * the sidebar (Sidebar.tsx, role-based) hid these, which didn't stop a
 * typed URL.
 *
 * Post-launch addition: /ipd was a static "coming soon" placeholder
 * (replacing what used to be a dead nav link) until the full IPD/Wards
 * module — Ward/Bed catalog + the admit/discharge workflow, with an
 * atomic bed-occupancy guard closing the exact "two admissions, one bed"
 * race the blueprint calls out — was built out on top of the
 * already-seeded ipd.read/ipd.write permissions. Billing also gained a
 * printable payment receipt (/billing/:id/receipt) and Reports gained an
 * Excel export + browser print-to-PDF at the same time.
 */
export default function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="/login" element={<LoginPage />} />

      <Route element={<ProtectedRoute />}>
        <Route element={<DashboardLayout />}>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/forbidden" element={<ForbiddenPage />} />

          {/* Departments and the lab test catalog are readable by every
              authenticated role on the backend (gated on dashboard.view,
              not a module-specific permission) — only their nav items are
              Admin-only, so no route-level guard is needed here. */}
          <Route path="/settings/departments" element={<DepartmentsPage />} />
          <Route path="/settings/lab-tests" element={<LabTestsPage />} />

          {/* Every group below requires the same permission the backend's
              read endpoint for that module enforces — see
              PermissionRoute's doc comment. Write actions inside each page
              stay gated by their own useHasPermission checks, unchanged. */}
          <Route element={<PermissionRoute permission={PERMISSIONS.PATIENT_READ} />}>
            <Route path="/patients" element={<PatientsListPage />} />
            <Route path="/patients/new" element={<PatientFormPage />} />
            <Route path="/patients/:id" element={<PatientProfilePage />} />
            <Route path="/patients/:id/edit" element={<PatientFormPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.DOCTOR_READ} />}>
            <Route path="/doctors" element={<DoctorsListPage />} />
            <Route path="/doctors/new" element={<DoctorFormPage />} />
            <Route path="/doctors/:id" element={<DoctorProfilePage />} />
            <Route path="/doctors/:id/edit" element={<DoctorFormPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.APPOINTMENT_VIEW} />}>
            <Route path="/appointments" element={<AppointmentsListPage />} />
            <Route path="/appointments/new" element={<AppointmentBookingPage />} />
            <Route path="/appointments/:id" element={<AppointmentDetailPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.EMR_READ} />}>
            <Route path="/emr" element={<MedicalRecordsListPage />} />
            <Route path="/emr/new" element={<MedicalRecordFormPage />} />
            <Route path="/emr/:id" element={<MedicalRecordDetailPage />} />
            <Route path="/emr/:id/edit" element={<MedicalRecordFormPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.PRESCRIPTION_READ} />}>
            <Route path="/prescriptions" element={<PrescriptionsListPage />} />
            <Route path="/prescriptions/new" element={<PrescriptionFormPage />} />
            <Route path="/prescriptions/:id" element={<PrescriptionDetailPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.LAB_ORDER_READ} />}>
            <Route path="/laboratory" element={<LabOrdersListPage />} />
            <Route path="/laboratory/new" element={<LabOrderFormPage />} />
            <Route path="/laboratory/:id" element={<LabOrderDetailPage />} />
            <Route path="/laboratory/:id/report" element={<LabReportPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.PHARMACY_READ} />}>
            <Route path="/pharmacy" element={<MedicinesListPage />} />
            <Route path="/pharmacy/dispense" element={<DispensePage />} />
            <Route path="/pharmacy/:id" element={<MedicineDetailPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.BILLING_READ} />}>
            <Route path="/billing" element={<BillingListPage />} />
            <Route path="/billing/new" element={<BillFormPage />} />
            <Route path="/billing/:id" element={<BillDetailPage />} />
            <Route path="/billing/:id/receipt" element={<BillReceiptPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.IPD_READ} />}>
            <Route path="/ipd" element={<BedBoardPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.SETTINGS_MANAGE} />}>
            <Route path="/settings/audit-logs" element={<AuditLogsPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.USER_MANAGE} />}>
            <Route path="/settings/users" element={<UsersPage />} />
          </Route>

          <Route element={<PermissionRoute permission={PERMISSIONS.REPORT_EXPORT} />}>
            <Route
              path="/reports"
              element={
                <Suspense fallback={<RouteFallback />}>
                  <ReportsDashboardPage />
                </Suspense>
              }
            />
          </Route>
        </Route>
      </Route>

      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}

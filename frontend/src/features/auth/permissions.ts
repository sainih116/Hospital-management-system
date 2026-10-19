/**
 * Permission codes, mirrored from backend/internal/modules/auth/domain/permission.go.
 * The backend's RBAC middleware is the actual enforcement point — these are
 * used only to drive frontend UI gating (show/hide actions and nav items).
 */
export const PERMISSIONS = {
  DASHBOARD_VIEW: "dashboard.view",
  USER_MANAGE: "user.manage",
  DEPARTMENT_MANAGE: "department.manage",
  DOCTOR_READ: "doctor.read",
  DOCTOR_MANAGE: "doctor.manage",
  PATIENT_CREATE: "patient.create",
  PATIENT_READ: "patient.read",
  PATIENT_UPDATE: "patient.update",
  PATIENT_DELETE: "patient.delete",
  EMR_READ: "emr.read",
  EMR_WRITE: "emr.write",
  PRESCRIPTION_READ: "prescription.read",
  PRESCRIPTION_WRITE: "prescription.write",
  APPOINTMENT_VIEW: "appointment.view",
  APPOINTMENT_MANAGE: "appointment.manage",
  LAB_TEST_MANAGE: "labtest.manage",
  LAB_ORDER_READ: "lab.order.read",
  LAB_ORDER_WRITE: "lab.order.write",
  LAB_RESULT_VERIFY: "lab.result.verify",
  PHARMACY_READ: "pharmacy.read",
  PHARMACY_DISPENSE: "pharmacy.dispense",
  BILLING_READ: "billing.read",
  BILLING_MANAGE: "billing.manage",
  IPD_READ: "ipd.read",
  IPD_WRITE: "ipd.write",
  HR_MANAGE: "hr.manage",
  SETTINGS_MANAGE: "settings.manage",
  // No bespoke audit-log permission exists in the blueprint's permission
  // list — the backend gates GET /audit-logs on settings.manage (see
  // auditlogs/routes.go), so the frontend reuses SETTINGS_MANAGE for
  // gating too rather than inventing a permission the backend won't check.
  REPORT_EXPORT: "report.export",
} as const;

export type Permission = (typeof PERMISSIONS)[keyof typeof PERMISSIONS];

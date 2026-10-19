import type { AppointmentStatus } from "@/features/appointments/appointmentsApi";

/**
 * Mirrors backend/internal/modules/appointments/domain's state machine, for
 * UI purposes only (deciding which action buttons to show) — the backend
 * is the actual source of truth and re-validates every transition.
 */
const TRANSITIONS: Record<AppointmentStatus, AppointmentStatus[]> = {
  SCHEDULED: ["CONFIRMED", "CANCELLED", "NO_SHOW"],
  CONFIRMED: ["IN_PROGRESS", "CANCELLED", "NO_SHOW"],
  IN_PROGRESS: ["COMPLETED", "CANCELLED"],
  COMPLETED: [],
  CANCELLED: [],
  NO_SHOW: [],
  RESCHEDULED: [],
};

export function allowedNextStatuses(current: AppointmentStatus): AppointmentStatus[] {
  return TRANSITIONS[current] ?? [];
}

export function isTerminal(status: AppointmentStatus): boolean {
  return TRANSITIONS[status].length === 0;
}

/** Statuses a DOCTOR-role user may set (clinical progression only — never
 * cancel/confirm, and only for their own appointment; the backend enforces
 * ownership). */
export const DOCTOR_ALLOWED_STATUSES: AppointmentStatus[] = ["IN_PROGRESS", "COMPLETED", "NO_SHOW"];

export const STATUS_LABELS: Record<AppointmentStatus, string> = {
  SCHEDULED: "Scheduled",
  CONFIRMED: "Confirmed",
  IN_PROGRESS: "In Progress",
  COMPLETED: "Completed",
  CANCELLED: "Cancelled",
  NO_SHOW: "No Show",
  RESCHEDULED: "Rescheduled",
};

export const STATUS_COLORS: Record<AppointmentStatus, "default" | "primary" | "success" | "error" | "warning" | "info"> = {
  SCHEDULED: "info",
  CONFIRMED: "primary",
  IN_PROGRESS: "warning",
  COMPLETED: "success",
  CANCELLED: "error",
  NO_SHOW: "error",
  RESCHEDULED: "default",
};

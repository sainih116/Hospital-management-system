import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope } from "@/types/api";

export interface Summary {
  totalPatients: number;
  totalDoctors: number;
  appointmentsToday: number;
  totalRevenueCollected: number;
  outstandingBalance: number;
  lowStockMedicineCount: number;
  pendingLabVerifications: number;
  activePrescriptions: number;
}

export interface StatusCount {
  status: string;
  count: number;
}

export interface TrendPoint {
  date: string;
  value: number;
}

export interface DoctorWorkload {
  doctorId: string;
  doctorName: string;
  count: number;
}

export interface NameCount {
  name: string;
  count: number;
}

export interface DateRangeParams {
  from?: string;
  to?: string;
}

export interface LimitedRangeParams extends DateRangeParams {
  limit?: number;
}

async function summary(): Promise<Summary> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<Summary>>("/reports/summary");
  return data.data;
}

async function appointmentsByStatus(params: DateRangeParams): Promise<StatusCount[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ items: StatusCount[] }>>("/reports/appointments-by-status", { params });
  return data.data.items;
}

async function revenueTrend(params: DateRangeParams): Promise<TrendPoint[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ items: TrendPoint[] }>>("/reports/revenue-trend", { params });
  return data.data.items;
}

async function patientRegistrationTrend(params: DateRangeParams): Promise<TrendPoint[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ items: TrendPoint[] }>>("/reports/patient-registrations-trend", { params });
  return data.data.items;
}

async function doctorWorkload(params: LimitedRangeParams): Promise<DoctorWorkload[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ items: DoctorWorkload[] }>>("/reports/doctor-workload", { params });
  return data.data.items;
}

async function labTestVolume(params: LimitedRangeParams): Promise<NameCount[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ items: NameCount[] }>>("/reports/lab-test-volume", { params });
  return data.data.items;
}

/**
 * Downloads the full report set as a single .xlsx workbook. Uses
 * responseType "blob" (not JSON) since this endpoint returns a raw file,
 * not an envelope — the caller is responsible for triggering the actual
 * browser save (see ReportsDashboardPage's handleExport).
 */
async function exportWorkbook(params: DateRangeParams): Promise<Blob> {
  const { data } = await apiClient.get<Blob>("/reports/export", { params, responseType: "blob" });
  return data;
}

export const reportsApi = {
  summary,
  appointmentsByStatus,
  revenueTrend,
  patientRegistrationTrend,
  doctorWorkload,
  labTestVolume,
  exportWorkbook,
};

import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";

export type AppointmentStatus =
  | "SCHEDULED"
  | "CONFIRMED"
  | "IN_PROGRESS"
  | "COMPLETED"
  | "CANCELLED"
  | "NO_SHOW"
  | "RESCHEDULED";

export interface Appointment {
  id: string;
  patientId: string;
  patientName: string;
  doctorId: string;
  doctorName: string;
  startAt: string;
  endAt: string;
  status: AppointmentStatus;
  reason?: string;
  notes?: string;
  cancelReason?: string;
  rescheduledFromId?: string;
  rescheduledToId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface BookAppointmentPayload {
  patientId: string;
  doctorId: string;
  startAt: string;
  endAt: string;
  reason?: string;
}

export interface ChangeStatusPayload {
  status: Exclude<AppointmentStatus, "SCHEDULED" | "RESCHEDULED">;
  reason?: string;
}

export interface ReschedulePayload {
  startAt: string;
  endAt: string;
}

export interface SearchAppointmentsParams {
  patientId?: string;
  doctorId?: string;
  status?: AppointmentStatus;
  from?: string;
  to?: string;
  page?: number;
  pageSize?: number;
}

async function search(params: SearchAppointmentsParams): Promise<PaginatedResult<Appointment>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<Appointment>>>("/appointments", { params });
  return data.data;
}

async function getById(id: string): Promise<Appointment> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<Appointment>>(`/appointments/${id}`);
  return data.data;
}

async function book(payload: BookAppointmentPayload): Promise<Appointment> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Appointment>>("/appointments", payload);
  return data.data;
}

async function changeStatus(id: string, payload: ChangeStatusPayload): Promise<Appointment> {
  const { data } = await apiClient.patch<ApiSuccessEnvelope<Appointment>>(`/appointments/${id}/status`, payload);
  return data.data;
}

async function reschedule(id: string, payload: ReschedulePayload): Promise<Appointment> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Appointment>>(`/appointments/${id}/reschedule`, payload);
  return data.data;
}

export const appointmentsApi = { search, getById, book, changeStatus, reschedule };

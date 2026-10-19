import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";

export type PrescriptionStatus = "ACTIVE" | "CANCELLED";

export interface PrescriptionItem {
  medicineName: string;
  dose: string;
  frequency: string;
  duration: string;
  instructions?: string;
}

export interface Prescription {
  id: string;
  patientId: string;
  patientName: string;
  doctorId: string;
  doctorName: string;
  medicalRecordId?: string;
  items: PrescriptionItem[];
  status: PrescriptionStatus;
  issuedAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreatePrescriptionPayload {
  patientId: string;
  doctorId: string;
  medicalRecordId?: string;
  items: PrescriptionItem[];
}

export interface SearchPrescriptionsParams {
  patientId?: string;
  doctorId?: string;
  status?: PrescriptionStatus;
  page?: number;
  pageSize?: number;
}

async function search(params: SearchPrescriptionsParams): Promise<PaginatedResult<Prescription>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<Prescription>>>("/prescriptions", { params });
  return data.data;
}

async function getById(id: string): Promise<Prescription> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<Prescription>>(`/prescriptions/${id}`);
  return data.data;
}

async function create(payload: CreatePrescriptionPayload): Promise<Prescription> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Prescription>>("/prescriptions", payload);
  return data.data;
}

async function cancel(id: string): Promise<Prescription> {
  const { data } = await apiClient.patch<ApiSuccessEnvelope<Prescription>>(`/prescriptions/${id}/status`, { status: "CANCELLED" });
  return data.data;
}

export const prescriptionsApi = { search, getById, create, cancel };

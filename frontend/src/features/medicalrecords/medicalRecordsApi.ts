import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";

export interface MedicalRecord {
  id: string;
  patientId: string;
  patientName: string;
  doctorId: string;
  doctorName: string;
  complaint?: string;
  hpi?: string;
  history?: string;
  examination?: string;
  diagnosis: string;
  treatment?: string;
  followUp?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateMedicalRecordPayload {
  patientId: string;
  doctorId: string;
  complaint?: string;
  hpi?: string;
  history?: string;
  examination?: string;
  diagnosis: string;
  treatment?: string;
  followUp?: string;
}

export type UpdateMedicalRecordPayload = Partial<Omit<CreateMedicalRecordPayload, "patientId" | "doctorId">>;

export interface SearchMedicalRecordsParams {
  patientId?: string;
  doctorId?: string;
  page?: number;
  pageSize?: number;
}

async function search(params: SearchMedicalRecordsParams): Promise<PaginatedResult<MedicalRecord>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<MedicalRecord>>>("/medical-records", { params });
  return data.data;
}

async function getById(id: string): Promise<MedicalRecord> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<MedicalRecord>>(`/medical-records/${id}`);
  return data.data;
}

async function create(payload: CreateMedicalRecordPayload): Promise<MedicalRecord> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<MedicalRecord>>("/medical-records", payload);
  return data.data;
}

async function update(id: string, payload: UpdateMedicalRecordPayload): Promise<MedicalRecord> {
  const { data } = await apiClient.patch<ApiSuccessEnvelope<MedicalRecord>>(`/medical-records/${id}`, payload);
  return data.data;
}

export const medicalRecordsApi = { search, getById, create, update };

import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";

export type Gender = "M" | "F" | "O";
export type PatientType = "OPD" | "IPD";

export interface EmergencyContact {
  name: string;
  relationship?: string;
  phone: string;
}

export interface Insurance {
  provider?: string;
  policyNumber?: string;
}

export interface Patient {
  id: string;
  patientCode: string;
  name: string;
  dob: string; // "YYYY-MM-DD"
  gender: Gender;
  phone: string;
  email?: string;
  address?: string;
  bloodGroup?: string;
  allergies?: string[];
  emergencyContact?: EmergencyContact;
  insurance?: Insurance;
  type: PatientType;
  status: "ACTIVE" | "INACTIVE";
  createdAt: string;
  updatedAt: string;
}

export interface PatientFormValues {
  name: string;
  dob: string;
  gender: Gender;
  phone: string;
  email?: string;
  address?: string;
  bloodGroup?: string;
  allergies?: string[];
  emergencyContact?: EmergencyContact;
  insurance?: Insurance;
  type: PatientType;
}

export interface SearchPatientsParams {
  query?: string;
  gender?: Gender;
  bloodGroup?: string;
  type?: PatientType;
  page?: number;
  pageSize?: number;
}

async function search(params: SearchPatientsParams): Promise<PaginatedResult<Patient>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<Patient>>>("/patients", { params });
  return data.data;
}

async function getById(id: string): Promise<Patient> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<Patient>>(`/patients/${id}`);
  return data.data;
}

async function create(payload: PatientFormValues): Promise<Patient> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Patient>>("/patients", payload);
  return data.data;
}

async function update(id: string, payload: Partial<PatientFormValues>): Promise<Patient> {
  const { data } = await apiClient.patch<ApiSuccessEnvelope<Patient>>(`/patients/${id}`, payload);
  return data.data;
}

async function remove(id: string): Promise<void> {
  await apiClient.delete(`/patients/${id}`);
}

export const patientsApi = { search, getById, create, update, remove };

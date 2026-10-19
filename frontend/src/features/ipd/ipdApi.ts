import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";

export type WardStatus = "ACTIVE" | "INACTIVE";
export type BedStatus = "AVAILABLE" | "OCCUPIED" | "MAINTENANCE";
export type AdmissionStatus = "ADMITTED" | "DISCHARGED" | "CANCELLED";

export interface Ward {
  id: string;
  name: string;
  code: string;
  status: WardStatus;
  createdAt: string;
  updatedAt: string;
}

export interface Bed {
  id: string;
  wardId: string;
  wardName: string;
  bedNumber: string;
  status: BedStatus;
  createdAt: string;
  updatedAt: string;
}

export interface Admission {
  id: string;
  patientId: string;
  patientName: string;
  doctorId: string;
  doctorName: string;
  bedId: string;
  bedNumber: string;
  wardName: string;
  reason?: string;
  status: AdmissionStatus;
  admittedAt: string;
  dischargedAt?: string;
  dischargeNote?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateWardPayload {
  name: string;
  code: string;
}

export interface CreateBedPayload {
  wardId: string;
  bedNumber: string;
}

export interface AdmitPatientPayload {
  patientId: string;
  doctorId: string;
  bedId: string;
  reason?: string;
}

export interface DischargePayload {
  note?: string;
}

export interface SearchAdmissionsParams {
  patientId?: string;
  doctorId?: string;
  status?: AdmissionStatus;
  page?: number;
  pageSize?: number;
}

async function listWards(): Promise<Ward[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ wards: Ward[] }>>("/ipd/wards");
  return data.data.wards;
}

async function createWard(payload: CreateWardPayload): Promise<Ward> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Ward>>("/ipd/wards", payload);
  return data.data;
}

async function listBeds(params: { wardId?: string; status?: BedStatus } = {}): Promise<Bed[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ beds: Bed[] }>>("/ipd/beds", { params });
  return data.data.beds;
}

async function createBed(payload: CreateBedPayload): Promise<Bed> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Bed>>("/ipd/beds", payload);
  return data.data;
}

/** Direct status change (AVAILABLE/MAINTENANCE only) — occupying/freeing a
 * bed as part of an admission goes through admitPatient/dischargePatient
 * instead, never this. */
async function setBedStatus(id: string, status: "AVAILABLE" | "MAINTENANCE"): Promise<Bed> {
  const { data } = await apiClient.patch<ApiSuccessEnvelope<Bed>>(`/ipd/beds/${id}`, { status });
  return data.data;
}

async function admitPatient(payload: AdmitPatientPayload): Promise<Admission> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Admission>>("/ipd/admissions", payload);
  return data.data;
}

async function dischargePatient(admissionId: string, payload: DischargePayload): Promise<Admission> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Admission>>(`/ipd/admissions/${admissionId}/discharge`, payload);
  return data.data;
}

async function getAdmission(id: string): Promise<Admission> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<Admission>>(`/ipd/admissions/${id}`);
  return data.data;
}

async function searchAdmissions(params: SearchAdmissionsParams): Promise<PaginatedResult<Admission>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<Admission>>>("/ipd/admissions", { params });
  return data.data;
}

export const ipdApi = {
  listWards,
  createWard,
  listBeds,
  createBed,
  setBedStatus,
  admitPatient,
  dischargePatient,
  getAdmission,
  searchAdmissions,
};

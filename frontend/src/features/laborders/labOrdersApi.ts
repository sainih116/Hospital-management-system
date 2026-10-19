import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";

export type LabOrderStatus = "ORDERED" | "AWAITING_VERIFICATION" | "COMPLETED" | "CANCELLED";
export type ResultFlag = "NORMAL" | "LOW" | "HIGH" | "CRITICAL";

export interface LabResult {
  id: string;
  resultValue: string;
  flag: ResultFlag;
  notes?: string;
  enteredByName?: string;
  enteredAt: string;
  verifiedByName?: string;
  verifiedAt?: string;
}

export interface LabOrder {
  id: string;
  patientId: string;
  patientName: string;
  doctorId: string;
  doctorName: string;
  testId: string;
  testCode: string;
  testName: string;
  unit?: string;
  normalRange?: string;
  sampleCode: string;
  status: LabOrderStatus;
  result?: LabResult;
  createdAt: string;
  updatedAt: string;
}

export interface CreateLabOrderPayload {
  patientId: string;
  doctorId: string;
  testId: string;
}

export interface EnterResultPayload {
  resultValue: string;
  flag: ResultFlag;
  notes?: string;
}

export interface SearchLabOrdersParams {
  patientId?: string;
  doctorId?: string;
  status?: LabOrderStatus;
  page?: number;
  pageSize?: number;
}

async function search(params: SearchLabOrdersParams): Promise<PaginatedResult<LabOrder>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<LabOrder>>>("/lab-orders", { params });
  return data.data;
}

async function getById(id: string): Promise<LabOrder> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<LabOrder>>(`/lab-orders/${id}`);
  return data.data;
}

async function create(payload: CreateLabOrderPayload): Promise<LabOrder> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<LabOrder>>("/lab-orders", payload);
  return data.data;
}

async function cancel(id: string): Promise<LabOrder> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<LabOrder>>(`/lab-orders/${id}/cancel`);
  return data.data;
}

async function enterResult(id: string, payload: EnterResultPayload): Promise<LabOrder> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<LabOrder>>(`/lab-orders/${id}/result`, payload);
  return data.data;
}

async function verifyResult(resultId: string): Promise<LabOrder> {
  const { data } = await apiClient.patch<ApiSuccessEnvelope<LabOrder>>(`/lab-results/${resultId}/verify`);
  return data.data;
}

async function getReport(id: string): Promise<{ order: LabOrder }> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ order: LabOrder }>>(`/lab-orders/${id}/report`);
  return data.data;
}

export const labOrdersApi = { search, getById, create, cancel, enterResult, verifyResult, getReport };

import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope } from "@/types/api";

export interface LabTest {
  id: string;
  code: string;
  name: string;
  category?: string;
  sampleType?: string;
  unit?: string;
  normalRange?: string;
  cost: number;
  status: "ACTIVE" | "INACTIVE";
  createdAt: string;
  updatedAt: string;
}

export interface CreateLabTestPayload {
  code: string;
  name: string;
  category?: string;
  sampleType?: string;
  unit?: string;
  normalRange?: string;
  cost: number;
}

export interface UpdateLabTestPayload {
  name?: string;
  category?: string;
  sampleType?: string;
  unit?: string;
  normalRange?: string;
  cost?: number;
  status?: "ACTIVE" | "INACTIVE";
}

async function list(): Promise<LabTest[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ labTests: LabTest[] }>>("/lab-tests");
  return data.data.labTests;
}

async function getById(id: string): Promise<LabTest> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<LabTest>>(`/lab-tests/${id}`);
  return data.data;
}

async function create(payload: CreateLabTestPayload): Promise<LabTest> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<LabTest>>("/lab-tests", payload);
  return data.data;
}

async function update(id: string, payload: UpdateLabTestPayload): Promise<LabTest> {
  const { data } = await apiClient.patch<ApiSuccessEnvelope<LabTest>>(`/lab-tests/${id}`, payload);
  return data.data;
}

export const labTestsApi = { list, getById, create, update };

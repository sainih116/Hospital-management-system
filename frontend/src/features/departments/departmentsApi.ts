import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope } from "@/types/api";

export interface Department {
  id: string;
  name: string;
  code: string;
  status: "ACTIVE" | "INACTIVE";
  createdAt: string;
  updatedAt: string;
}

async function list(): Promise<Department[]> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ departments: Department[] }>>("/departments");
  return data.data.departments;
}

async function getById(id: string): Promise<Department> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<Department>>(`/departments/${id}`);
  return data.data;
}

async function create(payload: { name: string; code: string }): Promise<Department> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Department>>("/departments", payload);
  return data.data;
}

async function update(id: string, payload: { name?: string; status?: "ACTIVE" | "INACTIVE" }): Promise<Department> {
  const { data } = await apiClient.patch<ApiSuccessEnvelope<Department>>(`/departments/${id}`, payload);
  return data.data;
}

export const departmentsApi = { list, getById, create, update };

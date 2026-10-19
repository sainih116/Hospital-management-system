import { apiClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";

export interface AvailabilitySlot {
  dayOfWeek: number; // 0=Sunday .. 6=Saturday
  startTime: string; // "HH:MM"
  endTime: string;
}

export interface Doctor {
  id: string;
  userId: string;
  name: string;
  email: string;
  departmentId: string;
  departmentName: string;
  licenseNo: string;
  specialization: string;
  fee: number;
  bio?: string;
  availability?: AvailabilitySlot[];
  status: "ACTIVE" | "INACTIVE";
  createdAt: string;
  updatedAt: string;
}

export interface CreateDoctorPayload {
  userId: string;
  departmentId: string;
  licenseNo: string;
  specialization: string;
  fee: number;
  bio?: string;
  availability?: AvailabilitySlot[];
}

export type UpdateDoctorPayload = Partial<Omit<CreateDoctorPayload, "userId">>;

export interface SearchDoctorsParams {
  query?: string;
  departmentId?: string;
  specialization?: string;
  page?: number;
  pageSize?: number;
}

async function search(params: SearchDoctorsParams): Promise<PaginatedResult<Doctor>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<Doctor>>>("/doctors", { params });
  return data.data;
}

async function getById(id: string): Promise<Doctor> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<Doctor>>(`/doctors/${id}`);
  return data.data;
}

async function create(payload: CreateDoctorPayload): Promise<Doctor> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<Doctor>>("/doctors", payload);
  return data.data;
}

async function update(id: string, payload: UpdateDoctorPayload): Promise<Doctor> {
  const { data } = await apiClient.patch<ApiSuccessEnvelope<Doctor>>(`/doctors/${id}`, payload);
  return data.data;
}

async function remove(id: string): Promise<void> {
  await apiClient.delete(`/doctors/${id}`);
}

async function getSchedule(id: string): Promise<{ doctorId: string; availability: AvailabilitySlot[] }> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<{ doctorId: string; availability: AvailabilitySlot[] }>>(
    `/doctors/${id}/schedule`,
  );
  return data.data;
}

export const doctorsApi = { search, getById, create, update, remove, getSchedule };

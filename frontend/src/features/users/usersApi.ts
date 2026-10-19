import { apiClient } from "@/services/apiClient";
import type { UserProfile } from "@/features/auth/authApi";
import type { ApiSuccessEnvelope, PaginatedResult } from "@/types/api";
import type { RoleCode } from "@/types/roles";

export interface UserSummary {
  id: string;
  name: string;
  email: string;
  role: RoleCode;
  status: "ACTIVE" | "INACTIVE" | "LOCKED";
}

export interface CreateUserPayload {
  name: string;
  email: string;
  password: string;
  roleCode: RoleCode;
}

/**
 * Admin-only. Used mainly to pick an existing DOCTOR-role account when
 * attaching a clinical profile (see PatientFormPage's doctor counterpart,
 * DoctorFormPage), and by the Staff/Users management page.
 */
async function list(params: { role?: RoleCode; page?: number; pageSize?: number }): Promise<PaginatedResult<UserSummary>> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<PaginatedResult<UserSummary>>>("/users", { params });
  return data.data;
}

/**
 * Admin-only. Creates a new staff account — the backend endpoint this
 * calls (POST /auth/register) already existed; only the Staff/Users page
 * consuming it is new. Returns the same UserProfile shape as GET /auth/me.
 */
async function create(payload: CreateUserPayload): Promise<UserProfile> {
  const { data } = await apiClient.post<ApiSuccessEnvelope<UserProfile>>("/auth/register", payload);
  return data.data;
}

export const usersApi = { list, create };

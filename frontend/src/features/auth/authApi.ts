import { apiClient, rawClient } from "@/services/apiClient";
import type { ApiSuccessEnvelope } from "@/types/api";
import type { RoleCode } from "@/types/roles";

export interface UserProfile {
  id: string;
  name: string;
  email: string;
  role: RoleCode;
  permissions: string[];
  status: string;
  createdAt: string;
}

export interface LoginResult {
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  user: UserProfile;
}

async function login(email: string, password: string): Promise<LoginResult> {
  const { data } = await rawClient.post<ApiSuccessEnvelope<LoginResult>>("/auth/login", { email, password });
  return data.data;
}

async function me(): Promise<UserProfile> {
  const { data } = await apiClient.get<ApiSuccessEnvelope<UserProfile>>("/auth/me");
  return data.data;
}

async function logout(refreshToken: string): Promise<void> {
  await rawClient.post("/auth/logout", { refreshToken });
}

async function forgotPassword(email: string): Promise<void> {
  await rawClient.post("/auth/password/forgot", { email });
}

async function resetPassword(token: string, newPassword: string): Promise<void> {
  await rawClient.post("/auth/password/reset", { token, newPassword });
}

export const authApi = { login, me, logout, forgotPassword, resetPassword };

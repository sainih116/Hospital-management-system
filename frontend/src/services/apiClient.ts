import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";

import type { ApiErrorBody } from "@/types/api";

const ACCESS_TOKEN_STORAGE_KEY = "hms.accessToken";
const REFRESH_TOKEN_STORAGE_KEY = "hms.refreshToken";
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_STORAGE_KEY);
}

export function setAccessToken(token: string | null): void {
  if (token) {
    localStorage.setItem(ACCESS_TOKEN_STORAGE_KEY, token);
  } else {
    localStorage.removeItem(ACCESS_TOKEN_STORAGE_KEY);
  }
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_STORAGE_KEY);
}

export function setRefreshToken(token: string | null): void {
  if (token) {
    localStorage.setItem(REFRESH_TOKEN_STORAGE_KEY, token);
  } else {
    localStorage.removeItem(REFRESH_TOKEN_STORAGE_KEY);
  }
}

export function setSession(accessToken: string, refreshToken: string): void {
  setAccessToken(accessToken);
  setRefreshToken(refreshToken);
}

export function clearSession(): void {
  setAccessToken(null);
  setRefreshToken(null);
}

/** Normalized client-side error shape derived from the backend envelope. */
export class ApiRequestError extends Error {
  code: string;
  status?: number;
  details?: unknown;

  constructor(body: ApiErrorBody, status?: number) {
    super(body.message);
    this.name = "ApiRequestError";
    this.code = body.code;
    this.status = status;
    this.details = body.details;
  }
}

function toApiRequestError(error: AxiosError<{ error?: ApiErrorBody }>): ApiRequestError {
  const body = error.response?.data?.error;
  if (body) {
    return new ApiRequestError(body, error.response?.status);
  }
  return new ApiRequestError(
    { code: "NETWORK_ERROR", message: error.message || "Network error. Please try again." },
    error.response?.status,
  );
}

/**
 * rawClient is used for the handful of calls that must never carry a bearer
 * token or trigger the silent-refresh flow below (login, refresh, logout) —
 * routing them through apiClient would risk infinite retry loops.
 */
export const rawClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15_000,
  headers: { "Content-Type": "application/json" },
});

rawClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ error?: ApiErrorBody }>) => Promise.reject(toApiRequestError(error)),
);

/** Invoked once a refresh attempt has definitively failed, so the app can
 * drop back to a logged-out state (see app/store.ts wiring in App.tsx). */
let sessionExpiredHandler: (() => void) | null = null;
export function setSessionExpiredHandler(handler: () => void): void {
  sessionExpiredHandler = handler;
}

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15_000,
  headers: { "Content-Type": "application/json" },
});

apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getAccessToken();
  if (token) {
    config.headers.set("Authorization", `Bearer ${token}`);
  }
  return config;
});

// Deduplicates concurrent refresh attempts: if five requests 401 at once,
// only one POST /auth/refresh is made and every caller awaits it.
let refreshPromise: Promise<string> | null = null;

async function refreshAccessToken(): Promise<string> {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      const refreshToken = getRefreshToken();
      if (!refreshToken) {
        throw new Error("no refresh token available");
      }
      const { data } = await rawClient.post<{ data: { accessToken: string; refreshToken: string } }>(
        "/auth/refresh",
        { refreshToken },
      );
      setSession(data.data.accessToken, data.data.refreshToken);
      return data.data.accessToken;
    })().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

interface RetriableConfig extends InternalAxiosRequestConfig {
  _retried?: boolean;
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<{ error?: ApiErrorBody }>) => {
    const config = error.config as RetriableConfig | undefined;

    if (error.response?.status === 401 && config && !config._retried) {
      config._retried = true;
      try {
        const newAccessToken = await refreshAccessToken();
        config.headers.set("Authorization", `Bearer ${newAccessToken}`);
        return apiClient.request(config);
      } catch {
        clearSession();
        sessionExpiredHandler?.();
        return Promise.reject(toApiRequestError(error));
      }
    }

    return Promise.reject(toApiRequestError(error));
  },
);

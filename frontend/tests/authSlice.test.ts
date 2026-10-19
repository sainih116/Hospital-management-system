import { configureStore } from "@reduxjs/toolkit";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/features/auth/authApi", () => ({
  authApi: {
    login: vi.fn(),
    me: vi.fn(),
    logout: vi.fn(),
    forgotPassword: vi.fn(),
    resetPassword: vi.fn(),
  },
}));

import { authApi } from "@/features/auth/authApi";
import authReducer, { bootstrapAuth, login, logout, sessionExpired } from "@/features/auth/authSlice";
import { getAccessToken, getRefreshToken } from "@/services/apiClient";

function buildStore() {
  return configureStore({ reducer: { auth: authReducer } });
}

describe("authSlice", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.mocked(authApi.login).mockReset();
    vi.mocked(authApi.me).mockReset();
    vi.mocked(authApi.logout).mockReset();
  });

  it("login stores the session tokens and marks the user authenticated", async () => {
    vi.mocked(authApi.login).mockResolvedValue({
      accessToken: "access-123",
      refreshToken: "refresh-456",
      expiresAt: new Date().toISOString(),
      user: {
        id: "1",
        name: "Dr. Sharma",
        email: "sharma@hms.local",
        role: "DOCTOR",
        permissions: ["dashboard.view", "patient.read", "emr.write"],
        status: "ACTIVE",
        createdAt: new Date().toISOString(),
      },
    });

    const store = buildStore();
    await store.dispatch(login({ email: "sharma@hms.local", password: "secret" }));

    const state = store.getState().auth;
    expect(state.isAuthenticated).toBe(true);
    expect(state.user?.role).toBe("DOCTOR");
    expect(getAccessToken()).toBe("access-123");
    expect(getRefreshToken()).toBe("refresh-456");
  });

  it("login failure surfaces the error message without authenticating", async () => {
    vi.mocked(authApi.login).mockRejectedValue(new Error("network down"));

    const store = buildStore();
    await store.dispatch(login({ email: "x@hms.local", password: "wrong" }));

    const state = store.getState().auth;
    expect(state.isAuthenticated).toBe(false);
    expect(state.status).toBe("failed");
  });

  it("bootstrapAuth resolves to an unauthenticated state when no token is stored", async () => {
    const store = buildStore();
    await store.dispatch(bootstrapAuth());

    const state = store.getState().auth;
    expect(state.bootstrapped).toBe(true);
    expect(state.isAuthenticated).toBe(false);
    expect(authApi.me).not.toHaveBeenCalled();
  });

  it("bootstrapAuth restores the session when a stored token is still valid", async () => {
    localStorage.setItem("hms.accessToken", "stored-token");
    vi.mocked(authApi.me).mockResolvedValue({
      id: "2",
      name: "Nurse Priya",
      email: "priya@hms.local",
      role: "NURSE",
      permissions: ["dashboard.view", "patient.read"],
      status: "ACTIVE",
      createdAt: new Date().toISOString(),
    });

    const store = buildStore();
    await store.dispatch(bootstrapAuth());

    const state = store.getState().auth;
    expect(state.bootstrapped).toBe(true);
    expect(state.isAuthenticated).toBe(true);
    expect(state.user?.name).toBe("Nurse Priya");
  });

  it("logout clears the session even if the server call fails", async () => {
    localStorage.setItem("hms.accessToken", "a");
    localStorage.setItem("hms.refreshToken", "r");
    vi.mocked(authApi.logout).mockRejectedValue(new Error("server unreachable"));

    const store = buildStore();
    await store.dispatch(logout());

    expect(getAccessToken()).toBeNull();
    expect(getRefreshToken()).toBeNull();
    expect(store.getState().auth.isAuthenticated).toBe(false);
  });

  it("sessionExpired synchronously drops the authenticated user", () => {
    const store = buildStore();
    store.dispatch(sessionExpired());
    expect(store.getState().auth.isAuthenticated).toBe(false);
  });
});

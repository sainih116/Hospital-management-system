import { createAsyncThunk, createSlice } from "@reduxjs/toolkit";

import { authApi, type UserProfile } from "@/features/auth/authApi";
import {
  ApiRequestError,
  clearSession,
  getAccessToken,
  getRefreshToken,
  setSession,
} from "@/services/apiClient";
import type { RoleCode } from "@/types/roles";

export interface AuthUser {
  id: string;
  name: string;
  email: string;
  role: RoleCode;
  permissions: string[];
  status: string;
}

interface AuthState {
  user: AuthUser | null;
  isAuthenticated: boolean;
  /** Whether the initial "restore session from storage" attempt has completed. */
  bootstrapped: boolean;
  status: "idle" | "loading" | "succeeded" | "failed";
  error: string | null;
}

const initialState: AuthState = {
  user: null,
  isAuthenticated: false,
  bootstrapped: false,
  status: "idle",
  error: null,
};

function toAuthUser(profile: UserProfile): AuthUser {
  return {
    id: profile.id,
    name: profile.name,
    email: profile.email,
    role: profile.role,
    permissions: profile.permissions,
    status: profile.status,
  };
}

export const login = createAsyncThunk<AuthUser, { email: string; password: string }, { rejectValue: string }>(
  "auth/login",
  async ({ email, password }, { rejectWithValue }) => {
    try {
      const result = await authApi.login(email, password);
      setSession(result.accessToken, result.refreshToken);
      return toAuthUser(result.user);
    } catch (err) {
      const message = err instanceof ApiRequestError ? err.message : "Unable to sign in. Please try again.";
      return rejectWithValue(message);
    }
  },
);

/** Restores a session from a previously stored access token on app load. */
export const bootstrapAuth = createAsyncThunk<AuthUser | null>("auth/bootstrap", async () => {
  if (!getAccessToken()) {
    return null;
  }
  try {
    const profile = await authApi.me();
    return toAuthUser(profile);
  } catch {
    clearSession();
    return null;
  }
});

export const logout = createAsyncThunk<void>("auth/logout", async () => {
  const refreshToken = getRefreshToken();
  if (refreshToken) {
    await authApi.logout(refreshToken).catch(() => undefined);
  }
  clearSession();
});

const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    /** Synchronous, e.g. for the interceptor-driven "session expired" path. */
    sessionExpired(state) {
      state.user = null;
      state.isAuthenticated = false;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(login.pending, (state) => {
        state.status = "loading";
        state.error = null;
      })
      .addCase(login.fulfilled, (state, action) => {
        state.status = "succeeded";
        state.user = action.payload;
        state.isAuthenticated = true;
      })
      .addCase(login.rejected, (state, action) => {
        state.status = "failed";
        state.error = action.payload ?? "Unable to sign in. Please try again.";
      })
      .addCase(bootstrapAuth.fulfilled, (state, action) => {
        state.bootstrapped = true;
        state.user = action.payload;
        state.isAuthenticated = action.payload !== null;
      })
      .addCase(bootstrapAuth.rejected, (state) => {
        state.bootstrapped = true;
        state.user = null;
        state.isAuthenticated = false;
      })
      .addCase(logout.fulfilled, (state) => {
        state.user = null;
        state.isAuthenticated = false;
        state.status = "idle";
      });
  },
});

export const { sessionExpired } = authSlice.actions;
export default authSlice.reducer;

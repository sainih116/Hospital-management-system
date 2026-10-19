import { configureStore } from "@reduxjs/toolkit";
import { render, screen } from "@testing-library/react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import App from "@/App";
import authReducer from "@/features/auth/authSlice";
import uiReducer from "@/features/ui/uiSlice";

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

function renderApp(initialPath: string) {
  const store = configureStore({ reducer: { auth: authReducer, ui: uiReducer } });
  return render(
    <Provider store={store}>
      <MemoryRouter initialEntries={[initialPath]}>
        <App />
      </MemoryRouter>
    </Provider>,
  );
}

describe("App shell", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.mocked(authApi.me).mockReset();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it("redirects an unauthenticated visitor from the dashboard to the login page", async () => {
    renderApp("/dashboard");

    expect(await screen.findByRole("heading", { name: /hms/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /sign in/i })).toBeInTheDocument();
  });

  it("renders the dashboard shell for a restored (authenticated) session", async () => {
    localStorage.setItem("hms.accessToken", "fake-access-token");
    vi.mocked(authApi.me).mockResolvedValue({
      id: "1",
      name: "Ada Admin",
      email: "ada@hms.local",
      role: "ADMIN",
      permissions: ["dashboard.view", "patient.read"],
      status: "ACTIVE",
      createdAt: new Date().toISOString(),
    });

    renderApp("/dashboard");

    expect(await screen.findByRole("heading", { name: /dashboard/i })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: /main navigation/i })).toBeInTheDocument();
  });

  it("renders a 404 page for unknown routes regardless of auth state", async () => {
    renderApp("/unknown-route");

    expect(await screen.findByText("404")).toBeInTheDocument();
  });
});

import { configureStore } from "@reduxjs/toolkit";
import { render, screen } from "@testing-library/react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/features/patients/patientsApi", () => ({
  patientsApi: {
    search: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
  },
}));

import authReducer, { type AuthUser } from "@/features/auth/authSlice";
import { patientsApi } from "@/features/patients/patientsApi";
import uiReducer from "@/features/ui/uiSlice";
import PatientsListPage from "@/pages/patients/PatientsListPage";

function renderWithAuth(user: AuthUser | null) {
  const store = configureStore({
    reducer: { auth: authReducer, ui: uiReducer },
    preloadedState: {
      auth: { user, isAuthenticated: user !== null, bootstrapped: true, status: "idle" as const, error: null },
      ui: { sidebarOpen: true, themeMode: "light" as const },
    },
  });

  return render(
    <Provider store={store}>
      <MemoryRouter initialEntries={["/patients"]}>
        <PatientsListPage />
      </MemoryRouter>
    </Provider>,
  );
}

const receptionist: AuthUser = {
  id: "1",
  name: "Amit",
  email: "amit@hms.local",
  role: "RECEPTIONIST",
  permissions: ["dashboard.view", "patient.create", "patient.read"],
  status: "ACTIVE",
};

const doctor: AuthUser = {
  id: "2",
  name: "Dr. Sharma",
  email: "sharma@hms.local",
  role: "DOCTOR",
  permissions: ["dashboard.view", "patient.read"],
  status: "ACTIVE",
};

describe("PatientsListPage", () => {
  beforeEach(() => {
    vi.mocked(patientsApi.search).mockReset();
  });

  it("shows the New Patient button for a role with patient.create", async () => {
    vi.mocked(patientsApi.search).mockResolvedValue({ items: [], page: 1, pageSize: 20, total: 0, totalPages: 0 });

    renderWithAuth(receptionist);

    expect(await screen.findByRole("button", { name: /new patient/i })).toBeInTheDocument();
  });

  it("hides the New Patient button for a role without patient.create", async () => {
    vi.mocked(patientsApi.search).mockResolvedValue({ items: [], page: 1, pageSize: 20, total: 0, totalPages: 0 });

    renderWithAuth(doctor);

    await screen.findByText(/no patients registered yet/i);
    expect(screen.queryByRole("button", { name: /new patient/i })).not.toBeInTheDocument();
  });

  it("renders returned patients in the table", async () => {
    vi.mocked(patientsApi.search).mockResolvedValue({
      items: [
        {
          id: "p1",
          patientCode: "PAT-10001",
          name: "Rohan Kumar",
          dob: "1992-04-15",
          gender: "M",
          phone: "9000000001",
          type: "OPD",
          status: "ACTIVE",
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
      ],
      page: 1,
      pageSize: 20,
      total: 1,
      totalPages: 1,
    });

    renderWithAuth(doctor);

    expect(await screen.findByText("PAT-10001")).toBeInTheDocument();
    expect(screen.getByText("Rohan Kumar")).toBeInTheDocument();
  });
});

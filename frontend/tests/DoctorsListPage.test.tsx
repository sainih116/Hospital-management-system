import { configureStore } from "@reduxjs/toolkit";
import { render, screen } from "@testing-library/react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/features/doctors/doctorsApi", () => ({
  doctorsApi: {
    search: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
    getSchedule: vi.fn(),
  },
}));

vi.mock("@/features/departments/departmentsApi", () => ({
  departmentsApi: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
  },
}));

import authReducer, { type AuthUser } from "@/features/auth/authSlice";
import { departmentsApi } from "@/features/departments/departmentsApi";
import { doctorsApi } from "@/features/doctors/doctorsApi";
import uiReducer from "@/features/ui/uiSlice";
import DoctorsListPage from "@/pages/doctors/DoctorsListPage";

function renderWithAuth(user: AuthUser) {
  const store = configureStore({
    reducer: { auth: authReducer, ui: uiReducer },
    preloadedState: {
      auth: { user, isAuthenticated: true, bootstrapped: true, status: "idle" as const, error: null },
      ui: { sidebarOpen: true, themeMode: "light" as const },
    },
  });

  return render(
    <Provider store={store}>
      <MemoryRouter initialEntries={["/doctors"]}>
        <DoctorsListPage />
      </MemoryRouter>
    </Provider>,
  );
}

const admin: AuthUser = {
  id: "1",
  name: "Ada Admin",
  email: "ada@hms.local",
  role: "ADMIN",
  permissions: ["dashboard.view", "doctor.read", "doctor.manage"],
  status: "ACTIVE",
};

const receptionist: AuthUser = {
  id: "2",
  name: "Amit",
  email: "amit@hms.local",
  role: "RECEPTIONIST",
  permissions: ["dashboard.view", "doctor.read"],
  status: "ACTIVE",
};

describe("DoctorsListPage", () => {
  beforeEach(() => {
    vi.mocked(doctorsApi.search).mockReset();
    vi.mocked(departmentsApi.list).mockReset();
    vi.mocked(departmentsApi.list).mockResolvedValue([]);
  });

  it("shows the New Doctor button for a role with doctor.manage", async () => {
    vi.mocked(doctorsApi.search).mockResolvedValue({ items: [], page: 1, pageSize: 20, total: 0, totalPages: 0 });

    renderWithAuth(admin);

    expect(await screen.findByRole("button", { name: /new doctor/i })).toBeInTheDocument();
  });

  it("hides the New Doctor button for a role without doctor.manage", async () => {
    vi.mocked(doctorsApi.search).mockResolvedValue({ items: [], page: 1, pageSize: 20, total: 0, totalPages: 0 });

    renderWithAuth(receptionist);

    await screen.findByText(/no doctors match your search/i);
    expect(screen.queryByRole("button", { name: /new doctor/i })).not.toBeInTheDocument();
  });

  it("renders returned doctors in the table", async () => {
    vi.mocked(doctorsApi.search).mockResolvedValue({
      items: [
        {
          id: "d1",
          userId: "u1",
          name: "Dr. Anil Sharma",
          email: "anil.sharma@hms.local",
          departmentId: "dept1",
          departmentName: "Heart Institute",
          licenseNo: "LIC-CARD-001",
          specialization: "Cardiology",
          fee: 800,
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

    renderWithAuth(admin);

    expect(await screen.findByText("Dr. Anil Sharma")).toBeInTheDocument();
    expect(screen.getByText("Heart Institute")).toBeInTheDocument();
  });
});

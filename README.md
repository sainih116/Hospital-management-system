
# Hospital Management System (HMS)

Production-grade Hospital Management System built with a Go/Gin/MongoDB backend
and a React/TypeScript/MUI frontend, developed phase by phase.

## Tech Stack

| Layer    | Technology |
|----------|------------|
| Backend  | Go 1.24+, Gin, MongoDB, JWT, Redis (optional cache), Swagger/OpenAPI, Docker |
| Frontend | React 19, TypeScript, Vite, MUI, Redux Toolkit, React Router, Axios, React Hook Form, Zod |
| Architecture | Clean Architecture, DDD, Repository Pattern, Service Layer, Dependency Injection (backend) · Feature-based, reusable components (frontend) |

## Repository Layout

```
project-root/
├── backend/
│   ├── cmd/api/            # main.go entry point
│   ├── internal/           # config, logger, database, middleware, server, handlers, response, apperror
│   ├── pkg/                # shared, reusable non-domain utilities
│   ├── configs/            # non-secret static configuration
│   ├── docs/                # generated Swagger (swag init) — do not hand-edit
│   ├── deployments/         # staging/production deployment manifests (Phase 11)
│   ├── scripts/             # dev/ops scripts (seed loaders, etc.)
│   ├── tests/                # cross-cutting integration tests
│   └── docker/               # Dockerfile
└── frontend/
    ├── src/
    │   ├── app/              # Redux store + typed hooks
    │   ├── components/       # shared/reusable UI (layout, common)
    │   ├── features/         # feature-scoped Redux slices
    │   ├── pages/             # route-level screens
    │   ├── routes/            # route table + guards
    │   ├── services/          # API clients
    │   ├── theme/              # MUI theme
    │   └── types/               # shared TS types
    ├── public/
    └── tests/
```

## Phase Status

- [x] **Phase 0 — Project Foundation**: repo scaffolding, Gin server, MongoDB/Redis wiring, centralized logging & error handling, health/readiness endpoints, Swagger setup, Docker, and the React dashboard shell (theme, Redux store, routing, sidebar/navbar).
- [x] **Phase 1 — Authentication & RBAC**: JWT access + rotating/revocable refresh tokens, bcrypt password hashing, RBAC middleware enforcing the six-role permission matrix, register (Admin-only)/login/refresh/logout/forgot-password/reset-password/me, `cmd/seed` bootstrap command, and a working login → session-restore → protected-route → logout flow on the frontend.
- [x] **Phase 2 — Patient Management**: full CRUD (soft-delete) with sequential `PAT-NNNNN` codes, paginated/filterable search, RBAC-gated create/update/delete, every mutation written to `audit_logs`, and a working patient list/registration/profile/edit UI with permission-gated actions.
- [x] **Phase 3 — Doctor Management**: a new `departments` reference module (Admin-managed) plus Doctors — clinical profiles layered on existing DOCTOR-role user accounts, with department/specialization/fee/weekly-availability, RBAC-gated CRUD, a dedicated schedule endpoint, and a working doctors directory/profile/create-edit UI (department management UI under Settings). Also added the small Admin-only `GET /users` endpoint this phase's "attach a profile to an existing account" flow needed.
- [x] **Phase 4 — Appointment Management**: booking with patient/doctor/availability validation, a documented double-booking defense appropriate for a standalone (non-replica-set) MongoDB deployment, the full SCHEDULED→CONFIRMED→IN_PROGRESS→COMPLETED state machine with CANCELLED/NO_SHOW/RESCHEDULED alternate paths, reschedule-as-a-new-record with safe rollback, fine-grained Doctor-can-progress-their-own-appointments authorization, and a working booking/list/detail UI with role-aware action buttons.
- [x] **Phase 5 — EMR Module**: Medical Records (complaint/HPI/history/examination/diagnosis/treatment/follow-up, amendable but never deleted, `?patientId=` search doubling as consultation history) and Prescriptions (free-text medicine items pending Phase 7's catalog, optional link back to the clinical note, ACTIVE→CANCELLED only — DISPENSED is reserved for Pharmacy), both with RBAC already correctly seeded from Phase 1, and working list/create/detail UI including a dynamic medicine-item form.
- [x] **Phase 6 — Laboratory Module**: a new `lab-tests` catalog module (Admin-managed, same pattern as Departments) plus Lab Orders — order → result entry → verification → printable report, with a deliberately narrow `lab.result.verify` permission (Admin+LabTech only, excluding Doctor per the blueprint's REST table — an explicit exception to the module's general Doctor+LabTech grant), atomic sample-code generation, and a browser-print-based report (no server PDF library).
- [x] **Phase 7 — Pharmacy Module**: a `medicines` catalog (name/generic name/category/manufacturer/reorder level, materialized `currentStock` for O(1) low-stock queries) plus batch-based inventory (`inventory` collection) and a `stock_movements` ledger, and FEFO (first-expiry-first-out) dispensing against Prescriptions that marks them `DISPENSED` — with a documented compensating-rollback pattern (no Mongo transactions, same constraint as Appointments/Lab Orders) and a working catalog/receive-batch/dispense UI.
- [x] **Phase 8 — Billing Module**: itemized bills (sequential `INV-NNNNN` numbers, free-text/categorized charge lines, discount/tax) against a patient, a `payments` ledger supporting partial payments with atomic overpayment-proof recording, an UNPAID→PARTIALLY_PAID→PAID status derived from `amountPaid` on every payment, cancellation restricted to bills with no recorded payments, an outstanding-balance search filter, and a working bill list/create/detail UI with payment recording.
- [x] **Phase 9 — Audit Logging**: a read-only query/reporting API (`GET /audit-logs`, filterable by actor/entity type/entity id/action/date range) over the `audit_logs` collection every module has been writing to since Phase 2 via `internal/audit.Recorder` — no new write path, no new domain model, just search on data that already existed. Gated on the existing Admin-only `settings.manage` permission rather than a new one, since the blueprint's permission list never named a dedicated audit-read code. A working filterable log viewer under Settings rounds out the phase.
- [x] **Phase 10 — Reporting Dashboard**: a `reports` module running live MongoDB aggregation pipelines (no materialized/pre-computed report tables) across every operational collection — a summary snapshot (patients, doctors, today's appointments, revenue collected, outstanding balance, low-stock medicines, pending lab verifications, active prescriptions) plus five date-ranged breakdowns (appointments by status, revenue trend, patient registration trend, doctor workload, lab test volume) — gated on the existing Admin-only `report.export` permission, and a Recharts-based dashboard UI (pie/area/line/bar charts, code-split from the main bundle since recharts is a ~400KB dependency only this one Admin-only page needs). The Phase 0 dashboard shell's placeholder KPI cards are also wired to live data for users who hold `report.export`.
- [x] **Post-launch addition — IPD/Wards, Payment Receipts, Report Exports**: three gaps identified against the original blueprint after Phase 10, added without touching any prior module's API contract. **IPD/Wards** is a full new module (Wards, Beds, Admissions) built on the `ipd.read`/`ipd.write` permissions already seeded in Phase 1 — a Ward/Bed board UI where admitting a patient atomically occupies a bed (a guard-filter update, not a transaction, closing the exact "two admissions, one bed" race the blueprint's critical-test-case list calls out) and discharging releases it, with the same compensating-rollback pattern used by Appointments/Pharmacy if admission creation fails after the bed was already occupied. **Payment receipts** reuse the existing `GET /billing/bills/{id}` response as-is — a pure frontend print view, no new endpoint, same browser-print pattern as the Lab report. **Report exports** add one `GET /reports/export` endpoint producing a multi-sheet `.xlsx` workbook (via `excelize`, the only new backend dependency added in this pass) covering every metric the dashboard shows, plus a browser print-to-PDF action on the dashboard itself — deliberately not a server-rendered PDF, for the same reason Lab/Billing already avoid one.
- [ ] Phase 11 — Deployment

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 20+ and npm
- MongoDB (local or Docker)
- Redis (optional; only needed if `REDIS_ENABLED=true`)
- Docker + Docker Compose (for containerized runs)

### Backend

```bash
cd backend
cp .env.example .env      # edit as needed
go mod tidy
go run ./cmd/api
```

- Health check: `GET http://localhost:5000/health`
- Readiness check: `GET http://localhost:5000/health/ready`
- Swagger UI: `http://localhost:5000/swagger/index.html`

Regenerate Swagger docs after adding/changing `@`-annotated handlers:

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.3
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

Run tests:

```bash
go test ./...
```

Bootstrap the database (RBAC roles + the initial Admin account — idempotent, safe to re-run):

```bash
go run ./cmd/seed
```

This upserts the six roles (`ADMIN`, `DOCTOR`, `NURSE`, `RECEPTIONIST`, `PHARMACIST`, `LAB_TECHNICIAN`) with the permissions from `internal/modules/auth/domain/permission.go`, and — only if no user exists yet with that email — creates an Admin at `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` (defaults: `admin@hms.local` / `ChangeMe123!`). It also creates the blueprint's sample departments (Cardiology, General Medicine, Pediatrics, Laboratory, Pharmacy, Emergency) and three sample doctors (Dr. Anil Sharma / Cardiology, Dr. Neha Verma / General Medicine, Dr. Rahul Mehta / Pediatrics), each a DOCTOR-role login account (password `SEED_DOCTOR_PASSWORD`, same default as the admin) plus its linked clinical profile with a Mon–Fri 09:00–17:00 schedule. **Change these default passwords immediately in any non-local environment.** Every other user is created afterwards via `POST /auth/register`, which requires an authenticated Admin (`user.manage` permission) — there is no public self-registration endpoint.

### Frontend

```bash
cd frontend
cp .env.example .env.local  # edit as needed
npm install
npm run dev
```

- App: `http://localhost:5173`
- Run tests: `npm run test`
- Lint: `npm run lint`
- Production build: `npm run build`

### Full stack via Docker Compose

```bash
docker compose up --build
```

- Frontend: `http://localhost:8080`
- Backend API: `http://localhost:5000`
- MongoDB: `localhost:27017`
- Redis: `localhost:6379`

## Environment Variables

### Backend (`backend/.env`)

| Variable | Default | Notes |
|---|---|---|
| `APP_ENV` | `development` | `production` enforces JWT secrets are set |
| `PORT` | `5000` | HTTP listen port |
| `APP_BASE_URL` | `http://localhost:5000` | Used for building absolute links |
| `FRONTEND_BASE_URL` | `http://localhost:5173` | Used to build the password-reset link sent to users |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | Comma-separated CORS allow-list |
| `LOG_LEVEL` | `info` (dev: `debug`) | `debug`\|`info`\|`warn`\|`error` |
| `MONGO_URI` | `mongodb://localhost:27017` | |
| `MONGO_DATABASE` | `hms` | |
| `MONGO_CONNECT_TIMEOUT_SECONDS` | `10` | |
| `REDIS_ENABLED` | `false` | App runs without a cache layer when `false` |
| `REDIS_ADDR` | `localhost:6379` | |
| `REDIS_PASSWORD` | *(empty)* | |
| `REDIS_DB` | `0` | |
| `JWT_ACCESS_SECRET` | *(dev default; required in prod)* | |
| `JWT_REFRESH_SECRET` | *(dev default; required in prod)* | |
| `JWT_ACCESS_TTL_MINUTES` | `15` | |
| `JWT_REFRESH_TTL_MINUTES` | `10080` (7 days) | |
| `SHUTDOWN_TIMEOUT_SECONDS` | `15` | Graceful shutdown grace period |
| `SEED_ADMIN_EMAIL` | `admin@hms.local` | `cmd/seed` only — initial Admin login |
| `SEED_ADMIN_NAME` | `System Administrator` | `cmd/seed` only |
| `SEED_ADMIN_PASSWORD` | `ChangeMe123!` | `cmd/seed` only — change before any shared/prod use |
| `SEED_DOCTOR_PASSWORD` | `ChangeMe123!` | `cmd/seed` only — password for the 3 sample doctor accounts |

### Frontend (`frontend/.env.local`)

| Variable | Default | Notes |
|---|---|---|
| `VITE_API_BASE_URL` | `/api/v1` (dev proxy) | Base URL the Axios client targets |

## Conventions

- API responses use a consistent envelope: `{ success, data, meta }` on success, `{ success: false, error: { code, message, details } }` on failure — see `backend/internal/response`.
- Every domain error is represented as an `*apperror.AppError` (`backend/internal/apperror`) so handlers never leak raw Go errors to clients.
- All protected endpoints flow through: `middleware.Authenticate` (JWT + active-account check) → `middleware.RequirePermission`/`RequireRole` → handler binds/validates the request → service executes the business rule → sensitive mutations are recorded via `audit.Recorder` (`backend/internal/audit`) into `audit_logs`. Phase 9 adds the query/reporting API and admin UI on top of that same collection — the write path already exists from Phase 2 onward.
- Business modules are organized Clean-Architecture style under `backend/internal/modules/<module>/{domain,dto,repository,service,handler}`, each exposing a `RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) Module` that the composition root (`internal/server/router.go`) wires up — see `internal/modules/auth` as the reference implementation.
- Frontend features are organized by domain under `src/features/*`; shared, reusable UI lives under `src/components/*`.

## Authentication & RBAC (Phase 1)

- **Roles**: `ADMIN`, `DOCTOR`, `NURSE`, `RECEPTIONIST`, `PHARMACIST`, `LAB_TECHNICIAN`. Permission codes and the default role→permission matrix live in `backend/internal/modules/auth/domain/permission.go` and are written to the `roles` collection by `cmd/seed`.
- **Access tokens** are short-lived JWTs (`JWT_ACCESS_TTL_MINUTES`, default 15 min) carrying the user's role and permissions, so most requests need zero extra DB round-trips for authorization — `middleware.Authenticate` still re-checks the account is still `ACTIVE` on every request.
- **Refresh tokens** are opaque, single-use, and rotated on every `/auth/refresh` call; only their SHA-256 hash is stored (`refresh_tokens`, TTL-indexed for automatic cleanup). Replaying an already-rotated (revoked) refresh token is treated as a compromised-session signal and revokes every session for that user.
- **Password reset** issues a single-use, 30-minute token (`password_reset_tokens`, also TTL-indexed) delivered via the pluggable `notifier.Notifier` interface — currently `LogNotifier`, which writes the reset link to the application log; swap in an SMTP/SMS-backed implementation in `cmd/api/main.go` without touching any caller.
- **Rate limiting**: `/auth/login`, `/auth/password/forgot` and `/auth/password/reset` are limited per-IP (in-memory token bucket, `internal/middleware/ratelimit.go`) — sized for this project's single-process/modular-monolith deployment model; back it with Redis instead if you ever split into multiple backend instances.
- **Frontend session flow**: `POST /auth/login` → tokens stored in `localStorage` → `App.tsx` restores the session on every load via `GET /auth/me` (gated behind a full-page spinner so `ProtectedRoute` never flash-redirects to `/login`) → `apiClient` transparently refreshes an expired access token once on any `401` and retries the original request → `Navbar`'s logout calls `POST /auth/logout` to revoke the refresh token server-side.
- Both the login response and `GET /auth/me` include the user's `permissions: string[]`; the frontend uses `useHasPermission()` (`src/features/auth/useHasPermission.ts`) purely to gate UI (show/hide buttons and nav items) — the backend's RBAC middleware remains the actual enforcement point, so a tampered/stale client can never do more than the server allows.

## Patient Management (Phase 2)

- **Endpoints**: `POST /patients` (create, requires `patient.create`), `GET /patients` (paginated/filterable search, `patient.read`), `GET /patients/{id}` (profile, `patient.read`), `PATCH /patients/{id}` (partial update, `patient.update`), `DELETE /patients/{id}` (soft-delete, `patient.delete`). Per the RBAC matrix, Admin has full CRUD, Receptionist has create+read only, every other clinical role has read-only access.
- **Patient codes** (`PAT-10001`, `PAT-10002`, ...) are generated atomically via `internal/pkg/sequence` (a MongoDB `counters` collection with an atomic `$inc`), the same mechanism later phases reuse for lab sample codes and invoice numbers.
- **Deletion is a soft-delete**: a `DELETE` sets `deletedAt`/`status: INACTIVE` rather than removing the document, since medical/appointment/billing history in later phases will reference patients by ID and retention is expected for compliance. Deleted patients are excluded from search/get by default but the data is retained.
- **Search** supports free-text `query` (matched against name/patient code/phone) plus `gender`, `bloodGroup` and `type` filters, with standard `page`/`pageSize` pagination (`internal/pkg/pagination`, reused by every future list endpoint).
- **Validation split**: structural validation (required fields, enums, string lengths) lives in request-binding tags; business-rule validation that binding tags can't express (date-of-birth not in the future/implausible, phone format) lives in the service layer — see `internal/modules/patients/service/patient_service.go`.

## Doctor & Department Management (Phase 3)

- **Departments** (`internal/modules/departments`) are simple reference data (`GET /departments` — any authenticated user, needed for dropdowns everywhere; `POST`/`PATCH` — Admin-only `department.manage`). Never hard-deleted, only deactivated, so existing doctor records always resolve.
- **Doctors are profiles layered on existing accounts, not a parallel identity system**: `POST /doctors` (Admin-only `doctor.manage`) requires an existing, active user with the `DOCTOR` role and links a clinical profile (department, license number, specialization, consultation fee, bio, weekly availability) to it — enforced with a one-to-one `userId` unique index. This is why `GET /users?role=DOCTOR` (Admin-only `user.manage`, added this phase) exists: the Doctor-creation screen needs it to populate the "which account is this?" dropdown.
- **Denormalized reads, by design**: a doctor's `name`/`email` are snapshotted from the linked user at creation (there is no user-profile-edit feature yet, so staleness risk is currently zero), and `departmentName` is re-snapshotted whenever the doctor's department assignment changes — both to keep list/search endpoints join-free. Documented as a deliberate tradeoff in `internal/modules/doctors/domain/doctor.go`.
- **Availability** is a recurring weekly schedule (`{dayOfWeek, startTime, endTime}[]`, `time.Weekday`-numbered), exposed both inline on the doctor profile and via the spec's dedicated `GET /doctors/{id}/schedule`. Dated exceptions (leave, on-call overrides) are out of scope here — Phase 4 (Appointments) revisits this only if conflict-checking needs finer granularity.
- **Deletion is a soft-delete**, same rationale and mechanics as Patients.

## Appointment Management (Phase 4)

- **Double-booking prevention is deliberately two-layered**, because this deployment runs MongoDB standalone (see root `docker-compose.yml`) rather than as a replica set, so multi-document ACID transactions (`session.WithTransaction`) are not available:
  1. A pre-insert overlap query (`HasConflict`) rejects any new/rescheduled time range that intersects another calendar-holding appointment for the same doctor — the general "overlapping appointments" case.
  2. A **partial unique index** on `(doctorId, startAt)`, scoped to calendar-holding statuses, is the atomic backstop for the literal race in the blueprint's test case ("two users book the same doctor/time slot simultaneously"). Unique-index enforcement is atomic per document even without transactions, so of two concurrent requests for the identical slot, exactly one insert succeeds.
  - The gap this leaves — two *overlapping-but-not-identical* start times raced in the same instant — is an accepted tradeoff of a single-node deployment; full documentation and reasoning is in `internal/modules/appointments/repository/appointment_repository.go`'s package doc.
- **State machine**: `SCHEDULED → CONFIRMED → IN_PROGRESS → COMPLETED`, with `CANCELLED`/`NO_SHOW` reachable from any non-terminal state and enforced by `domain.Status.CanTransitionTo`. **Reschedule is not a same-record transition** — it marks the original `RESCHEDULED` and creates a new `SCHEDULED` appointment (preserving full history), with the original safely reverted to its previous status if creating the replacement fails for any reason (no transactions means this is an explicit compensating action, not a rollback).
- **Resolving a real spec ambiguity**: the RBAC matrix lists Doctor as "View" only for Appointments, while the REST endpoint table lists Doctor among the roles allowed to `PATCH` status. The reconciliation implemented here — `authorizeStatusChange` in `appointment_service.go` — is that Admin/Receptionist (`appointment.manage`) may make any transition, while a Doctor may only progress *their own* appointments through the clinical states (`IN_PROGRESS`/`COMPLETED`/`NO_SHOW`), never confirm, cancel, or touch another doctor's appointment. The route itself is gated on `appointment.view` (which Doctors have); the finer-grained ownership/transition check happens inside the service.
- **"Doctor schedule view"** from the spec's REST table is just `GET /appointments?doctorId=&from=&to=` — no separate endpoint was needed since Search already covers it.
- Patient/doctor names are snapshotted onto the appointment at booking time, same join-free-reads rationale as Doctors' denormalization.

## EMR Module — Medical Records & Prescriptions (Phase 5)

- **Medical records are amendable, never deleted or replaced in place**: `PATCH /medical-records/{id}` updates the existing document (every change audit-logged as `MEDICAL_RECORD_AMENDED`) rather than versioning or soft-deleting — clinical documentation is corrected, not erased. No ownership restriction is applied (unlike Appointments): any Doctor can read/amend any record, matching the RBAC matrix's blanket role-based grant (continuity-of-care coverage by another physician is standard EMR behavior, not an edge case to lock down).
- **"Consultation history"** from the phase checklist is `GET /medical-records?patientId=` — no bespoke endpoint, same pattern as Appointments' "doctor schedule view".
- **Prescriptions deliberately use free-text medicine names**, not a `medicineId` reference. Now that Phase 7 (Pharmacy) has built the real medicines catalog, the schema stayed additive rather than retrofitting a link: a doctor's free-text order ("Paracetamol 500mg") and a pharmacist's stocked product name won't always match exactly, so Pharmacy's dispense flow lets the pharmacist independently search and pick the actual catalog medicine (and quantity) rather than this module attempting unreliable text-matching. `StatusDispensed` was added to the `Status` enum in Phase 7, set directly by Pharmacy via the repository (this module's own `ChangeStatus` still only ever allows `ACTIVE`→`CANCELLED`).
- Prescriptions may optionally link back to the medical record they were issued from (`medicalRecordId`, validated against the Medical Records module built earlier in this same phase).
- Both modules snapshot patient/doctor names at creation time, the same join-free-reads tradeoff used throughout (Doctors, Appointments).
- File/image attachments (imaging, lab reports) mentioned in the blueprint's broader EMR section were deliberately **not** built this phase — they're outside this phase's stated checklist (Diagnosis, Notes, Prescriptions, Consultation History) and belong with whichever later phase actually needs object storage wiring (most plausibly Lab report generation, Phase 6).

## Laboratory Module (Phase 6)

- **Two modules, not one**, mirroring the Doctors/Departments split from Phase 3: `lab-tests` is a small Admin-managed reference catalog (code, name, category, sample type, unit, normal range, cost — never hard-deleted, only deactivated), while `lab-orders` owns the actual order→result→verification workflow and treats `lab_orders`/`lab_results` as one tightly-coupled domain (verifying a result directly drives its parent order's status) even though they're separate Mongo collections, matching the blueprint's DB design.
- **Status is deliberately just four states** — `ORDERED → AWAITING_VERIFICATION → COMPLETED`, with `CANCELLED` (only before a result exists) as the alternate path. The blueprint's broader workflow narrative mentions granular sample-collection tracking (collected/received/processing); that's out of this phase's actual checklist (Create Lab Order, Result Entry, Report Generation) and was deliberately not built, same scope discipline applied to EMR attachments in Phase 5.
- **`lab.result.verify` is its own permission**, not reused from the general `lab.order.write` grant — the spec's own example permission list (Section 3) names it explicitly, and the REST table restricts verification to Admin/Lab Technician, *excluding* Doctor even though Doctor holds `lab.order.write`. This is the second explicit spec-table exception found so far (the first being Appointments' Doctor-can-progress-own-appointments carve-out in Phase 4) — both are honored precisely rather than smoothed over.
- **Sample codes** (`SMP-00001`, ...) reuse the same atomic Mongo-counter mechanism `internal/pkg/sequence` already built for patient codes in Phase 2.
- **"Report Generation"** is a printable browser page (`GET /lab-orders/{id}/report` returning structured JSON, rendered by the frontend with print-specific CSS and `window.print()`), not a server-rendered PDF binary — avoids adding an untested PDF-generation dependency in this environment while still fully satisfying the requirement for a hospital to produce a physical/PDF report via "print to PDF". The report is only available once a result is verified (`REPORT_NOT_READY` otherwise).
- The acting user's display name (for "Entered by" / "Verified by" on the report) is resolved via a small, justified cross-module dependency on the auth module's `UserRepository` — the JWT claims carry only email/role, and a report reading "Entered by admin@hms.local" is worse than one reading "Entered by Suresh Kumar".

## Pharmacy Module (Phase 7)

- **No Mongo transactions, same constraint as Appointments (Phase 4) and Lab Orders (Phase 6)**: this deployment runs MongoDB standalone (see root `docker-compose.yml`), so `Dispense` cannot wrap "check prescription, decrement N batches, adjust materialized stock, mark prescription DISPENSED" in a single atomic unit. Instead: every batch decrement is individually atomic and guarded against going negative (`repository.ErrInsufficientBatchStock`/`ErrInsufficientStock`, the same `$gte` guard-filter pattern used by Appointments' partial unique index and Pharmacy's own stock fields); the prescription is marked `DISPENSED` *before* any stock is touched so a failure never leaves a partially-dispensed ACTIVE prescription; and if anything after that fails, every batch decrement already applied *by that call* is explicitly reverted and the prescription reverted to `ACTIVE` — a compensating action, not a rollback, documented in full in `dispense_service.go`'s package doc alongside the one residual race window it accepts (two pharmacists dispensing the literal same prescription in the same instant).
- **FEFO (first-expiry-first-out) allocation**: `BatchRepository.FindAvailableForMedicine` returns only non-expired, in-stock batches sorted by ascending expiry date, so a single dispensed quantity can span multiple batches (oldest-expiring first) and the response reports exactly which batch(es) supplied each item.
- **`CurrentStock` on `Medicine` is a materialized running total**, not aggregated from batches on every read — kept in sync by the same atomic `AdjustStock` guard-filter method on both the receive-batch (+) and dispense (-) paths, so "low stock" search filtering (`$expr: currentStock <= reorderLevel`) stays O(1) instead of a batch aggregation per medicine per list request.
- **Prescription items stay free-text** (see EMR Module note above) — the pharmacist picks the actual catalog medicine and quantity independently at dispense time rather than this module attempting to text-match a doctor's free-text order against the catalog.
- **Every stock change is ledgered**: `stock_movements` records one `IN` entry per batch received and one `OUT` entry per batch actually drawn from during a dispense (so a single dispensed line item can produce multiple ledger rows if FEFO spanned batches), each carrying the acting user's resolved display name, same "Entered by Suresh Kumar, not admin@hms.local" pattern as Lab Orders.
- Batches live in their own `inventory` collection (per the blueprint's DB design) but are modeled as the Go type `Batch`, not `Inventory` — the collection name and the domain concept it represents aren't required to share a name.

## Billing Module (Phase 8)

- **`AmountPaid` is a materialized running total**, kept in sync by an atomic guard-filter increment (`BillRepository.RecordPayment`, using `$expr: amountPaid + amount <= totalAmount` the same way Pharmacy's `AdjustStock` guards against negative stock) — an overpayment is rejected atomically rather than caught after the fact, so a bill's ledger and its `amountPaid` field can never disagree.
- **Status is derived, not chosen by the caller**: after every payment, `domain.StatusFor(totalAmount, amountPaid)` recomputes `UNPAID`/`PARTIALLY_PAID`/`PAID` from the actual numbers — there's no `PATCH .../status` endpoint for a human to set it directly, unlike Prescriptions/Lab Orders/Appointments where status is a workflow choice. `CANCELLED` is the one status a person sets, and only when `AmountPaid == 0` (a bill with any recorded payment can no longer be cancelled — it must be handled as a real financial transaction, not erased).
- **Bills and Payments are separate collections**, the same "tightly-coupled workflow, split collections" pattern Lab Orders uses for Orders/Results — recording a payment directly drives its bill's status, but a bill can have many payments (partial payments over time) so it isn't a single embedded sub-document.
- **`INV-NNNNN` bill numbers** reuse the same atomic Mongo-counter mechanism (`internal/pkg/sequence`) already built for patient codes (Phase 2) and lab sample codes (Phase 6).
- **Charge line items stay free-text with a `Category` label** (Consultation/Lab/Pharmacy/Procedure/Room Charge/Other) rather than linking to a specific Appointment/LabOrder/Prescription record — consistent with this codebase's established position (see Prescriptions, EMR) that free-text-at-entry beats a premature hard link when the source of a charge doesn't map cleanly onto a single upstream entity (e.g. a bundled procedure fee). Whoever raises the bill (Admin, per the RBAC matrix — see below) enters what the charge is for and its price.
- **RBAC as seeded in Phase 1 gives only Admin `billing.manage`**; Receptionist has `billing.read` only. This means bill creation and payment recording are Admin-only in this build, which may read as unusual for a receptionist-facing billing desk — it's implemented exactly as the blueprint's seeded role matrix specifies (`internal/modules/auth/domain/permission.go`) rather than silently widened, the same "honor the spec's RBAC table precisely, flag anything that looks like an exception" discipline applied to the Lab Orders/Appointments carve-outs in Phases 4 and 6.

## Audit Logging Module (Phase 9)

- **No new write path**: every module has recorded audit entries through `internal/audit.Recorder` since Phase 2 (Patients was the first). This phase adds only a read side — `internal/modules/auditlogs` is a thin query module: a repository that searches the existing `audit_logs` collection (already indexed on `(entityType, entityId, createdAt)` and `(actorId, createdAt)` since Phase 0/2), a service that parses/validates the `from`/`to` RFC3339 query params, and a handler. There's deliberately no domain package of its own — it reuses `audit.Log` from the shared `internal/audit` package rather than duplicating that type.
- **Gated on `settings.manage`, not a new permission**: the blueprint's own permission list (mirrored in `auth/domain/permission.go`) never named an audit-specific code, unlike Pharmacy/Billing which had `pharmacy.*`/`billing.*` pre-defined from Phase 1. Rather than inventing one, this phase reuses the existing Admin-only `settings.manage` grant — it was defined in Phase 1 but never actually gated anything until now.
- **Filters are exact-match plus a date range** (`actorId`, `entityType`, `entityId`, `action`, `from`, `to`) — no free-text search across metadata, since Mongo would need a text index over an arbitrary `map[string]any` field to support that well, and nothing in this phase's checklist calls for it.

## Reporting Dashboard (Phase 10)

- **Every metric is computed live from the operational collections** on each request — no scheduled aggregation job, no separate `reports` collection kept in sync. At this system's scale that's simpler and always-correct; it trades a few aggregation pipelines per dashboard load for never having a report that's stale or out of sync with the source data, consistent with this codebase's general bias toward correctness-and-simplicity over premature optimization.
- **The reporting module reaches directly into sibling collections** (`patients`, `doctors`, `appointments`, `bills`, `payments`, `medicines`, `lab_orders`, `prescriptions`) rather than going through each module's own repository interface — the one deliberate exception to this codebase's "modules depend on each other's repository interfaces, never raw collections" rule elsewhere (Pharmacy→Prescriptions, Lab Orders→Lab Tests, Billing→Patients, ...). A cross-cutting `$group`/`$sum`/`$dateToString` aggregation doesn't fit any single module's existing repository contract, and duplicating six new aggregation methods across six modules' repositories just to keep the "never raw collections" rule would be exactly the kind of premature abstraction this project has avoided everywhere else.
- **Gated on `report.export`** (Admin-only, pre-defined in Phase 1's permission list) for both viewing and exporting the dashboard — there is no separate `report.view`, the same "reuse what the blueprint already named rather than split it further" reasoning applied to Billing and Audit Logging.
- **Date-range endpoints default to the last 30 days** when `from`/`to` are omitted, parsed as plain `YYYY-MM-DD` dates (not RFC3339 — a report date range is a calendar concept, not an instant) with `to` extended to end-of-day so same-day records aren't excluded. `doctor-workload` and `lab-test-volume` additionally cap results at a `limit` (default 10, max 50) since a "top N" chart with no cap would grow unbounded.
- **The dashboard is code-split**: `recharts` is a ~400KB dependency (before this phase, unused anywhere in the app), so `ReportsDashboardPage` is loaded via `React.lazy`/`Suspense` rather than the app's normal eager-import pattern used by every other route — every other role never pays for a chart library they can't even reach (the page is Admin-only).
- **The Phase 0 dashboard shell's placeholder cards are now live** for users who hold `report.export` (reusing the same `/reports/summary` endpoint the full dashboard's cards use); other roles keep seeing the original static placeholders with an explanatory note, since `report.export` is Admin-only and the dashboard route itself is not.

## Post-Launch Additions — IPD/Wards, Receipts, Report Exports

- **IPD/Wards models three entities as one module** (`Ward` → `Bed` → `Admission`), not split the way Departments/Doctors are, because admitting a patient directly changes a bed's status — the same tightly-coupled relationship Lab Orders has with Lab Results. Wards are simple reference data (like Departments); Beds belong to a Ward and track `AVAILABLE`/`OCCUPIED`/`MAINTENANCE`; Admissions snapshot patient/doctor/bed names at creation time (the same join-free-reads tradeoff used everywhere else).
- **The "two admissions, one bed" race is closed by an atomic guard-filter update** (`BedRepository.Occupy`, a `FindOneAndUpdate` that only succeeds if the bed is still `AVAILABLE`), not an application-level check-then-act — the same pattern Pharmacy uses for `Medicine.CurrentStock` and Appointments uses for its partial unique index. If creating the admission record fails *after* the bed was successfully occupied, the bed is explicitly released as a compensating action (no Mongo transactions on this standalone deployment, same constraint documented for Appointments/Pharmacy). This exact scenario is covered by an automated test (`TestAdmitPatientRejectsAnAlreadyOccupiedBed`).
- **No new permissions were needed** — `ipd.read`/`ipd.write` were already seeded in Phase 1 (Admin/Doctor/Nurse have both; Receptionist has read-only), matching the blueprint's own IPD/Wards RBAC row exactly.
- **Scope is deliberately narrower than the blueprint's full IPD workflow narrative** ("nursing/vitals, medication administration, procedure schedule") — only "Ward/room/bed inventory; admission; doctor assignment; ... discharge and bed release" (the phase's actual checklist) is modeled, the same scope discipline applied to EMR attachments in Phase 5 and Lab's sample-collection tracking in Phase 6.
- **Payment receipts need no new backend endpoint** — `GET /billing/bills/{id}` already returns everything a receipt needs (items, totals, full payment history), so `BillReceiptPage` is a pure presentation layer over data `BillDetailPage` already fetches, printed via `window.print()` exactly like the Lab report.
- **Report exports add exactly one new backend dependency**, `excelize`, to produce a real multi-sheet `.xlsx` workbook (Summary, Appointments by Status, Revenue Trend, Patient Registrations, Doctor Workload, Lab Test Volume) for the dashboard's current date range — reusing the `reports` service's own already-validated methods rather than re-querying the repository directly. A companion "Print / Save as PDF" button on the dashboard itself covers the PDF half without a server-side PDF library, consistent with Lab/Billing.

## Frontend routes (through Phase 10)

| Path | Screen | Guard |
|---|---|---|
| `/login` | Sign in | Public |
| `/dashboard` | Dashboard shell | Authenticated |
| `/patients` | Search/list | Authenticated (`patient.read`) |
| `/patients/new` | Registration form | Authenticated; "New Patient" button only shown with `patient.create` |
| `/patients/:id` | Profile | Authenticated; Edit/Deactivate buttons only shown with `patient.update`/`patient.delete` |
| `/patients/:id/edit` | Edit form | Authenticated (`patient.update`) |
| `/doctors` | Directory/search | Authenticated (`doctor.read`) |
| `/doctors/new` | New profile form | Authenticated; only shown with `doctor.manage` |
| `/doctors/:id` | Profile + weekly schedule | Authenticated; Edit/Deactivate only shown with `doctor.manage` |
| `/doctors/:id/edit` | Edit form | Authenticated (`doctor.manage`) |
| `/appointments` | Search/list (filter by doctor/status/date) | Authenticated (`appointment.view`) |
| `/appointments/new` | Booking form (patient/doctor autocomplete) | Authenticated; only shown with `appointment.manage` |
| `/appointments/:id` | Detail + role-aware status actions + reschedule | Authenticated (`appointment.view`) |
| `/emr` | Medical records list (filter by patient = consultation history) | Authenticated (`emr.read`) |
| `/emr/new` | New clinical note | Authenticated; only shown with `emr.write` |
| `/emr/:id` | Note detail | Authenticated (`emr.read`) |
| `/emr/:id/edit` | Amend note | Authenticated (`emr.write`) |
| `/prescriptions` | Prescriptions list (filter by patient/status) | Authenticated (`prescription.read`) |
| `/prescriptions/new` | Issue prescription (dynamic medicine rows) | Authenticated; only shown with `prescription.write` |
| `/prescriptions/:id` | Detail + Cancel action | Authenticated (`prescription.read`); Cancel only shown with `prescription.write` |
| `/laboratory` | Lab orders list (filter by patient/status) | Authenticated (`lab.order.read`) |
| `/laboratory/new` | New lab order (patient/doctor autocomplete + test select) | Authenticated; only shown with `lab.order.write` |
| `/laboratory/:id` | Order detail + result entry + verify + cancel | Authenticated; result entry needs `lab.order.write`, Verify needs `lab.result.verify` |
| `/laboratory/:id/report` | Printable report (browser print-to-PDF) | Authenticated (`lab.order.read`); only reachable once `COMPLETED` |
| `/settings/departments` | Department list + create + activate/deactivate | Admin only (nav-gated; backend allows any authenticated read) |
| `/settings/lab-tests` | Lab test catalog + create + activate/deactivate | Admin only (nav-gated; backend allows any authenticated read) |
| `/pharmacy` | Medicine catalog (search + low-stock filter) | Authenticated (`pharmacy.read`); "Add Medicine"/"Dispense" only shown with `pharmacy.dispense` |
| `/pharmacy/:id` | Medicine detail + batch list + Receive Batch | Authenticated (`pharmacy.read`); Receive Batch only shown with `pharmacy.dispense` |
| `/pharmacy/dispense` | Dispense a patient's active prescription (FEFO, per-item catalog medicine picker) | Authenticated (`pharmacy.dispense`); optional `?prescriptionId=` prefill from the Prescription detail page |
| `/billing` | Bill list (filter by patient/status/outstanding-only) | Authenticated (`billing.read`); "New Bill" only shown with `billing.manage` |
| `/billing/new` | Raise a bill (dynamic charge-line rows, live subtotal/discount/tax/total) | Authenticated (`billing.manage`) |
| `/billing/:id` | Bill detail + payment history + Record Payment / Cancel Bill | Authenticated (`billing.read`); Record Payment/Cancel only shown with `billing.manage` |
| `/billing/:id/receipt` | Printable payment receipt (browser print-to-PDF) | Authenticated (`billing.read`) |
| `/settings/audit-logs` | Filterable audit log viewer (actor/entity type/entity id/action/date range) | Admin only (nav-gated; backend enforces `settings.manage`) |
| `/settings/users` | Staff/Users — list + create (no role change/deactivate yet, no backend support) | Admin only (nav-gated; backend enforces `user.manage`) |
| `/reports` | Reporting Dashboard — summary cards + charts + Export to Excel + Print/Save as PDF | Admin only (nav-gated; backend enforces `report.export`); code-split via `React.lazy` |
| `/ipd` | Ward/Bed board — click a bed to admit or view/discharge; Active Admissions table | Authenticated (`ipd.read`); Admit/Discharge/Add Ward/Add Bed only shown with `ipd.write` |
=======
# Hospital-management-system
Full-stack Hospital Management System using Go, MongoDB, React and TypeScript


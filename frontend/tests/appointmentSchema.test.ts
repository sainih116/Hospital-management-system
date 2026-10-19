import { describe, expect, it } from "vitest";

import {
  bookingFormSchema,
  buildBookingPayload,
  buildReschedulePayload,
  emptyBookingFormValues,
  rescheduleFormSchema,
} from "@/features/appointments/appointmentSchema";
import {
  DOCTOR_ALLOWED_STATUSES,
  allowedNextStatuses,
  isTerminal,
} from "@/features/appointments/appointmentStateMachine";

function futureLocalDateTime(hoursFromNow: number): string {
  const d = new Date(Date.now() + hoursFromNow * 60 * 60 * 1000);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

describe("bookingFormSchema", () => {
  function validInput() {
    return { ...emptyBookingFormValues, patientId: "p1", doctorId: "d1", startAt: futureLocalDateTime(24), durationMinutes: 30 };
  }

  it("accepts a valid future booking", () => {
    expect(bookingFormSchema.safeParse(validInput()).success).toBe(true);
  });

  it("rejects a start time in the past", () => {
    const result = bookingFormSchema.safeParse({ ...validInput(), startAt: futureLocalDateTime(-1) });
    expect(result.success).toBe(false);
  });

  it("rejects a missing patient", () => {
    expect(bookingFormSchema.safeParse({ ...validInput(), patientId: "" }).success).toBe(false);
  });

  it("rejects a duration outside 5-240 minutes", () => {
    expect(bookingFormSchema.safeParse({ ...validInput(), durationMinutes: 1 }).success).toBe(false);
    expect(bookingFormSchema.safeParse({ ...validInput(), durationMinutes: 300 }).success).toBe(false);
  });
});

describe("buildBookingPayload", () => {
  it("computes endAt from startAt + durationMinutes", () => {
    const startAt = futureLocalDateTime(24);
    const payload = buildBookingPayload({ ...emptyBookingFormValues, patientId: "p1", doctorId: "d1", startAt, durationMinutes: 45 });

    const start = new Date(payload.startAt).getTime();
    const end = new Date(payload.endAt).getTime();
    expect(end - start).toBe(45 * 60_000);
  });

  it("omits an empty reason", () => {
    const payload = buildBookingPayload({ ...emptyBookingFormValues, patientId: "p1", doctorId: "d1", startAt: futureLocalDateTime(24) });
    expect(payload.reason).toBeUndefined();
  });
});

describe("rescheduleFormSchema / buildReschedulePayload", () => {
  it("rejects a past start time", () => {
    const result = rescheduleFormSchema.safeParse({ startAt: futureLocalDateTime(-2), durationMinutes: 30 });
    expect(result.success).toBe(false);
  });

  it("computes a new end time from the duration", () => {
    const startAt = futureLocalDateTime(48);
    const payload = buildReschedulePayload({ startAt, durationMinutes: 60 });
    const start = new Date(payload.startAt).getTime();
    const end = new Date(payload.endAt).getTime();
    expect(end - start).toBe(60 * 60_000);
  });
});

describe("appointment state machine", () => {
  it("allows the documented happy-path transitions", () => {
    expect(allowedNextStatuses("SCHEDULED")).toContain("CONFIRMED");
    expect(allowedNextStatuses("CONFIRMED")).toContain("IN_PROGRESS");
    expect(allowedNextStatuses("IN_PROGRESS")).toContain("COMPLETED");
  });

  it("does not allow skipping straight from SCHEDULED to COMPLETED", () => {
    expect(allowedNextStatuses("SCHEDULED")).not.toContain("COMPLETED");
  });

  it("treats COMPLETED, CANCELLED, NO_SHOW and RESCHEDULED as terminal", () => {
    expect(isTerminal("COMPLETED")).toBe(true);
    expect(isTerminal("CANCELLED")).toBe(true);
    expect(isTerminal("NO_SHOW")).toBe(true);
    expect(isTerminal("RESCHEDULED")).toBe(true);
    expect(isTerminal("SCHEDULED")).toBe(false);
  });

  it("restricts the doctor-allowed statuses to clinical progression only", () => {
    expect(DOCTOR_ALLOWED_STATUSES).toEqual(expect.arrayContaining(["IN_PROGRESS", "COMPLETED", "NO_SHOW"]));
    expect(DOCTOR_ALLOWED_STATUSES).not.toContain("CANCELLED");
    expect(DOCTOR_ALLOWED_STATUSES).not.toContain("CONFIRMED");
  });
});

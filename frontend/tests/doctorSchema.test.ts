import { describe, expect, it } from "vitest";

import {
  DAYS_OF_WEEK,
  buildCreateDoctorPayload,
  buildUpdateDoctorPayload,
  doctorFormSchema,
  doctorToFormValues,
  emptyDoctorFormValues,
} from "@/features/doctors/doctorSchema";

function validInput() {
  return {
    ...emptyDoctorFormValues(),
    userId: "user-1",
    departmentId: "dept-1",
    licenseNo: "LIC-001",
    specialization: "Cardiology",
    fee: 500,
  };
}

describe("doctorFormSchema", () => {
  it("accepts a minimal valid submission with no availability enabled", () => {
    const result = doctorFormSchema.safeParse(validInput());
    expect(result.success).toBe(true);
  });

  it("rejects a missing department", () => {
    const result = doctorFormSchema.safeParse({ ...validInput(), departmentId: "" });
    expect(result.success).toBe(false);
  });

  it("rejects a negative fee", () => {
    const result = doctorFormSchema.safeParse({ ...validInput(), fee: -10 });
    expect(result.success).toBe(false);
  });

  it("rejects an enabled day whose end time is before its start time", () => {
    const input = validInput();
    input.availability[1] = { enabled: true, startTime: "17:00", endTime: "09:00" };
    const result = doctorFormSchema.safeParse(input);
    expect(result.success).toBe(false);
  });

  it("ignores time validity for a disabled day", () => {
    const input = validInput();
    input.availability[1] = { enabled: false, startTime: "not-a-time", endTime: "also-not-a-time" };
    const result = doctorFormSchema.safeParse(input);
    expect(result.success).toBe(true);
  });

  it("accepts an enabled day with a valid time range", () => {
    const input = validInput();
    input.availability[1] = { enabled: true, startTime: "09:00", endTime: "17:00" };
    const result = doctorFormSchema.safeParse(input);
    expect(result.success).toBe(true);
  });
});

describe("buildCreateDoctorPayload / buildUpdateDoctorPayload", () => {
  it("includes only enabled days as availability slots, keyed by day index", () => {
    const input = validInput();
    input.availability[1] = { enabled: true, startTime: "09:00", endTime: "17:00" }; // Monday
    input.availability[3] = { enabled: true, startTime: "10:00", endTime: "14:00" }; // Wednesday

    const payload = buildCreateDoctorPayload(input);
    expect(payload.availability).toEqual([
      { dayOfWeek: 1, startTime: "09:00", endTime: "17:00" },
      { dayOfWeek: 3, startTime: "10:00", endTime: "14:00" },
    ]);
  });

  it("returns an empty availability array when no day is enabled", () => {
    const payload = buildCreateDoctorPayload(validInput());
    expect(payload.availability).toEqual([]);
  });

  it("update payload never includes userId (doctor-user linkage is immutable)", () => {
    const payload = buildUpdateDoctorPayload(validInput());
    expect(payload).not.toHaveProperty("userId");
  });

  it("omits an empty bio rather than sending an empty string", () => {
    const payload = buildCreateDoctorPayload({ ...validInput(), bio: "" });
    expect(payload.bio).toBeUndefined();
  });
});

describe("doctorToFormValues", () => {
  it("round-trips a persisted doctor's availability into the 7-day flat array", () => {
    const form = doctorToFormValues({
      departmentId: "dept-1",
      licenseNo: "LIC-001",
      specialization: "Cardiology",
      fee: 500,
      availability: [{ dayOfWeek: 2, startTime: "08:00", endTime: "12:00" }],
    });

    expect(form.availability).toHaveLength(DAYS_OF_WEEK.length);
    expect(form.availability[2]).toEqual({ enabled: true, startTime: "08:00", endTime: "12:00" });
    expect(form.availability[0].enabled).toBe(false);
  });

  it("defaults every day to disabled when no availability is persisted", () => {
    const form = doctorToFormValues({
      departmentId: "dept-1",
      licenseNo: "LIC-001",
      specialization: "Cardiology",
      fee: 500,
    });

    expect(form.availability.every((slot) => !slot.enabled)).toBe(true);
  });
});

import { describe, expect, it } from "vitest";

import { buildPatientPayload, emptyPatientFormValues, patientFormSchema, patientToFormValues } from "@/features/patients/patientSchema";

function validInput() {
  return {
    ...emptyPatientFormValues,
    name: "Rohan Kumar",
    dob: "1992-04-15",
    gender: "M",
    phone: "9000000001",
    type: "OPD",
  };
}

describe("patientFormSchema", () => {
  it("accepts a minimal valid submission", () => {
    const result = patientFormSchema.safeParse(validInput());
    expect(result.success).toBe(true);
  });

  it("rejects a future date of birth", () => {
    const future = new Date();
    future.setFullYear(future.getFullYear() + 1);
    const result = patientFormSchema.safeParse({ ...validInput(), dob: future.toISOString().slice(0, 10) });
    expect(result.success).toBe(false);
  });

  it("rejects an invalid phone number", () => {
    const result = patientFormSchema.safeParse({ ...validInput(), phone: "not-a-phone" });
    expect(result.success).toBe(false);
  });

  it("rejects an unknown gender code", () => {
    const result = patientFormSchema.safeParse({ ...validInput(), gender: "X" });
    expect(result.success).toBe(false);
  });

  it("requires an emergency contact phone when a contact name is given", () => {
    const result = patientFormSchema.safeParse({ ...validInput(), emergencyContactName: "Jane Doe" });
    expect(result.success).toBe(false);
  });

  it("allows an emergency contact when both name and phone are given", () => {
    const result = patientFormSchema.safeParse({
      ...validInput(),
      emergencyContactName: "Jane Doe",
      emergencyContactPhone: "9111111111",
    });
    expect(result.success).toBe(true);
  });

  it("rejects a malformed emergency contact phone number", () => {
    const result = patientFormSchema.safeParse({
      ...validInput(),
      emergencyContactName: "Jane Doe",
      emergencyContactPhone: "call-jane",
    });
    expect(result.success).toBe(false);
  });
});

describe("buildPatientPayload", () => {
  it("splits the comma-separated allergies field into an array", () => {
    const payload = buildPatientPayload({ ...validInput(), allergiesText: "Penicillin,  Peanuts ,  " });
    expect(payload.allergies).toEqual(["Penicillin", "Peanuts"]);
  });

  it("omits empty optional fields rather than sending empty strings", () => {
    const payload = buildPatientPayload(validInput());
    expect(payload.email).toBeUndefined();
    expect(payload.emergencyContact).toBeUndefined();
    expect(payload.insurance).toBeUndefined();
  });

  it("builds a nested emergencyContact only when a name is present", () => {
    const payload = buildPatientPayload({
      ...validInput(),
      emergencyContactName: "Jane Doe",
      emergencyContactPhone: "9111111111",
      emergencyContactRelationship: "Spouse",
    });
    expect(payload.emergencyContact).toEqual({ name: "Jane Doe", phone: "9111111111", relationship: "Spouse" });
  });

  it("builds a nested insurance object when either field is present", () => {
    const payload = buildPatientPayload({ ...validInput(), insuranceProvider: "Acme Health" });
    expect(payload.insurance).toEqual({ provider: "Acme Health", policyNumber: undefined });
  });
});

describe("patientToFormValues", () => {
  it("round-trips a persisted patient into flat, editable form state", () => {
    const form = patientToFormValues({
      name: "Neha Singh",
      dob: "1995-01-01",
      gender: "F",
      phone: "9000000002",
      allergies: ["Latex"],
      emergencyContact: { name: "Amit", phone: "9000000003", relationship: "Brother" },
      insurance: { provider: "Acme", policyNumber: "P123" },
      type: "OPD",
    });

    expect(form.allergiesText).toBe("Latex");
    expect(form.emergencyContactName).toBe("Amit");
    expect(form.insurancePolicyNumber).toBe("P123");

    // Round-tripping back through buildPatientPayload should reproduce
    // the same nested shape the API expects.
    const payload = buildPatientPayload(form);
    expect(payload.emergencyContact).toEqual({ name: "Amit", phone: "9000000003", relationship: "Brother" });
  });
});

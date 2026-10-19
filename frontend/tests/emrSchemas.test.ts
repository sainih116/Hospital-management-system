import { describe, expect, it } from "vitest";

import { emptyMedicalRecordFormValues, medicalRecordFormSchema } from "@/features/medicalrecords/medicalRecordSchema";
import { emptyPrescriptionFormValues, emptyPrescriptionItem, prescriptionFormSchema } from "@/features/prescriptions/prescriptionSchema";

describe("medicalRecordFormSchema", () => {
  function validInput() {
    return { ...emptyMedicalRecordFormValues, patientId: "p1", doctorId: "d1", diagnosis: "Suspected angina" };
  }

  it("accepts a minimal valid submission", () => {
    expect(medicalRecordFormSchema.safeParse(validInput()).success).toBe(true);
  });

  it("requires a patient", () => {
    expect(medicalRecordFormSchema.safeParse({ ...validInput(), patientId: "" }).success).toBe(false);
  });

  it("requires a doctor", () => {
    expect(medicalRecordFormSchema.safeParse({ ...validInput(), doctorId: "" }).success).toBe(false);
  });

  it("requires a diagnosis of at least 2 characters", () => {
    expect(medicalRecordFormSchema.safeParse({ ...validInput(), diagnosis: "x" }).success).toBe(false);
  });

  it("allows optional clinical fields to be empty", () => {
    const result = medicalRecordFormSchema.safeParse({ ...validInput(), complaint: "", hpi: "", history: "" });
    expect(result.success).toBe(true);
  });
});

describe("prescriptionFormSchema", () => {
  function validInput() {
    return {
      ...emptyPrescriptionFormValues,
      patientId: "p1",
      doctorId: "d1",
      items: [{ medicineName: "Paracetamol 500mg", dose: "1 tablet", frequency: "Twice daily", duration: "5 days" }],
    };
  }

  it("accepts a submission with one complete medicine item", () => {
    const result = prescriptionFormSchema.safeParse(validInput());
    expect(result.success).toBe(true);
  });

  it("rejects an empty items array", () => {
    const result = prescriptionFormSchema.safeParse({ ...validInput(), items: [] });
    expect(result.success).toBe(false);
  });

  it("rejects an item missing required fields", () => {
    const result = prescriptionFormSchema.safeParse({
      ...validInput(),
      items: [{ ...emptyPrescriptionItem, medicineName: "" }],
    });
    expect(result.success).toBe(false);
  });

  it("accepts multiple medicine items", () => {
    const result = prescriptionFormSchema.safeParse({
      ...validInput(),
      items: [
        { medicineName: "Paracetamol 500mg", dose: "1 tablet", frequency: "Twice daily", duration: "5 days" },
        { medicineName: "Amoxicillin 500mg", dose: "1 capsule", frequency: "Thrice daily", duration: "7 days" },
      ],
    });
    expect(result.success).toBe(true);
  });

  it("allows instructions to be omitted", () => {
    const result = prescriptionFormSchema.safeParse({
      ...validInput(),
      items: [{ medicineName: "Paracetamol", dose: "1", frequency: "1x", duration: "3 days" }],
    });
    expect(result.success).toBe(true);
  });
});

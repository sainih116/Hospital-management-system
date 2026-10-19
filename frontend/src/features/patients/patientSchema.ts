import { z } from "zod";

import type { PatientFormValues as ApiPatientPayload } from "@/features/patients/patientsApi";

export const GENDERS = ["M", "F", "O"] as const;
export const BLOOD_GROUPS = ["A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"] as const;
export const PATIENT_TYPES = ["OPD", "IPD"] as const;

const phoneRegex = /^[0-9]{7,15}$/;
const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** Flat form shape (one field per input) — mapped to the nested API
 * payload shape by buildPatientPayload below. */
export const patientFormSchema = z
  .object({
    name: z.string().min(2, "Name must be at least 2 characters").max(150, "Name is too long"),
    dob: z.string().min(1, "Date of birth is required"),
    gender: z.string().refine((v) => (GENDERS as readonly string[]).includes(v), "Select a gender"),
    phone: z.string().regex(phoneRegex, "Phone must be 7-15 digits"),
    email: z.string().optional(),
    address: z.string().max(300).optional(),
    bloodGroup: z.string().optional(),
    allergiesText: z.string().optional(),
    emergencyContactName: z.string().max(120).optional(),
    emergencyContactPhone: z.string().optional(),
    emergencyContactRelationship: z.string().max(60).optional(),
    insuranceProvider: z.string().max(120).optional(),
    insurancePolicyNumber: z.string().max(60).optional(),
    type: z.string().refine((v) => (PATIENT_TYPES as readonly string[]).includes(v), "Select a patient type"),
  })
  .superRefine((data, ctx) => {
    if (data.dob) {
      const parsed = new Date(data.dob);
      if (Number.isNaN(parsed.getTime())) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["dob"], message: "Enter a valid date" });
      } else if (parsed.getTime() > Date.now()) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["dob"], message: "Date of birth cannot be in the future" });
      }
    }
    if (data.email && !emailRegex.test(data.email)) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["email"], message: "Enter a valid email" });
    }
    if (data.bloodGroup && !(BLOOD_GROUPS as readonly string[]).includes(data.bloodGroup)) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["bloodGroup"], message: "Select a valid blood group" });
    }
    if (data.emergencyContactName && !data.emergencyContactPhone) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["emergencyContactPhone"],
        message: "Required when a contact name is given",
      });
    } else if (data.emergencyContactPhone && !phoneRegex.test(data.emergencyContactPhone)) {
      // Reuses the same phoneRegex already applied to the patient's own
      // phone field above — the backend now enforces this format for the
      // emergency contact too (see patient_service.go's
      // validateEmergencyContact), so catch it client-side as well.
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["emergencyContactPhone"],
        message: "Phone must be 7-15 digits",
      });
    }
  });

export type PatientFormInput = z.infer<typeof patientFormSchema>;

export const emptyPatientFormValues: PatientFormInput = {
  name: "",
  dob: "",
  gender: "",
  phone: "",
  email: "",
  address: "",
  bloodGroup: "",
  allergiesText: "",
  emergencyContactName: "",
  emergencyContactPhone: "",
  emergencyContactRelationship: "",
  insuranceProvider: "",
  insurancePolicyNumber: "",
  type: "",
};

/** Converts the flat form state into the nested shape the API expects. */
export function buildPatientPayload(input: PatientFormInput): ApiPatientPayload {
  const allergies = (input.allergiesText ?? "")
    .split(",")
    .map((a) => a.trim())
    .filter(Boolean);

  return {
    name: input.name,
    dob: input.dob,
    gender: input.gender as ApiPatientPayload["gender"],
    phone: input.phone,
    email: input.email || undefined,
    address: input.address || undefined,
    bloodGroup: input.bloodGroup || undefined,
    allergies: allergies.length > 0 ? allergies : undefined,
    emergencyContact: input.emergencyContactName
      ? {
          name: input.emergencyContactName,
          phone: input.emergencyContactPhone ?? "",
          relationship: input.emergencyContactRelationship || undefined,
        }
      : undefined,
    insurance:
      input.insuranceProvider || input.insurancePolicyNumber
        ? { provider: input.insuranceProvider || undefined, policyNumber: input.insurancePolicyNumber || undefined }
        : undefined,
    type: input.type as ApiPatientPayload["type"],
  };
}

/** Converts a persisted Patient's nested shape back into flat form state for editing. */
export function patientToFormValues(patient: {
  name: string;
  dob: string;
  gender: string;
  phone: string;
  email?: string;
  address?: string;
  bloodGroup?: string;
  allergies?: string[];
  emergencyContact?: { name: string; phone: string; relationship?: string };
  insurance?: { provider?: string; policyNumber?: string };
  type: string;
}): PatientFormInput {
  return {
    name: patient.name,
    dob: patient.dob,
    gender: patient.gender,
    phone: patient.phone,
    email: patient.email ?? "",
    address: patient.address ?? "",
    bloodGroup: patient.bloodGroup ?? "",
    allergiesText: (patient.allergies ?? []).join(", "),
    emergencyContactName: patient.emergencyContact?.name ?? "",
    emergencyContactPhone: patient.emergencyContact?.phone ?? "",
    emergencyContactRelationship: patient.emergencyContact?.relationship ?? "",
    insuranceProvider: patient.insurance?.provider ?? "",
    insurancePolicyNumber: patient.insurance?.policyNumber ?? "",
    type: patient.type,
  };
}

import { z } from "zod";

export const medicalRecordFormSchema = z.object({
  patientId: z.string().min(1, "Select a patient"),
  patientLabel: z.string().optional(),
  doctorId: z.string().min(1, "Select a doctor"),
  doctorLabel: z.string().optional(),
  complaint: z.string().max(1000).optional(),
  hpi: z.string().max(2000).optional(),
  history: z.string().max(2000).optional(),
  examination: z.string().max(2000).optional(),
  diagnosis: z.string().min(2, "Diagnosis is required").max(1000),
  treatment: z.string().max(2000).optional(),
  followUp: z.string().max(1000).optional(),
});

export type MedicalRecordFormInput = z.infer<typeof medicalRecordFormSchema>;

export const emptyMedicalRecordFormValues: MedicalRecordFormInput = {
  patientId: "",
  patientLabel: "",
  doctorId: "",
  doctorLabel: "",
  complaint: "",
  hpi: "",
  history: "",
  examination: "",
  diagnosis: "",
  treatment: "",
  followUp: "",
};

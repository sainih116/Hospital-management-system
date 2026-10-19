import { z } from "zod";

const itemSchema = z.object({
  medicineName: z.string().min(2, "Medicine name is required").max(200),
  dose: z.string().min(1, "Dose is required").max(60),
  frequency: z.string().min(1, "Frequency is required").max(60),
  duration: z.string().min(1, "Duration is required").max(60),
  instructions: z.string().max(300).optional(),
});

export const prescriptionFormSchema = z.object({
  patientId: z.string().min(1, "Select a patient"),
  patientLabel: z.string().optional(),
  doctorId: z.string().min(1, "Select a doctor"),
  doctorLabel: z.string().optional(),
  items: z.array(itemSchema).min(1, "Add at least one medicine"),
});

export type PrescriptionFormInput = z.infer<typeof prescriptionFormSchema>;

export const emptyPrescriptionItem = { medicineName: "", dose: "", frequency: "", duration: "", instructions: "" };

export const emptyPrescriptionFormValues: PrescriptionFormInput = {
  patientId: "",
  patientLabel: "",
  doctorId: "",
  doctorLabel: "",
  items: [emptyPrescriptionItem],
};

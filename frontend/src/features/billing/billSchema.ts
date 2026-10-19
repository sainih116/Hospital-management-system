import { z } from "zod";

export const BILL_ITEM_CATEGORIES = ["CONSULTATION", "LAB", "PHARMACY", "PROCEDURE", "ROOM_CHARGE", "OTHER"] as const;

const itemSchema = z.object({
  description: z.string().min(1, "Description is required").max(200),
  category: z.enum(BILL_ITEM_CATEGORIES),
  quantity: z.coerce.number().int().min(1, "Must be at least 1"),
  unitPrice: z.coerce.number().min(0, "Cannot be negative"),
});

export const billFormSchema = z.object({
  patientId: z.string().min(1, "Select a patient"),
  patientLabel: z.string().optional(),
  items: z.array(itemSchema).min(1, "Add at least one charge"),
  discount: z.coerce.number().min(0, "Cannot be negative"),
  tax: z.coerce.number().min(0, "Cannot be negative"),
});

export type BillFormInput = z.infer<typeof billFormSchema>;

export const emptyBillItem = { description: "", category: "CONSULTATION" as const, quantity: 1, unitPrice: 0 };

export const emptyBillFormValues: BillFormInput = {
  patientId: "",
  patientLabel: "",
  items: [emptyBillItem],
  discount: 0,
  tax: 0,
};

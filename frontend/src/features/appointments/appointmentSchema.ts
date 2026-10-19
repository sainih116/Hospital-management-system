import { z } from "zod";

import type { BookAppointmentPayload, ReschedulePayload } from "@/features/appointments/appointmentsApi";

export const bookingFormSchema = z
  .object({
    patientId: z.string().min(1, "Select a patient"),
    patientLabel: z.string().optional(),
    doctorId: z.string().min(1, "Select a doctor"),
    doctorLabel: z.string().optional(),
    startAt: z.string().min(1, "Start time is required"),
    durationMinutes: z.coerce.number().min(5, "Minimum 5 minutes").max(240, "Maximum 4 hours"),
    reason: z.string().max(500).optional(),
  })
  .superRefine((data, ctx) => {
    const start = new Date(data.startAt);
    if (Number.isNaN(start.getTime())) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["startAt"], message: "Enter a valid date/time" });
    } else if (start.getTime() <= Date.now()) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["startAt"], message: "Must be in the future" });
    }
  });

export type BookingFormInput = z.infer<typeof bookingFormSchema>;

export const emptyBookingFormValues: BookingFormInput = {
  patientId: "",
  patientLabel: "",
  doctorId: "",
  doctorLabel: "",
  startAt: "",
  durationMinutes: 30,
  reason: "",
};

export function buildBookingPayload(input: BookingFormInput): BookAppointmentPayload {
  const start = new Date(input.startAt);
  const end = new Date(start.getTime() + input.durationMinutes * 60_000);
  return {
    patientId: input.patientId,
    doctorId: input.doctorId,
    startAt: start.toISOString(),
    endAt: end.toISOString(),
    reason: input.reason || undefined,
  };
}

export const rescheduleFormSchema = z
  .object({
    startAt: z.string().min(1, "Start time is required"),
    durationMinutes: z.coerce.number().min(5, "Minimum 5 minutes").max(240, "Maximum 4 hours"),
  })
  .superRefine((data, ctx) => {
    const start = new Date(data.startAt);
    if (Number.isNaN(start.getTime())) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["startAt"], message: "Enter a valid date/time" });
    } else if (start.getTime() <= Date.now()) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["startAt"], message: "Must be in the future" });
    }
  });

export type RescheduleFormInput = z.infer<typeof rescheduleFormSchema>;

export function buildReschedulePayload(input: RescheduleFormInput): ReschedulePayload {
  const start = new Date(input.startAt);
  const end = new Date(start.getTime() + input.durationMinutes * 60_000);
  return { startAt: start.toISOString(), endAt: end.toISOString() };
}

import { z } from "zod";

import type { AvailabilitySlot, CreateDoctorPayload, UpdateDoctorPayload } from "@/features/doctors/doctorsApi";

export const DAYS_OF_WEEK = [
  { value: 0, label: "Sunday" },
  { value: 1, label: "Monday" },
  { value: 2, label: "Tuesday" },
  { value: 3, label: "Wednesday" },
  { value: 4, label: "Thursday" },
  { value: 5, label: "Friday" },
  { value: 6, label: "Saturday" },
] as const;

const timePattern = /^([01]\d|2[0-3]):([0-5]\d)$/;

const daySlotSchema = z
  .object({
    enabled: z.boolean(),
    startTime: z.string(),
    endTime: z.string(),
  })
  .superRefine((slot, ctx) => {
    if (!slot.enabled) return;
    if (!timePattern.test(slot.startTime)) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["startTime"], message: "Invalid time" });
    }
    if (!timePattern.test(slot.endTime)) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["endTime"], message: "Invalid time" });
    }
    if (timePattern.test(slot.startTime) && timePattern.test(slot.endTime) && slot.startTime >= slot.endTime) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["endTime"], message: "Must be after start time" });
    }
  });

export const doctorFormSchema = z.object({
  userId: z.string().optional(),
  departmentId: z.string().min(1, "Select a department"),
  licenseNo: z.string().min(2, "License number is required").max(60),
  specialization: z.string().min(2, "Specialization is required").max(120),
  fee: z.coerce.number().min(0, "Fee must be 0 or more"),
  bio: z.string().max(1000).optional(),
  availability: z.array(daySlotSchema).length(7),
});

export type DoctorFormInput = z.infer<typeof doctorFormSchema>;

export function emptyDoctorFormValues(): DoctorFormInput {
  return {
    userId: "",
    departmentId: "",
    licenseNo: "",
    specialization: "",
    fee: 0,
    bio: "",
    availability: DAYS_OF_WEEK.map(() => ({ enabled: false, startTime: "09:00", endTime: "17:00" })),
  };
}

export function buildCreateDoctorPayload(input: DoctorFormInput): CreateDoctorPayload {
  return {
    userId: input.userId ?? "",
    departmentId: input.departmentId,
    licenseNo: input.licenseNo,
    specialization: input.specialization,
    fee: input.fee,
    bio: input.bio || undefined,
    availability: buildAvailability(input.availability),
  };
}

export function buildUpdateDoctorPayload(input: DoctorFormInput): UpdateDoctorPayload {
  return {
    departmentId: input.departmentId,
    licenseNo: input.licenseNo,
    specialization: input.specialization,
    fee: input.fee,
    bio: input.bio || undefined,
    availability: buildAvailability(input.availability),
  };
}

function buildAvailability(slots: DoctorFormInput["availability"]): AvailabilitySlot[] {
  return slots
    .map((slot, dayOfWeek) => ({ ...slot, dayOfWeek }))
    .filter((slot) => slot.enabled)
    .map((slot) => ({ dayOfWeek: slot.dayOfWeek, startTime: slot.startTime, endTime: slot.endTime }));
}

export function doctorToFormValues(doctor: {
  departmentId: string;
  licenseNo: string;
  specialization: string;
  fee: number;
  bio?: string;
  availability?: AvailabilitySlot[];
}): DoctorFormInput {
  const availability = DAYS_OF_WEEK.map((day) => {
    const match = doctor.availability?.find((slot) => slot.dayOfWeek === day.value);
    return match
      ? { enabled: true, startTime: match.startTime, endTime: match.endTime }
      : { enabled: false, startTime: "09:00", endTime: "17:00" };
  });

  return {
    userId: "",
    departmentId: doctor.departmentId,
    licenseNo: doctor.licenseNo,
    specialization: doctor.specialization,
    fee: doctor.fee,
    bio: doctor.bio ?? "",
    availability,
  };
}

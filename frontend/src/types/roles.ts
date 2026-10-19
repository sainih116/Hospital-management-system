/**
 * Role codes shared with the backend RBAC model. Kept in sync manually with
 * backend/internal/config and the roles seeded in Phase 1.
 */
export const ROLES = {
  ADMIN: "ADMIN",
  DOCTOR: "DOCTOR",
  NURSE: "NURSE",
  RECEPTIONIST: "RECEPTIONIST",
  PHARMACIST: "PHARMACIST",
  LAB_TECHNICIAN: "LAB_TECHNICIAN",
} as const;

export type RoleCode = (typeof ROLES)[keyof typeof ROLES];

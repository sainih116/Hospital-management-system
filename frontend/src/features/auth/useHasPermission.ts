import { useAppSelector } from "@/app/hooks";
import type { Permission } from "@/features/auth/permissions";

/** Returns whether the current user's role grants the given permission. */
export function useHasPermission(permission: Permission): boolean {
  return useAppSelector((state) => state.auth.user?.permissions.includes(permission) ?? false);
}

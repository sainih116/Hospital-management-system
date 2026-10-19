import type { ReactElement } from "react";
import { Navigate, Outlet } from "react-router-dom";

import { useHasPermission } from "@/features/auth/useHasPermission";
import type { Permission } from "@/features/auth/permissions";

interface PermissionRouteProps {
  /** The permission required to reach any nested route. Reuses the same
   * PERMISSIONS codes the backend's RequirePermission middleware checks
   * and that page components already use to hide individual buttons —
   * see useHasPermission. */
  permission: Permission;
}

/**
 * Route-level counterpart to ProtectedRoute: ProtectedRoute only checks
 * "is authenticated", so before this guard existed any authenticated user
 * could reach a module's page shell by typing its URL even with zero
 * permission for it (the sidebar only hides the nav item — it doesn't
 * block direct navigation). This redirects to a proper 403 page instead
 * of rendering a page that would just fail its first API call.
 */
export default function PermissionRoute({ permission }: PermissionRouteProps): ReactElement {
  const hasPermission = useHasPermission(permission);

  if (!hasPermission) {
    return <Navigate to="/forbidden" replace />;
  }

  return <Outlet />;
}

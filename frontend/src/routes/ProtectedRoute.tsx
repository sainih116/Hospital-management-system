import type { ReactElement } from "react";
import { Navigate, Outlet, useLocation } from "react-router-dom";

import { useAppSelector } from "@/app/hooks";

/**
 * Gate for authenticated-only routes. Redirects to /login (preserving the
 * attempted destination in location state) when there is no active session.
 */
export default function ProtectedRoute(): ReactElement {
  const isAuthenticated = useAppSelector((state) => state.auth.isAuthenticated);
  const location = useLocation();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  return <Outlet />;
}

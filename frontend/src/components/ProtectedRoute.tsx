import { Navigate, Outlet } from "react-router";
import { useAuth } from "../context/AuthContext";

// Espera a que termine la rehidratación de localStorage (isReady) antes de decidir — si no,
// un refresh de página redirigiría a /login por una fracción de segundo aunque sí había sesión.
export function ProtectedRoute() {
  const { isAuthenticated, isReady } = useAuth();

  if (!isReady) return null;
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <Outlet />;
}

import { Navigate, Outlet } from "react-router";
import { useAuth } from "../context/AuthContext";
import { Spinner } from "./ui/Spinner";

// Espera a que termine la rehidratación de localStorage (isReady) antes de decidir — si no,
// un refresh de página redirigiría a /login por una fracción de segundo aunque sí había sesión.
export function ProtectedRoute() {
  const { isAuthenticated, isReady } = useAuth();

  if (!isReady) return (
    <div className="flex min-h-screen items-center justify-center">
      <Spinner className="h-6 w-6 text-(--text-muted)" />
    </div>
  );
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <Outlet />;
}

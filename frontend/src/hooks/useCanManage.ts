import { useAuth } from "../context/AuthContext";

// useCanManage refleja shared/tenant.CanManage del backend (owner/admin pueden gestionar la
// empresa activa; member no — ver erp/internal/shared/tenant/context.go). Es el mismo criterio,
// del lado del frontend, para ocultar/deshabilitar botones de gestión que el backend rechazaría
// con 403 de todos modos — antes esos botones quedaban visibles y clicables para cualquier rol,
// y el member solo se enteraba del rechazo al hacer clic.
export function useCanManage(): boolean {
  const { user } = useAuth();
  return user?.role === "owner" || user?.role === "admin";
}

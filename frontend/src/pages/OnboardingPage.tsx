import { Building2 } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { Card } from "../components/ui/Card";
import { CompanyManager } from "../components/CompanyManager";

// Gate previo al dashboard cuando el usuario no tiene empresa activa (0 o 2+ vinculadas, ver
// middleware.RequireTenant en apidian). La lógica de listar/crear/seleccionar empresa vive en
// CompanyManager (reusada también dentro del dashboard en CompaniesPage, /companies) — aquí solo el
// chrome de página de un gate previo al login (Card centrada + logout).
export function OnboardingPage() {
  const { logout } = useAuth();

  return (
    <div className="flex min-h-screen items-center justify-center bg-(--bg-primary) px-4 py-8">
      <Card className="w-full max-w-xl">
        <div className="flex items-center justify-between border-b border-(--border-light) px-4 py-3">
          <div className="flex items-center gap-2">
            <Building2 className="h-5 w-5 text-(--accent-primary)" />
            <h1 className="text-sm font-semibold text-(--text-primary)">Tu empresa</h1>
          </div>
          <button type="button" onClick={logout} className="text-xs text-(--text-secondary) hover:text-(--text-primary)">
            Cerrar sesión
          </button>
        </div>

        <div className="p-4">
          <CompanyManager />
        </div>
      </Card>
    </div>
  );
}

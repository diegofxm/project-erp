import { CheckCircle2 } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { Card } from "../components/ui/Card";

// Primer pantallazo del dashboard (Fase 1) — placeholder intencional: el contenido real
// (resumen de facturación, accesos rápidos) llega en una fase posterior, ver
// docs/frontend-architecture.md.
export function DashboardPage() {
  const { user, activeIssuer } = useAuth();

  return (
    <div className="p-4">
      <Card className="p-4">
        <div className="mb-3 flex items-center gap-2 text-(--color-success)">
          <CheckCircle2 className="h-4 w-4" />
          <h2 className="text-sm font-semibold text-(--text-primary)">Sesión activa</h2>
        </div>
        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
          <dt className="text-(--text-secondary)">Usuario</dt>
          <dd className="text-(--text-primary)">
            {user?.name} ({user?.email})
          </dd>
          <dt className="text-(--text-secondary)">Empresa activa</dt>
          <dd className="text-(--text-primary)">
            {activeIssuer?.business_name} — NIT {activeIssuer?.nit}
          </dd>
          <dt className="text-(--text-secondary)">Ambiente</dt>
          <dd className="text-(--text-primary)">{activeIssuer?.environment === "1" ? "Producción" : "Habilitación"}</dd>
        </dl>
      </Card>
    </div>
  );
}

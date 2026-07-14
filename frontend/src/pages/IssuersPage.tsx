import { Building2 } from "lucide-react";
import { CompanyDataPanel } from "../components/issuer-settings/CompanyDataPanel";
import { Card } from "../components/ui/Card";
import { IssuerManager } from "../components/IssuerManager";

export function IssuersPage() {
  return (
    <div className="p-4 flex flex-col gap-4">
      <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <Building2 className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Mis empresas
      </h1>

      <div className="flex gap-4 items-start">
        {/* Perfil de la empresa activa — columna ancha */}
        <div className="flex-3 min-w-0">
          <CompanyDataPanel />
        </div>

        {/* Cambiar de empresa o crear una nueva — columna estrecha */}
        <div className="flex-2 min-w-0">
          <Card className="p-4">
            <IssuerManager />
          </Card>
        </div>
      </div>
    </div>
  );
}

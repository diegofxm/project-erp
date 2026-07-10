import { Building2 } from "lucide-react";
import { Card } from "../components/ui/Card";
import { IssuerManager } from "../components/IssuerManager";

// Gestionar empresas desde dentro del dashboard (ya con una empresa activa) — cambiar a otra
// vinculada o crear una nueva, mismo IssuerManager que usa OnboardingPage antes de entrar al
// dashboard. Ruta /issuers, accesible desde el icono de empresas del Navbar.
//
// Sin max-w aquí a propósito (ver docs/frontend-architecture.md, regla de ancho): es un
// listado, ocupa todo el ancho disponible. El CompanyForm que IssuerManager muestra al crear
// una empresa se acota a sí mismo, no necesita que esta página lo haga por él.
export function IssuersPage() {
  return (
    <div className="p-4">
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <Building2 className="h-4 w-4 shrink-0 text-(--text-secondary)" />
        Mis empresas
      </h1>
      <Card className="p-4">
        <IssuerManager />
      </Card>
    </div>
  );
}

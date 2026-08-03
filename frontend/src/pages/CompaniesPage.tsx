import { Building2 } from "lucide-react";
import { Card } from "../components/ui/Card";
import { CompanyManager } from "../components/CompanyManager";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";

export function CompaniesPage() {
  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Mis empresas" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <Building2 className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Mis empresas
      </h1>
      <Card className="p-4">
        <CompanyManager />
      </Card>
    </div>
  );
}

import { useEffect, useState } from "react";
import { BadgeCheck, Lock, PackageCheck } from "lucide-react";
import { Card } from "../components/ui/Card";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { Spinner } from "../components/ui/Spinner";
import { Banner } from "../components/ui/Banner";
import { getMyPlan } from "../lib/saas";
import { ApiError } from "../lib/apiClient";
import type { MyPlan } from "../lib/types";

// Mismo catálogo fijo que erp/internal/saas/infrastructure/persistence/postgres/seed/seed.go —
// solo 3 módulos posibles, no vale la pena traer nombre/descripción del backend para esto.
const ALL_MODULES = [
  { code: "electronic_invoicing", name: "Documentos Electrónicos", description: "Facturación electrónica, notas crédito/débito y documento soporte ante la DIAN" },
  { code: "erp_core", name: "ERP completo", description: "Ventas, compras, inventario, contabilidad, terceros y productos" },
  { code: "payroll_hr", name: "Nómina y RRHH", description: "Liquidación de nómina y gestión de empleados" },
];

function formatCents(cents: number) {
  return `$ ${Math.round(cents / 100).toLocaleString("es-CO")}`;
}

function formatDate(iso: string | undefined) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("es-CO", { day: "2-digit", month: "short", year: "numeric" });
}

export function MyPlanPage() {
  const [plan, setPlan] = useState<MyPlan | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getMyPlan()
      .then(setPlan)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar tu plan"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Configuración", to: "/settings/general" }, { label: "Mi plan" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <PackageCheck className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Mi plan
      </h1>

      {loading ? (
        <div className="flex min-h-32 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
      ) : error ? (
        <Banner tone="danger">{error}</Banner>
      ) : plan ? (
        <div className="flex flex-col gap-3">
          <Card className="p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p className="text-xs text-(--text-secondary)">Plan actual</p>
                <p className="text-lg font-semibold text-(--text-primary)">{plan.plan_name}</p>
              </div>
              <div className="text-right">
                <p className="text-xs text-(--text-secondary)">Valor contratado</p>
                <p className="text-lg font-semibold text-(--text-primary)">{formatCents(plan.contracted_cents)}</p>
              </div>
            </div>
            <div className="mt-3 grid grid-cols-2 gap-3 border-t border-(--border-color) pt-3 text-xs sm:grid-cols-3">
              <div>
                <p className="text-(--text-secondary)">Documentos usados</p>
                <p className="font-medium text-(--text-primary)">
                  {plan.documents_used}{plan.included_documents != null ? ` / ${plan.included_documents}` : " (ilimitado)"}
                </p>
              </div>
              <div>
                <p className="text-(--text-secondary)">Próxima renovación</p>
                <p className="font-medium text-(--text-primary)">{formatDate(plan.current_period_end)}</p>
              </div>
              <div>
                <p className="text-(--text-secondary)">Certificado DIAN</p>
                <p className="font-medium text-(--text-primary)">
                  {plan.has_own_certificate ? "Propio" : plan.cert_expires_at ? `Nuestro — vence ${formatDate(plan.cert_expires_at)}` : "—"}
                </p>
              </div>
            </div>
          </Card>

          <Card className="p-4">
            <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Módulos</p>
            <div className="flex flex-col gap-2">
              {ALL_MODULES.map((m) => {
                const unlocked = plan.modules.includes(m.code);
                return (
                  <div key={m.code} className={`flex items-start gap-2 rounded border p-2.5 ${unlocked ? "border-(--border-color)" : "border-(--border-color) opacity-60"}`}>
                    {unlocked ? (
                      <BadgeCheck className="mt-0.5 h-4 w-4 shrink-0 text-(--color-success)" />
                    ) : (
                      <Lock className="mt-0.5 h-4 w-4 shrink-0 text-(--text-muted)" />
                    )}
                    <div>
                      <p className="text-xs font-medium text-(--text-primary)">{m.name}</p>
                      <p className="text-xs text-(--text-secondary)">{m.description}</p>
                    </div>
                  </div>
                );
              })}
            </div>
          </Card>
        </div>
      ) : null}
    </div>
  );
}

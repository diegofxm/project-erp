import { useEffect, useState } from "react";
import { Shield, RefreshCw, ChevronRight } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { useToast } from "../context/ToastContext";
import { ApiError } from "../lib/apiClient";
import {
  adminListPlans,
  adminGetIssuer,
  adminGetSubscription,
  adminAssignPlan,
  adminUpdatePlan,
} from "../lib/admin";
import type { Plan, Subscription, Issuer } from "../lib/types";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";

function formatCOP(cents: number) {
  return `$ ${(cents / 100).toLocaleString("es-CO")}`;
}

// ── Planes ────────────────────────────────────────────────────────────────────
function PlansPanel() {
  const toast = useToast();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    adminListPlans()
      .then(setPlans)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  async function toggleActive(plan: Plan) {
    try {
      const updated = await adminUpdatePlan(plan.id, { ...plan, is_active: !plan.is_active });
      setPlans((ps) => ps.map((p) => (p.id === updated.id ? updated : p)));
      toast.success(`Plan "${updated.name}" ${updated.is_active ? "activado" : "desactivado"}.`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo actualizar el plan");
    }
  }

  if (loading) return <p className="text-xs text-(--text-secondary)">Cargando planes…</p>;

  return (
    <div className="flex flex-col gap-2">
      {plans.map((p) => (
        <div key={p.id} className="flex items-center justify-between rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2">
          <div className="flex flex-col gap-0.5">
            <span className="text-xs font-semibold text-(--text-primary)">{p.name}</span>
            <span className="text-xs text-(--text-secondary)">
              {p.max_documents_per_month == null ? "Ilimitado" : `${p.max_documents_per_month} docs/mes`}
              {p.price_cop > 0 ? ` · ${formatCOP(p.price_cop * 100)}/mes` : " · Gratis"}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${p.is_active ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400"}`}>
              {p.is_active ? "Activo" : "Inactivo"}
            </span>
            <Button variant="secondary" onClick={() => toggleActive(p)}>
              {p.is_active ? "Desactivar" : "Activar"}
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}

// ── Suscripciones por emisor ──────────────────────────────────────────────────
function SubscriptionPanel() {
  const toast = useToast();
  const [issuerId, setIssuerId] = useState("");
  const [issuer, setIssuer] = useState<Issuer | null>(null);
  const [sub, setSub] = useState<Subscription | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [selectedPlanId, setSelectedPlanId] = useState("");
  const [searching, setSearching] = useState(false);
  const [assigning, setAssigning] = useState(false);

  useEffect(() => {
    adminListPlans().then(setPlans).catch(() => {});
  }, []);

  async function handleSearch() {
    if (!issuerId.trim()) return;
    setSearching(true);
    setIssuer(null);
    setSub(null);
    try {
      const [iss, s] = await Promise.allSettled([
        adminGetIssuer(issuerId.trim()),
        adminGetSubscription(issuerId.trim()),
      ]);
      if (iss.status === "fulfilled") setIssuer(iss.value);
      if (s.status === "fulfilled") {
        setSub(s.value);
        setSelectedPlanId(s.value.plan_id);
      }
      if (iss.status === "rejected") toast.error("No se encontró el emisor");
    } finally {
      setSearching(false);
    }
  }

  async function handleAssign() {
    if (!issuer || !selectedPlanId) return;
    setAssigning(true);
    try {
      const s = await adminAssignPlan(issuer.id, selectedPlanId);
      setSub(s);
      toast.success(`Plan asignado: ${s.plan_name}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo asignar el plan");
    } finally {
      setAssigning(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-end gap-2">
        <Input
          label="ID del emisor (UUID)"
          value={issuerId}
          onChange={(e) => setIssuerId(e.target.value)}
          placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
          className="flex-1 font-mono text-xs"
        />
        <Button loading={searching} icon={<ChevronRight className="h-3.5 w-3.5" />} onClick={handleSearch}>
          Buscar
        </Button>
      </div>

      {issuer && (
        <div className="rounded border border-(--border-color) bg-(--bg-primary) p-3">
          <p className="text-xs font-semibold text-(--text-primary)">{issuer.business_name}</p>
          <p className="text-xs text-(--text-secondary)">NIT {issuer.nit}-{issuer.check_digit}</p>
          {sub && (
            <p className="mt-1 text-xs text-(--text-secondary)">
              Plan actual: <strong className="text-(--text-primary)">{sub.plan_name}</strong>
              {sub.max_documents_per_month != null ? ` (${sub.max_documents_per_month} docs/mes)` : " (ilimitado)"}
            </p>
          )}
          {!sub && <p className="mt-1 text-xs text-(--text-secondary)">Sin suscripción activa</p>}

          <div className="mt-3 flex items-end gap-2">
            <label className="flex flex-col gap-1 flex-1">
              <span className="text-xs font-medium text-(--text-secondary)">Asignar plan</span>
              <select
                value={selectedPlanId}
                onChange={(e) => setSelectedPlanId(e.target.value)}
                className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-xs text-(--text-primary)"
              >
                <option value="">— elegir plan —</option>
                {plans.filter((p) => p.is_active).map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </label>
            <Button loading={assigning} disabled={!selectedPlanId} onClick={handleAssign} icon={<RefreshCw className="h-3.5 w-3.5" />}>
              Asignar
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Página principal ─────────────────────────────────────────────────────────
export function AdminPage() {
  const { user } = useAuth();

  if (!user?.is_superadmin) {
    return (
      <div className="p-8 text-center">
        <Shield className="mx-auto mb-2 h-8 w-8 text-(--text-secondary)" />
        <p className="text-sm text-(--text-secondary)">Acceso restringido a superadministradores.</p>
      </div>
    );
  }

  return (
    <div className="p-4">
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <Shield className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Panel de administración
      </h1>
      <div className="flex flex-col gap-4">
        <Card className="flex flex-col gap-3 p-4">
          <h2 className="text-xs font-semibold text-(--text-primary)">Planes</h2>
          <PlansPanel />
        </Card>
        <Card className="flex flex-col gap-3 p-4">
          <h2 className="text-xs font-semibold text-(--text-primary)">Suscripción por emisor</h2>
          <p className="text-xs text-(--text-secondary)">Busca un emisor por su UUID para consultar o cambiar su plan activo.</p>
          <SubscriptionPanel />
        </Card>
      </div>
    </div>
  );
}

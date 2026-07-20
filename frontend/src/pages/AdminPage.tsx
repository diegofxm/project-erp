import { useEffect, useState } from "react";
import { BarChart2, Building2, Layers, RefreshCw, ChevronRight } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { useToast } from "../context/ToastContext";
import { ApiError } from "../lib/apiClient";
import {
  adminListPlans,
  adminGetIssuer,
  adminGetSubscription,
  adminGetIssuerSettings,
  adminAssignPlan,
  adminUpdatePlan,
  adminUpdateIssuerSettings,
  adminGetBillingSummary,
} from "../lib/admin";
import type { BillingEntry, Plan, Subscription, Issuer } from "../lib/types";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";

function formatCOP(pesos: number) {
  return `$ ${pesos.toLocaleString("es-CO")}`;
}

function AccessDenied() {
  return (
    <div className="flex h-full items-center justify-center p-8 text-center">
      <p className="text-sm text-(--text-secondary)">Acceso restringido a superadministradores.</p>
    </div>
  );
}

function withSuperAdmin<P extends object>(Component: React.ComponentType<P>) {
  return function SuperAdminGuard(props: P) {
    const { user } = useAuth();
    if (!user?.is_superadmin) return <AccessDenied />;
    return <Component {...props} />;
  };
}

// ── Facturación ──────────────────────────────────────────────────────────────
function BillingContent() {
  const [entries, setEntries] = useState<BillingEntry[]>([]);
  const [loading, setLoading] = useState(true);

  function load() {
    setLoading(true);
    adminGetBillingSummary()
      .then(setEntries)
      .catch(() => {})
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  const totalDocs = entries.reduce((s, e) => s + e.DocsThisMonth, 0);
  const totalCOP = entries.reduce((s, e) => s + e.TotalCOP, 0);

  return (
    <>
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <BarChart2 className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Facturación
        </h1>
        <Button variant="secondary" onClick={load} icon={<RefreshCw className="h-3.5 w-3.5" />}>
          Actualizar
        </Button>
      </div>

      <p className="mb-3 text-xs text-(--text-secondary)">
        Documentos emitidos en el mes actual por todos los emisores.
      </p>

      {loading ? (
        <p className="text-xs text-(--text-secondary)">Cargando…</p>
      ) : entries.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Ningún emisor ha emitido documentos este mes.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Empresa</th>
                <th className="px-3 py-2 font-medium">NIT</th>
                <th className="px-3 py-2 text-right font-medium">Docs</th>
                <th className="px-3 py-2 text-right font-medium">$/doc</th>
                <th className="px-3 py-2 text-right font-medium">Subtotal</th>
                <th className="px-3 py-2 text-right font-medium">IVA (19%)</th>
                <th className="px-3 py-2 text-right font-medium">Total</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e, i) => (
                <tr
                  key={e.IssuerID}
                  className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}
                >
                  <td className="px-3 py-2 text-(--text-primary)">{e.BusinessName}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{e.NIT}</td>
                  <td className="px-3 py-2 text-right text-(--text-primary)">{e.DocsThisMonth}</td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">{formatCOP(e.PricePerDocumentCOP)}</td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">{formatCOP(e.SubtotalCOP)}</td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">{formatCOP(e.IVA)}</td>
                  <td className="px-3 py-2 text-right font-semibold text-(--text-primary)">{formatCOP(e.TotalCOP)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot className="bg-(--bg-tertiary) font-semibold text-(--text-primary)">
              <tr className="border-t-2 border-(--border-color)">
                <td colSpan={2} className="px-3 py-2">Total ({entries.length} empresas)</td>
                <td className="px-3 py-2 text-right">{totalDocs}</td>
                <td colSpan={3} />
                <td className="px-3 py-2 text-right">{formatCOP(totalCOP)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      )}
    </>
  );
}

// ── Por emisor ───────────────────────────────────────────────────────────────
function IssuerContent() {
  const toast = useToast();
  const [issuerId, setIssuerId] = useState("");
  const [issuer, setIssuer] = useState<Issuer | null>(null);
  const [sub, setSub] = useState<Subscription | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [selectedPlanId, setSelectedPlanId] = useState("");
  const [priceInput, setPriceInput] = useState("");
  const [searching, setSearching] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [savingPrice, setSavingPrice] = useState(false);

  useEffect(() => {
    adminListPlans().then(setPlans).catch(() => {});
  }, []);

  async function handleSearch() {
    if (!issuerId.trim()) return;
    setSearching(true);
    setIssuer(null);
    setSub(null);
    setPriceInput("");
    try {
      const [issRes, subRes] = await Promise.allSettled([
        adminGetIssuer(issuerId.trim()),
        adminGetSubscription(issuerId.trim()),
      ]);
      if (issRes.status === "fulfilled") {
        setIssuer(issRes.value);
        const st = await adminGetIssuerSettings(issRes.value.id).catch(() => null);
        if (st) setPriceInput(String(st.price_per_document_cop));
      } else {
        toast.error("No se encontró el emisor");
      }
      if (subRes.status === "fulfilled") {
        setSub(subRes.value);
        setSelectedPlanId(subRes.value.plan_id);
      }
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

  async function handleSavePrice() {
    if (!issuer) return;
    const price = parseInt(priceInput, 10);
    if (isNaN(price) || price < 0) {
      toast.error("El precio debe ser un número positivo.");
      return;
    }
    setSavingPrice(true);
    try {
      await adminUpdateIssuerSettings(issuer.id, { price_per_document_cop: price });
      toast.success(`Precio actualizado: ${formatCOP(price)} por documento.`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo actualizar el precio");
    } finally {
      setSavingPrice(false);
    }
  }

  return (
    <>
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Building2 className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Por emisor
        </h1>
      </div>

      <p className="mb-3 text-xs text-(--text-secondary)">
        Busca un emisor por su UUID para gestionar su plan y precio por documento.
      </p>

      <div className="flex items-end gap-2 mb-4">
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
        <div className="rounded border border-(--border-color) overflow-hidden">
          <div className="bg-(--bg-tertiary) px-3 py-2 border-b border-(--border-color)">
            <p className="text-xs font-semibold text-(--text-primary)">{issuer.business_name}</p>
            <p className="text-xs text-(--text-secondary)">NIT {issuer.nit}-{issuer.check_digit}</p>
            {sub
              ? <p className="mt-0.5 text-xs text-(--text-secondary)">Plan: <strong className="text-(--text-primary)">{sub.plan_name}</strong></p>
              : <p className="mt-0.5 text-xs text-(--text-secondary)">Sin suscripción activa</p>
            }
          </div>

          <div className="flex items-end gap-3 px-3 py-3 border-b border-(--border-color)">
            <Input
              label="Precio por documento (COP)"
              type="number"
              min="0"
              value={priceInput}
              onChange={(e) => setPriceInput(e.target.value)}
              className="w-44"
            />
            <Button loading={savingPrice} onClick={handleSavePrice}>Guardar precio</Button>
          </div>

          <div className="flex items-end gap-3 px-3 py-3">
            <label className="flex flex-col gap-1 flex-1">
              <span className="text-xs font-medium text-(--text-secondary)">Cambiar plan</span>
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
    </>
  );
}

// ── Planes ───────────────────────────────────────────────────────────────────
function PlansContent() {
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

  return (
    <>
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Layers className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Planes
        </h1>
      </div>

      {loading ? (
        <p className="text-xs text-(--text-secondary)">Cargando planes…</p>
      ) : (
        <div className="rounded border border-(--border-color) overflow-hidden">
          {plans.map((p, i) => (
            <div
              key={p.id}
              className={`flex items-center justify-between px-3 py-2 ${
                i > 0 ? "border-t border-(--border-color)" : ""
              } ${i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}`}
            >
              <div className="flex flex-col gap-0.5">
                <span className="text-xs font-semibold text-(--text-primary)">{p.name}</span>
                <span className="text-xs text-(--text-secondary)">
                  {p.max_documents_per_month == null ? "Ilimitado" : `${p.max_documents_per_month} docs/mes`}
                  {p.price_cop > 0 ? ` · ${formatCOP(p.price_cop)}/mes` : " · Gratis"}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                  p.is_active
                    ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                    : "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400"
                }`}>
                  {p.is_active ? "Activo" : "Inactivo"}
                </span>
                <Button variant="secondary" onClick={() => toggleActive(p)}>
                  {p.is_active ? "Desactivar" : "Activar"}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </>
  );
}

// ── Exports de página ────────────────────────────────────────────────────────
export const AdminBillingPage = withSuperAdmin(function AdminBillingPage() {
  return <div className="p-4"><BillingContent /></div>;
});

export const AdminIssuerPage = withSuperAdmin(function AdminIssuerPage() {
  return <div className="p-4"><IssuerContent /></div>;
});

export const AdminPlansPage = withSuperAdmin(function AdminPlansPage() {
  return <div className="p-4"><PlansContent /></div>;
});

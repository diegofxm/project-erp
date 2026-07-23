import { useEffect, useState } from "react";
import { Navigate } from "react-router";
import { BarChart2, Building2, CalendarClock, ClipboardList, Layers, RefreshCw, ChevronRight, Users, Plus } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import { apiClient, ApiError } from "../lib/apiClient";
import {
  adminListPlans,
  adminGetIssuer,
  adminGetSubscription,
  adminGetIssuerSettings,
  adminAssignPlan,
  adminUpdatePlan,
  adminApplyPlanIncrement,
  adminUpdateIssuerSettings,
  adminAffiliateIssuer,
  adminRenewIssuer,
  adminGetBillingSummary,
  adminGetRenewalsSummary,
  adminListUsers,
  adminCreateUser,
  adminListIssuerPayments,
  adminListProspects,
  adminApproveProspect,
  adminRejectProspect,
  adminProspectCedulaUrl,
  adminProspectRutUrl,
} from "../lib/admin";
import type { AdminUser, BillingEntry, IssuerSettings, Payment, Plan, Prospect, RenewalEntry, Subscription, Issuer } from "../lib/types";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";

function formatCOP(pesos: number) {
  return `$ ${pesos.toLocaleString("es-CO")}`;
}

const PAYMENT_TYPE_LABEL: Record<string, string> = {
  affiliation: "Afiliación",
  renewal:     "Renovación",
  documents:   "Documentos",
};

function formatDate(iso: string | null | undefined) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("es-CO", { day: "2-digit", month: "short", year: "numeric" });
}

function withSuperAdmin<P extends object>(Component: React.ComponentType<P>) {
  return function SuperAdminGuard(props: P) {
    const { user } = useAuth();
    if (!user?.is_superadmin) return <Navigate to="/" replace />;
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
        <Button variant="secondary" onClick={load} icon={<RefreshCw className="h-3.5 w-3.5" />}>Actualizar</Button>
      </div>
      <p className="mb-3 text-xs text-(--text-secondary)">Documentos emitidos en el mes actual por todos los emisores.</p>

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
                <th className="px-3 py-2 text-right font-medium">IVA 19%</th>
                <th className="px-3 py-2 text-right font-medium">Total</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e, i) => (
                <tr key={e.IssuerID} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 text-(--text-primary)">{e.BusinessName}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{e.NIT}</td>
                  <td className="px-3 py-2 text-right">{e.DocsThisMonth}</td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">{formatCOP(e.PricePerDocumentCOP)}</td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">{formatCOP(e.SubtotalCOP)}</td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">{formatCOP(e.IVA)}</td>
                  <td className="px-3 py-2 text-right font-semibold">{formatCOP(e.TotalCOP)}</td>
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

// ── Renovaciones ─────────────────────────────────────────────────────────────
function RenewalsContent() {
  const [entries, setEntries] = useState<RenewalEntry[]>([]);
  const [loading, setLoading] = useState(true);

  function load() {
    setLoading(true);
    adminGetRenewalsSummary()
      .then(setEntries)
      .catch(() => {})
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  function urgencyClass(days: number) {
    if (days === 0) return "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400";
    if (days <= 15) return "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400";
    if (days <= 30) return "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400";
    return "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400";
  }

  return (
    <>
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <CalendarClock className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Renovaciones
        </h1>
        <Button variant="secondary" onClick={load} icon={<RefreshCw className="h-3.5 w-3.5" />}>Actualizar</Button>
      </div>
      <p className="mb-3 text-xs text-(--text-secondary)">Emisores cuya renovación anual vence en los próximos 90 días (o ya venció).</p>

      {loading ? (
        <p className="text-xs text-(--text-secondary)">Cargando…</p>
      ) : entries.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Ninguna renovación próxima en los próximos 90 días.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Empresa</th>
                <th className="px-3 py-2 font-medium">NIT</th>
                <th className="px-3 py-2 font-medium">Afiliación</th>
                <th className="px-3 py-2 font-medium">Vencimiento</th>
                <th className="px-3 py-2 text-right font-medium">Tarifa renovación</th>
                <th className="px-3 py-2 text-center font-medium">Estado</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e, i) => (
                <tr key={e.IssuerID} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 text-(--text-primary)">{e.BusinessName}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{e.NIT}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{formatDate(e.AffiliatedAt)}</td>
                  <td className="px-3 py-2 font-medium text-(--text-primary)">{formatDate(e.RenewalDueAt)}</td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">{formatCOP(e.RenewalFeeCOP)}</td>
                  <td className="px-3 py-2 text-center">
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${urgencyClass(e.DaysUntilRenewal)}`}>
                      {e.DaysUntilRenewal === 0 ? "Vencido" : `${e.DaysUntilRenewal}d`}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
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
  const [settings, setSettings] = useState<IssuerSettings | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [selectedPlanId, setSelectedPlanId] = useState("");
  const [priceInput, setPriceInput] = useState("");
  const [affFeeInput, setAffFeeInput] = useState("");
  const [renFeeInput, setRenFeeInput] = useState("");
  const [paymentsHistory, setPaymentsHistory] = useState<Payment[]>([]);
  const [searching, setSearching] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [savingSettings, setSavingSettings] = useState(false);
  const [affiliating, setAffiliating] = useState(false);
  const [renewing, setRenewing] = useState(false);

  useEffect(() => { adminListPlans().then(setPlans).catch(() => {}); }, []);

  async function handleSearch() {
    if (!issuerId.trim()) return;
    setSearching(true);
    setIssuer(null);
    setSub(null);
    setSettings(null);
    setPaymentsHistory([]);
    try {
      const [issRes, subRes] = await Promise.allSettled([
        adminGetIssuer(issuerId.trim()),
        adminGetSubscription(issuerId.trim()),
      ]);
      if (issRes.status === "fulfilled") {
        setIssuer(issRes.value);
        const [st, pmts] = await Promise.all([
          adminGetIssuerSettings(issRes.value.id).catch(() => null),
          adminListIssuerPayments(issRes.value.id).catch(() => []),
        ]);
        if (st) {
          setSettings(st);
          setPriceInput(String(st.price_per_document_cop));
          setAffFeeInput(String(st.affiliation_fee_cop));
          setRenFeeInput(String(st.renewal_fee_cop));
        }
        setPaymentsHistory(pmts);
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

  async function handleSaveSettings() {
    if (!issuer) return;
    const price = parseInt(priceInput, 10);
    const affFee = parseInt(affFeeInput, 10);
    const renFee = parseInt(renFeeInput, 10);
    if ([price, affFee, renFee].some((v) => isNaN(v) || v < 0)) {
      toast.error("Las tarifas deben ser números positivos.");
      return;
    }
    setSavingSettings(true);
    try {
      const st = await adminUpdateIssuerSettings(issuer.id, {
        price_per_document_cop: price,
        affiliation_fee_cop: affFee,
        renewal_fee_cop: renFee,
      });
      setSettings(st);
      toast.success("Tarifas actualizadas.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudieron actualizar las tarifas");
    } finally {
      setSavingSettings(false);
    }
  }

  async function handleAffiliate() {
    if (!issuer) return;
    const fee = parseInt(affFeeInput, 10);
    if (isNaN(fee) || fee < 0) { toast.error("Ingresa una tarifa válida."); return; }
    setAffiliating(true);
    try {
      const st = await adminAffiliateIssuer(issuer.id, fee);
      setSettings(st);
      adminListIssuerPayments(issuer.id).then(setPaymentsHistory).catch(() => {});
      toast.success(`Afiliación registrada. Vigente hasta ${formatDate(st.renewal_due_at)}.`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo registrar la afiliación");
    } finally {
      setAffiliating(false);
    }
  }

  async function handleRenew() {
    if (!issuer) return;
    const fee = parseInt(renFeeInput, 10);
    if (isNaN(fee) || fee < 0) { toast.error("Ingresa una tarifa válida."); return; }
    setRenewing(true);
    try {
      const st = await adminRenewIssuer(issuer.id, fee);
      setSettings(st);
      adminListIssuerPayments(issuer.id).then(setPaymentsHistory).catch(() => {});
      toast.success(`Renovación registrada. Nueva vigencia hasta ${formatDate(st.renewal_due_at)}.`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo registrar la renovación");
    } finally {
      setRenewing(false);
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

  const isAffiliated = !!settings?.affiliated_at;

  return (
    <>
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Building2 className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Por emisor
        </h1>
      </div>
      <p className="mb-3 text-xs text-(--text-secondary)">Busca un emisor por su UUID para gestionar su plan, tarifas y vigencia.</p>

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
        <>
        <div className="rounded border border-(--border-color) overflow-hidden">
          {/* Cabecera del emisor */}
          <div className="bg-(--bg-tertiary) px-3 py-2 border-b border-(--border-color)">
            <p className="text-xs font-semibold text-(--text-primary)">{issuer.business_name}</p>
            <p className="text-xs text-(--text-secondary)">NIT {issuer.nit}-{issuer.check_digit}</p>
            <div className="mt-1 flex flex-wrap gap-3 text-xs text-(--text-secondary)">
              <span>Plan: <strong className="text-(--text-primary)">{sub?.plan_name ?? "sin suscripción"}</strong></span>
              <span>Afiliado: <strong className="text-(--text-primary)">{formatDate(settings?.affiliated_at)}</strong></span>
              <span>Vence: <strong className={settings?.renewal_due_at && new Date(settings.renewal_due_at) < new Date() ? "text-red-600" : "text-(--text-primary)"}>
                {formatDate(settings?.renewal_due_at)}
              </strong></span>
            </div>
          </div>

          {/* Tarifas */}
          <div className="border-b border-(--border-color) px-3 py-3">
            <p className="mb-2 text-xs font-medium text-(--text-secondary)">Tarifas pactadas</p>
            <div className="flex flex-wrap items-end gap-3">
              <Input label="$/documento (COP)" type="number" min="0" value={priceInput}
                onChange={(e) => setPriceInput(e.target.value)} className="w-36" />
              <Input label="Afiliación (COP)" type="number" min="0" value={affFeeInput}
                onChange={(e) => setAffFeeInput(e.target.value)} className="w-36" />
              <Input label="Renovación (COP)" type="number" min="0" value={renFeeInput}
                onChange={(e) => setRenFeeInput(e.target.value)} className="w-36" />
              <Button loading={savingSettings} onClick={handleSaveSettings}>Guardar tarifas</Button>
            </div>
          </div>

          {/* Afiliación / Renovación */}
          <div className="border-b border-(--border-color) px-3 py-3 flex flex-wrap gap-2">
            <Button
              loading={affiliating}
              onClick={handleAffiliate}
              variant={isAffiliated ? "secondary" : "primary"}
            >
              {isAffiliated ? "Re-afiliar" : "Registrar afiliación"}
            </Button>
            <Button
              loading={renewing}
              onClick={handleRenew}
              disabled={!isAffiliated}
              variant="secondary"
            >
              Renovar (+1 año)
            </Button>
          </div>

          {/* Plan */}
          <div className="px-3 py-3 flex items-end gap-3">
            <label className="flex flex-col gap-1 flex-1">
              <span className="text-xs font-medium text-(--text-secondary)">Cambiar plan</span>
              <select value={selectedPlanId} onChange={(e) => setSelectedPlanId(e.target.value)}
                className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-xs text-(--text-primary)">
                <option value="">— elegir plan —</option>
                {plans.filter((p) => p.is_active).map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </label>
            <Button loading={assigning} disabled={!selectedPlanId} onClick={handleAssign}
              icon={<RefreshCw className="h-3.5 w-3.5" />}>
              Asignar
            </Button>
          </div>
        </div>

        {/* Historial de pagos */}
        <div className="mt-4">
          <p className="mb-2 text-xs font-medium text-(--text-secondary)">Historial de pagos</p>
          {paymentsHistory.length === 0 ? (
            <p className="text-xs text-(--text-secondary)">Sin pagos registrados.</p>
          ) : (
            <div className="overflow-x-auto rounded border border-(--border-color)">
              <table className="w-full text-xs">
                <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium">Tipo</th>
                    <th className="px-3 py-2 text-right font-medium">Valor (COP)</th>
                    <th className="px-3 py-2 text-left font-medium">Fecha</th>
                  </tr>
                </thead>
                <tbody>
                  {paymentsHistory.map((p, i) => (
                    <tr key={p.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                      <td className="px-3 py-2 text-(--text-primary)">
                        {PAYMENT_TYPE_LABEL[p.type] ?? p.type}
                      </td>
                      <td className="px-3 py-2 text-right font-medium text-(--text-primary)">{formatCOP(p.amount_cop)}</td>
                      <td className="px-3 py-2 text-(--text-secondary)">{formatDate(p.paid_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
        </>
      )}
    </>
  );
}

// ── Planes ───────────────────────────────────────────────────────────────────
function PlansContent() {
  const toast = useToast();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [applyingId, setApplyingId] = useState<string | null>(null);

  useEffect(() => {
    adminListPlans().then(setPlans).catch(() => {}).finally(() => setLoading(false));
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

  async function applyIncrement(plan: Plan) {
    if (plan.annual_increment_pct <= 0) {
      toast.error("Este plan no tiene tasa de incremento configurada.");
      return;
    }
    setApplyingId(plan.id);
    try {
      const updated = await adminApplyPlanIncrement(plan.id);
      setPlans((ps) => ps.map((p) => (p.id === updated.id ? updated : p)));
      toast.success(`Incremento de ${plan.annual_increment_pct}% aplicado a "${updated.name}".`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo aplicar el incremento");
    } finally {
      setApplyingId(null);
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
            <div key={p.id} className={`border-b border-(--border-color) last:border-0 px-3 py-3 ${i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}`}>
              <div className="flex items-start justify-between gap-2 mb-2">
                <div>
                  <span className="text-xs font-semibold text-(--text-primary)">{p.name}</span>
                  {p.description && <p className="text-xs text-(--text-secondary) mt-0.5">{p.description}</p>}
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${p.is_active ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400"}`}>
                    {p.is_active ? "Activo" : "Inactivo"}
                  </span>
                  <Button variant="secondary" onClick={() => toggleActive(p)}>
                    {p.is_active ? "Desactivar" : "Activar"}
                  </Button>
                </div>
              </div>
              <div className="flex flex-wrap gap-4 text-xs text-(--text-secondary)">
                <span>Docs/mes: <strong className="text-(--text-primary)">{p.max_documents_per_month ?? "Ilimitado"}</strong></span>
                <span>Afiliación: <strong className="text-(--text-primary)">{formatCOP(p.affiliation_fee_cop)}</strong></span>
                <span>Renovación: <strong className="text-(--text-primary)">{formatCOP(p.renewal_fee_cop)}</strong></span>
                <span>Incremento anual: <strong className="text-(--text-primary)">{p.annual_increment_pct > 0 ? `${p.annual_increment_pct}%` : "—"}</strong></span>
              </div>
              {p.annual_increment_pct > 0 && (
                <div className="mt-2">
                  <Button
                    variant="secondary"
                    loading={applyingId === p.id}
                    onClick={() => applyIncrement(p)}
                  >
                    Aplicar incremento {p.annual_increment_pct}%
                  </Button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </>
  );
}

// ── Usuarios ─────────────────────────────────────────────────────────────────
function UsersContent() {
  const toast = useToast();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);

  function load() {
    setLoading(true);
    adminListUsers().then(setUsers).catch(() => {}).finally(() => setLoading(false));
  }

  useEffect(() => { load(); }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    try {
      const u = await adminCreateUser({ email, name });
      setUsers((prev) => [u, ...prev]);
      setEmail("");
      setName("");
      toast.success(`Invitación enviada a ${u.email}.`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo crear el usuario");
    } finally {
      setCreating(false);
    }
  }

  return (
    <>
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Users className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Usuarios
        </h1>
      </div>

      {/* Formulario para crear usuario invitado */}
      <form onSubmit={handleCreate} className="mb-4 flex flex-wrap items-end gap-2 rounded border border-(--border-color) bg-(--bg-secondary) p-3">
        <Input label="Nombre" value={name} onChange={(e) => setName(e.target.value)} required className="w-44" />
        <Input label="Correo" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required className="w-56" />
        <Button type="submit" loading={creating} icon={<Plus className="h-3.5 w-3.5" />}>
          Invitar usuario
        </Button>
      </form>

      {loading ? (
        <p className="text-xs text-(--text-secondary)">Cargando usuarios…</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 text-left font-medium">Nombre</th>
                <th className="px-3 py-2 text-left font-medium">Correo</th>
                <th className="px-3 py-2 text-left font-medium">Estado</th>
                <th className="px-3 py-2 text-left font-medium">Creado</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u, i) => (
                <tr key={u.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 font-medium text-(--text-primary)">
                    {u.name}
                    {u.is_superadmin && <span className="ml-1.5 rounded-full bg-amber-100 px-1.5 text-[10px] font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">Super</span>}
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">{u.email}</td>
                  <td className="px-3 py-2">
                    {u.invite_accepted_at ? (
                      <span className="rounded-full bg-green-100 px-2 py-0.5 text-[10px] font-medium text-green-700 dark:bg-green-900/30 dark:text-green-400">Activo</span>
                    ) : (
                      <span className="rounded-full bg-yellow-100 px-2 py-0.5 text-[10px] font-medium text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400">Invitación pendiente</span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">{formatDate(u.created_at)}</td>
                </tr>
              ))}
              {users.length === 0 && (
                <tr><td colSpan={4} className="px-3 py-4 text-center text-(--text-secondary)">No hay usuarios.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}

// ── Solicitudes (prospects) ───────────────────────────────────────────────────
const STATUS_LABEL: Record<string, string> = {
  pending:  "Pendiente",
  approved: "Aprobado",
  rejected: "Rechazado",
};
const STATUS_CLASS: Record<string, string> = {
  pending:  "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
  approved: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  rejected: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
};

function ProspectsContent() {
  const [prospects, setProspects] = useState<Prospect[]>([]);
  const [loading, setLoading] = useState(true);
  const confirm = useConfirm();
  const toast = useToast();

  function load() {
    setLoading(true);
    adminListProspects().then(setProspects).catch(() => {}).finally(() => setLoading(false));
  }

  useEffect(() => { load(); }, []);

  async function openPDF(url: string, filename: string) {
    try {
      const blob = await apiClient.getBlob(url);
      const objectUrl = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = objectUrl;
      a.target = "_blank";
      a.download = filename;
      a.click();
      setTimeout(() => URL.revokeObjectURL(objectUrl), 5000);
    } catch {
      toast.error("No se pudo descargar el documento.");
    }
  }

  async function handleApprove(p: Prospect) {
    if (!(await confirm(`¿Aprobar la solicitud de ${p.name} y enviar invitación a ${p.email}?`, { confirmLabel: "Aprobar y enviar" }))) return;
    try {
      const updated = await adminApproveProspect(p.id);
      setProspects((prev) => prev.map((x) => (x.id === updated.id ? updated : x)));
      toast.success(`Invitación enviada a ${p.email}.`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo aprobar la solicitud.");
    }
  }

  async function handleReject(p: Prospect) {
    if (!(await confirm(`¿Rechazar la solicitud de ${p.name}?`, { tone: "danger", confirmLabel: "Rechazar" }))) return;
    try {
      const updated = await adminRejectProspect(p.id);
      setProspects((prev) => prev.map((x) => (x.id === updated.id ? updated : x)));
      toast.success("Solicitud rechazada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo rechazar la solicitud.");
    }
  }

  return (
    <>
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <ClipboardList className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Solicitudes de acceso
        </h1>
        <button onClick={load} className="flex items-center gap-1 rounded px-2 py-1 text-xs text-(--text-secondary) hover:bg-(--bg-hover)" title="Recargar">
          <RefreshCw className="h-3.5 w-3.5" />
        </button>
      </div>

      {loading ? (
        <p className="text-xs text-(--text-secondary)">Cargando solicitudes…</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 text-left font-medium">Nombre</th>
                <th className="px-3 py-2 text-left font-medium">Correo</th>
                <th className="px-3 py-2 text-left font-medium">NIT</th>
                <th className="px-3 py-2 text-left font-medium">Documentos</th>
                <th className="px-3 py-2 text-left font-medium">Estado</th>
                <th className="px-3 py-2 text-left font-medium">Fecha</th>
                <th className="px-3 py-2 text-left font-medium">Acciones</th>
              </tr>
            </thead>
            <tbody>
              {prospects.map((p, i) => (
                <tr key={p.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 font-medium text-(--text-primary)">{p.name}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{p.email}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{p.nit || "—"}</td>
                  <td className="px-3 py-2">
                    <div className="flex gap-1">
                      {p.has_cedula && (
                        <button onClick={() => openPDF(adminProspectCedulaUrl(p.id), `cedula-${p.id}.pdf`)}
                          className="rounded bg-(--bg-hover) px-1.5 py-0.5 text-[10px] font-medium text-(--text-secondary) hover:text-(--text-primary)">
                          Cédula
                        </button>
                      )}
                      {p.has_rut && (
                        <button onClick={() => openPDF(adminProspectRutUrl(p.id), `rut-${p.id}.pdf`)}
                          className="rounded bg-(--bg-hover) px-1.5 py-0.5 text-[10px] font-medium text-(--text-secondary) hover:text-(--text-primary)">
                          RUT
                        </button>
                      )}
                      {!p.has_cedula && !p.has_rut && <span className="text-(--text-secondary)">—</span>}
                    </div>
                  </td>
                  <td className="px-3 py-2">
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${STATUS_CLASS[p.status]}`}>
                      {STATUS_LABEL[p.status]}
                    </span>
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">{formatDate(p.created_at)}</td>
                  <td className="px-3 py-2">
                    {p.status === "pending" && (
                      <div className="flex gap-1.5">
                        <button onClick={() => handleApprove(p)}
                          className="rounded bg-green-100 px-2 py-0.5 text-[10px] font-medium text-green-700 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400 dark:hover:bg-green-900/50">
                          Aprobar
                        </button>
                        <button onClick={() => handleReject(p)}
                          className="rounded bg-red-100 px-2 py-0.5 text-[10px] font-medium text-red-700 hover:bg-red-200 dark:bg-red-900/30 dark:text-red-400 dark:hover:bg-red-900/50">
                          Rechazar
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
              {prospects.length === 0 && (
                <tr><td colSpan={7} className="px-3 py-4 text-center text-(--text-secondary)">No hay solicitudes.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}

// ── Exports de página ────────────────────────────────────────────────────────
export const AdminBillingPage = withSuperAdmin(function AdminBillingPage() {
  return <div className="p-4"><BillingContent /></div>;
});

export const AdminRenewalsPage = withSuperAdmin(function AdminRenewalsPage() {
  return <div className="p-4"><RenewalsContent /></div>;
});

export const AdminIssuerPage = withSuperAdmin(function AdminIssuerPage() {
  return <div className="p-4"><IssuerContent /></div>;
});

export const AdminPlansPage = withSuperAdmin(function AdminPlansPage() {
  return <div className="p-4"><PlansContent /></div>;
});

export const AdminUsersPage = withSuperAdmin(function AdminUsersPage() {
  return <div className="p-4"><UsersContent /></div>;
});

export const AdminProspectsPage = withSuperAdmin(function AdminProspectsPage() {
  return <div className="p-4"><ProspectsContent /></div>;
});

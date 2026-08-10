import { useEffect, useState } from "react";
import { Navigate } from "react-router";
import { BarChart2, Building2, CalendarClock, ClipboardList, Layers, RefreshCw, ChevronRight, Users, Plus, Settings2, Pencil } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import { apiClient, ApiError } from "../lib/apiClient";
import {
  adminListModules,
  adminListPlans,
  adminGetCompanyInfo,
  adminGetSubscription,
  adminAssignPlan,
  adminRenewSubscription,
  adminCreatePlan,
  adminUpdatePlan,
  adminApplyPlanIncrement,
  adminGetSettings,
  adminUpdateSettings,
  adminGetBillingSummary,
  adminGetRenewalsSummary,
  adminListUsers,
  adminSetUserSuperAdmin,
  adminListCompanyPayments,
  adminRecordPayment,
  adminListProspects,
  adminApproveProspect,
  adminRejectProspect,
  adminProspectCedulaUrl,
  adminProspectRutUrl,
} from "../lib/admin";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { InfoTip } from "../components/ui/InfoTip";
import type {
  AdminUser, BillingEntry, BillingCycle, CompanyInfo, Payment, Plan, Prospect, RenewalEntry,
  SaasModule, SaasSettings, Subscription,
} from "../lib/types";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";
import { Select } from "../components/ui/Select";

// Los precios del módulo SaaS viajan en centavos (igual que accounting/electronic) — helpers
// separados de formatCOP (pesos) que usa el resto de la app para no mezclar unidades por error.
function formatCents(cents: number) {
  return `$ ${Math.round(cents / 100).toLocaleString("es-CO")}`;
}
function pesosToCents(pesos: string) {
  const n = parseFloat(pesos);
  return isNaN(n) ? 0 : Math.round(n * 100);
}
function centsToPesosInput(cents: number) {
  return cents === 0 ? "" : String(cents / 100);
}

const PAYMENT_TYPE_LABEL: Record<string, string> = {
  plan: "Plan",
  certificate: "Certificado",
  overage: "Excedente de documentos",
};

const BILLING_CYCLE_LABEL: Record<BillingCycle, string> = {
  monthly: "Mensual",
  annual: "Anual",
  none: "Sin ciclo (gratis)",
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
  const [ivaRateBP, setIvaRateBP] = useState(0);
  const [loading, setLoading] = useState(true);

  function load() {
    setLoading(true);
    Promise.all([adminGetBillingSummary(), adminGetSettings()])
      .then(([e, s]) => { setEntries(e); setIvaRateBP(s.iva_rate_bp); })
      .catch(() => {})
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  const totalDocs = entries.reduce((s, e) => s + e.documents_used, 0);
  const totalCents = entries.reduce((s, e) => s + e.total_cents, 0);
  const ivaPct = (ivaRateBP / 100).toFixed(2);

  return (
    <>
      <Breadcrumbs items={[{ label: "Comando", to: "/admin/billing" }, { label: "Facturación" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <BarChart2 className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Facturación
          <InfoTip>Empresas con suscripción activa — cargos del período vigente de cada una.</InfoTip>
        </h1>
        <Button variant="secondary" onClick={load} icon={<RefreshCw className="h-3.5 w-3.5" />}>Actualizar</Button>
      </div>

      {loading ? (
        <p className="text-xs text-(--text-secondary)">Cargando…</p>
      ) : entries.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Ninguna empresa con suscripción activa.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Empresa</th>
                <th className="px-3 py-2 font-medium">NIT</th>
                <th className="px-3 py-2 font-medium">Plan</th>
                <th className="px-3 py-2 text-right font-medium">Docs usados</th>
                <th className="px-3 py-2 text-right font-medium">Base</th>
                <th className="px-3 py-2 text-right font-medium">Excedente</th>
                <th className="px-3 py-2 text-right font-medium">IVA {ivaPct}%</th>
                <th className="px-3 py-2 text-right font-medium">Total</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e, i) => (
                <tr key={e.company_id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 text-(--text-primary)">{e.business_name || "—"}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{e.nit}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{e.plan_name}</td>
                  <td className="px-3 py-2 text-right">
                    {e.documents_used}{e.documents_included != null ? ` / ${e.documents_included}` : " (ilimitado)"}
                  </td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">{formatCents(e.base_cents)}</td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">
                    {e.overage_cents > 0 ? `${formatCents(e.overage_cents)} (${e.overage_documents})` : "—"}
                  </td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">{formatCents(e.iva_cents)}</td>
                  <td className="px-3 py-2 text-right font-semibold">{formatCents(e.total_cents)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot className="bg-(--bg-tertiary) font-semibold text-(--text-primary)">
              <tr className="border-t-2 border-(--border-color)">
                <td colSpan={3} className="px-3 py-2">Total ({entries.length} empresas)</td>
                <td className="px-3 py-2 text-right">{totalDocs}</td>
                <td colSpan={3} />
                <td className="px-3 py-2 text-right">{formatCents(totalCents)}</td>
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
    if (days <= 0) return "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400";
    if (days <= 15) return "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400";
    if (days <= 30) return "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400";
    return "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400";
  }

  return (
    <>
      <Breadcrumbs items={[{ label: "Comando", to: "/admin/billing" }, { label: "Renovaciones" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <CalendarClock className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Renovaciones
          <InfoTip>Suscripciones cuyo período vence en los próximos 90 días (o ya venció).</InfoTip>
        </h1>
        <Button variant="secondary" onClick={load} icon={<RefreshCw className="h-3.5 w-3.5" />}>Actualizar</Button>
      </div>

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
                <th className="px-3 py-2 font-medium">Plan</th>
                <th className="px-3 py-2 font-medium">Vencimiento</th>
                <th className="px-3 py-2 text-right font-medium">Tarifa renovación</th>
                <th className="px-3 py-2 text-center font-medium">Estado</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e, i) => (
                <tr key={e.company_id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 text-(--text-primary)">{e.business_name || "—"}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{e.nit}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{e.plan_name}</td>
                  <td className="px-3 py-2 font-medium text-(--text-primary)">{formatDate(e.current_period_end)}</td>
                  <td className="px-3 py-2 text-right text-(--text-secondary)">{formatCents(e.renewal_cents)}</td>
                  <td className="px-3 py-2 text-center">
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${urgencyClass(e.days_until_renewal)}`}>
                      {e.days_until_renewal <= 0 ? "Vencido" : `${e.days_until_renewal}d`}
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

// ── Por empresa ───────────────────────────────────────────────────────────────
function CompanyContent() {
  const toast = useToast();
  const [companyId, setCompanyId] = useState("");
  const [company, setCompany] = useState<CompanyInfo | null>(null);
  const [sub, setSub] = useState<Subscription | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [selectedPlanId, setSelectedPlanId] = useState("");
  const [hasOwnCert, setHasOwnCert] = useState(true);
  const [paymentsHistory, setPaymentsHistory] = useState<Payment[]>([]);
  const [searching, setSearching] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [renewing, setRenewing] = useState(false);
  const [payType, setPayType] = useState<Payment["type"]>("plan");
  const [payAmount, setPayAmount] = useState("");
  const [payNote, setPayNote] = useState("");
  const [recordingPayment, setRecordingPayment] = useState(false);

  useEffect(() => { adminListPlans().then(setPlans).catch(() => {}); }, []);

  async function handleSearch() {
    if (!companyId.trim()) return;
    setSearching(true);
    setCompany(null);
    setSub(null);
    setPaymentsHistory([]);
    try {
      const [compRes, subRes] = await Promise.allSettled([
        adminGetCompanyInfo(companyId.trim()),
        adminGetSubscription(companyId.trim()),
      ]);
      if (compRes.status === "fulfilled") {
        setCompany(compRes.value);
        adminListCompanyPayments(compRes.value.id).then(setPaymentsHistory).catch(() => {});
      } else {
        toast.error("No se encontró la empresa");
      }
      if (subRes.status === "fulfilled") {
        setSub(subRes.value);
        setSelectedPlanId(subRes.value.plan_id);
        setHasOwnCert(subRes.value.has_own_certificate);
      }
    } finally {
      setSearching(false);
    }
  }

  async function handleAssign() {
    if (!company || !selectedPlanId) return;
    setAssigning(true);
    try {
      const s = await adminAssignPlan(company.id, selectedPlanId, hasOwnCert);
      setSub(s);
      const planName = plans.find((p) => p.id === selectedPlanId)?.name ?? "";
      toast.success(`Plan asignado: ${planName}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo asignar el plan");
    } finally {
      setAssigning(false);
    }
  }

  async function handleRenew() {
    if (!company) return;
    setRenewing(true);
    try {
      const s = await adminRenewSubscription(company.id);
      setSub(s);
      toast.success(`Suscripción renovada. Nueva vigencia hasta ${formatDate(s.current_period_end)}.`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo renovar la suscripción");
    } finally {
      setRenewing(false);
    }
  }

  async function handleRecordPayment() {
    if (!company) return;
    const amountCents = pesosToCents(payAmount);
    if (amountCents <= 0) { toast.error("Ingresa un valor válido."); return; }
    setRecordingPayment(true);
    try {
      const p = await adminRecordPayment(company.id, {
        subscription_id: sub?.id, type: payType, amount_cents: amountCents, note: payNote,
      });
      setPaymentsHistory((prev) => [p, ...prev]);
      setPayAmount("");
      setPayNote("");
      toast.success("Pago registrado.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo registrar el pago");
    } finally {
      setRecordingPayment(false);
    }
  }

  const selectedPlan = plans.find((p) => p.id === selectedPlanId);

  return (
    <>
      <Breadcrumbs items={[{ label: "Comando", to: "/admin/billing" }, { label: "Por empresa" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Building2 className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Por empresa
          <InfoTip>Busca una empresa por su UUID para gestionar su plan y pagos.</InfoTip>
        </h1>
      </div>

      <div className="flex items-end gap-2 mb-4">
        <Input
          label="ID de la empresa (UUID)"
          value={companyId}
          onChange={(e) => setCompanyId(e.target.value)}
          placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
          className="flex-1 font-mono text-xs"
        />
        <Button loading={searching} icon={<ChevronRight className="h-3.5 w-3.5" />} onClick={handleSearch}>
          Buscar
        </Button>
      </div>

      {company && (
        <>
        <div className="rounded border border-(--border-color) overflow-hidden">
          {/* Cabecera de la empresa */}
          <div className="bg-(--bg-tertiary) px-3 py-2 border-b border-(--border-color)">
            <p className="text-xs font-semibold text-(--text-primary)">{company.business_name || company.trade_name || "(sin nombre)"}</p>
            <p className="text-xs text-(--text-secondary)">NIT {company.nit}</p>
            <div className="mt-1 flex flex-wrap gap-3 text-xs text-(--text-secondary)">
              <span>Plan: <strong className="text-(--text-primary)">{sub ? plans.find((p) => p.id === sub.plan_id)?.name ?? sub.plan_id : "sin suscripción"}</strong></span>
              {sub && (
                <>
                  <span>Vence: <strong className={new Date(sub.current_period_end) < new Date() ? "text-red-600" : "text-(--text-primary)"}>
                    {formatDate(sub.current_period_end)}
                  </strong></span>
                  <span>Certificado: <strong className="text-(--text-primary)">{sub.has_own_certificate ? "propio" : "vendido por nosotros"}</strong></span>
                </>
              )}
            </div>
          </div>

          {/* Plan */}
          <div className="border-b border-(--border-color) px-3 py-3">
            <p className="mb-2 text-xs font-medium text-(--text-secondary)">Contratar / cambiar plan</p>
            <div className="flex flex-wrap items-end gap-3">
              <label className="flex flex-col gap-1">
                <span className="text-xs font-medium text-(--text-secondary)">Plan</span>
                <select value={selectedPlanId} onChange={(e) => setSelectedPlanId(e.target.value)}
                  className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-xs text-(--text-primary)">
                  <option value="">— elegir plan —</option>
                  {plans.filter((p) => p.is_active).map((p) => (
                    <option key={p.id} value={p.id}>{p.name}</option>
                  ))}
                </select>
              </label>
              {selectedPlan?.requires_certificate && (
                <label className="flex items-center gap-1.5 pb-1.5 text-xs text-(--text-secondary)">
                  <input type="checkbox" checked={hasOwnCert} onChange={(e) => setHasOwnCert(e.target.checked)} />
                  El cliente ya tiene su propio certificado
                </label>
              )}
              <Button loading={assigning} disabled={!selectedPlanId} onClick={handleAssign}
                icon={<RefreshCw className="h-3.5 w-3.5" />}>
                Asignar
              </Button>
              {sub && (
                <Button variant="secondary" loading={renewing} onClick={handleRenew}>
                  Renovar suscripción
                </Button>
              )}
            </div>
          </div>

          {/* Registrar pago */}
          <div className="px-3 py-3 flex flex-wrap items-end gap-3">
            <p className="w-full text-xs font-medium text-(--text-secondary)">Registrar pago manual</p>
            <Select label="Tipo" value={payType} onChange={(e) => setPayType(e.target.value as Payment["type"])} className="w-40">
              <option value="plan">Plan</option>
              <option value="certificate">Certificado</option>
              <option value="overage">Excedente de documentos</option>
            </Select>
            <Input label="Valor (COP)" type="number" min="0" value={payAmount} onChange={(e) => setPayAmount(e.target.value)} className="w-36" />
            <Input label="Nota" value={payNote} onChange={(e) => setPayNote(e.target.value)} className="w-56" />
            <Button loading={recordingPayment} onClick={handleRecordPayment}>Registrar</Button>
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
                    <th className="px-3 py-2 text-left font-medium">Nota</th>
                    <th className="px-3 py-2 text-left font-medium">Fecha</th>
                  </tr>
                </thead>
                <tbody>
                  {paymentsHistory.map((p, i) => (
                    <tr key={p.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                      <td className="px-3 py-2 text-(--text-primary)">
                        {PAYMENT_TYPE_LABEL[p.type] ?? p.type}
                      </td>
                      <td className="px-3 py-2 text-right font-medium text-(--text-primary)">{formatCents(p.amount_cents)}</td>
                      <td className="px-3 py-2 text-(--text-secondary)">{p.note || "—"}</td>
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

interface PlanFormState {
  code: string; name: string; description: string; billingCycle: BillingCycle;
  price: string; includedDocuments: string; pricePerExtraDoc: string;
  requiresCertificate: boolean; certificatePrice: string; annualIncrementPct: string;
  modules: string[];
}

const EMPTY_PLAN_FORM: PlanFormState = {
  code: "", name: "", description: "", billingCycle: "monthly",
  price: "", includedDocuments: "", pricePerExtraDoc: "",
  requiresCertificate: false, certificatePrice: "", annualIncrementPct: "",
  modules: [],
};

function planToForm(p: Plan): PlanFormState {
  return {
    code: p.code, name: p.name, description: p.description, billingCycle: p.billing_cycle,
    price: centsToPesosInput(p.price_cents),
    includedDocuments: p.included_documents == null ? "" : String(p.included_documents),
    pricePerExtraDoc: centsToPesosInput(p.price_per_extra_document_cents),
    requiresCertificate: p.requires_certificate,
    certificatePrice: centsToPesosInput(p.certificate_price_cents),
    annualIncrementPct: p.annual_increment_pct === 0 ? "" : String(p.annual_increment_pct),
    modules: p.modules,
  };
}

function PlanFormModal({
  modules, initial, onClose, onSaved,
}: {
  modules: SaasModule[];
  initial: Plan | null; // null = crear
  onClose: () => void;
  onSaved: (p: Plan) => void;
}) {
  const toast = useToast();
  const [form, setForm] = useState<PlanFormState>(initial ? planToForm(initial) : EMPTY_PLAN_FORM);
  const [saving, setSaving] = useState(false);

  function toggleModule(code: string) {
    setForm((f) => ({
      ...f,
      modules: f.modules.includes(code) ? f.modules.filter((c) => c !== code) : [...f.modules, code],
    }));
  }

  async function handleSave() {
    if (!form.code.trim() || !form.name.trim()) {
      toast.error("Código y nombre son requeridos.");
      return;
    }
    setSaving(true);
    try {
      const payload = {
        code: form.code.trim(), name: form.name.trim(), description: form.description,
        billing_cycle: form.billingCycle,
        price_cents: pesosToCents(form.price),
        included_documents: form.includedDocuments.trim() === "" ? null : parseInt(form.includedDocuments, 10),
        price_per_extra_document_cents: pesosToCents(form.pricePerExtraDoc),
        requires_certificate: form.requiresCertificate,
        certificate_price_cents: form.requiresCertificate ? pesosToCents(form.certificatePrice) : 0,
        annual_increment_pct: form.annualIncrementPct.trim() === "" ? 0 : parseFloat(form.annualIncrementPct),
        is_internal: initial?.is_internal ?? false,
        modules: form.modules,
      };
      const saved = initial
        ? await adminUpdatePlan(initial.id, { ...payload, is_active: initial.is_active })
        : await adminCreatePlan(payload);
      toast.success(`Plan "${saved.name}" guardado.`);
      onSaved(saved);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar el plan");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <div className="w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-lg border border-(--border-color) bg-(--bg-primary) p-5 shadow-xl" onClick={(e) => e.stopPropagation()}>
        <h2 className="mb-4 text-sm font-semibold text-(--text-primary)">{initial ? `Editar plan "${initial.name}"` : "Nuevo plan"}</h2>

        <div className="space-y-3">
          <div className="flex gap-2">
            <Input label="Código" value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} disabled={!!initial} className="w-32" />
            <Input label="Nombre" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="flex-1" />
          </div>
          <Input label="Descripción" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />

          <div className="flex gap-2">
            <Select label="Ciclo de cobro" value={form.billingCycle} onChange={(e) => setForm({ ...form, billingCycle: e.target.value as BillingCycle })} className="w-40">
              <option value="monthly">Mensual</option>
              <option value="annual">Anual</option>
              <option value="none">Sin ciclo (gratis)</option>
            </Select>
            <Input label="Precio (COP)" type="number" min="0" value={form.price} onChange={(e) => setForm({ ...form, price: e.target.value })} className="flex-1" />
          </div>

          <div className="flex gap-2">
            <Input label="Documentos incluidos" type="number" min="0" placeholder="vacío = ilimitado"
              value={form.includedDocuments} onChange={(e) => setForm({ ...form, includedDocuments: e.target.value })} className="flex-1" />
            <Input label="Excedente $/documento" type="number" min="0" value={form.pricePerExtraDoc}
              onChange={(e) => setForm({ ...form, pricePerExtraDoc: e.target.value })} className="flex-1" />
          </div>

          <label className="flex items-center gap-1.5 text-xs text-(--text-secondary)">
            <input type="checkbox" checked={form.requiresCertificate} onChange={(e) => setForm({ ...form, requiresCertificate: e.target.checked })} />
            Requiere certificado DIAN
          </label>
          {form.requiresCertificate && (
            <Input label="Precio si nosotros vendemos el certificado (COP/año)" type="number" min="0"
              value={form.certificatePrice} onChange={(e) => setForm({ ...form, certificatePrice: e.target.value })} />
          )}

          <Input label="Incremento anual (%)" type="number" min="0" step="0.1" placeholder="0 = sin incremento"
            value={form.annualIncrementPct} onChange={(e) => setForm({ ...form, annualIncrementPct: e.target.value })} className="w-40" />

          <div>
            <span className="mb-1 block text-xs font-medium text-(--text-secondary)">Módulos que desbloquea</span>
            <div className="flex flex-col gap-1.5">
              {modules.map((m) => (
                <label key={m.code} className="flex items-center gap-1.5 text-xs text-(--text-primary)">
                  <input type="checkbox" checked={form.modules.includes(m.code)} onChange={() => toggleModule(m.code)} />
                  {m.name}
                </label>
              ))}
            </div>
          </div>
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose}>Cancelar</Button>
          <Button type="button" loading={saving} onClick={handleSave}>Guardar</Button>
        </div>
      </div>
    </div>
  );
}

function PlansContent() {
  const toast = useToast();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [modules, setModules] = useState<SaasModule[]>([]);
  const [loading, setLoading] = useState(true);
  const [applyingId, setApplyingId] = useState<string | null>(null);
  const [editing, setEditing] = useState<Plan | null | "new">(null);

  function load() {
    setLoading(true);
    Promise.all([adminListPlans(), adminListModules()])
      .then(([p, m]) => { setPlans(p); setModules(m); })
      .catch(() => {})
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

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
      <Breadcrumbs items={[{ label: "Comando", to: "/admin/billing" }, { label: "Planes" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Layers className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Planes
        </h1>
        <Button icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setEditing("new")}>Nuevo plan</Button>
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
                  {p.is_internal && <span className="ml-1.5 rounded-full bg-blue-100 px-1.5 text-[10px] font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">Interno</span>}
                  {p.description && <p className="text-xs text-(--text-secondary) mt-0.5">{p.description}</p>}
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${p.is_active ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400"}`}>
                    {p.is_active ? "Activo" : "Inactivo"}
                  </span>
                  <Button variant="secondary" icon={<Pencil className="h-3 w-3" />} onClick={() => setEditing(p)}>Editar</Button>
                  <Button variant="secondary" onClick={() => toggleActive(p)}>
                    {p.is_active ? "Desactivar" : "Activar"}
                  </Button>
                </div>
              </div>
              <div className="flex flex-wrap gap-4 text-xs text-(--text-secondary)">
                <span>Ciclo: <strong className="text-(--text-primary)">{BILLING_CYCLE_LABEL[p.billing_cycle]}</strong></span>
                <span>Precio: <strong className="text-(--text-primary)">{formatCents(p.price_cents)}</strong></span>
                <span>Docs incluidos: <strong className="text-(--text-primary)">{p.included_documents ?? "Ilimitado"}</strong></span>
                {p.requires_certificate && (
                  <span>Certificado (si lo vendemos): <strong className="text-(--text-primary)">{formatCents(p.certificate_price_cents)}/año</strong></span>
                )}
                <span>Incremento anual: <strong className="text-(--text-primary)">{p.annual_increment_pct > 0 ? `${p.annual_increment_pct}%` : "—"}</strong></span>
                <span>Módulos: <strong className="text-(--text-primary)">{p.modules.length > 0 ? p.modules.join(", ") : "ninguno"}</strong></span>
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

      {editing !== null && (
        <PlanFormModal
          modules={modules}
          initial={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={(saved) => {
            setPlans((ps) => (ps.some((p) => p.id === saved.id) ? ps.map((p) => (p.id === saved.id ? saved : p)) : [...ps, saved]));
            setEditing(null);
          }}
        />
      )}
    </>
  );
}

// ── Configuración (IVA) ────────────────────────────────────────────────────────
function SettingsContent() {
  const toast = useToast();
  const [settings, setSettings] = useState<SaasSettings | null>(null);
  const [ivaPct, setIvaPct] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    adminGetSettings().then((s) => { setSettings(s); setIvaPct(String(s.iva_rate_bp / 100)); }).catch(() => {});
  }, []);

  async function handleSave() {
    const pct = parseFloat(ivaPct);
    if (isNaN(pct) || pct < 0) { toast.error("Ingresa un porcentaje válido."); return; }
    setSaving(true);
    try {
      const s = await adminUpdateSettings(Math.round(pct * 100));
      setSettings(s);
      toast.success("Configuración actualizada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo actualizar la configuración");
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <Breadcrumbs items={[{ label: "Comando", to: "/admin/billing" }, { label: "Configuración" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <Settings2 className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Configuración de la plataforma
        <InfoTip>Tasa de IVA aplicada a todos los cobros (planes, certificados, excedente de documentos).</InfoTip>
      </h1>

      {settings && (
        <div className="flex items-end gap-2">
          <Input label="IVA (%)" type="number" min="0" step="0.01" value={ivaPct} onChange={(e) => setIvaPct(e.target.value)} className="w-32" />
          <Button loading={saving} onClick={handleSave}>Guardar</Button>
        </div>
      )}
    </>
  );
}

// ── Usuarios ─────────────────────────────────────────────────────────────────
function UsersContent() {
  const toast = useToast();
  const confirm = useConfirm();
  const { user: me } = useAuth();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  function load() {
    setLoading(true);
    adminListUsers().then(setUsers).catch(() => {}).finally(() => setLoading(false));
  }

  useEffect(load, []);

  async function toggleSuperAdmin(u: AdminUser) {
    const makeSuper = !u.is_superadmin;
    const label = makeSuper ? "convertir en superadmin" : "quitarle el acceso de superadmin";
    if (!(await confirm(`¿Seguro que quieres ${label} a ${u.name || u.email}?`, { tone: makeSuper ? undefined : "danger", confirmLabel: "Confirmar" }))) return;
    setTogglingId(u.id);
    try {
      await adminSetUserSuperAdmin(u.id, makeSuper);
      setUsers((prev) => prev.map((x) => (x.id === u.id ? { ...x, is_superadmin: makeSuper } : x)));
      toast.success(makeSuper ? "Ahora es superadmin." : "Ya no es superadmin.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo actualizar el usuario");
    } finally {
      setTogglingId(null);
    }
  }

  return (
    <>
      <Breadcrumbs items={[{ label: "Comando", to: "/admin/billing" }, { label: "Usuarios" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Users className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Usuarios
          <InfoTip>
            Usuarios de toda la plataforma. Para invitar uno nuevo a una empresa, usa Configuración → Empresa dentro
            de esa empresa. El acceso de superadmin <strong>nunca se otorga solo</strong> — un superadmin existente
            tiene que dárselo a otro aquí.
          </InfoTip>
        </h1>
      </div>

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
                <th className="px-3 py-2 text-left font-medium">Acciones</th>
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
                  <td className="px-3 py-2">
                    <Button
                      variant="secondary"
                      loading={togglingId === u.id}
                      disabled={u.id === me?.id}
                      onClick={() => toggleSuperAdmin(u)}
                    >
                      {u.is_superadmin ? "Quitar superadmin" : "Hacer superadmin"}
                    </Button>
                  </td>
                </tr>
              ))}
              {users.length === 0 && (
                <tr><td colSpan={5} className="px-3 py-4 text-center text-(--text-secondary)">No hay usuarios.</td></tr>
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
    if (!(await confirm(`¿Aprobar la solicitud de ${p.name}?`, { confirmLabel: "Aprobar" }))) return;
    try {
      const updated = await adminApproveProspect(p.id);
      setProspects((prev) => prev.map((x) => (x.id === updated.id ? updated : x)));
      toast.success("Solicitud aprobada.");
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
      <Breadcrumbs items={[{ label: "Comando", to: "/admin/billing" }, { label: "Solicitudes" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <ClipboardList className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Solicitudes de acceso
        </h1>
        <button onClick={load} className="flex items-center gap-1 rounded px-2 py-1 text-xs text-(--text-secondary) hover:bg-(--bg-hover)" title="Recargar" aria-label="Recargar">
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

export const AdminCompanyPage = withSuperAdmin(function AdminCompanyPage() {
  return <div className="p-4"><CompanyContent /></div>;
});

export const AdminPlansPage = withSuperAdmin(function AdminPlansPage() {
  return <div className="p-4"><PlansContent /></div>;
});

export const AdminSettingsPage = withSuperAdmin(function AdminSettingsPage() {
  return <div className="p-4"><SettingsContent /></div>;
});

export const AdminUsersPage = withSuperAdmin(function AdminUsersPage() {
  return <div className="p-4"><UsersContent /></div>;
});

export const AdminProspectsPage = withSuperAdmin(function AdminProspectsPage() {
  return <div className="p-4"><ProspectsContent /></div>;
});

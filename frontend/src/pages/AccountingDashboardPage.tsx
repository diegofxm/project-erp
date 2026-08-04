import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import {
  ArrowRight, BookText, Calculator, Landmark, Lock, ShoppingBag, ShoppingCart, Users, Wallet,
} from "lucide-react";
import { listJournals, getBSReport, getTrialBalance, listPeriods } from "../lib/accounting";
import { getReceivables } from "../lib/payments";
import { getPayables } from "../lib/purchasePayments";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { useAuth } from "../context/AuthContext";
import type { AccountingPeriod, JournalEntry, JournalStatus, PayableBalance, ReceivableBalance } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

const MONTHS = ["Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"];
const STATUS_LABEL: Record<JournalStatus, string> = { DRAFT: "Borrador", POSTED: "Contabilizado", VOID: "Anulado" };
const STATUS_TONE: Record<JournalStatus, StatusTone> = { DRAFT: "neutral", POSTED: "success", VOID: "danger" };

const JOURNAL_BATCH = 300;
const EXPENSE_CATEGORIES = ["Gasto", "Costo", "Costo de Producción"];

function money(cents: number): string {
  return formatCOP.format(cents / 100);
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

function monthKey(d: Date): string {
  return `${d.getFullYear()}-${d.getMonth()}`;
}

function entryTotal(e: JournalEntry): number {
  return e.lines.reduce((sum, l) => sum + l.debit, 0);
}

function trend(current: number, prev: number): { label: string; positive: boolean } | null {
  if (prev === 0 && current === 0) return null;
  if (prev === 0) return { label: "nuevo", positive: true };
  const pct = Math.round(((current - prev) / prev) * 100);
  return { label: `${pct > 0 ? "+" : ""}${pct}% vs. mes anterior`, positive: pct >= 0 };
}

interface DiarioConfig {
  key: string;
  label: string;
  icon: React.ReactNode;
  newLabel?: string;
  onNew?: () => void;
}

export function AccountingDashboardPage() {
  const { activeCompany } = useAuth();
  const navigate = useNavigate();

  const [journals, setJournals] = useState<JournalEntry[] | null>(null);
  const [receivables, setReceivables] = useState<ReceivableBalance[] | null>(null);
  const [payables, setPayables] = useState<PayableBalance[] | null>(null);
  const [bsRows, setBsRows] = useState<{ account_code: string; balance: number }[] | null>(null);
  const [plRows, setPlRows] = useState<{ category: string; debit: number; credit: number; balance: number }[] | null>(null);
  const [periods, setPeriods] = useState<AccountingPeriod[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const now = new Date();
    const firstOfMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-01`;
    Promise.all([
      listJournals(JOURNAL_BATCH, 0),
      getReceivables(),
      getPayables(),
      getBSReport(todayISO()),
      getTrialBalance(firstOfMonth, todayISO()),
      listPeriods(),
    ])
      .then(([j, r, p, bs, trial, per]) => {
        setJournals(j);
        setReceivables(r);
        setPayables(p);
        setBsRows(bs);
        setPlRows(trial);
        setPeriods(per);
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el panel de contabilidad"));
  }, []);

  const now = new Date();
  const currentPeriod = useMemo(
    () => periods?.find((p) => p.year === now.getFullYear() && p.month === now.getMonth() + 1) ?? null,
    [periods, now],
  );

  const receivablesTotals = useMemo(() => {
    if (!receivables) return null;
    const today = todayISO();
    const total = receivables.reduce((s, r) => s + r.balance, 0);
    const overdue = receivables.filter((r) => r.due_date && r.due_date < today);
    return { total, overdueCount: overdue.length, overdueTotal: overdue.reduce((s, r) => s + r.balance, 0) };
  }, [receivables]);

  const payablesTotals = useMemo(() => {
    if (!payables) return null;
    const today = todayISO();
    const in7 = new Date(now.getTime() + 7 * 86_400_000).toISOString().slice(0, 10);
    const total = payables.reduce((s, p) => s + p.balance, 0);
    const dueSoon = payables.filter((p) => p.due_date && p.due_date >= today && p.due_date <= in7);
    return { total, dueSoonCount: dueSoon.length };
  }, [payables, now]);

  const cashBalance = useMemo(() => {
    if (!bsRows) return null;
    return bsRows.filter((r) => r.account_code.startsWith("11")).reduce((s, r) => s + r.balance, 0);
  }, [bsRows]);

  const monthResult = useMemo(() => {
    if (!plRows) return null;
    const income = plRows.filter((r) => r.category === "Ingreso").reduce((s, r) => s + r.credit - r.debit, 0);
    const expenses = plRows.filter((r) => EXPENSE_CATEGORIES.includes(r.category)).reduce((s, r) => s + r.debit - r.credit, 0);
    return { income, expenses, net: income - expenses };
  }, [plRows]);

  const posted = useMemo(() => (journals ?? []).filter((e) => e.status === "POSTED"), [journals]);
  const curKey = monthKey(now);
  const prevMonthDate = new Date(now.getFullYear(), now.getMonth() - 1, 1);
  const prevKey = monthKey(prevMonthDate);

  function bySource(source: string) {
    const cur = posted.filter((e) => e.source === source && monthKey(new Date(e.date)) === curKey);
    const prev = posted.filter((e) => e.source === source && monthKey(new Date(e.date)) === prevKey);
    return {
      total: cur.reduce((s, e) => s + entryTotal(e), 0),
      count: cur.length,
      trend: trend(cur.reduce((s, e) => s + entryTotal(e), 0), prev.reduce((s, e) => s + entryTotal(e), 0)),
    };
  }

  const diarios: DiarioConfig[] = [
    { key: "sales", label: "Ventas", icon: <ShoppingCart className="h-4 w-4" />, newLabel: "Nueva venta", onNew: () => navigate("/sales/new") },
    { key: "purchase", label: "Compras", icon: <ShoppingBag className="h-4 w-4" />, newLabel: "Nueva orden", onNew: () => navigate("/purchases/new") },
    { key: "payroll", label: "Nómina", icon: <Users className="h-4 w-4" /> },
    { key: "manual", label: "Ajustes manuales", icon: <BookText className="h-4 w-4" />, newLabel: "Nuevo asiento", onNew: () => navigate("/accounting/journals/new") },
  ];

  const recent = (journals ?? []).slice(0, 5);
  const loading = journals === null || receivables === null || payables === null || bsRows === null || plRows === null;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad" }]} />
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
            <Calculator className="h-4 w-4 shrink-0 text-(--accent-primary)" />
            Panel de contabilidad
          </h1>
          <p className="mt-0.5 text-xs text-(--text-secondary)">
            {activeCompany?.trade_name || activeCompany?.business_name} · Periodo {MONTHS[now.getMonth()]} {now.getFullYear()}
            {currentPeriod && (
              <span className={`ml-1.5 ${currentPeriod.status === "OPEN" ? "text-(--color-success-text)" : "text-(--text-muted)"}`}>
                · {currentPeriod.status === "OPEN" ? "Abierto" : "Cerrado"}
              </span>
            )}
          </p>
        </div>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {loading ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : (
        <>
          <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Por cobrar <Wallet className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{money(receivablesTotals!.total)}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">
                {receivablesTotals!.overdueCount > 0 ? (
                  <span className="text-(--color-danger-text)">{receivablesTotals!.overdueCount} vencidas · {money(receivablesTotals!.overdueTotal)}</span>
                ) : "Sin vencidas"}
              </div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Por pagar <Wallet className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{money(payablesTotals!.total)}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">
                {payablesTotals!.dueSoonCount > 0 ? `${payablesTotals!.dueSoonCount} esta semana` : "Nada vence esta semana"}
              </div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Resultado del mes <Calculator className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className={`mt-2 text-xl font-bold tabular-nums ${monthResult!.net >= 0 ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>
                {monthResult!.net >= 0 ? "+" : ""}{money(monthResult!.net)}
              </div>
              <div className="mt-1 text-xs text-(--text-secondary)">Ingresos {money(monthResult!.income)} — Gastos {money(monthResult!.expenses)}</div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Efectivo y bancos <Landmark className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{money(cashBalance!)}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">Saldo consolidado (caja y bancos)</div>
            </Card>
          </div>

          <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Diarios</p>
          <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {diarios.map((d) => {
              const s = bySource(d.key);
              return (
                <Card key={d.key} className="flex flex-col p-4">
                  <div className="flex items-center gap-2 text-xs font-semibold text-(--text-primary)">
                    <span className="text-(--accent-primary)">{d.icon}</span>
                    {d.label}
                  </div>
                  <div className="mt-2 text-lg font-bold tabular-nums text-(--text-primary)">{money(s.total)}</div>
                  {s.trend ? (
                    <span className={`mt-0.5 text-xs ${s.trend.positive ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>{s.trend.label}</span>
                  ) : (
                    <span className="mt-0.5 text-xs text-(--text-muted)">{s.count} asiento{s.count === 1 ? "" : "s"} este mes</span>
                  )}
                  <div className="mt-3 flex flex-1 items-end justify-between gap-2 border-t border-(--border-light) pt-2 text-xs">
                    {d.onNew ? (
                      <button type="button" onClick={d.onNew} className="font-medium text-(--accent-primary) hover:underline">{d.newLabel}</button>
                    ) : <span />}
                    <button type="button" onClick={() => navigate("/accounting/journals")} className="flex items-center gap-1 text-(--text-secondary) hover:text-(--accent-primary)">
                      Ver asientos <ArrowRight className="h-3 w-3" />
                    </button>
                  </div>
                </Card>
              );
            })}
          </div>

          <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
            <Card className="p-4 lg:col-span-2">
              <div className="mb-2 flex items-center justify-between">
                <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Actividad reciente</h2>
                <button type="button" onClick={() => navigate("/accounting/journals")} className="flex items-center gap-1 text-xs text-(--accent-primary) hover:underline">
                  Ver todos <ArrowRight className="h-3 w-3" />
                </button>
              </div>
              {recent.length === 0 ? (
                <p className="py-4 text-center text-xs text-(--text-muted)">Aún no hay asientos registrados</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-xs">
                    <thead className="text-(--text-secondary)">
                      <tr>
                        <th className="py-1.5 pr-3 font-medium">Fecha</th>
                        <th className="py-1.5 pr-3 font-medium">Descripción</th>
                        <th className="py-1.5 pr-3 font-medium">Origen</th>
                        <th className="py-1.5 pr-3 text-right font-medium">Monto</th>
                        <th className="py-1.5 font-medium">Estado</th>
                      </tr>
                    </thead>
                    <tbody>
                      {recent.map((e, i) => (
                        <tr
                          key={e.id}
                          className={`cursor-pointer border-t border-(--border-light) hover:bg-(--bg-hover) ${i % 2 === 1 ? "bg-(--bg-secondary)" : ""}`}
                          onClick={() => navigate(`/accounting/journals/${e.id}`)}
                        >
                          <td className="py-1.5 pr-3 text-(--text-secondary)">{new Date(e.date).toLocaleDateString("es-CO")}</td>
                          <td className="py-1.5 pr-3 text-(--text-primary)">{e.description}</td>
                          <td className="py-1.5 pr-3 text-(--text-secondary)">{e.source}</td>
                          <td className="py-1.5 pr-3 text-right font-mono text-(--text-primary)">{money(entryTotal(e))}</td>
                          <td className="py-1.5"><StatusPill tone={STATUS_TONE[e.status]} label={STATUS_LABEL[e.status]} /></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </Card>

            <Card className="p-4">
              <h2 className="mb-3 text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Accesos rápidos</h2>
              <div className="flex flex-col gap-0.5">
                <QuickLink icon={<Landmark className="h-3.5 w-3.5" />} label="Plan de cuentas (PUC)" onClick={() => navigate("/accounting/accounts")} />
                <QuickLink
                  icon={<Lock className="h-3.5 w-3.5" />}
                  label={currentPeriod?.status === "OPEN" ? `Cerrar periodo ${MONTHS[now.getMonth()]}` : "Ver periodos"}
                  onClick={() => navigate("/accounting/periods")}
                />
                <QuickLink icon={<Calculator className="h-3.5 w-3.5" />} label="Estado de resultados" onClick={() => navigate("/accounting/reports?type=pl")} />
                <QuickLink icon={<Landmark className="h-3.5 w-3.5" />} label="Balance general" onClick={() => navigate("/accounting/reports?type=bs")} />
              </div>
            </Card>
          </div>
        </>
      )}
    </div>
  );
}

function QuickLink({ icon, label, onClick }: { icon: React.ReactNode; label: string; onClick: () => void }) {
  return (
    <button type="button" onClick={onClick} className="flex items-center gap-2 rounded px-1.5 py-2 text-left text-xs text-(--text-secondary) transition-colors hover:bg-(--bg-hover) hover:text-(--text-primary)">
      {icon}
      {label}
    </button>
  );
}

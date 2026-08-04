import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { ArrowRight, FileEdit, ShoppingCart, Wallet, FileWarning } from "lucide-react";
import { listSales } from "../lib/sales";
import { listQuotes } from "../lib/quotes";
import { getReceivables } from "../lib/payments";
import { listCustomers } from "../lib/customers";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { useAuth } from "../context/AuthContext";
import type { Customer, Quote, ReceivableBalance, Sale, SaleStatus } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

const STATUS_LABEL: Record<SaleStatus, string> = { draft: "Borrador", confirmed: "Confirmada", cancelled: "Cancelada" };
const STATUS_TONE: Record<SaleStatus, StatusTone> = { draft: "neutral", confirmed: "success", cancelled: "danger" };

function money(v: number): string {
  return formatCOP.format(v);
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

function monthKey(d: Date): string {
  return `${d.getFullYear()}-${d.getMonth()}`;
}

function saleTotal(s: Sale): number {
  return s.lines.reduce((sum, l) => sum + l.total, 0);
}

function quoteTotal(q: Quote): number {
  return q.lines.reduce((sum, l) => sum + l.total, 0);
}

function trend(current: number, prev: number): { label: string; positive: boolean } | null {
  if (prev === 0 && current === 0) return null;
  if (prev === 0) return { label: "nuevo", positive: true };
  const pct = Math.round(((current - prev) / prev) * 100);
  return { label: `${pct > 0 ? "+" : ""}${pct}% vs. mes anterior`, positive: pct >= 0 };
}

export function SalesDashboardPage() {
  const { activeCompany } = useAuth();
  const navigate = useNavigate();

  const [sales, setSales] = useState<Sale[] | null>(null);
  const [quotes, setQuotes] = useState<Quote[] | null>(null);
  const [receivables, setReceivables] = useState<ReceivableBalance[] | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([listSales(), listQuotes(), getReceivables(), listCustomers()])
      .then(([s, q, r, c]) => { setSales(s); setQuotes(q); setReceivables(r); setCustomers(c); })
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el panel de ventas"));
  }, []);

  const customerName = useMemo(() => {
    const map = new Map(customers.map((c) => [c.id, c.name]));
    return (id: string) => map.get(id) ?? "—";
  }, [customers]);

  const now = new Date();
  const curKey = monthKey(now);
  const prevKey = monthKey(new Date(now.getFullYear(), now.getMonth() - 1, 1));

  const confirmedThisMonth = useMemo(
    () => (sales ?? []).filter((s) => s.status === "confirmed" && monthKey(new Date(s.issue_date)) === curKey),
    [sales, curKey],
  );
  const confirmedPrevMonth = useMemo(
    () => (sales ?? []).filter((s) => s.status === "confirmed" && monthKey(new Date(s.issue_date)) === prevKey),
    [sales, prevKey],
  );
  const salesTrend = trend(
    confirmedThisMonth.reduce((s, x) => s + saleTotal(x), 0),
    confirmedPrevMonth.reduce((s, x) => s + saleTotal(x), 0),
  );

  const draftSales = useMemo(() => (sales ?? []).filter((s) => s.status === "draft"), [sales]);
  const unbilled = useMemo(() => (sales ?? []).filter((s) => s.status === "confirmed" && !s.invoice_document_id), [sales]);

  const openQuotes = useMemo(() => (quotes ?? []).filter((q) => q.status === "sent"), [quotes]);
  const acceptedThisMonth = useMemo(
    () => (quotes ?? []).filter((q) => q.status === "accepted" && monthKey(new Date(q.issue_date)) === curKey),
    [quotes, curKey],
  );

  const receivablesTotals = useMemo(() => {
    if (!receivables) return null;
    const today = todayISO();
    const overdue = receivables.filter((r) => r.due_date && r.due_date < today);
    return { total: receivables.reduce((s, r) => s + r.balance, 0), overdueCount: overdue.length, overdueTotal: overdue.reduce((s, r) => s + r.balance, 0) };
  }, [receivables]);

  const recent = useMemo(
    () => [...(sales ?? [])].sort((a, b) => b.created_at.localeCompare(a.created_at)).slice(0, 5),
    [sales],
  );

  const loading = sales === null || quotes === null || receivables === null;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Ventas" }]} />
      <div className="mb-3">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <ShoppingCart className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Panel de ventas
        </h1>
        <p className="mt-0.5 text-xs text-(--text-secondary)">{activeCompany?.trade_name || activeCompany?.business_name}</p>
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
                Ventas del mes <ShoppingCart className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{money(confirmedThisMonth.reduce((s, x) => s + saleTotal(x), 0))}</div>
              {salesTrend ? (
                <div className={`mt-1 text-xs ${salesTrend.positive ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>{salesTrend.label}</div>
              ) : (
                <div className="mt-1 text-xs text-(--text-secondary)">{confirmedThisMonth.length} venta{confirmedThisMonth.length === 1 ? "" : "s"}</div>
              )}
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Cotizaciones abiertas <FileEdit className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{openQuotes.length}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">{money(openQuotes.reduce((s, q) => s + quoteTotal(q), 0))} · {acceptedThisMonth.length} aceptadas este mes</div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Cartera <Wallet className="h-3.5 w-3.5 text-(--accent-primary)" />
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
                Sin facturar <FileWarning className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{unbilled.length}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">{money(unbilled.reduce((s, x) => s + saleTotal(x), 0))} sin factura electrónica</div>
            </Card>
          </div>

          <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
            <Card className="flex flex-col p-4">
              <div className="text-xs font-semibold text-(--text-primary)">Cotizaciones</div>
              <div className="mt-2 text-lg font-bold tabular-nums text-(--text-primary)">{openQuotes.length}</div>
              <span className="mt-0.5 text-xs text-(--text-muted)">pendientes de respuesta</span>
              <div className="mt-3 flex flex-1 items-end justify-between gap-2 border-t border-(--border-light) pt-2 text-xs">
                <button type="button" onClick={() => navigate("/sales/quotes/new")} className="font-medium text-(--accent-primary) hover:underline">Nueva cotización</button>
                <button type="button" onClick={() => navigate("/sales/quotes")} className="flex items-center gap-1 text-(--text-secondary) hover:text-(--accent-primary)">Ver <ArrowRight className="h-3 w-3" /></button>
              </div>
            </Card>
            <Card className="flex flex-col p-4">
              <div className="text-xs font-semibold text-(--text-primary)">Ventas</div>
              <div className="mt-2 text-lg font-bold tabular-nums text-(--text-primary)">{draftSales.length}</div>
              <span className="mt-0.5 text-xs text-(--text-muted)">borradores sin confirmar</span>
              <div className="mt-3 flex flex-1 items-end justify-between gap-2 border-t border-(--border-light) pt-2 text-xs">
                <button type="button" onClick={() => navigate("/sales/new")} className="font-medium text-(--accent-primary) hover:underline">Nueva venta</button>
                <button type="button" onClick={() => navigate("/sales/orders")} className="flex items-center gap-1 text-(--text-secondary) hover:text-(--accent-primary)">Ver <ArrowRight className="h-3 w-3" /></button>
              </div>
            </Card>
            <Card className="flex flex-col p-4">
              <div className="text-xs font-semibold text-(--text-primary)">Cartera</div>
              <div className="mt-2 text-lg font-bold tabular-nums text-(--text-primary)">{receivablesTotals!.overdueCount}</div>
              <span className="mt-0.5 text-xs text-(--text-muted)">facturas vencidas</span>
              <div className="mt-3 flex flex-1 items-end justify-end border-t border-(--border-light) pt-2 text-xs">
                <button type="button" onClick={() => navigate("/sales/receivables")} className="flex items-center gap-1 text-(--text-secondary) hover:text-(--accent-primary)">Ver cartera <ArrowRight className="h-3 w-3" /></button>
              </div>
            </Card>
          </div>

          <Card className="p-4">
            <div className="mb-2 flex items-center justify-between">
              <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Actividad reciente</h2>
              <button type="button" onClick={() => navigate("/sales/orders")} className="flex items-center gap-1 text-xs text-(--accent-primary) hover:underline">
                Ver todas <ArrowRight className="h-3 w-3" />
              </button>
            </div>
            {recent.length === 0 ? (
              <p className="py-4 text-center text-xs text-(--text-muted)">Aún no hay ventas registradas</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="text-(--text-secondary)">
                    <tr>
                      <th className="py-1.5 pr-3 font-medium">Fecha</th>
                      <th className="py-1.5 pr-3 font-medium">Número</th>
                      <th className="py-1.5 pr-3 font-medium">Cliente</th>
                      <th className="py-1.5 pr-3 text-right font-medium">Total</th>
                      <th className="py-1.5 font-medium">Estado</th>
                    </tr>
                  </thead>
                  <tbody>
                    {recent.map((s, i) => (
                      <tr key={s.id} className={`cursor-pointer border-t border-(--border-light) hover:bg-(--bg-hover) ${i % 2 === 1 ? "bg-(--bg-secondary)" : ""}`} onClick={() => navigate(`/sales/${s.id}`)}>
                        <td className="py-1.5 pr-3 text-(--text-secondary)">{new Date(s.issue_date).toLocaleDateString("es-CO")}</td>
                        <td className="py-1.5 pr-3 font-mono text-(--text-primary)">{s.number || "Borrador"}</td>
                        <td className="py-1.5 pr-3 text-(--text-primary)">{customerName(s.customer_id)}</td>
                        <td className="py-1.5 pr-3 text-right font-mono text-(--text-primary)">{money(saleTotal(s))}</td>
                        <td className="py-1.5"><StatusPill tone={STATUS_TONE[s.status]} label={STATUS_LABEL[s.status]} /></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </>
      )}
    </div>
  );
}

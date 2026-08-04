import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { ArrowRight, PackageCheck, ShoppingBag, Wallet, FileEdit } from "lucide-react";
import { listPurchases } from "../lib/purchases";
import { getPayables } from "../lib/purchasePayments";
import { listSuppliers } from "../lib/suppliers";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { useAuth } from "../context/AuthContext";
import type { PayableBalance, Purchase, PurchaseStatus, Supplier } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

const STATUS_LABEL: Record<PurchaseStatus, string> = { draft: "Borrador", confirmed: "Confirmada", received: "Recibida", cancelled: "Cancelada" };
const STATUS_TONE: Record<PurchaseStatus, StatusTone> = { draft: "neutral", confirmed: "info", received: "success", cancelled: "danger" };

function money(v: number): string {
  return formatCOP.format(v);
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

function monthKey(d: Date): string {
  return `${d.getFullYear()}-${d.getMonth()}`;
}

function purchaseTotal(p: Purchase): number {
  return p.lines.reduce((sum, l) => sum + l.total, 0);
}

function trend(current: number, prev: number): { label: string; positive: boolean } | null {
  if (prev === 0 && current === 0) return null;
  if (prev === 0) return { label: "nuevo", positive: true };
  const pct = Math.round(((current - prev) / prev) * 100);
  return { label: `${pct > 0 ? "+" : ""}${pct}% vs. mes anterior`, positive: pct >= 0 };
}

export function PurchaseDashboardPage() {
  const { activeCompany } = useAuth();
  const navigate = useNavigate();

  const [purchases, setPurchases] = useState<Purchase[] | null>(null);
  const [payables, setPayables] = useState<PayableBalance[] | null>(null);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([listPurchases(), getPayables(), listSuppliers()])
      .then(([p, pay, s]) => { setPurchases(p); setPayables(pay); setSuppliers(s); })
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el panel de compras"));
  }, []);

  const supplierName = useMemo(() => {
    const map = new Map(suppliers.map((s) => [s.id, s.name]));
    return (id: string) => map.get(id) ?? "—";
  }, [suppliers]);

  const now = new Date();
  const curKey = monthKey(now);
  const prevKey = monthKey(new Date(now.getFullYear(), now.getMonth() - 1, 1));

  const receivedThisMonth = useMemo(
    () => (purchases ?? []).filter((p) => p.status === "received" && monthKey(new Date(p.issue_date)) === curKey),
    [purchases, curKey],
  );
  const receivedPrevMonth = useMemo(
    () => (purchases ?? []).filter((p) => p.status === "received" && monthKey(new Date(p.issue_date)) === prevKey),
    [purchases, prevKey],
  );
  const purchasesTrend = trend(
    receivedThisMonth.reduce((s, x) => s + purchaseTotal(x), 0),
    receivedPrevMonth.reduce((s, x) => s + purchaseTotal(x), 0),
  );

  const draftPurchases = useMemo(() => (purchases ?? []).filter((p) => p.status === "draft"), [purchases]);
  const pendingReceipt = useMemo(() => (purchases ?? []).filter((p) => p.status === "confirmed"), [purchases]);

  const payablesTotals = useMemo(() => {
    if (!payables) return null;
    const today = todayISO();
    const in7 = new Date(now.getTime() + 7 * 86_400_000).toISOString().slice(0, 10);
    const dueSoon = payables.filter((p) => p.due_date && p.due_date >= today && p.due_date <= in7);
    return { total: payables.reduce((s, p) => s + p.balance, 0), dueSoonCount: dueSoon.length };
  }, [payables, now]);

  const recent = useMemo(
    () => [...(purchases ?? [])].sort((a, b) => b.created_at.localeCompare(a.created_at)).slice(0, 5),
    [purchases],
  );

  const loading = purchases === null || payables === null;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Compras" }]} />
      <div className="mb-3">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <ShoppingBag className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Panel de compras
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
                Compras del mes <ShoppingBag className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{money(receivedThisMonth.reduce((s, x) => s + purchaseTotal(x), 0))}</div>
              {purchasesTrend ? (
                <div className={`mt-1 text-xs ${purchasesTrend.positive ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>{purchasesTrend.label}</div>
              ) : (
                <div className="mt-1 text-xs text-(--text-secondary)">{receivedThisMonth.length} recibida{receivedThisMonth.length === 1 ? "" : "s"}</div>
              )}
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Por recibir <PackageCheck className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{pendingReceipt.length}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">{money(pendingReceipt.reduce((s, x) => s + purchaseTotal(x), 0))} confirmadas sin recibir</div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Cuentas por pagar <Wallet className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{money(payablesTotals!.total)}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">
                {payablesTotals!.dueSoonCount > 0 ? `${payablesTotals!.dueSoonCount} esta semana` : "Nada vence esta semana"}
              </div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Borradores <FileEdit className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{draftPurchases.length}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">órdenes sin confirmar</div>
            </Card>
          </div>

          <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
            <Card className="flex flex-col p-4">
              <div className="text-xs font-semibold text-(--text-primary)">Órdenes de compra</div>
              <div className="mt-2 text-lg font-bold tabular-nums text-(--text-primary)">{draftPurchases.length}</div>
              <span className="mt-0.5 text-xs text-(--text-muted)">en borrador</span>
              <div className="mt-3 flex flex-1 items-end justify-between gap-2 border-t border-(--border-light) pt-2 text-xs">
                <button type="button" onClick={() => navigate("/purchases/new")} className="font-medium text-(--accent-primary) hover:underline">Nueva orden</button>
                <button type="button" onClick={() => navigate("/purchases/orders")} className="flex items-center gap-1 text-(--text-secondary) hover:text-(--accent-primary)">Ver <ArrowRight className="h-3 w-3" /></button>
              </div>
            </Card>
            <Card className="flex flex-col p-4">
              <div className="text-xs font-semibold text-(--text-primary)">Recepciones pendientes</div>
              <div className="mt-2 text-lg font-bold tabular-nums text-(--text-primary)">{pendingReceipt.length}</div>
              <span className="mt-0.5 text-xs text-(--text-muted)">confirmadas, esperando mercancía</span>
              <div className="mt-3 flex flex-1 items-end justify-end border-t border-(--border-light) pt-2 text-xs">
                <button type="button" onClick={() => navigate("/purchases/orders")} className="flex items-center gap-1 text-(--text-secondary) hover:text-(--accent-primary)">Ver <ArrowRight className="h-3 w-3" /></button>
              </div>
            </Card>
            <Card className="flex flex-col p-4">
              <div className="text-xs font-semibold text-(--text-primary)">Cuentas por pagar</div>
              <div className="mt-2 text-lg font-bold tabular-nums text-(--text-primary)">{payablesTotals!.dueSoonCount}</div>
              <span className="mt-0.5 text-xs text-(--text-muted)">vencen esta semana</span>
              <div className="mt-3 flex flex-1 items-end justify-end border-t border-(--border-light) pt-2 text-xs">
                <button type="button" onClick={() => navigate("/purchases/payables")} className="flex items-center gap-1 text-(--text-secondary) hover:text-(--accent-primary)">Ver <ArrowRight className="h-3 w-3" /></button>
              </div>
            </Card>
          </div>

          <Card className="p-4">
            <div className="mb-2 flex items-center justify-between">
              <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Actividad reciente</h2>
              <button type="button" onClick={() => navigate("/purchases/orders")} className="flex items-center gap-1 text-xs text-(--accent-primary) hover:underline">
                Ver todas <ArrowRight className="h-3 w-3" />
              </button>
            </div>
            {recent.length === 0 ? (
              <p className="py-4 text-center text-xs text-(--text-muted)">Aún no hay órdenes de compra registradas</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="text-(--text-secondary)">
                    <tr>
                      <th className="py-1.5 pr-3 font-medium">Fecha</th>
                      <th className="py-1.5 pr-3 font-medium">Número</th>
                      <th className="py-1.5 pr-3 font-medium">Proveedor</th>
                      <th className="py-1.5 pr-3 text-right font-medium">Total</th>
                      <th className="py-1.5 font-medium">Estado</th>
                    </tr>
                  </thead>
                  <tbody>
                    {recent.map((p, i) => (
                      <tr key={p.id} className={`cursor-pointer border-t border-(--border-light) hover:bg-(--bg-hover) ${i % 2 === 1 ? "bg-(--bg-secondary)" : ""}`} onClick={() => navigate(`/purchases/${p.id}`)}>
                        <td className="py-1.5 pr-3 text-(--text-secondary)">{new Date(p.issue_date).toLocaleDateString("es-CO")}</td>
                        <td className="py-1.5 pr-3 font-mono text-(--text-primary)">{p.number || "Borrador"}</td>
                        <td className="py-1.5 pr-3 text-(--text-primary)">{supplierName(p.supplier_id)}</td>
                        <td className="py-1.5 pr-3 text-right font-mono text-(--text-primary)">{money(purchaseTotal(p))}</td>
                        <td className="py-1.5"><StatusPill tone={STATUS_TONE[p.status]} label={STATUS_LABEL[p.status]} /></td>
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

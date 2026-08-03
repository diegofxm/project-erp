import { useEffect, useMemo, useState } from "react";
import { Wallet, DollarSign } from "lucide-react";
import { getReceivables, recordPayment } from "../lib/payments";
import { listCustomers } from "../lib/customers";
import { ApiError } from "../lib/apiClient";
import { useToast } from "../context/ToastContext";
import { formatCOP } from "../lib/currency";
import type { Customer, PaymentMethod, ReceivableBalance } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill } from "../components/ui/StatusPill";
import { RecordPaymentModal } from "../components/sales/RecordPaymentModal";

function isOverdue(dueDate: string | undefined): boolean {
  if (!dueDate) return false;
  return new Date(dueDate).getTime() < Date.now();
}

export function ReceivablesPage() {
  const [receivables, setReceivables] = useState<ReceivableBalance[] | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [paying, setPaying] = useState<ReceivableBalance | null>(null);
  const toast = useToast();

  function refresh() {
    getReceivables()
      .then(setReceivables)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la cartera"));
  }

  useEffect(() => {
    refresh();
    listCustomers().then(setCustomers).catch(() => setCustomers([]));
  }, []);

  const customerName = useMemo(() => {
    const map = new Map(customers.map((c) => [c.id, c.name]));
    return (id: string) => map.get(id) ?? "—";
  }, [customers]);

  const totalPending = receivables?.reduce((s, r) => s + r.balance, 0) ?? 0;
  const totalOverdue = receivables?.filter((r) => isOverdue(r.due_date)).reduce((s, r) => s + r.balance, 0) ?? 0;

  async function handleRecordPayment(payload: { amount: number; payment_date: string; payment_method: PaymentMethod; reference: string; notes: string }) {
    if (!paying) return;
    await recordPayment({ sale_id: paying.sale_id, ...payload });
    toast.success(`Pago registrado para ${paying.sale_number || "la venta"}.`);
    setPaying(null);
    refresh();
  }

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Ventas", to: "/sales" }, { label: "Cartera" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Wallet className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Cartera
        </h1>
      </div>

      {receivables && receivables.length > 0 && (
        <div className="mb-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Card className="p-3">
            <p className="text-xs font-medium text-(--text-secondary)">Total por cobrar</p>
            <p className="font-mono text-lg font-semibold text-(--text-primary)">{formatCOP.format(totalPending)}</p>
          </Card>
          <Card className="p-3">
            <p className="text-xs font-medium text-(--text-secondary)">Vencido</p>
            <p className="font-mono text-lg font-semibold text-(--color-danger-text)">{formatCOP.format(totalOverdue)}</p>
          </Card>
        </div>
      )}

      {error && <Banner tone="danger">{error}</Banner>}

      {receivables === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : receivables.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">No tienes cartera pendiente — todas las ventas confirmadas están al día.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Venta</th>
                <th className="px-3 py-2 font-medium">Cliente</th>
                <th className="px-3 py-2 font-medium">Fecha</th>
                <th className="px-3 py-2 font-medium">Vence</th>
                <th className="px-3 py-2 font-medium">Total</th>
                <th className="px-3 py-2 font-medium">Pagado</th>
                <th className="px-3 py-2 font-medium">Saldo</th>
                <th className="px-3 py-2 font-medium" />
              </tr>
            </thead>
            <tbody>
              {receivables.map((r, i) => {
                const overdue = isOverdue(r.due_date);
                return (
                  <tr key={r.sale_id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{r.sale_number || "—"}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{customerName(r.customer_id)}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{new Date(r.issue_date).toLocaleDateString("es-CO")}</td>
                    <td className="px-3 py-2">
                      {r.due_date ? (
                        overdue ? (
                          <StatusPill tone="danger" label={new Date(r.due_date).toLocaleDateString("es-CO")} />
                        ) : (
                          <span className="text-(--text-secondary)">{new Date(r.due_date).toLocaleDateString("es-CO")}</span>
                        )
                      ) : "—"}
                    </td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(r.total)}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(r.paid)}</td>
                    <td className="px-3 py-2 font-mono font-semibold text-(--text-primary)">{formatCOP.format(r.balance)}</td>
                    <td className="px-3 py-2">
                      <button
                        type="button"
                        onClick={() => setPaying(r)}
                        className="inline-flex items-center gap-1 rounded px-2 py-1 text-(--accent-primary) hover:bg-(--bg-hover)"
                      >
                        <DollarSign className="h-3.5 w-3.5" />
                        Registrar pago
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {paying && (
        <RecordPaymentModal receivable={paying} onSubmit={handleRecordPayment} onClose={() => setPaying(null)} />
      )}
    </div>
  );
}

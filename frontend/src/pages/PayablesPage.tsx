import { useEffect, useMemo, useState } from "react";
import { Wallet, DollarSign } from "lucide-react";
import { getPayables, recordPurchasePayment } from "../lib/purchasePayments";
import { listSuppliers } from "../lib/suppliers";
import { ApiError } from "../lib/apiClient";
import { useToast } from "../context/ToastContext";
import { formatCOP } from "../lib/currency";
import { formatDateOnly } from "../lib/dateFormat";
import type { PayableBalance, PaymentMethod, Supplier } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill } from "../components/ui/StatusPill";
import { RecordPurchasePaymentModal } from "../components/purchase/RecordPurchasePaymentModal";

function isOverdue(dueDate: string | undefined): boolean {
  if (!dueDate) return false;
  return new Date(dueDate).getTime() < Date.now();
}

export function PayablesPage() {
  const [payables, setPayables] = useState<PayableBalance[] | null>(null);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [paying, setPaying] = useState<PayableBalance | null>(null);
  const toast = useToast();

  function refresh() {
    getPayables()
      .then(setPayables)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar las cuentas por pagar"));
  }

  useEffect(() => {
    refresh();
    listSuppliers().then(setSuppliers).catch(() => setSuppliers([]));
  }, []);

  const supplierName = useMemo(() => {
    const map = new Map(suppliers.map((s) => [s.id, s.name]));
    return (id: string) => map.get(id) ?? "—";
  }, [suppliers]);

  const totalPending = payables?.reduce((s, p) => s + p.balance, 0) ?? 0;
  const totalOverdue = payables?.filter((p) => isOverdue(p.due_date)).reduce((s, p) => s + p.balance, 0) ?? 0;

  async function handleRecordPayment(payload: { amount: number; payment_date: string; payment_method: PaymentMethod; reference: string; notes: string }) {
    if (!paying) return;
    await recordPurchasePayment({ purchase_id: paying.purchase_id, ...payload });
    toast.success(`Pago registrado para ${paying.purchase_number || "la orden"}.`);
    setPaying(null);
    refresh();
  }

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Compras", to: "/purchases" }, { label: "Cuentas por pagar" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Wallet className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Cuentas por pagar
        </h1>
      </div>

      {payables && payables.length > 0 && (
        <div className="mb-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Card className="p-3">
            <p className="text-xs font-medium text-(--text-secondary)">Total por pagar</p>
            <p className="font-mono text-lg font-semibold text-(--text-primary)">{formatCOP.format(totalPending)}</p>
          </Card>
          <Card className="p-3">
            <p className="text-xs font-medium text-(--text-secondary)">Vencido</p>
            <p className="font-mono text-lg font-semibold text-(--color-danger-text)">{formatCOP.format(totalOverdue)}</p>
          </Card>
        </div>
      )}

      {error && <Banner tone="danger">{error}</Banner>}

      {payables === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : payables.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">No tienes cuentas por pagar pendientes — todas las órdenes recibidas están al día.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Orden</th>
                <th className="px-3 py-2 font-medium">Proveedor</th>
                <th className="px-3 py-2 font-medium">Fecha</th>
                <th className="px-3 py-2 font-medium">Vence</th>
                <th className="px-3 py-2 font-medium">Total</th>
                <th className="px-3 py-2 font-medium">Pagado</th>
                <th className="px-3 py-2 font-medium">Saldo</th>
                <th className="px-3 py-2 font-medium" />
              </tr>
            </thead>
            <tbody>
              {payables.map((p, i) => {
                const overdue = isOverdue(p.due_date);
                return (
                  <tr key={p.purchase_id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{p.purchase_number || "—"}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{supplierName(p.supplier_id)}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{formatDateOnly(p.issue_date)}</td>
                    <td className="px-3 py-2">
                      {p.due_date ? (
                        overdue ? (
                          <StatusPill tone="danger" label={formatDateOnly(p.due_date)} />
                        ) : (
                          <span className="text-(--text-secondary)">{formatDateOnly(p.due_date)}</span>
                        )
                      ) : "—"}
                    </td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(p.total)}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(p.paid)}</td>
                    <td className="px-3 py-2 font-mono font-semibold text-(--text-primary)">{formatCOP.format(p.balance)}</td>
                    <td className="px-3 py-2">
                      <button
                        type="button"
                        onClick={() => setPaying(p)}
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
        <RecordPurchasePaymentModal payable={paying} onSubmit={handleRecordPayment} onClose={() => setPaying(null)} />
      )}
    </div>
  );
}

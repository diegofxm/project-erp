import { useState } from "react";
import { X, DollarSign } from "lucide-react";
import { formatCOP } from "../../lib/currency";
import type { PaymentMethod, ReceivableBalance } from "../../lib/types";
import { Button } from "../ui/Button";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { Banner } from "../ui/Banner";

const METHOD_LABEL: Record<PaymentMethod, string> = {
  cash: "Efectivo", transfer: "Transferencia", check: "Cheque", card: "Tarjeta", other: "Otro",
};

interface Props {
  receivable: ReceivableBalance;
  onSubmit: (payload: { amount: number; payment_date: string; payment_method: PaymentMethod; reference: string; notes: string }) => Promise<void>;
  onClose: () => void;
}

const today = () => new Date().toISOString().slice(0, 10);

export function RecordPaymentModal({ receivable, onSubmit, onClose }: Props) {
  const [amount, setAmount] = useState(receivable.balance.toString());
  const [date, setDate] = useState(today());
  const [method, setMethod] = useState<PaymentMethod>("transfer");
  const [reference, setReference] = useState("");
  const [notes, setNotes] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const amountNum = Number(amount || 0);
  const valid = amountNum > 0 && amountNum <= receivable.balance + 0.01;

  async function handleSubmit() {
    if (!valid) return;
    setError(null);
    setSaving(true);
    try {
      await onSubmit({ amount: amountNum, payment_date: date, payment_method: method, reference, notes });
    } catch {
      setError("No se pudo registrar el pago.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div className="w-full max-w-md rounded-lg border border-(--border-color) bg-(--bg-primary) p-5 shadow-xl" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <DollarSign className="h-4 w-4 text-(--accent-primary)" />
            <h2 className="text-sm font-semibold text-(--text-primary)">Registrar pago — {receivable.sale_number || "venta"}</h2>
          </div>
          <button type="button" onClick={onClose} className="text-(--text-muted) hover:text-(--text-primary) transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <p className="mb-3 text-xs text-(--text-secondary)">
          Saldo pendiente: <span className="font-mono font-semibold text-(--text-primary)">{formatCOP.format(receivable.balance)}</span>
        </p>

        {error && <Banner tone="danger">{error}</Banner>}

        <div className="space-y-3">
          <Input
            label="Monto (COP)" type="number" min="0" step="0.01" required
            value={amount} onChange={(e) => setAmount(e.target.value)}
            error={!valid && amount ? "El monto debe ser mayor a cero y no superar el saldo pendiente." : undefined}
          />
          <Input label="Fecha de pago" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          <Select label="Medio de pago" value={method} onChange={(e) => setMethod(e.target.value as PaymentMethod)}>
            {(Object.keys(METHOD_LABEL) as PaymentMethod[]).map((m) => (
              <option key={m} value={m}>{METHOD_LABEL[m]}</option>
            ))}
          </Select>
          <Input label="Referencia (opcional)" value={reference} onChange={(e) => setReference(e.target.value)} placeholder="Nº de consignación, cheque…" />
          <Input label="Notas (opcional)" value={notes} onChange={(e) => setNotes(e.target.value)} />
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose}>Cancelar</Button>
          <Button type="button" disabled={!valid} loading={saving} onClick={handleSubmit}>Registrar pago</Button>
        </div>
      </div>
    </div>
  );
}

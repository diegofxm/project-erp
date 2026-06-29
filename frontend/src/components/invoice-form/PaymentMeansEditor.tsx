import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { listPaymentMethods, listPaymentTerms } from "../../lib/catalogs";
import { useCatalog } from "../../lib/useCatalog";
import type { PaymentMean } from "../../lib/types";
import { Button } from "../ui/Button";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";

interface PaymentMeansEditorProps {
  paymentMeans: PaymentMean[];
  onChange: (paymentMeans: PaymentMean[]) => void;
}

interface Draft {
  code: string;
  paymentMethodCode: string;
  dueDate: string;
  reference: string;
}

const EMPTY_DRAFT: Draft = { code: "", paymentMethodCode: "", dueDate: "", reference: "" };

// cac:PaymentMeans es obligatorio (1..N) para Factura/Notas según el Anexo Técnico — la DIAN
// rechaza con "errores en campos mandatorios" si no se informa ninguno (ver
// docs/apidian-architecture.md sección 9.30). code es la forma de pago (catálogo
// payment_terms, "1" contado/"2" crédito); payment_method_code es el medio de pago (catálogo
// payment_methods). due_date solo tiene sentido si la forma de pago es crédito.
export function PaymentMeansEditor({ paymentMeans, onChange }: PaymentMeansEditorProps) {
  const { data: paymentTerms, loading: loadingPaymentTerms } = useCatalog(listPaymentTerms);
  const { data: paymentMethods, loading: loadingPaymentMethods } = useCatalog(listPaymentMethods);
  const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);

  const isCredit = draft.code === "2";

  function handleAdd() {
    onChange([
      ...paymentMeans,
      {
        code: draft.code,
        payment_method_code: draft.paymentMethodCode,
        due_date: isCredit && draft.dueDate ? draft.dueDate : undefined,
        payment_reference: draft.reference || undefined,
      },
    ]);
    setDraft(EMPTY_DRAFT);
  }

  function handleRemove(index: number) {
    onChange(paymentMeans.filter((_, i) => i !== index));
  }

  const isDuplicate = paymentMeans.some((pm) => pm.code === draft.code && pm.payment_method_code === draft.paymentMethodCode);
  const canAdd = draft.code !== "" && draft.paymentMethodCode !== "" && !isDuplicate;

  return (
    <div className="flex flex-col gap-3">
      {paymentMeans.length > 0 && (
        <div className="overflow-hidden rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Forma de pago</th>
                <th className="px-3 py-2 font-medium">Medio de pago</th>
                <th className="px-3 py-2 font-medium">Vencimiento</th>
                <th className="px-3 py-2 font-medium">Referencia</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {paymentMeans.map((pm, i) => (
                <tr key={i} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 text-(--text-primary)">
                    {paymentTerms.find((t) => t.code === pm.code)?.name ?? pm.code}
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">
                    {paymentMethods.find((m) => m.code === pm.payment_method_code)?.name ?? pm.payment_method_code}
                  </td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{pm.due_date || "—"}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{pm.payment_reference || "—"}</td>
                  <td className="px-3 py-2">
                    <button
                      type="button"
                      title="Quitar"
                      onClick={() => handleRemove(i)}
                      className="rounded p-1 text-(--color-danger) hover:bg-(--bg-hover)"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="rounded border border-(--border-color) bg-(--bg-primary) p-3">
        <div className="grid grid-cols-12 gap-3">
          <div className="col-span-4">
            <Select
              label="Forma de pago"
              required
              disabled={loadingPaymentTerms}
              value={draft.code}
              onChange={(e) => setDraft({ ...draft, code: e.target.value })}
            >
              {loadingPaymentTerms ? (
                <option>Cargando…</option>
              ) : (
                <>
                  <option value="">Selecciona…</option>
                  {paymentTerms.map((t) => (
                    <option key={t.code} value={t.code}>
                      {t.name}
                    </option>
                  ))}
                </>
              )}
            </Select>
          </div>
          <div className="col-span-4">
            <Select
              label="Medio de pago"
              required
              disabled={loadingPaymentMethods}
              value={draft.paymentMethodCode}
              onChange={(e) => setDraft({ ...draft, paymentMethodCode: e.target.value })}
            >
              {loadingPaymentMethods ? (
                <option>Cargando…</option>
              ) : (
                <>
                  <option value="">Selecciona…</option>
                  {paymentMethods.map((m) => (
                    <option key={m.code} value={m.code}>
                      {m.name}
                    </option>
                  ))}
                </>
              )}
            </Select>
          </div>
          {isCredit && (
            <div className="col-span-2">
              <Input label="Vencimiento" type="date" value={draft.dueDate} onChange={(e) => setDraft({ ...draft, dueDate: e.target.value })} />
            </div>
          )}
          <div className={isCredit ? "col-span-2" : "col-span-4"}>
            <Input label="Referencia (opcional)" value={draft.reference} onChange={(e) => setDraft({ ...draft, reference: e.target.value })} />
          </div>
          <div className="col-span-12 flex items-center gap-2">
            <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} disabled={!canAdd} onClick={handleAdd}>
              Agregar forma de pago
            </Button>
            {isDuplicate && <span className="text-xs text-(--text-muted)">Esa combinación ya está agregada.</span>}
          </div>
        </div>
      </div>
    </div>
  );
}
